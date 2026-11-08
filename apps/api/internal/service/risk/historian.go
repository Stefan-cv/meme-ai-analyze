package risk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Historian tracks the founder/creator history of meme tokens.
// This is the "soul module" of the project — the only engine tracking
// meme token creator behavior history across all tokens.
type Historian struct {
	db     *pgxpool.Pool
	client *ethclient.Client
	apiKey string
}

// CreatorRecord represents a founder's profile from the database.
type CreatorRecord struct {
	Address          string    `db:"address"`
	TotalTokens      int       `db:"total_tokens"`
	RuggedTokens     int       `db:"rugged_tokens"`
	AvgLifetimeHours float64   `db:"avg_lifetime_hours"`
	FirstSeenAt      time.Time `db:"first_seen_at"`
	LastSeenAt       time.Time `db:"last_seen_at"`
	RiskLabel        string    `db:"risk_label"`
	RugRate          float64   // computed: ruggedTokens / totalTokens
	RecentTokens     []TokenRecord
}

// TokenRecord represents a token created by a founder.
type TokenRecord struct {
	Address   string    `db:"token_address"`
	Name      string    `db:"token_name"`
	Symbol    string    `db:"token_symbol"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	RuggedAt  *time.Time `db:"rugged_at"`
}

// NewHistorian creates a new Historian instance.
func NewHistorian(pool *pgxpool.Pool, apiKey string) *Historian {
	return &Historian{db: pool, apiKey: apiKey}
}

// SetClient sets an optional ethclient for on-chain deployer lookups.
func (h *Historian) SetClient(client *ethclient.Client) {
	h.client = client
}

// TraceCreator traces the creator of a given token and returns their profile.
func (h *Historian) TraceCreator(ctx context.Context, tokenAddr common.Address) (*CreatorRecord, error) {
	// Step 1: Find the deployer address for this token
	creator, err := h.GetDeployer(ctx, tokenAddr)
	if err != nil {
		// If we can't find the deployer, return a minimal profile
		return &CreatorRecord{
			Address:  tokenAddr.Hex(),
			RiskLabel: "⚠️ 无法确定创建者",
		}, nil
	}

	// Step 2: Look up the creator record from the database
	record, err := h.getCreatorRecord(ctx, creator)
	if err != nil {
		// New address, no history
		return &CreatorRecord{
			Address:     creator.Hex(),
			TotalTokens: 0,
			RiskLabel:   "✅ 新地址，无不良记录",
		}, nil
	}

	// Step 3: Compute rug rate and classify
	if record.TotalTokens > 0 {
		record.RugRate = float64(record.RuggedTokens) / float64(record.TotalTokens)
	}

	// Step 4: Get recent tokens
	record.RecentTokens, _ = h.getRecentTokens(ctx, creator, 20)

	// Step 5: Generate risk label
	record.RiskLabel = h.classifyCreator(record.TotalTokens, record.RugRate)

	return record, nil
}

// GetDeployer finds the original deployer of a token contract (public for scanner use).
func (h *Historian) GetDeployer(ctx context.Context, tokenAddr common.Address) (common.Address, error) {
	// Step 1: Query the database first for a cached deployer
	var deployerHex string
	err := h.db.QueryRow(ctx,
		`SELECT creator_address FROM creator_tokens WHERE token_address = $1`,
		tokenAddr.Hex(),
	).Scan(&deployerHex)

	if err == nil {
		return common.HexToAddress(deployerHex), nil
	}

	// Step 2: Try BscScan API to find the contract creator
	deployer, err := h.getCreatorFromBscScan(ctx, tokenAddr)
	if err == nil {
		// Cache the result for future use
		go func() {
			h.UpdateCreatorTokens(context.Background(), deployer, tokenAddr, "", "")
		}()
		return deployer, nil
	}

	// Step 3: If we have an RPC client, try on-chain lookup via receipt
	if h.client != nil {
		// For now, return not found — on-chain deployer discovery
		// requires knowing the creation tx hash, which needs archive node
		log.Printf("[Historian] On-chain deployer lookup not available without creation tx hash")
	}

	return common.Address{}, fmt.Errorf("deployer not found for %s", tokenAddr.Hex())
}

// bscScanCreationResponse is the API response for contract creation lookup.
type bscScanCreationResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  []struct {
		ContractAddress string `json:"contractAddress"`
		ContractCreator string `json:"contractCreator"`
		TxHash          string `json:"txHash"`
	} `json:"result"`
}

// getCreatorFromBscScan queries BscScan API for the contract creator.
func (h *Historian) getCreatorFromBscScan(ctx context.Context, tokenAddr common.Address) (common.Address, error) {
	if h.apiKey == "" {
		return common.Address{}, fmt.Errorf("bscscan: no API key configured")
	}

	url := fmt.Sprintf(
		"https://api.bscscan.com/api?module=contract&action=getcontractcreation&contractaddresses=%s&apikey=%s",
		tokenAddr.Hex(), h.apiKey,
	)

	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
	if err != nil {
		return common.Address{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return common.Address{}, fmt.Errorf("bscscan api error: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result bscScanCreationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return common.Address{}, fmt.Errorf("bscscan parse error: %w", err)
	}

	if result.Status != "1" || len(result.Result) == 0 {
		return common.Address{}, fmt.Errorf("bscscan: no creation data found")
	}

	creatorHex := result.Result[0].ContractCreator
	if creatorHex == "" {
		return common.Address{}, fmt.Errorf("bscscan: empty creator")
	}

	return common.HexToAddress(creatorHex), nil
}

// getCreatorRecord fetches the founder profile from the database.
func (h *Historian) getCreatorRecord(ctx context.Context, creator common.Address) (*CreatorRecord, error) {
	row := h.db.QueryRow(ctx, `
		SELECT address, total_tokens, rugged_tokens, avg_lifetime_hours,
		       first_seen_at, last_seen_at, risk_label
		FROM creator_records
		WHERE address = $1
	`, creator.Hex())

	var record CreatorRecord
	err := row.Scan(
		&record.Address,
		&record.TotalTokens,
		&record.RuggedTokens,
		&record.AvgLifetimeHours,
		&record.FirstSeenAt,
		&record.LastSeenAt,
		&record.RiskLabel,
	)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// getRecentTokens fetches the most recent tokens created by a founder.
func (h *Historian) getRecentTokens(ctx context.Context, creator common.Address, limit int) ([]TokenRecord, error) {
	rows, err := h.db.Query(ctx, `
		SELECT token_address, token_name, token_symbol, status, created_at, rugged_at
		FROM creator_tokens
		WHERE creator_address = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, creator.Hex(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []TokenRecord
	for rows.Next() {
		var t TokenRecord
		err := rows.Scan(&t.Address, &t.Name, &t.Symbol, &t.Status, &t.CreatedAt, &t.RuggedAt)
		if err != nil {
			continue
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

// classifyCreator generates a human-readable risk label for a founder.
func (h *Historian) classifyCreator(totalTokens int, rugRate float64) string {
	switch {
	case totalTokens == 0:
		return "✅ 新地址，无不良记录"
	case totalTokens >= 10 && rugRate >= 0.7:
		return fmt.Sprintf("⚠️ 惯犯：该地址创建了大量已归零代币（归零率 %.0f%%）", rugRate*100)
	case rugRate >= 0.5:
		return "⚠️ 可疑：超过半数创建的代币已归零"
	case totalTokens >= 5:
		return "ℹ️ 该地址较活跃，但暂未发现明显风险模式"
	default:
		return "✅ 新地址，无不良记录"
	}
}

// UpdateCreatorTokens updates the creator_tokens table when a new token is discovered.
func (h *Historian) UpdateCreatorTokens(ctx context.Context, creator, tokenAddr common.Address, name, symbol string) error {
	// Upsert creator record
	_, err := h.db.Exec(ctx, `
		INSERT INTO creator_records (address, total_tokens, first_seen_at, last_seen_at)
		VALUES ($1, 1, NOW(), NOW())
		ON CONFLICT (address) DO UPDATE SET
			total_tokens = creator_records.total_tokens + 1,
			last_seen_at = NOW()
	`, creator.Hex())
	if err != nil {
		return fmt.Errorf("upsert creator record: %w", err)
	}

	// Insert token record
	_, err = h.db.Exec(ctx, `
		INSERT INTO creator_tokens (creator_address, token_address, token_name, token_symbol, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (token_address) DO NOTHING
	`, creator.Hex(), tokenAddr.Hex(), name, symbol)
	if err != nil {
		return fmt.Errorf("insert creator token: %w", err)
	}

	return nil
}

// MarkTokenAsRugged updates a token's status to 'rugged'.
func (h *Historian) MarkTokenAsRugged(ctx context.Context, tokenAddr common.Address) error {
	_, err := h.db.Exec(ctx, `
		UPDATE creator_tokens
		SET status = 'rugged', rugged_at = NOW()
		WHERE token_address = $1 AND status = 'active'
	`, tokenAddr.Hex())
	return err
}

// GetDeployerFromReceipt extracts the deployer from a contract creation transaction.
func (h *Historian) GetDeployerFromReceipt(ctx context.Context, txHash common.Hash) (common.Address, error) {
	if h.client == nil {
		return common.Address{}, fmt.Errorf("no RPC client available")
	}

	tx, _, err := h.client.TransactionByHash(ctx, txHash)
	if err != nil {
		return common.Address{}, fmt.Errorf("get transaction: %w", err)
	}

	chainID := big.NewInt(56) // BSC
	signer := types.NewCancunSigner(chainID)
	from, err := types.Sender(signer, tx)
	if err != nil {
		return common.Address{}, fmt.Errorf("get sender: %w", err)
	}

	return from, nil
}
