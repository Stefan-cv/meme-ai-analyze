package risk

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"
)

// Indexer scans BSC blocks for new token creation events and tracks them.
type Indexer struct {
	client     *ethclient.Client
	db         *pgxpool.Pool
	chainID    int64
	rpcLimiter *rate.Limiter // separate limiter so indexing doesn't compete with user scans
}

// NewIndexer creates a new Indexer.
func NewIndexer(client *ethclient.Client, db *pgxpool.Pool) *Indexer {
	return &Indexer{
		client:     client,
		db:         db,
		chainID:    56,
		rpcLimiter: rate.NewLimiter(rate.Limit(2), 1), // 2 RPC calls/sec for indexing
	}
}

// PairCreatedEventSignature is keccak256("PairCreated(address,address,address,uint256)")
var PairCreatedEventSignature = common.HexToHash("0x0d3648bd0f6ba80134a33ba9275ac585d9d315f0ad8355cddefde31afa28d0e9")

// Start begins the background indexing loop.
func (idx *Indexer) Start(ctx context.Context) {
	log.Println("[Indexer] Starting block scanner...")
	go idx.scanLoop(ctx)
}

// scanLoop continuously scans recent blocks for new PairCreated events.
func (idx *Indexer) scanLoop(ctx context.Context) {
	ticker := time.NewTicker(12 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Indexer] Shutting down")
			return
		case <-ticker.C:
			idx.scanRecentBlocks(ctx)
		}
	}
}

// scanRecentBlocks fetches recent blocks one-by-one and checks their transactions.
// Uses eth_getBlockByNumber (not eth_getLogs) to work with free RPC nodes.
func (idx *Indexer) scanRecentBlocks(ctx context.Context) {
	if idx.client == nil {
		return
	}

	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	header, err := idx.client.HeaderByNumber(rpcCtx, nil)
	cancel()
	if err != nil {
		log.Printf("[Indexer] Failed to get header: %v", err)
		return
	}

	currentBlock := header.Number.Uint64()
	lastBlock := idx.getLastProcessedBlock(ctx)
	if lastBlock == 0 {
		lastBlock = currentBlock - 5
	}
	if lastBlock >= currentBlock {
		return
	}

	toBlock := currentBlock
	if toBlock-lastBlock > 10 {
		lastBlock = toBlock - 10
	}

	found := 0
	for blockNum := lastBlock + 1; blockNum <= toBlock; blockNum++ {
		if idx.scanBlock(ctx, blockNum) {
			found++
		}
	}

	if found > 0 {
		log.Printf("[Indexer] Blocks %d-%d: found %d new tokens", lastBlock+1, toBlock, found)
	}
	idx.setLastProcessedBlock(ctx, toBlock)
}

// scanBlock scans a single block for new token deployments via PancakeSwap Factory.
func (idx *Indexer) scanBlock(ctx context.Context, blockNum uint64) bool {
	if err := idx.rpcLimiter.Wait(ctx); err != nil {
		return false
	}
	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	block, err := idx.client.BlockByNumber(rpcCtx, new(big.Int).SetUint64(blockNum))
	if err != nil {
		return false
	}

	found := false
	for _, tx := range block.Transactions() {
		// Check if transaction is to PancakeSwap Factory
		if tx.To() != nil && *tx.To() == pancakeSwapFactory {
			// Check receipt for PairCreated events
			receipt, err := idx.client.TransactionReceipt(rpcCtx, tx.Hash())
			if err != nil {
				continue
			}
			for _, vLog := range receipt.Logs {
				if len(vLog.Topics) > 0 && vLog.Topics[0] == PairCreatedEventSignature {
					idx.processPairCreated(ctx, *vLog)
					found = true
				}
			}
		}
	}
	return found
}

// processPairCreated processes a single PairCreated event.
func (idx *Indexer) processPairCreated(ctx context.Context, vLog types.Log) {
	if len(vLog.Topics) < 3 {
		return
	}

	token0 := common.BytesToAddress(vLog.Topics[1].Bytes())
	token1 := common.BytesToAddress(vLog.Topics[2].Bytes())

	// Determine which token is new (not WBNB/BUSD/USDT)
	var newToken common.Address
	if token0 != wbnbAddress && token0 != usdtAddress && token0 != busdAddress {
		newToken = token0
	} else if token1 != wbnbAddress && token1 != usdtAddress && token1 != busdAddress {
		newToken = token1
	} else {
		return // stablecoin pair, skip
	}

	// Check if already tracked
	var exists bool
	err := idx.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM creator_tokens WHERE token_address=$1)`,
		newToken.Hex(),
	).Scan(&exists)

	if err != nil || exists {
		return // already tracked or error
	}

	// Get deployer from transaction
	deployer, err := idx.getDeployerFromTx(ctx, vLog.TxHash)
	if err != nil {
		log.Printf("[Indexer] Failed to get deployer for %s: %v", newToken.Hex(), err)
		return
	}

	// Get token info
	name, symbol := idx.getTokenInfo(ctx, newToken)

	// Write to DB
	_, err = idx.db.Exec(ctx, `
		INSERT INTO creator_records (address, total_tokens, first_seen_at, last_seen_at)
		VALUES ($1, 1, NOW(), NOW())
		ON CONFLICT (address) DO UPDATE SET
			total_tokens = creator_records.total_tokens + 1,
			last_seen_at = NOW()
	`, deployer.Hex())
	if err != nil {
		log.Printf("[Indexer] Failed to upsert creator %s: %v", deployer.Hex(), err)
	}

	_, err = idx.db.Exec(ctx, `
		INSERT INTO creator_tokens (creator_address, token_address, token_name, token_symbol, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (token_address) DO NOTHING
	`, deployer.Hex(), newToken.Hex(), name, symbol)
	if err != nil {
		log.Printf("[Indexer] Failed to insert token %s: %v", newToken.Hex(), err)
	}

	log.Printf("[Indexer] 🆕 New token: %s ($%s) by %s", name, symbol, deployer.Hex()[:10]+"...")
}

// getDeployerFromTx extracts the sender of a transaction.
func (idx *Indexer) getDeployerFromTx(ctx context.Context, txHash common.Hash) (common.Address, error) {
	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, _, err := idx.client.TransactionByHash(rpcCtx, txHash)
	if err != nil {
		return common.Address{}, err
	}

	chainID := big.NewInt(56)
	signer := types.NewCancunSigner(chainID)
	from, err := types.Sender(signer, tx)
	if err != nil {
		return common.Address{}, err
	}
	return from, nil
}

// getTokenInfo fetches token name and symbol via ERC20 calls.
func (idx *Indexer) getTokenInfo(ctx context.Context, tokenAddr common.Address) (string, string) {
	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	name := "Unknown"
	symbol := "???"

	// name()
	callData, _ := erc20ABI.Pack("name")
	msg := ethereum.CallMsg{To: &tokenAddr, Data: callData}
	if result, err := idx.client.CallContract(rpcCtx, msg, nil); err == nil && len(result) >= 64 {
		if n, err2 := abiDecodeString(result); err2 == nil {
			name = n
		}
	}

	// symbol()
	callData2, _ := erc20ABI.Pack("symbol")
	msg2 := ethereum.CallMsg{To: &tokenAddr, Data: callData2}
	if result, err := idx.client.CallContract(rpcCtx, msg2, nil); err == nil && len(result) >= 64 {
		if s, err2 := abiDecodeString(result); err2 == nil {
			symbol = s
		}
	}

	return name, symbol
}

// getLastProcessedBlock reads the last processed block from the DB.
func (idx *Indexer) getLastProcessedBlock(ctx context.Context) uint64 {
	var lastBlock int64
	err := idx.db.QueryRow(ctx,
		`SELECT last_block FROM indexer_state WHERE chain_id = $1`, idx.chainID,
	).Scan(&lastBlock)
	if err != nil {
		return 0
	}
	return uint64(lastBlock)
}

// setLastProcessedBlock updates the last processed block in the DB.
func (idx *Indexer) setLastProcessedBlock(ctx context.Context, block uint64) {
	_, err := idx.db.Exec(ctx, `
		INSERT INTO indexer_state (chain_id, last_block, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (chain_id) DO UPDATE SET last_block = $2, updated_at = NOW()
	`, idx.chainID, block)
	if err != nil {
		log.Printf("[Indexer] Failed to update last block: %v", err)
	}
}

// GetTrackedTokens returns all active tracked tokens.
func (idx *Indexer) GetTrackedTokens(ctx context.Context) ([]common.Address, error) {
	rows, err := idx.db.Query(ctx,
		`SELECT token_address FROM creator_tokens WHERE status = 'active'`)
	if err != nil {
		return nil, fmt.Errorf("get tracked tokens: %w", err)
	}
	defer rows.Close()

	var tokens []common.Address
	for rows.Next() {
		var addr string
		if err := rows.Scan(&addr); err == nil {
			tokens = append(tokens, common.HexToAddress(addr))
		}
	}
	return tokens, rows.Err()
}
