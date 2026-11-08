package risk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Reporter generates natural-language risk analysis using AI (OpenAI / Gemini).
type Reporter struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewReporter creates a new Reporter instance.
func NewReporter(apiKey, model string) *Reporter {
	if model == "" {
		model = "gpt-4o"
	}

	return &Reporter{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.openai.com/v1",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SystemPrompt is the AI system prompt for risk analysis.
const SystemPrompt = `你是一位专业的链上安全分析师，擅长把复杂的链上数据翻译成散户能听懂的话。

分析规则：
1. 先看创始人历史——如果 rug 率 > 50%，这是最危险的信号，必须警告
2. 再看 LP 锁定——未锁定的池子创始人随时可以跑路
3. 再看合约安全——有 mint/blacklist/pause 函数的代币风险更高
4. 最后看持仓——前 3 地址持有 > 50% 说明高度集中

输出要求：
- 150-200 字
- 第一句给结论（"该代币风险较高，建议谨慎" 或 "暂未发现明显风险信号"）
- 用 2-3 条具体数据支撑结论
- 如果创始人历史可疑，重点强调
- 语言直白。使用"建议不要参与"，而不是"存在较高的投资风险"
- 如果代币一切正常，最后加上"Meme 代币天然高风险，请自行判断"

不要输出 markdown 格式。不要用 bullet point。用自然段落。`

// Generate creates an AI-powered natural-language risk report.
func (r *Reporter) Generate(ctx context.Context, profile *TokenRiskProfile) (string, error) {
	if r.apiKey == "" {
		log.Println("[Reporter] No API key configured, returning fallback summary")
		return r.fallbackSummary(profile), nil
	}

	// Build the prompt with structured data
	dataJSON := r.formatProfileForAI(profile)
	userPrompt := fmt.Sprintf("请分析以下 Meme 代币的风险数据：\n\n%s", dataJSON)

	// Call OpenAI Chat Completion API
	reqBody := map[string]interface{}{
		"model": r.model,
		"messages": []map[string]string{
			{"role": "system", "content": SystemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"max_tokens":  500,
		"temperature": 0.3,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		r.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.apiKey)

	resp, err := r.client.Do(req)
	if err != nil {
		log.Printf("[Reporter] API call failed: %v, using fallback", err)
		return r.fallbackSummary(profile), nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Reporter] API returned %d: %s, using fallback", resp.StatusCode, string(body))
		return r.fallbackSummary(profile), nil
	}

	// Parse OpenAI response
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(result.Choices) == 0 {
		return r.fallbackSummary(profile), nil
	}

	return result.Choices[0].Message.Content, nil
}

// formatProfileForAI converts the risk profile to AI-friendly JSON.
func (r *Reporter) formatProfileForAI(profile *TokenRiskProfile) string {
	data := map[string]interface{}{
		"token": profile.Address,
	}

	// Contract safety
	if profile.ContractSafety != nil {
		data["contract"] = map[string]interface{}{
			"is_honeypot":       profile.ContractSafety.IsHoneypot,
			"has_mint_function": profile.ContractSafety.HasMint,
			"has_blacklist":     profile.ContractSafety.HasBlacklist,
			"has_pause":         profile.ContractSafety.HasPause,
			"has_high_tax":      profile.ContractSafety.HasHighTax,
		}
	}

	// LP status
	if profile.LPStatus != nil {
		data["liquidity_pool"] = map[string]interface{}{
			"found":          profile.LPStatus.Found,
			"is_locked":      profile.LPStatus.Locked,
			"lock_remaining": profile.LPStatus.LockRemaining,
		}
	}

	// Holder concentration
	if profile.HolderConcentration != nil {
		data["holders"] = map[string]interface{}{
			"total_holders":    profile.HolderConcentration.TotalHolders,
			"top3_percentage":  profile.HolderConcentration.Top3Percent,
			"top10_percentage": profile.HolderConcentration.Top10Percent,
		}
	}

	// Creator history (most important)
	if profile.CreatorProfile != nil {
		data["creator"] = map[string]interface{}{
			"total_tokens":      profile.CreatorProfile.TotalTokens,
			"rugged_tokens":     profile.CreatorProfile.RuggedTokens,
			"rug_rate":          profile.CreatorProfile.RugRate,
			"avg_lifetime_hrs":  profile.CreatorProfile.AvgLifetimeHours,
			"first_seen_days":   int(time.Since(profile.CreatorProfile.FirstSeenAt).Hours() / 24),
			"risk_label":        profile.CreatorProfile.RiskLabel,
		}
	}

	// Liquidity
	if profile.Liquidity != nil {
		data["liquidity"] = map[string]interface{}{
			"total_liquidity_usd": profile.Liquidity.TotalLiquidityUSD,
			"daily_volume_usd":    profile.Liquidity.DailyVolumeUSD,
		}
	}

	b, _ := json.Marshal(data)
	return string(b)
}

// fallbackSummary generates a rule-based summary when AI is unavailable.
func (r *Reporter) fallbackSummary(profile *TokenRiskProfile) string {
	if profile == nil {
		return "暂无足够数据进行分析。"
	}

	var parts []string

	// Check creator history (most critical)
	if profile.CreatorProfile != nil && profile.CreatorProfile.TotalTokens > 0 {
		if profile.CreatorProfile.RugRate >= 0.7 {
			parts = append(parts, fmt.Sprintf(
				"该代币的创建者曾部署过 %d 个代币，其中 %d 个已归零（归零率 %.0f%%），属于高风险创建者。",
				profile.CreatorProfile.TotalTokens,
				profile.CreatorProfile.RuggedTokens,
				profile.CreatorProfile.RugRate*100,
			))
		}
	}

	// Check LP status
	if profile.LPStatus != nil {
		if !profile.LPStatus.Locked {
			parts = append(parts, "该代币的流动性池未锁定，创始人可随时撤走流动性。")
		}
	}

	// Conclusion
	if len(parts) == 0 {
		return "暂未发现明显风险信号。Meme 代币天然高风险，请自行判断。"
	}

	parts = append(parts, "强烈建议不要参与该代币。")
	return joinStrings(parts, " ")
}

func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}
