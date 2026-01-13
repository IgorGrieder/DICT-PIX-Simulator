package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/dict-simulator/go/internal/config"
	"github.com/dict-simulator/go/internal/db"
)

// RateLimitResult contains rate limit information
type RateLimitResult struct {
	Allowed   bool
	Limit     int
	Remaining int
	Reset     int64
}

// RateLimiterMiddleware implements token bucket rate limiting using Redis
func RateLimiterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting if disabled (for benchmarks)
		if !config.Env.RateLimitEnabled {
			next.ServeHTTP(w, r)
			return
		}

		userID := r.Header.Get("X-User-Id")
		if userID == "" {
			userID = "anonymous"
		}

		result, err := checkRateLimit(r.Context(), userID)
		if err != nil {
			// If Redis fails, allow the request but log the error
			next.ServeHTTP(w, r)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.Reset, 10))

		if !result.Allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "TOO_MANY_REQUESTS",
				"message": "Rate limit exceeded. Please try again later.",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func checkRateLimit(ctx context.Context, userID string) (*RateLimitResult, error) {
	bucketSize := config.Env.RateLimitBucketSize
	refillSeconds := config.Env.RateLimitRefillSeconds

	key := fmt.Sprintf("rate_limit:%s", userID)

	currentTokens, err := db.RedisClient.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	resetTime := time.Now().Unix() + int64(refillSeconds)

	// Key doesn't exist, create new bucket
	if currentTokens == "" {
		remaining := bucketSize - 1
		err = db.RedisClient.Set(ctx, key, strconv.Itoa(remaining), time.Duration(refillSeconds)*time.Second).Err()
		if err != nil {
			return nil, err
		}
		return &RateLimitResult{
			Allowed:   true,
			Limit:     bucketSize,
			Remaining: remaining,
			Reset:     resetTime,
		}, nil
	}

	tokens, _ := strconv.Atoi(currentTokens)

	if tokens <= 0 {
		return &RateLimitResult{
			Allowed:   false,
			Limit:     bucketSize,
			Remaining: 0,
			Reset:     resetTime,
		}, nil
	}

	remaining := tokens - 1
	err = db.RedisClient.Set(ctx, key, strconv.Itoa(remaining), time.Duration(refillSeconds)*time.Second).Err()
	if err != nil {
		return nil, err
	}

	return &RateLimitResult{
		Allowed:   true,
		Limit:     bucketSize,
		Remaining: remaining,
		Reset:     resetTime,
	}, nil
}
