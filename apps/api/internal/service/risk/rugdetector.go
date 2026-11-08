package risk

import (
	"context"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RugDetector periodically checks tracked tokens for rug pulls.
type RugDetector struct {
	client  *ethclient.Client
	db      *pgxpool.Pool
	indexer *Indexer
}

// NewRugDetector creates a new RugDetector.
func NewRugDetector(client *ethclient.Client, db *pgxpool.Pool, indexer *Indexer) *RugDetector {
	return &RugDetector{client: client, db: db, indexer: indexer}
}

// Start begins the periodic rug detection loop.
func (rd *RugDetector) Start(ctx context.Context) {
	log.Println("[RugDetector] Starting rug detection loop...")
	go rd.detectionLoop(ctx)
}

// detectionLoop runs every 5 minutes to check for rug pulls.
func (rd *RugDetector) detectionLoop(ctx context.Context) {
	// Run once immediately
	rd.checkAllTokens(ctx)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rd.checkAllTokens(ctx)
		}
	}
}

// checkAllTokens checks all active tracked tokens for rug pulls.
func (rd *RugDetector) checkAllTokens(ctx context.Context) {
	if rd.client == nil {
		return
	}

	tokens, err := rd.indexer.GetTrackedTokens(ctx)
	if err != nil {
		log.Printf("[RugDetector] Failed to get tracked tokens: %v", err)
		return
	}

	if len(tokens) == 0 {
		return
	}

	log.Printf("[RugDetector] Checking %d tracked tokens for rug status...", len(tokens))
	rugged := 0

	for _, tokenAddr := range tokens {
		if rd.isRugged(ctx, tokenAddr) {
			rd.markAsRugged(ctx, tokenAddr)
			rugged++
		}
	}

	if rugged > 0 {
		log.Printf("[RugDetector] 💀 Detected %d new rug pulls", rugged)
		// Update creator stats
		rd.updateCreatorStats(ctx)
	}
}

// isRugged checks if a token has been rugged by checking its PancakeSwap pair reserves.
func (rd *RugDetector) isRugged(ctx context.Context, tokenAddr common.Address) bool {
	// Get pair address
	callData, err := factoryABI.Pack("getPair", tokenAddr, wbnbAddress)
	if err != nil {
		return false
	}

	rpcCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	msg := ethereum.CallMsg{To: &pancakeSwapFactory, Data: callData}
	result, err := rd.client.CallContract(rpcCtx, msg, nil)
	if err != nil || len(result) < 32 {
		return false
	}

	pairAddr := common.BytesToAddress(result)
	if pairAddr == (common.Address{}) {
		return false
	}

	// Get reserves
	reserveData, err := pairABI.Pack("getReserves")
	if err != nil {
		return false
	}

	msg2 := ethereum.CallMsg{To: &pairAddr, Data: reserveData}
	result2, err := rd.client.CallContract(rpcCtx, msg2, nil)
	if err != nil {
		return false
	}

	values, err := pairABI.Unpack("getReserves", result2)
	if err != nil || len(values) < 2 {
		return false
	}

	r0 := values[0].(*big.Int)
	r1 := values[1].(*big.Int)

	// Either reserve is 0 = rugged
	return r0.Cmp(big.NewInt(0)) == 0 || r1.Cmp(big.NewInt(0)) == 0
}

// markAsRugged marks a token as rugged in the database.
func (rd *RugDetector) markAsRugged(ctx context.Context, tokenAddr common.Address) {
	result, err := rd.db.Exec(ctx, `
		UPDATE creator_tokens
		SET status = 'rugged', rugged_at = NOW()
		WHERE token_address = $1 AND status = 'active'
	`, tokenAddr.Hex())
	if err != nil {
		log.Printf("[RugDetector] Failed to mark rugged %s: %v", tokenAddr.Hex(), err)
		return
	}
	if result.RowsAffected() > 0 {
		log.Printf("[RugDetector] 💀 Marked as rugged: %s", tokenAddr.Hex())
	}
}

// updateCreatorStats recalculates rugged_tokens and avg_lifetime for all creators.
func (rd *RugDetector) updateCreatorStats(ctx context.Context) {
	_, err := rd.db.Exec(ctx, `
		UPDATE creator_records cr
		SET
			rugged_tokens = (
				SELECT COUNT(*) FROM creator_tokens ct
				WHERE ct.creator_address = cr.address AND ct.status = 'rugged'
			),
			avg_lifetime_hours = (
				SELECT COALESCE(AVG(
					EXTRACT(EPOCH FROM (COALESCE(rugged_at, NOW()) - created_at)) / 3600
				), 0)
				FROM creator_tokens ct
				WHERE ct.creator_address = cr.address AND ct.status = 'rugged'
			),
			last_seen_at = NOW()
	`)
	if err != nil {
		log.Printf("[RugDetector] Failed to update creator stats: %v", err)
	} else {
		log.Println("[RugDetector] ✅ Creator stats updated")
	}
}
