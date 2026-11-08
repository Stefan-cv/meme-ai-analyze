package model

import "time"

// RiskReport represents a stored risk report in the database.
type RiskReport struct {
	ID             int64   `db:"id"`
	ChainID        int     `db:"chain_id"`
	TokenAddress   string  `db:"token_address"`
	TokenName      string  `db:"token_name"`
	TokenSymbol    string  `db:"token_symbol"`
	RiskScore      float64 `db:"risk_score"`
	RiskLevel      string  `db:"risk_level"`
	DimensionJSON  string  `db:"dimension_json"` // JSONB
	AIReport       string  `db:"ai_report"`
	ReportHash     string  `db:"report_hash"`
	OracleTxHash   string  `db:"oracle_tx_hash"`
	CreatorAddress string  `db:"creator_address"`
	ScannedAt      int64   `db:"scanned_at"`
	CreatedAt      int64   `db:"created_at"`
}

// ScanLog represents a scan event record.
type ScanLog struct {
	ID           int64   `db:"id"`
	TokenAddress string  `db:"token_address"`
	ScannedBy    string  `db:"scanned_by"`
	RiskScore    float64 `db:"risk_score"`
	ScannedAt    int64   `db:"scanned_at"`
}
