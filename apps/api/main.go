package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/meme-ai-analyze/api/internal/config"
	"github.com/meme-ai-analyze/api/internal/handler"
	"github.com/meme-ai-analyze/api/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/api.yaml", "the config file")

func main() {
	flag.Parse()

	// Load configuration
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Create service context (DI container)
	ctx := svc.NewServiceContext(c)
	defer ctx.Close()

	// Create HTTP server
	server := rest.MustNewServer(c.RestConf)

	// Add CORS middleware
	server.Use(corsMiddleware)

	// Register routes
	handler.RegisterRoutes(server, ctx)

	fmt.Printf("🚀 Meme AI Analyze API starting on %s:%d\n", c.Host, c.Port)
	log.Printf("Database: %s:%d/%s", c.Postgres.Host, c.Postgres.Port, c.Postgres.Database)
	log.Printf("Redis: %s:%d", c.Redis.Host, c.Redis.Port)

	server.Start()
}

// corsMiddleware adds CORS headers to allow frontend requests from any origin (dev mode).
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}
