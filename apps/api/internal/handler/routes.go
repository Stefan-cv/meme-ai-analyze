package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest"

	"github.com/meme-ai-analyze/api/internal/handler/risk"
	"github.com/meme-ai-analyze/api/internal/svc"
)

// corsHandler returns 204 for OPTIONS preflight requests to allow CORS.
func corsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	w.Header().Set("Access-Control-Max-Age", "86400")
	w.WriteHeader(http.StatusNoContent)
}

// RegisterRoutes registers all API routes on the go-zero server.
func RegisterRoutes(server *rest.Server, ctx *svc.ServiceContext) {
	// Health check
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/health",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok","service":"meme-ai-analyze-api"}`))
		},
	})

	// CORS preflight handlers for API paths
	server.AddRoute(rest.Route{
		Method:  http.MethodOptions,
		Path:    "/api/v1/scan",
		Handler: corsHandler,
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodOptions,
		Path:    "/api/v1/report/:tokenAddress",
		Handler: corsHandler,
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodOptions,
		Path:    "/api/v1/creator/:address",
		Handler: corsHandler,
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodOptions,
		Path:    "/api/v1/stats",
		Handler: corsHandler,
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodOptions,
		Path:    "/api/v1/hot",
		Handler: corsHandler,
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodOptions,
		Path:    "/api/v1/risk-wall",
		Handler: corsHandler,
	})

	// Risk scan handlers
	riskHandler := risk.NewHandlers(ctx)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/api/v1/scan",
				Handler: riskHandler.Scan,
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/report/:tokenAddress",
				Handler: riskHandler.GetReport,
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/creator/:address",
				Handler: riskHandler.GetCreator,
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/stats",
				Handler: riskHandler.GetStats,
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/hot",
				Handler: riskHandler.GetHotTokens,
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/risk-wall",
				Handler: riskHandler.GetRiskWall,
			},
		},
	)

	// WebSocket endpoint
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/ws/scan",
		Handler: riskHandler.ScanWS,
	})
	// WebSocket CORS preflight
	server.AddRoute(rest.Route{
		Method:  http.MethodOptions,
		Path:    "/ws/scan",
		Handler: corsHandler,
	})
}
