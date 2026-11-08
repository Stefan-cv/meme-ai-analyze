package risk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-redis/redis/v8"
	"golang.org/x/time/rate"
)

// Scanner fetches all on-chain data for a given token in parallel.
type Scanner struct {
	client       *ethclient.Client
	redis        *redis.Client
	historian    *Historian
	chainID      int64
	bscScanKey   string
	honeypot     *HoneypotDetector
	progressCh   chan<- ScanProgress
	rpcLimiter   *rate.Limiter              // rate limits RPC calls
	fixedCache   map[string]fixedCacheEntry // cache for fixed/expensive data
	fixedCacheMu sync.RWMutex
}

type fixedCacheEntry struct {
	value   interface{}
	expires time.Time
}

// ScanProgress represents a progress update during a scan.
type ScanProgress struct {
	TokenAddress string  `json:"token_address"`
	Step         string  `json:"step"`
	Progress     float64 `json:"progress"`
	TotalSteps   int     `json:"total_steps"`
}

// TokenRiskProfile holds all raw data gathered for a token across all six dimensions.
type TokenRiskProfile struct {
	Address             string
	TokenName           string
	TokenSymbol         string
	TokenDecimals       uint8
	ContractSafety      *ContractSafetyData
	LPStatus            *LPStatusData
	HolderConcentration *HolderData
	CreatorProfile      *CreatorRecord
	Liquidity           *LiquidityData
	SocialSignals       *SocialSignalData
}

// ---- Dimension Sub-Types ----

type ContractSafetyData struct {
	IsHoneypot        bool     `json:"is_honeypot"`
	HasMint           bool     `json:"has_mint_function"`
	HasBlacklist      bool     `json:"has_blacklist"`
	HasPause          bool     `json:"has_pause"`
	HasHighTax        bool     `json:"has_high_tax"`
	HasSetMaxTx       bool     `json:"has_set_max_tx"`
	OwnerPrivileges   []string `json:"owner_privileges"`
	ContractVerified  bool     `json:"contract_verified"`
	IsProxy           bool     `json:"is_proxy"`
}

type LPStatusData struct {
	Found          bool    `json:"found"`
	Locked         bool    `json:"locked"`
	LockRemaining  int64   `json:"lock_remaining_seconds"`
	TokenAmount    float64 `json:"lp_token_amount"`
	PairAddress    string  `json:"pair_address"`
	LockerName     string  `json:"locker_name"`
}

type HolderData struct {
	TotalHolders  int     `json:"total_holders"`
	Top3Percent   float64 `json:"top3_percentage"`
	Top10Percent  float64 `json:"top10_percentage"`
}

type LiquidityData struct {
	TotalLiquidityUSD float64 `json:"total_liquidity_usd"`
	TokenReserve      float64 `json:"token_reserve"`
	BNBReserve        float64 `json:"bnb_reserve"`
	DailyVolumeUSD    float64 `json:"daily_volume_usd"`
	BNBPriceUSD       float64 `json:"bnb_price_usd"`
}

type SocialSignalData struct {
	HasWebsite      bool    `json:"has_website"`
	TwitterAgeDays  int     `json:"twitter_age_days"`
	TelegramMembers int     `json:"telegram_members"`
	SentimentScore  float64 `json:"sentiment_score"`
}

// ============================================================
// ABI Definitions
// ============================================================

// minimal ERC20 ABI for name/symbol/decimals
var erc20ABI, _ = abi.JSON(strings.NewReader(`[
	{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"},
	{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"},
	{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"},
	{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},
	{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"type":"function"}
]`))

// Minimal PancakeSwap Pair ABI for getReserves + token0/token1
var pairABI, _ = abi.JSON(strings.NewReader(`[
	{"constant":true,"inputs":[],"name":"getReserves","outputs":[{"name":"_reserve0","type":"uint112"},{"name":"_reserve1","type":"uint112"},{"name":"_blockTimestampLast","type":"uint32"}],"type":"function"},
	{"constant":true,"inputs":[],"name":"token0","outputs":[{"name":"","type":"address"}],"type":"function"},
	{"constant":true,"inputs":[],"name":"token1","outputs":[{"name":"","type":"address"}],"type":"function"},
	{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"type":"function"}
]`))

// Minimal PancakeSwap Factory ABI for getPair
var factoryABI, _ = abi.JSON(strings.NewReader(`[
	{"constant":true,"inputs":[{"name":"tokenA","type":"address"},{"name":"tokenB","type":"address"}],"name":"getPair","outputs":[{"name":"pair","type":"address"}],"type":"function"}
]`))

// Minimal LP Locker ABI (Unicrypt/Mudra style)
var lockerABI, _ = abi.JSON(strings.NewReader(`[
	{"constant":true,"inputs":[{"name":"","type":"address"},{"name":"","type":"address"},{"name":"","type":"uint256"}],"name":"lockedLiquidity","outputs":[{"name":"amount","type":"uint256"},{"name":"unlockTime","type":"uint256"}],"type":"function"},
	{"constant":true,"inputs":[{"name":"lpToken","type":"address"},{"name":"user","type":"address"}],"name":"getLockInfo","outputs":[{"name":"amount","type":"uint256"},{"name":"unlockTime","type":"uint256"},{"name":"owner","type":"address"}],"type":"function"}
]`))

// Commonly used addresses on BSC
var (
	pancakeSwapFactory   = common.HexToAddress("0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73")
	pancakeSwapRouter    = common.HexToAddress("0x10ED43C718714eb63d5aA57B78B54704E256024E")
	wbnbAddress          = common.HexToAddress("0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c")
	usdtAddress          = common.HexToAddress("0x55d398326f99059fF775485246999027B3197955")
	busdAddress          = common.HexToAddress("0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56")

	// Common LP locker contracts on BSC
	unicryptLocker = common.HexToAddress("0xDba68f07d1b7Ca219f78aE8582C213337bAA2F4c")
	mudraLocker    = common.HexToAddress("0x39C0Fb0A5Ea8359E0E2D7e7a26e0E4e0B4308471")

	// Dangerous function selectors (4-byte)
	dangerousSelectors = map[string]string{
		"0x40c10f19": "mint",            // mint(address,uint256)
		"0x79cc6790": "burn",            // burn(address,uint256) - owner burn
		"0x42966c68": "burn_self",       // burn(uint256)
		"0x8456cb59": "pause",           // pause()
		"0x3f4ba83a": "unpause",         // unpause()
		"0xfcb0bbd9": "setMaxTxAmount",  // setMaxTxAmount(uint256)
		"0x6833c097": "setMaxTxPercent", // setMaxTxPercent(uint256)
		"0x2045ad49": "excludeFromFee",  // excludeFromFee(address)
		"0xbf56e42c": "includeInFee",    // includeInFee(address)
		"0xe2f45689": "setSwapAndLiquifyEnabled", // setSwapAndLiquifyEnabled(bool)
	}
)

// ============================================================
// Constructor
// ============================================================

// NewScanner creates a new Scanner with retryable RPC connection.
func NewScanner(rpcURL string, backupRPC string, redisClient *redis.Client, historian *Historian, bscScanKey string) *Scanner {
	client := dialWithRetry(rpcURL, backupRPC)
	return &Scanner{
		client:     client,
		redis:      redisClient,
		historian:  historian,
		chainID:    56,
		bscScanKey: bscScanKey,
		rpcLimiter: rate.NewLimiter(rate.Limit(10), 3), // 10 req/s, burst 3
		fixedCache: make(map[string]fixedCacheEntry),
	}
}

// waitRPC blocks until the rate limiter allows an RPC call. Public RPC nodes
// typically allow 10-20 req/s; we stay conservative at 10/s to avoid bans.
func (s *Scanner) waitRPC(ctx context.Context) error {
	return s.rpcLimiter.Wait(ctx)
}

// cachedOrFetch checks the in-memory fixed cache before making an expensive call.
func (s *Scanner) cachedOrFetch(key string, ttl time.Duration, fetch func() (interface{}, error)) (interface{}, error) {
	s.fixedCacheMu.RLock()
	entry, ok := s.fixedCache[key]
	s.fixedCacheMu.RUnlock()
	if ok && time.Now().Before(entry.expires) {
		return entry.value, nil
	}

	val, err := fetch()
	if err != nil {
		return nil, err
	}

	s.fixedCacheMu.Lock()
	s.fixedCache[key] = fixedCacheEntry{value: val, expires: time.Now().Add(ttl)}
	s.fixedCacheMu.Unlock()
	return val, nil
}

// GetClient returns the underlying ethclient (for sharing with other services).
func (s *Scanner) GetClient() *ethclient.Client {
	return s.client
}

// SetHoneypot sets the honeypot detector.
func (s *Scanner) SetHoneypot(h *HoneypotDetector) {
	s.honeypot = h
}

// SetProgressChannel sets the WebSocket progress channel.
func (s *Scanner) SetProgressChannel(ch chan<- ScanProgress) {
	s.progressCh = ch
}

func (s *Scanner) sendProgress(tokenAddr, step string, progress float64) {
	if s.progressCh == nil {
		return
	}
	select {
	case s.progressCh <- ScanProgress{TokenAddress: tokenAddr, Step: step, Progress: progress, TotalSteps: 8}:
	default:
	}
}

// dialWithRetry attempts to connect to primary RPC, falling back to backup.
func dialWithRetry(primary, backup string) *ethclient.Client {
	urls := []string{primary}
	if backup != "" {
		urls = append(urls, backup)
	}

	for _, url := range urls {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		client, err := ethclient.DialContext(ctx, url)
		cancel()
		if err == nil {
			log.Printf("[Scanner] Connected to RPC: %s", url)
			return client
		}
		log.Printf("[Scanner] Failed to connect to %s: %v", url, err)
	}

	log.Printf("[Scanner] WARNING: all RPC endpoints unreachable, running in offline mode")
	return nil
}

// ============================================================
// Main Scan Entry Point
// ============================================================

// Scan performs a full parallel scan of all six risk dimensions.
func (s *Scanner) Scan(ctx context.Context, tokenAddr string) (*TokenRiskProfile, error) {
	addr := common.HexToAddress(tokenAddr)
	profile := &TokenRiskProfile{Address: tokenAddr}

	// Step 0: Fill basic token info first
	s.sendProgress(tokenAddr, "正在读取合约信息...", 0.1)
	s.fillTokenInfo(ctx, addr, profile)

	var wg sync.WaitGroup
	errCh := make(chan error, 6)

	// Dimension 1: Contract Safety
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := s.analyzeContractSafety(ctx, addr)
		if err != nil {
			errCh <- fmt.Errorf("contract_safety: %w", err)
			profile.ContractSafety = &ContractSafetyData{}
		} else {
			profile.ContractSafety = data
		}
	}()

	// Dimension 2: LP Status
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := s.analyzeLPStatus(ctx, addr)
		if err != nil {
			errCh <- fmt.Errorf("lp_status: %w", err)
			profile.LPStatus = &LPStatusData{}
		} else {
			profile.LPStatus = data
		}
	}()

	// Dimension 3: Holder Concentration
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := s.analyzeHolderConcentration(ctx, addr)
		if err != nil {
			errCh <- fmt.Errorf("holder_concentration: %w", err)
			profile.HolderConcentration = &HolderData{}
		} else {
			profile.HolderConcentration = data
		}
	}()

	// Dimension 4: Creator History
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := s.historian.TraceCreator(ctx, addr)
		if err != nil {
			errCh <- fmt.Errorf("creator_history: %w", err)
			profile.CreatorProfile = &CreatorRecord{}
		} else {
			profile.CreatorProfile = data
		}
	}()

	// Dimension 5: Liquidity
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := s.analyzeLiquidity(ctx, addr)
		if err != nil {
			errCh <- fmt.Errorf("liquidity: %w", err)
			profile.Liquidity = &LiquidityData{}
		} else {
			profile.Liquidity = data
		}
	}()

	// Dimension 6: Social Signals
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, err := s.analyzeSocialSignals(ctx, addr)
		if err != nil {
			errCh <- fmt.Errorf("social_signals: %w", err)
			profile.SocialSignals = &SocialSignalData{}
		} else {
			profile.SocialSignals = data
		}
	}()

	wg.Wait()
	close(errCh)

	for err := range errCh {
		log.Printf("[Scanner] Non-fatal error: %v", err)
	}

	return profile, nil
}

// ============================================================
// Token Info (ERC20 name/symbol/decimals)
// ============================================================

func (s *Scanner) fillTokenInfo(ctx context.Context, addr common.Address, profile *TokenRiskProfile) {
	if s.client == nil {
		profile.TokenName = "Unknown"
		profile.TokenSymbol = "???"
		profile.TokenDecimals = 18
		return
	}

	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// name()
	if result, err := s.callContract(rpcCtx, addr, "name"); err == nil && len(result) > 0 {
		if name, err2 := abiDecodeString(result); err2 == nil {
			profile.TokenName = name
		}
	}
	if profile.TokenName == "" {
		profile.TokenName = "Unknown"
	}

	// symbol()
	if result, err := s.callContract(rpcCtx, addr, "symbol"); err == nil && len(result) > 0 {
		if sym, err2 := abiDecodeString(result); err2 == nil {
			profile.TokenSymbol = sym
		}
	}
	if profile.TokenSymbol == "" {
		profile.TokenSymbol = "???"
	}

	// decimals()
	if result, err := s.callContract(rpcCtx, addr, "decimals"); err == nil && len(result) >= 32 {
		profile.TokenDecimals = uint8(new(big.Int).SetBytes(result).Uint64())
	} else {
		profile.TokenDecimals = 18
	}

	log.Printf("[Scanner] Token: %s ($%s) decimals=%d", profile.TokenName, profile.TokenSymbol, profile.TokenDecimals)
}

// ============================================================
// Dimension 1: Contract Safety
// ============================================================

func (s *Scanner) analyzeContractSafety(ctx context.Context, addr common.Address) (*ContractSafetyData, error) {
	data := &ContractSafetyData{ContractVerified: false}

	if s.client == nil {
		return data, nil
	}

	rpcCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	// Get contract bytecode
	code, err := s.client.CodeAt(rpcCtx, addr, nil)
	if err != nil {
		return data, fmt.Errorf("get code: %w", err)
	}

	if len(code) == 0 {
		return data, nil // EOA, not a contract
	}

	// Convert bytecode to hex string for pattern matching
	codeHex := common.Bytes2Hex(code)

	// Scan for dangerous function selectors
	ownerPrivileges := []string{}
	for selector, name := range dangerousSelectors {
		if strings.Contains(codeHex, strings.TrimPrefix(selector, "0x")) {
			switch {
			case name == "mint":
				data.HasMint = true
				ownerPrivileges = append(ownerPrivileges, "可增发 (mint)")
			case strings.HasPrefix(name, "burn"):
				data.HasMint = true // burn from arbitrary address = owner privilege
				ownerPrivileges = append(ownerPrivileges, "可销毁他人代币 (burn)")
			case name == "pause":
				data.HasPause = true
				ownerPrivileges = append(ownerPrivileges, "可暂停交易 (pause)")
			case name == "unpause":
				data.HasPause = true
			case strings.HasPrefix(name, "setMaxTx"):
				data.HasSetMaxTx = true
				ownerPrivileges = append(ownerPrivileges, "可限制单笔交易额 (setMaxTx)")
			case strings.HasPrefix(name, "excludeFromFee") || strings.HasPrefix(name, "includeInFee"):
				ownerPrivileges = append(ownerPrivileges, "可设置手续费白名单")
			case strings.Contains(name, "SwapAndLiquify"):
				ownerPrivileges = append(ownerPrivileges, "可控制交易开关")
			}
		}
	}

	// Proxy detection: check for common proxy patterns
	if strings.Contains(codeHex, "360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc") {
		data.IsProxy = true
	}

	// Honeypot detection: use the HoneypotDetector for real simulation
	if s.honeypot != nil {
		// Swap simulation is complex and can have false positives.
		// Only flag as honeypot if transfer tax check is definitive.
		if highTax, taxReason := s.honeypot.CheckTransferTax(ctx, addr); highTax {
			data.HasHighTax = true
			ownerPrivileges = append(ownerPrivileges, taxReason)
			// If tax is 100% (zero received), it's effectively a honeypot
			if strings.Contains(taxReason, "100%") || strings.Contains(taxReason, "无法收到") {
				data.IsHoneypot = true
				ownerPrivileges = append(ownerPrivileges, "蜜罐: 卖出时完全无法收到代币")
			}
		}
	}

	data.OwnerPrivileges = ownerPrivileges
	data.ContractVerified = true
	return data, nil
}

// detectHoneypot simulates a sell to check if the token is a honeypot.
func (s *Scanner) detectHoneypot(ctx context.Context, tokenAddr common.Address) bool {
	if s.client == nil {
		return false
	}

	// Simple check: try to call transfer with a small amount
	// If the token has a maxTxAmount=0 or trading disabled, the call reverts
	pairAddr := s.getPairAddress(ctx, tokenAddr)
	if pairAddr == (common.Address{}) {
		return false // no pair = not necessarily honeypot
	}

	// Simulate: transfer 1 token from pair to dead address (would fail if sell restricted)
	deadAddr := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	callData, err := erc20ABI.Pack("transfer", deadAddr, big.NewInt(1))
	if err != nil {
		return false
	}

	msg := ethereum.CallMsg{
		To:   &tokenAddr,
		Data: callData,
		From: pairAddr, // simulate as if pair is calling (like a swap)
	}

	rpcCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err = s.client.CallContract(rpcCtx, msg, nil)
	if err != nil {
		log.Printf("[Scanner] Honeypot check: transfer from pair reverts → possible honeypot: %v", err)
		return true
	}
	return false
}

// ============================================================
// Dimension 2: LP Status (PancakeSwap + Lockers)
// ============================================================

func (s *Scanner) analyzeLPStatus(ctx context.Context, tokenAddr common.Address) (*LPStatusData, error) {
	data := &LPStatusData{}

	if s.client == nil {
		return data, nil
	}

	pairAddr := s.getPairAddress(ctx, tokenAddr)
	if pairAddr == (common.Address{}) {
		return data, nil // no LP pool found
	}

	data.Found = true
	data.PairAddress = pairAddr.Hex()

	// Check if LP tokens are locked
	s.checkLPLock(ctx, pairAddr, data)

	return data, nil
}

// getPairAddress queries PancakeSwap Factory for the WBNB pair.
func (s *Scanner) getPairAddress(ctx context.Context, tokenAddr common.Address) common.Address {
	callData, err := factoryABI.Pack("getPair", tokenAddr, wbnbAddress)
	if err != nil {
		return common.Address{}
	}

	msg := ethereum.CallMsg{To: &pancakeSwapFactory, Data: callData}
	rpcCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	result, err := s.client.CallContract(rpcCtx, msg, nil)
	if err != nil || len(result) < 32 {
		return common.Address{}
	}

	return common.BytesToAddress(result)
}

// checkLPLock queries locker contracts for LP token lock status.
func (s *Scanner) checkLPLock(ctx context.Context, pairAddr common.Address, data *LPStatusData) {
	lockers := []struct {
		addr common.Address
		name string
	}{
		{unicryptLocker, "Unicrypt"},
		{mudraLocker, "Mudra"},
	}

	rpcCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	for _, locker := range lockers {
		callData, err := lockerABI.Pack("getLockInfo", pairAddr, common.Address{})
		if err != nil {
			continue
		}

		msg := ethereum.CallMsg{To: &locker.addr, Data: callData}
		result, err := s.client.CallContract(rpcCtx, msg, nil)
		if err != nil || len(result) < 64 {
			continue
		}

		// Parse: (uint256 amount, uint256 unlockTime, address owner)
		values, err := lockerABI.Unpack("getLockInfo", result)
		if err != nil || len(values) < 2 {
			continue
		}

		amount := values[0].(*big.Int)
		unlockTime := values[1].(*big.Int)

		if amount.Cmp(big.NewInt(0)) > 0 {
			data.Locked = true
			data.LockerName = locker.name
			now := time.Now().Unix()
			remaining := unlockTime.Int64() - now
			if remaining > 0 {
				data.LockRemaining = remaining
			} else {
				data.LockRemaining = 0 // expired lock
			}
			return
		}
	}
}

// ============================================================
// Dimension 3: Holder Concentration
// ============================================================

func (s *Scanner) analyzeHolderConcentration(ctx context.Context, tokenAddr common.Address) (*HolderData, error) {
	// Try BscScan API first for accurate holder data
	if s.bscScanKey != "" {
		data, err := s.fetchTopHoldersFromBscScan(ctx, tokenAddr)
		if err == nil && data.TotalHolders > 0 {
			log.Printf("[Scanner] Holders (BscScan): total=%d, top3=%.1f%%, top10=%.1f%%", data.TotalHolders, data.Top3Percent, data.Top10Percent)
			return data, nil
		}
		log.Printf("[Scanner] BscScan holder fetch failed: %v, falling back to RPC", err)
	}

	// Fallback: RPC-based holder detection
	return s.analyzeHoldersRPC(ctx, tokenAddr)
}

// fetchTopHoldersFromBscScan queries BscScan API for token holders.
func (s *Scanner) fetchTopHoldersFromBscScan(ctx context.Context, tokenAddr common.Address) (*HolderData, error) {
	url := fmt.Sprintf(
		"https://api.bscscan.com/api?module=token&action=tokenholderlist&contractaddress=%s&page=1&offset=10&apikey=%s",
		tokenAddr.Hex(), s.bscScanKey,
	)

	reqCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bscscan request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  []struct {
			TokenHolderAddress  string `json:"TokenHolderAddress"`
			TokenHolderQuantity string `json:"TokenHolderQuantity"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("bscscan parse: %w", err)
	}

	if result.Status != "1" || len(result.Result) == 0 {
		return nil, fmt.Errorf("bscscan: %s", result.Message)
	}

	// Get total supply for percentage calculation
	totalSupply := s.getTotalSupply(ctx, tokenAddr)
	if totalSupply == nil || totalSupply.Cmp(big.NewInt(0)) == 0 {
		return nil, fmt.Errorf("no total supply")
	}

	data := &HolderData{TotalHolders: len(result.Result)}
	totalSupFloat := new(big.Float).SetInt(totalSupply)

	var top3Sum, top10Sum = new(big.Int), new(big.Int)
	for i, h := range result.Result {
		bal := new(big.Int)
		bal.SetString(h.TokenHolderQuantity, 10)

		if i < 3 {
			top3Sum.Add(top3Sum, bal)
		}
		if i < 10 {
			top10Sum.Add(top10Sum, bal)
		}
	}

	if top3Sum.Cmp(big.NewInt(0)) > 0 {
		ratio := new(big.Float).Quo(new(big.Float).SetInt(top3Sum), totalSupFloat)
		data.Top3Percent, _ = ratio.Float64()
		data.Top3Percent *= 100
	}
	if top10Sum.Cmp(big.NewInt(0)) > 0 {
		ratio := new(big.Float).Quo(new(big.Float).SetInt(top10Sum), totalSupFloat)
		data.Top10Percent, _ = ratio.Float64()
		data.Top10Percent *= 100
	}

	return data, nil
}

// analyzeHoldersRPC is the fallback RPC-based holder analysis.
func (s *Scanner) analyzeHoldersRPC(ctx context.Context, tokenAddr common.Address) (*HolderData, error) {
	data := &HolderData{}

	if s.client == nil {
		return data, nil
	}

	totalSupply := s.getTotalSupply(ctx, tokenAddr)
	if totalSupply == nil || totalSupply.Cmp(big.NewInt(0)) == 0 {
		return data, nil
	}

	// Collect balances from key addresses:
	// 1. PancakeSwap pair (usually the largest holder)
	// 2. Burn address (0xdead)
	// 3. Deployer address (from historian)
	// 4. Null address
	var topBalances []*big.Int

	pairAddr := s.getPairAddress(ctx, tokenAddr)
	burnAddr := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	nullAddr := common.HexToAddress("0x0000000000000000000000000000000000000000")

	addrs := []common.Address{burnAddr, nullAddr}
	if pairAddr != (common.Address{}) {
		addrs = append(addrs, pairAddr)
	}

	// Also try to get deployer from historian
	if s.historian != nil && s.historian.client != nil {
		creator, err := s.historian.GetDeployer(ctx, tokenAddr)
		if err == nil && creator != (common.Address{}) {
			addrs = append(addrs, creator)
		}
	}

	for _, addr := range addrs {
		bal := s.getBalance(ctx, tokenAddr, addr)
		if bal != nil && bal.Cmp(big.NewInt(0)) > 0 {
			topBalances = append(topBalances, bal)
			data.TotalHolders++
		}
	}

	if len(topBalances) == 0 {
		return data, nil
	}

	// Calculate top 3 / top 10 percentages
	totalSupFloat := new(big.Float).SetInt(totalSupply)
	var totalTop3, totalTop10 *big.Int

	for i, bal := range topBalances {
		if i < 3 {
			if totalTop3 == nil {
				totalTop3 = new(big.Int).Set(bal)
			} else {
				totalTop3.Add(totalTop3, bal)
			}
		}
		if totalTop10 == nil {
			totalTop10 = new(big.Int).Set(bal)
		} else {
			totalTop10.Add(totalTop10, bal)
		}
	}

	if totalTop3 != nil && totalSupply.Cmp(big.NewInt(0)) > 0 {
		top3Float := new(big.Float).SetInt(totalTop3)
		ratio := new(big.Float).Quo(top3Float, totalSupFloat)
		data.Top3Percent, _ = ratio.Float64()
		data.Top3Percent *= 100
	}

	if totalTop10 != nil && totalSupply.Cmp(big.NewInt(0)) > 0 {
		top10Float := new(big.Float).SetInt(totalTop10)
		ratio := new(big.Float).Quo(top10Float, totalSupFloat)
		data.Top10Percent, _ = ratio.Float64()
		data.Top10Percent *= 100
	}

	log.Printf("[Scanner] Holders (RPC): found=%d, top3=%.1f%%, top10=%.1f%%", data.TotalHolders, data.Top3Percent, data.Top10Percent)
	return data, nil
}

// getKeyHolderAddresses returns addresses likely to hold significant amounts.
func (s *Scanner) getKeyHolderAddresses(tokenAddr common.Address) []common.Address {
	// Common addresses to check: burn address, PancakeSwap pair, null address
	pairAddr := s.getPairAddress(context.Background(), tokenAddr)

	addresses := []common.Address{
		common.HexToAddress("0x000000000000000000000000000000000000dEaD"), // burn
		common.HexToAddress("0x0000000000000000000000000000000000000000"), // null
	}
	if pairAddr != (common.Address{}) {
		addresses = append(addresses, pairAddr) // PancakeSwap pair (usually largest holder)
	}
	return addresses
}

// getTotalSupply returns the token's total supply via ERC20 call.
func (s *Scanner) getTotalSupply(ctx context.Context, tokenAddr common.Address) *big.Int {
	rpcCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	result, err := s.callContract(rpcCtx, tokenAddr, "totalSupply")
	if err != nil || len(result) < 32 {
		return nil
	}
	return new(big.Int).SetBytes(result)
}

// getBalance returns an address' token balance via ERC20 balanceOf.
func (s *Scanner) getBalance(ctx context.Context, tokenAddr, holderAddr common.Address) *big.Int {
	callData, err := erc20ABI.Pack("balanceOf", holderAddr)
	if err != nil {
		return nil
	}

	rpcCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	msg := ethereum.CallMsg{To: &tokenAddr, Data: callData}
	result, err := s.client.CallContract(rpcCtx, msg, nil)
	if err != nil || len(result) < 32 {
		return nil
	}
	return new(big.Int).SetBytes(result)
}

// ============================================================
// Dimension 5: Liquidity Analysis
// ============================================================

func (s *Scanner) analyzeLiquidity(ctx context.Context, tokenAddr common.Address) (*LiquidityData, error) {
	data := &LiquidityData{}

	if s.client == nil {
		return data, nil
	}

	pairAddr := s.getPairAddress(ctx, tokenAddr)
	if pairAddr == (common.Address{}) {
		return data, nil
	}

	// Get pair reserves
	reserve0, reserve1, err := s.getPairReserves(ctx, pairAddr)
	if err != nil {
		return data, fmt.Errorf("get reserves: %w", err)
	}

	// Determine which reserve is the token vs WBNB
	token0, err := s.getToken0(ctx, pairAddr)
	if err != nil {
		return data, err
	}

	decimals := s.getTokenDecimals(ctx, tokenAddr)
	bnbPrice := s.getBNBPrice(ctx)

	// Convert reserves to human-readable values
	r0Float := tokenToFloat(reserve0, 18) // assume WBNB/BUSD has 18 decimals
	r1Float := tokenToFloat(reserve1, decimals)

	if token0 == wbnbAddress || token0 == busdAddress {
		data.BNBReserve = r0Float
		data.TokenReserve = r1Float
	} else {
		data.TokenReserve = r0Float
		data.BNBReserve = r1Float
	}

	data.BNBPriceUSD = bnbPrice
	data.TotalLiquidityUSD = data.BNBReserve * bnbPrice * 2 // both sides ~equal in AMM

	log.Printf("[Scanner] Liquidity: $%.2f (BNB: %.4f, Token: %.4f)", data.TotalLiquidityUSD, data.BNBReserve, data.TokenReserve)
	return data, nil
}

// getPairReserves returns the reserves from a PancakeSwap pair.
func (s *Scanner) getPairReserves(ctx context.Context, pairAddr common.Address) (*big.Int, *big.Int, error) {
	callData, err := pairABI.Pack("getReserves")
	if err != nil {
		return nil, nil, err
	}

	rpcCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	msg := ethereum.CallMsg{To: &pairAddr, Data: callData}
	result, err := s.client.CallContract(rpcCtx, msg, nil)
	if err != nil {
		return nil, nil, err
	}

	values, err := pairABI.Unpack("getReserves", result)
	if err != nil {
		return nil, nil, err
	}

	return values[0].(*big.Int), values[1].(*big.Int), nil
}

// getToken0 returns token0 of a pair.
func (s *Scanner) getToken0(ctx context.Context, pairAddr common.Address) (common.Address, error) {
	callData, err := pairABI.Pack("token0")
	if err != nil {
		return common.Address{}, err
	}

	rpcCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	msg := ethereum.CallMsg{To: &pairAddr, Data: callData}
	result, err := s.client.CallContract(rpcCtx, msg, nil)
	if err != nil || len(result) < 32 {
		return common.Address{}, err
	}
	return common.BytesToAddress(result), nil
}

// getBNBPrice gets current BNB price from USDT-WBNB pair (cached 5 min).
func (s *Scanner) getBNBPrice(ctx context.Context) float64 {
	val, err := s.cachedOrFetch("bnb_price", 5*time.Minute, func() (interface{}, error) {
		usdtWbnbPair := s.getPairAddress(ctx, usdtAddress)
		if usdtWbnbPair == (common.Address{}) {
			return 600.0, nil
		}
		r0, r1, err := s.getPairReserves(ctx, usdtWbnbPair)
		if err != nil {
			return 600.0, nil
		}
		token0, _ := s.getToken0(ctx, usdtWbnbPair)
		r0Float := tokenToFloat(r0, 18)
		r1Float := tokenToFloat(r1, 18)
		if token0 == wbnbAddress && r0Float > 0 {
			return r1Float / r0Float, nil
		}
		if r1Float > 0 {
			return r0Float / r1Float, nil
		}
		return 600.0, nil
	})
	if err != nil {
		return 600.0
	}
	return val.(float64)
}

// ============================================================
// Dimension 6: Social Signals
// ============================================================

func (s *Scanner) analyzeSocialSignals(_ context.Context, _ common.Address) (*SocialSignalData, error) {
	// Social signals typically come from off-chain data (BscScan API, Twitter, etc.)
	// For now, return default neutral scores.
	// In production: query BscScan token info for website/social links,
	// check Twitter account age, Telegram member count.
	return &SocialSignalData{
		HasWebsite:      false,
		TwitterAgeDays:  0,
		TelegramMembers: 0,
		SentimentScore:  0,
	}, nil
}

// ============================================================
// Helpers
// ============================================================

// callContract makes an eth_call for a named ERC20 method (no arguments).
func (s *Scanner) callContract(ctx context.Context, tokenAddr common.Address, method string) ([]byte, error) {
	if err := s.waitRPC(ctx); err != nil {
		return nil, err
	}
	callData, err := erc20ABI.Pack(method)
	if err != nil {
		return nil, err
	}
	msg := ethereum.CallMsg{To: &tokenAddr, Data: callData}
	return s.client.CallContract(ctx, msg, nil)
}

// getTokenDecimals returns the token's decimals via ERC20 call.
func (s *Scanner) getTokenDecimals(ctx context.Context, tokenAddr common.Address) uint8 {
	result, err := s.callContract(ctx, tokenAddr, "decimals")
	if err != nil || len(result) < 32 {
		return 18
	}
	return uint8(new(big.Int).SetBytes(result).Uint64())
}

// abiDecodeString decodes an ABI-encoded string from call result.
func abiDecodeString(data []byte) (string, error) {
	if len(data) < 64 {
		return "", fmt.Errorf("data too short")
	}
	// ABI-encoded string: first 32 bytes = offset, next 32 = length, then data
	strLen := new(big.Int).SetBytes(data[32:64]).Uint64()
	if uint64(len(data)) < 64+strLen {
		return "", fmt.Errorf("data truncated")
	}
	return string(data[64 : 64+strLen]), nil
}

// tokenToFloat converts a raw token amount to a float using decimals.
func tokenToFloat(amount *big.Int, decimals uint8) float64 {
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	amountFloat := new(big.Float).SetInt(amount)
	result := new(big.Float).Quo(amountFloat, divisor)
	f, _ := result.Float64()
	return f
}
