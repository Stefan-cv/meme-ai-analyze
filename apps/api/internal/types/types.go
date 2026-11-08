package types

// ============================================
// Request Types
// ============================================

// ScanRequest represents a token scan request
type ScanRequest struct {
	TokenAddress string `json:"token_address" validate:"required,len=42"`
	ChainID      int    `json:"chain_id,optional,default=56"`
}

// ============================================
// Response Types
// ============================================

// ScanResponse is the core output of a token scan
type ScanResponse struct {
	TokenAddress   string           `json:"token_address"`
	TokenName      string           `json:"token_name"`
	TokenSymbol    string           `json:"token_symbol"`
	RiskScore      float64          `json:"risk_score"`
	RiskLevel      string           `json:"risk_level"`
	Dimensions     [6]DimensionScore `json:"dimensions"`
	AIReport       string           `json:"ai_report"`
	CreatorHistory *CreatorProfile  `json:"creator_history"`
	ReportHash     string           `json:"report_hash"`
	TxHash         string           `json:"tx_hash"`
	ScannedAt      int64            `json:"scanned_at"`
}

// DimensionScore represents a single dimension's score in the six-dimension radar
type DimensionScore struct {
	Name    string  `json:"name"`
	Score   float64 `json:"score"`
	Weight  float64 `json:"weight"`
	Details string  `json:"details"`
	Status  string  `json:"status"` // "good" / "warning" / "danger"
}

// CreatorProfile represents a token creator's historical profile
type CreatorProfile struct {
	Address        string       `json:"address"`
	TotalTokens    int          `json:"total_tokens"`
	ActiveTokens   int          `json:"active_tokens"`
	RuggedTokens   int          `json:"rugged_tokens"`
	RugRate        float64      `json:"rug_rate"`
	AvgLifetimeHrs float64      `json:"avg_lifetime_hrs"`
	FirstSeenDays  int          `json:"first_seen_days"`
	RiskLabel      string       `json:"risk_label"`
	RecentTokens   []TokenBrief `json:"recent_tokens"`
}

// TokenBrief is a summary of a creator's token
type TokenBrief struct {
	Address   string  `json:"address"`
	Name      string  `json:"name"`
	Symbol    string  `json:"symbol"`
	Status    string  `json:"status"` // "active" / "rugged" / "unknown"
	Lifetime  float64 `json:"lifetime_hrs"`
	CreatedAt int64   `json:"created_at"`
}

// ============================================
// WebSocket Progress Types
// ============================================

// WSMessage represents a WebSocket progress message
type WSMessage struct {
	Status   string       `json:"status"` // "scanning" / "completed" / "error"
	Step     string       `json:"step,omitempty"`
	Progress float64      `json:"progress,omitempty"`
	Data     *ScanResponse `json:"data,omitempty"`
	Error    string       `json:"error,omitempty"`
}

// ============================================
// Stats Types
// ============================================

// StatsResponse represents global platform statistics
type StatsResponse struct {
	TotalScans      int64 `json:"total_scans"`
	TotalReports    int64 `json:"total_reports"`
	HighRiskTokens  int64 `json:"high_risk_tokens"`
	RuggedCreators  int64 `json:"rugged_creators"`
}

// ============================================
// Risk Wall Types
// ============================================

// RiskWallItem represents a single high-risk token on the wall
type RiskWallItem struct {
	TokenAddress string  `json:"token_address"`
	TokenName    string  `json:"token_name"`
	TokenSymbol  string  `json:"token_symbol"`
	RiskScore    float64 `json:"risk_score"`
	RiskLevel    string  `json:"risk_level"`
	CreatorLabel string  `json:"creator_label"`
	ScannedAt    int64   `json:"scanned_at"`
}

// ============================================
// API Response Wrapper
// ============================================

// APIResponse is the standard API response wrapper
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
