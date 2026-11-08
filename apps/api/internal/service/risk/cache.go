package risk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/meme-ai-analyze/api/internal/types"
)

// CacheManager manages the Redis caching layer for scan results.
//
// Caching strategy (TTL based on risk level):
//   - Low risk (1-3):     12 hours  — results change slowly
//   - Medium risk (4-6):  1 hour    — moderate volatility
//   - High risk (7-10):   10 min    — tokens can rug at any moment
type CacheManager struct {
	redis *redis.Client
}

// NewCacheManager creates a new CacheManager instance.
func NewCacheManager(rdb *redis.Client) *CacheManager {
	return &CacheManager{redis: rdb}
}

// cacheKey generates a Redis cache key for a token scan result.
func (c *CacheManager) cacheKey(chainID int, tokenAddr string) string {
	return fmt.Sprintf("risk:report:%d:%s", chainID, tokenAddr)
}

// GetOrScan attempts to retrieve a cached scan result.
// Returns the cached response and true if found and not expired.
func (c *CacheManager) GetOrScan(ctx context.Context, chainID int, tokenAddr string) (*types.ScanResponse, bool) {
	key := c.cacheKey(chainID, tokenAddr)

	cached, err := c.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, false // cache miss
	}

	var resp types.ScanResponse
	if err := json.Unmarshal([]byte(cached), &resp); err != nil {
		return nil, false // corrupted cache
	}

	// Check if the cache is expired based on risk level TTL
	ttl := c.calcExpiry(resp.RiskScore)
	if time.Now().Unix() > resp.ScannedAt+ttl {
		return nil, false // expired
	}

	return &resp, true
}

// Set stores a scan response in Redis with the appropriate TTL.
func (c *CacheManager) Set(ctx context.Context, chainID int, resp *types.ScanResponse) error {
	key := c.cacheKey(chainID, resp.TokenAddress)

	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal cache data: %w", err)
	}

	ttl := time.Duration(c.calcExpiry(resp.RiskScore)) * time.Second
	return c.redis.Set(ctx, key, data, ttl).Err()
}

// Invalidate removes a cached scan result (e.g., after a re-scan).
func (c *CacheManager) Invalidate(ctx context.Context, chainID int, tokenAddr string) error {
	key := c.cacheKey(chainID, tokenAddr)
	return c.redis.Del(ctx, key).Err()
}

// IsRateLimited checks if a request should be rate-limited.
// Simple implementation: 10 requests per minute per IP.
func (c *CacheManager) IsRateLimited(ctx context.Context, key string, limit int, window time.Duration) bool {
	count, err := c.redis.Incr(ctx, "ratelimit:"+key).Result()
	if err != nil {
		return false // fail open
	}

	// Set expiry on first request in the window
	if count == 1 {
		c.redis.Expire(ctx, "ratelimit:"+key, window)
	}

	return count > int64(limit)
}

// calcExpiry returns the cache TTL (in seconds) based on risk score.
func (c *CacheManager) calcExpiry(score float64) int64 {
	switch {
	case score <= 3:
		return 12 * 3600 // 12 hours
	case score <= 6:
		return 3600 // 1 hour
	default:
		return 600 // 10 minutes (high-risk tokens change quickly)
	}
}

// GetStats retrieves global platform statistics from Redis.
func (c *CacheManager) GetStats(ctx context.Context) map[string]int64 {
	stats := map[string]int64{
		"total_scans":   0,
		"total_reports": 0,
	}

	if val, err := c.redis.Get(ctx, "stats:total_scans").Int64(); err == nil {
		stats["total_scans"] = val
	}
	if val, err := c.redis.Get(ctx, "stats:total_reports").Int64(); err == nil {
		stats["total_reports"] = val
	}

	return stats
}

// IncrStats atomically increments a platform statistic.
func (c *CacheManager) IncrStats(ctx context.Context, stat string) error {
	return c.redis.Incr(ctx, "stats:"+stat).Err()
}
