package risk

import (
	"math"
)

// Analyzer computes the weighted six-dimension risk score.
type Analyzer struct{}

// ---- Dimension Weights ----
// Rationale:
//   - Creator history (30%): most important — tracks serial rug pullers
//   - LP lock (25%): liquidity not locked = founder can exit anytime
//   - Holder concentration (15%): top-heavy = whale manipulation risk
//   - Contract safety (10%): technical exploit risk
//   - Liquidity depth (10%): exit difficulty
//   - Social signals (10%): community authenticity indicators

const (
	WeightContract  = 0.10
	WeightLP        = 0.25
	WeightHolder    = 0.15
	WeightCreator   = 0.30
	WeightLiquidity = 0.10
	WeightSocial    = 0.10
)

// NewAnalyzer creates a new Analyzer instance.
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// CalculateScore computes the overall risk score using weighted dimensions.
// Returns a value between 1.0 (lowest risk) and 10.0 (highest risk).
func (a *Analyzer) CalculateScore(profile *TokenRiskProfile) float64 {
	// Collect all dimension scores with their weights
	type scoredDim struct {
		score  float64
		weight float64
	}

	scores := []scoredDim{
		{a.scoreContract(profile.ContractSafety), WeightContract},
		{a.scoreLP(profile.LPStatus), WeightLP},
		{a.scoreHolder(profile.HolderConcentration), WeightHolder},
		{a.scoreCreatorHistory(profile.CreatorProfile), WeightCreator},
		{a.scoreLiquidity(profile.Liquidity), WeightLiquidity},
		{a.scoreSocial(profile.SocialSignals), WeightSocial},
	}

	var total float64
	for _, s := range scores {
		total += s.score * s.weight
	}

	return math.Round(total*10) / 10
}

// CalculateDimensions returns the six individual dimension scores.
func (a *Analyzer) CalculateDimensions(profile *TokenRiskProfile) [6]DimensionScoreStruct {
	return [6]DimensionScoreStruct{
		{
			Name:    "合约安全",
			Score:   a.scoreContract(profile.ContractSafety),
			Weight:  WeightContract,
			Details: a.contractDetails(profile.ContractSafety),
			Status:  scoreStatus(a.scoreContract(profile.ContractSafety)),
		},
		{
			Name:    "LP锁定",
			Score:   a.scoreLP(profile.LPStatus),
			Weight:  WeightLP,
			Details: a.lpDetails(profile.LPStatus),
			Status:  scoreStatus(a.scoreLP(profile.LPStatus)),
		},
		{
			Name:    "持仓集中",
			Score:   a.scoreHolder(profile.HolderConcentration),
			Weight:  WeightHolder,
			Details: a.holderDetails(profile.HolderConcentration),
			Status:  scoreStatus(a.scoreHolder(profile.HolderConcentration)),
		},
		{
			Name:    "创始人",
			Score:   a.scoreCreatorHistory(profile.CreatorProfile),
			Weight:  WeightCreator,
			Details: a.creatorDetails(profile.CreatorProfile),
			Status:  scoreStatus(a.scoreCreatorHistory(profile.CreatorProfile)),
		},
		{
			Name:    "流动性",
			Score:   a.scoreLiquidity(profile.Liquidity),
			Weight:  WeightLiquidity,
			Details: a.liquidityDetails(profile.Liquidity),
			Status:  scoreStatus(a.scoreLiquidity(profile.Liquidity)),
		},
		{
			Name:    "社区信号",
			Score:   a.scoreSocial(profile.SocialSignals),
			Weight:  WeightSocial,
			Details: a.socialDetails(profile.SocialSignals),
			Status:  scoreStatus(a.scoreSocial(profile.SocialSignals)),
		},
	}
}

// GetRiskLevel maps a numeric score to a risk level label.
func (a *Analyzer) GetRiskLevel(score float64) string {
	switch {
	case score <= 3.0:
		return "低风险"
	case score <= 5.0:
		return "中风险"
	case score <= 7.0:
		return "高风险"
	default:
		return "极度危险"
	}
}

// ---- Dimension Scoring Functions ----

// scoreCreatorHistory scores the creator's historical behavior.
func (a *Analyzer) scoreCreatorHistory(profile *CreatorRecord) float64 {
	if profile == nil {
		return 5.0 // unknown creator
	}
	if profile.TotalTokens == 0 {
		return 2.0 // new address, no history
	}

	rugRate := profile.RugRate

	switch {
	case profile.TotalTokens >= 10 && rugRate >= 0.7:
		return 10.0 // serial rugger
	case rugRate >= 0.5:
		return 8.0 // high risk
	case profile.TotalTokens >= 5:
		return 5.0 // suspicious
	case profile.TotalTokens >= 2:
		return 3.0 // mild concern
	default:
		return 1.5 // likely safe
	}
}

// scoreLP scores the LP lock status.
func (a *Analyzer) scoreLP(lp *LPStatusData) float64 {
	if lp == nil || !lp.Found {
		return 10.0 // no LP pool — extreme danger
	}
	if !lp.Locked {
		return 9.0 // LP not locked — rug imminent
	}
	if lp.LockRemaining < 24*3600 {
		return 7.0 // unlocking within 24h
	}
	if lp.LockRemaining < 7*24*3600 {
		return 4.0 // unlocking within 1 week
	}
	return 1.0 // long-term locked, safe
}

// scoreContract scores the contract safety indicators.
func (a *Analyzer) scoreContract(safety *ContractSafetyData) float64 {
	if safety == nil {
		return 5.0
	}

	score := 0.0
	if safety.IsHoneypot {
		score += 10.0 // honeypot is maximum danger
	}
	if safety.HasMint {
		score += 3.0
	}
	if safety.HasBlacklist {
		score += 2.0
	}
	if safety.HasPause {
		score += 2.0
	}
	if safety.HasHighTax {
		score += 3.0 // high tax is concerning
	}
	return math.Min(score, 10.0)
}

// scoreHolder scores the holder concentration.
func (a *Analyzer) scoreHolder(h *HolderData) float64 {
	if h == nil || h.TotalHolders == 0 {
		return 5.0
	}
	if h.Top3Percent >= 80 {
		return 10.0
	}
	if h.Top3Percent >= 50 {
		return 7.0
	}
	if h.Top10Percent >= 80 {
		return 5.0
	}
	return 2.0
}

// scoreLiquidity scores the liquidity depth.
func (a *Analyzer) scoreLiquidity(l *LiquidityData) float64 {
	if l == nil {
		return 5.0
	}
	switch {
	case l.TotalLiquidityUSD < 1000:
		return 9.0
	case l.TotalLiquidityUSD < 10000:
		return 6.0
	case l.TotalLiquidityUSD < 100000:
		return 3.0
	default:
		return 1.0
	}
}

// scoreSocial scores community signals.
func (a *Analyzer) scoreSocial(s *SocialSignalData) float64 {
	if s == nil {
		return 5.0
	}
	score := 5.0 // default neutral
	if s.HasWebsite {
		score -= 1.0
	}
	if s.TwitterAgeDays > 30 {
		score -= 1.5
	}
	if s.TelegramMembers > 1000 {
		score -= 1.0
	}
	return math.Max(score, 1.0)
}

// ---- Detail Description Functions ----

func (a *Analyzer) contractDetails(s *ContractSafetyData) string {
	if s == nil {
		return "无法读取合约代码"
	}
	if s.IsHoneypot {
		return "⚠️ 检测为蜜罐合约，无法卖出"
	}
	details := "合约代码已验证"
	if s.HasMint {
		details += "，存在增发函数"
	}
	if s.HasBlacklist {
		details += "，存在黑名单功能"
	}
	if s.HasPause {
		details += "，可暂停交易"
	}
	return details
}

func (a *Analyzer) lpDetails(lp *LPStatusData) string {
	if lp == nil || !lp.Found {
		return "未找到流动性池"
	}
	if !lp.Locked {
		return "⚠️ LP 完全未锁定，创始人可随时撤池"
	}
	if lp.LockRemaining < 24*3600 {
		return "⚠️ LP 锁定即将到期（< 24h）"
	}
	return "LP 已锁定，状态安全"
}

func (a *Analyzer) holderDetails(h *HolderData) string {
	if h == nil || h.TotalHolders == 0 {
		return "暂无持仓数据"
	}
	if h.Top3Percent >= 80 {
		return "⚠️ 前 3 地址持有超过 80%"
	}
	if h.Top3Percent >= 50 {
		return "⚠️ 前 3 地址持有超过 50%"
	}
	return "持仓分布较分散"
}

func (a *Analyzer) creatorDetails(c *CreatorRecord) string {
	if c == nil {
		return "无法获取创始人信息"
	}
	if c.TotalTokens == 0 {
		return "新地址，无历史记录"
	}
	if c.RugRate >= 0.7 {
		return "⚠️ 高归零率创始人，创建过多个已归零代币"
	}
	if c.RugRate >= 0.5 {
		return "⚠️ 该创始人创建的代币半数以上已归零"
	}
	return "创始人无显著风险记录"
}

func (a *Analyzer) liquidityDetails(l *LiquidityData) string {
	if l == nil {
		return "暂无流动性数据"
	}
	if l.TotalLiquidityUSD < 1000 {
		return "⚠️ 流动性极低（<$1,000），难以退出"
	}
	if l.TotalLiquidityUSD < 10000 {
		return "流动性偏低（<$10,000）"
	}
	return "流动性充足"
}

func (a *Analyzer) socialDetails(s *SocialSignalData) string {
	if s == nil {
		return "暂无社区数据"
	}
	if !s.HasWebsite && s.TwitterAgeDays == 0 {
		return "未发现网站和社交媒体"
	}
	return "社区信号正常"
}

// scoreStatus maps a score to a status label.
func scoreStatus(score float64) string {
	switch {
	case score <= 3.0:
		return "good"
	case score <= 6.0:
		return "warning"
	default:
		return "danger"
	}
}

// DimensionScoreStruct is used internally for the 6-dimension array.
type DimensionScoreStruct struct {
	Name    string
	Score   float64
	Weight  float64
	Details string
	Status  string
}
