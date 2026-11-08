package risk

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/meme-ai-analyze/api/internal/service/risk"
	"github.com/meme-ai-analyze/api/internal/svc"
	"github.com/meme-ai-analyze/api/internal/types"
)

// ScanLogic orchestrates the complete scan pipeline:
//   Cache check → Scanner → Analyzer → Reporter → Oracle → Cache → DB
type ScanLogic struct {
	svcCtx *svc.ServiceContext
}

// NewScanLogic creates a new ScanLogic instance.
func NewScanLogic(ctx *svc.ServiceContext) *ScanLogic {
	return &ScanLogic{svcCtx: ctx}
}

// Execute runs the full scan pipeline for a given token address.
func (l *ScanLogic) Execute(ctx context.Context, req *types.ScanRequest) (*types.ScanResponse, error) {
	tokenAddr := req.TokenAddress
	chainID := req.ChainID

	log.Printf("[ScanLogic] Starting scan for %s (chain=%d)", tokenAddr, chainID)

	// Step 1: Check cache
	if cached, found := l.svcCtx.Cache.GetOrScan(ctx, chainID, tokenAddr); found {
		log.Printf("[ScanLogic] Cache hit for %s", tokenAddr)
		return cached, nil
	}

	// Step 2: Run scanner (6 goroutines in parallel)
	profile, err := l.svcCtx.Scanner.Scan(ctx, tokenAddr)
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	// Step 3: Calculate risk score
	totalScore := l.svcCtx.Analyzer.CalculateScore(profile)
	riskLevel := l.svcCtx.Analyzer.GetRiskLevel(totalScore)

	// Step 4: Generate per-dimension scores
	dims := l.svcCtx.Analyzer.CalculateDimensions(profile)

	// Convert dimensions to response format
	dimensions := [6]types.DimensionScore{}
	for i, d := range dims {
		dimensions[i] = types.DimensionScore{
			Name:    d.Name,
			Score:   d.Score,
			Weight:  d.Weight,
			Details: d.Details,
			Status:  d.Status,
		}
	}

	// Step 5: Generate AI report
	aiReport, err := l.svcCtx.Reporter.Generate(ctx, profile)
	if err != nil {
		log.Printf("[ScanLogic] AI report generation failed: %v", err)
		aiReport = "AI 分析暂时不可用，请参考各维度评分。"
	}

	// Step 6: Build creator history response
	creatorHistory := buildCreatorHistory(profile.CreatorProfile)

	// Step 7: Build the final response
	now := time.Now().Unix()
	resp := &types.ScanResponse{
		TokenAddress:   tokenAddr,
		TokenName:      profile.TokenName,
		TokenSymbol:    profile.TokenSymbol,
		RiskScore:      totalScore,
		RiskLevel:      riskLevel,
		Dimensions:     dimensions,
		AIReport:       aiReport,
		CreatorHistory: creatorHistory,
		ReportHash:     "",
		TxHash:         "",
		ScannedAt:      now,
	}

	// Step 8: Anchor report hash on-chain (async, non-blocking)
	go func() {
		reportJSON, _ := json.Marshal(resp)
		reportHash := risk.ReportHash(reportJSON)
		txHash, err := l.svcCtx.Oracle.SubmitReport(
			context.Background(),
			common.HexToAddress(tokenAddr),
			reportHash,
			uint8(totalScore),
			"v1.0.0",
		)
		if err != nil {
			log.Printf("[ScanLogic] Oracle submission failed: %v", err)
			return
		}
		resp.ReportHash = fmt.Sprintf("0x%x", reportHash)
		resp.TxHash = txHash

		// Update cached response with on-chain data
		l.svcCtx.Cache.Set(context.Background(), chainID, resp)
	}()

	// Step 9: Cache the result
	if err := l.svcCtx.Cache.Set(ctx, chainID, resp); err != nil {
		log.Printf("[ScanLogic] Cache set failed: %v", err)
	}

	// Step 10: Record scan in database and stats
	go func() {
		l.svcCtx.Cache.IncrStats(context.Background(), "total_scans")
		l.saveScanLog(context.Background(), tokenAddr, totalScore, now)
	}()

	// Step 11: Record scan on-chain (async, non-blocking)
	go func() {
		if _, err := l.svcCtx.Oracle.RecordScan(context.Background(), common.HexToAddress(tokenAddr)); err != nil {
			log.Printf("[ScanLogic] RecordScan failed: %v", err)
		}
	}()

	log.Printf("[ScanLogic] Scan complete for %s — score=%.1f (%s)", tokenAddr, totalScore, riskLevel)
	return resp, nil
}

// saveScanLog inserts a scan log record into PostgreSQL.
func (l *ScanLogic) saveScanLog(ctx context.Context, tokenAddr string, riskScore float64, scannedAt int64) {
	if l.svcCtx.DB == nil {
		return
	}

	_, err := l.svcCtx.DB.Exec(ctx, `
		INSERT INTO scan_logs (token_address, risk_score, scanned_at)
		VALUES ($1, $2, $3)
	`, tokenAddr, riskScore, scannedAt)
	if err != nil {
		log.Printf("[ScanLogic] Failed to save scan log: %v", err)
	}
}

// buildCreatorHistory converts the internal CreatorRecord to the API response type.
func buildCreatorHistory(record *risk.CreatorRecord) *types.CreatorProfile {
	if record == nil {
		return nil
	}

	// Convert recent tokens
	recentTokens := make([]types.TokenBrief, 0, len(record.RecentTokens))
	for _, t := range record.RecentTokens {
		lifetime := 0.0
		if t.RuggedAt != nil {
			lifetime = t.RuggedAt.Sub(t.CreatedAt).Hours()
		} else {
			lifetime = time.Since(t.CreatedAt).Hours()
		}

		recentTokens = append(recentTokens, types.TokenBrief{
			Address:   t.Address,
			Name:      t.Name,
			Symbol:    t.Symbol,
			Status:    t.Status,
			Lifetime:  lifetime,
			CreatedAt: t.CreatedAt.Unix(),
		})
	}

	// Calculate active tokens (total - rugged)
	activeTokens := record.TotalTokens - record.RuggedTokens

	return &types.CreatorProfile{
		Address:        record.Address,
		TotalTokens:    record.TotalTokens,
		ActiveTokens:   activeTokens,
		RuggedTokens:   record.RuggedTokens,
		RugRate:        record.RugRate,
		AvgLifetimeHrs: record.AvgLifetimeHours,
		FirstSeenDays:  int(time.Since(record.FirstSeenAt).Hours() / 24),
		RiskLabel:      record.RiskLabel,
		RecentTokens:   recentTokens,
	}
}
