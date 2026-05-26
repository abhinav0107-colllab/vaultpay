package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// We embed our raw Lua script directly into a string for clean executable distribution
const luaScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
local current = redis.call('ZCARD', key)
if current >= limit then
    return 0
else
    redis.call('ZADD', key, now, now)
    redis.call('EXPIRE', key, window)
    return 1
end`

type RateLimiterMiddleware struct {
	rdb *redis.Client
}

func NewRateLimiterMiddleware(rdb *redis.Client) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{rdb: rdb}
}

func (rl *RateLimiterMiddleware) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the authenticated key (In a full app, this comes from your Auth context middleware)
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = "anonymous_global_client"
		}

		redisKey := "ratelimit:" + apiKey
		now := time.Now().Unix()
		window := 60  // 60 seconds tracking interval
		limit := 1000 // Max 1000 requests per minute default

		// Execute our Lua code atomically directly on the Redis engine cache core
		res, err := rl.rdb.Eval(context.Background(), luaScript, []string{redisKey}, now, window, limit).Result()
		if err != nil {
			http.Error(w, "Rate limiting calculation failure", http.StatusInternalServerError)
			return
		}

		// If the script returns 0, the sliding window capacity is full! Block them.
		if res.(int64) == 0 {
			w.Header().Set("Retry-After", strconv.Itoa(window))
			http.Error(w, "Too Many Requests - Rate limit exceeded. Slow down your pipelines.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
