package risk

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/meme-ai-analyze/api/internal/logic/risk"
	"github.com/meme-ai-analyze/api/internal/svc"
	"github.com/meme-ai-analyze/api/internal/types"
)

// maxConcurrentScans limits how many scans can run simultaneously to avoid
// overwhelming the shared RPC node (10 req/s × 6 goroutines each).
const maxConcurrentScans = 5

// Handlers contains HTTP and WebSocket handlers for the risk analysis API.
type Handlers struct {
	svcCtx    *svc.ServiceContext
	semaphore chan struct{} // concurrency limiter
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(ctx *svc.ServiceContext) *Handlers {
	return &Handlers{
		svcCtx:    ctx,
		semaphore: make(chan struct{}, maxConcurrentScans),
	}
}

// ---- HTTP Handlers ----

// Scan handles POST /api/v1/scan — triggers a new token scan.
func (h *Handlers) Scan(w http.ResponseWriter, r *http.Request) {
	var req types.ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TokenAddress == "" {
		respondError(w, http.StatusBadRequest, "token_address is required")
		return
	}

	if req.ChainID == 0 {
		req.ChainID = 56
	}

	// Acquire concurrency slot; block with timeout
	select {
	case h.semaphore <- struct{}{}:
		defer func() { <-h.semaphore }()
	case <-time.After(30 * time.Second):
		respondError(w, http.StatusTooManyRequests, "server busy, please retry")
		return
	}

	logic := risk.NewScanLogic(h.svcCtx)
	resp, err := logic.Execute(r.Context(), &req)
	if err != nil {
		log.Printf("[Handler] Scan error for %s: %v", req.TokenAddress, err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, &types.APIResponse{
		Code:    0,
		Message: "success",
		Data:    resp,
	})
}

// GetReport handles GET /api/v1/report/:tokenAddress — retrieves a cached report.
func (h *Handlers) GetReport(w http.ResponseWriter, r *http.Request) {
	tokenAddr := r.PathValue("tokenAddress")
	if tokenAddr == "" {
		respondError(w, http.StatusBadRequest, "token address is required")
		return
	}

	chainID := 56 // default
	resp, found := h.svcCtx.Cache.GetOrScan(r.Context(), chainID, tokenAddr)
	if !found {
		respondError(w, http.StatusNotFound, "no cached report found for this token")
		return
	}

	respondJSON(w, http.StatusOK, &types.APIResponse{
		Code:    0,
		Message: "success",
		Data:    resp,
	})
}

// GetCreator handles GET /api/v1/creator/:address — retrieves a creator profile.
func (h *Handlers) GetCreator(w http.ResponseWriter, r *http.Request) {
	creatorAddr := r.PathValue("address")
	if creatorAddr == "" {
		respondError(w, http.StatusBadRequest, "creator address is required")
		return
	}

	respondJSON(w, http.StatusOK, &types.APIResponse{
		Code:    0,
		Message: "success",
		Data:    map[string]string{"address": creatorAddr, "note": "creator profile endpoint"},
	})
}

// GetStats handles GET /api/v1/stats — returns global platform statistics.
func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.svcCtx.Cache.GetStats(r.Context())

	respondJSON(w, http.StatusOK, &types.APIResponse{
		Code:    0,
		Message: "success",
		Data: types.StatsResponse{
			TotalScans:     stats["total_scans"],
			TotalReports:   stats["total_reports"],
			HighRiskTokens: stats["high_risk_tokens"],
			RuggedCreators: stats["rugged_creators"],
		},
	})
}

// GetHotTokens handles GET /api/v1/hot — returns trending tokens.
func (h *Handlers) GetHotTokens(w http.ResponseWriter, r *http.Request) {
	// In production: query most-scanned tokens from scan_logs
	respondJSON(w, http.StatusOK, &types.APIResponse{
		Code:    0,
		Message: "success",
		Data:    []types.RiskWallItem{},
	})
}

// GetRiskWall handles GET /api/v1/risk-wall — returns recent high-risk tokens.
func (h *Handlers) GetRiskWall(w http.ResponseWriter, r *http.Request) {
	// In production: query risk_reports where risk_score >= 7, ordered by scanned_at DESC
	respondJSON(w, http.StatusOK, &types.APIResponse{
		Code:    0,
		Message: "success",
		Data:    []types.RiskWallItem{},
	})
}

// ---- WebSocket Handler ----

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// ScanWS handles GET /ws/scan — provides real-time scan progress over WebSocket.
func (h *Handlers) ScanWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade error: %v", err)
		return
	}
	defer conn.Close()

	tokenAddr := r.URL.Query().Get("token")
	if tokenAddr == "" {
		conn.WriteJSON(types.WSMessage{
			Status: "error",
			Error:  "token parameter is required",
		})
		return
	}

	steps := []string{
		"正在读取合约代码...",
		"正在检查 LP 池锁定状态...",
		"正在分析持仓分布...",
		"正在追踪创始人地址历史...",
		"正在评估流动性深度...",
		"正在抓取社区信号...",
		"AI 正在分析生成报告...",
		"正在将报告锚定到链上...",
	}

	// Send progress updates
	for i, step := range steps {
		progress := float64(i+1) / float64(len(steps)) * 100
		msg := types.WSMessage{
			Status:   "scanning",
			Step:     step,
			Progress: progress,
		}

		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("[WS] Write error: %v", err)
			return
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Run the actual scan
	scanReq := &types.ScanRequest{
		TokenAddress: tokenAddr,
		ChainID:      56,
	}

	logic := risk.NewScanLogic(h.svcCtx)
	result, err := logic.Execute(r.Context(), scanReq)
	if err != nil {
		conn.WriteJSON(types.WSMessage{
			Status: "error",
			Error:  err.Error(),
		})
		return
	}

	conn.WriteJSON(types.WSMessage{
		Status: "completed",
		Data:   result,
	})
}

// ---- Helpers ----

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, &types.APIResponse{
		Code:    status,
		Message: message,
	})
}
