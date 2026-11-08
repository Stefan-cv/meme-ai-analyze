package model

// DimensionScoreData represents the six-dimension score data stored in JSONB.
type DimensionScoreData struct {
	ContractSafety  DimensionDetail `json:"contract_safety"`
	LPStatus        DimensionDetail `json:"lp_status"`
	HolderConcentration DimensionDetail `json:"holder_concentration"`
	CreatorHistory  DimensionDetail `json:"creator_history"`
	Liquidity       DimensionDetail `json:"liquidity"`
	SocialSignals   DimensionDetail `json:"social_signals"`
}

// DimensionDetail represents a single dimension's score details.
type DimensionDetail struct {
	Name    string  `json:"name"`
	Score   float64 `json:"score"`
	Weight  float64 `json:"weight"`
	Details string  `json:"details"`
	Status  string  `json:"status"`
}
