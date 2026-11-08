package svc

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/meme-ai-analyze/api/internal/config"
	riskService "github.com/meme-ai-analyze/api/internal/service/risk"
)

// ServiceContext is the dependency injection container.
type ServiceContext struct {
	Config config.Config

	// Database
	DB *pgxpool.Pool

	// Cache
	Redis *redis.Client

	// Services
	Scanner     *riskService.Scanner
	Analyzer    *riskService.Analyzer
	Historian   *riskService.Historian
	Honeypot    *riskService.HoneypotDetector
	Reporter    *riskService.Reporter
	Oracle      *riskService.Oracle
	Cache       *riskService.CacheManager
	Indexer     *riskService.Indexer
	RugDetector *riskService.RugDetector
}

// NewServiceContext creates a new service context with all dependencies.
func NewServiceContext(c config.Config) *ServiceContext {
	// Connect to PostgreSQL
	pgPool, err := pgxpool.New(context.Background(), c.Postgres.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	log.Println("✅ Connected to PostgreSQL")

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Addr(),
		Password: c.Redis.Password,
		DB:       c.Redis.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("✅ Connected to Redis")

	// Initialize services
	historian := riskService.NewHistorian(pgPool, c.BscScan.APIKey)
	scanner := riskService.NewScanner(c.BSC.RPCUrl, c.BSC.BackupRPCUrl, rdb, historian, c.BscScan.APIKey)
	historian.SetClient(scanner.GetClient())

	analyzer := riskService.NewAnalyzer()
	honeypot := riskService.NewHoneypotDetector(c.BSC.RPCUrl)
	honeypot.SetClient(scanner.GetClient())
	scanner.SetHoneypot(honeypot)

	reporter := riskService.NewReporter(c.AI.APIKey, c.AI.Model)
	oracle := riskService.NewOracle(c.Oracle.ContractAddress, c.Oracle.PrivateKey, c.BSC.RPCUrl)
	cache := riskService.NewCacheManager(rdb)

	// Initialize Indexer and RugDetector
	indexer := riskService.NewIndexer(scanner.GetClient(), pgPool)
	rugDetector := riskService.NewRugDetector(scanner.GetClient(), pgPool, indexer)

	svcCtx := &ServiceContext{
		Config:      c,
		DB:          pgPool,
		Redis:       rdb,
		Scanner:     scanner,
		Analyzer:    analyzer,
		Historian:   historian,
		Honeypot:    honeypot,
		Reporter:    reporter,
		Oracle:      oracle,
		Cache:       cache,
		Indexer:     indexer,
		RugDetector: rugDetector,
	}

	// Start background services
	go indexer.Start(context.Background())
	go rugDetector.Start(context.Background())

	return svcCtx
}

// Close gracefully shuts down all connections.
func (s *ServiceContext) Close() {
	if s.DB != nil {
		s.DB.Close()
	}
	if s.Redis != nil {
		s.Redis.Close()
	}
	log.Println("👋 Service context closed")
}
