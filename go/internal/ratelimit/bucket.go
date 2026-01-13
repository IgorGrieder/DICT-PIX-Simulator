package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Bucket implements a token bucket rate limiter using Redis
type Bucket struct {
	client *redis.Client
}

// BucketState represents the current state of a rate limit bucket
type BucketState struct {
	Allowed   bool       // whether the request is allowed
	Remaining int        // tokens remaining after this request
	Reset     int64      // unix timestamp when bucket refills
	Policy    PolicyName // which policy this state belongs to
}

// NewBucket creates a new rate limiter bucket backed by Redis
func NewBucket(client *redis.Client) *Bucket {
	return &Bucket{client: client}
}

// key generates the Redis key for a specific policy and identifier
// Format: rate_limit:{policy}:{identifier}
func (b *Bucket) key(policy PolicyName, identifier string) string {
	return fmt.Sprintf("rate_limit:%s:%s", policy, identifier)
}

// tokensKey stores the current token count
func (b *Bucket) tokensKey(policy PolicyName, identifier string) string {
	return b.key(policy, identifier) + ":tokens"
}

// lastRefillKey stores the last refill timestamp
func (b *Bucket) lastRefillKey(policy PolicyName, identifier string) string {
	return b.key(policy, identifier) + ":last_refill"
}

// Check verifies if a request is allowed (pre-request check)
// This does NOT deduct tokens - use Consume for that
func (b *Bucket) Check(ctx context.Context, policy Policy, identifier string) (*BucketState, error) {
	tokens, err := b.getTokensWithRefill(ctx, policy, identifier)
	if err != nil {
		return nil, err
	}

	// Calculate reset time (next minute boundary)
	resetTime := time.Now().Add(time.Minute).Unix()

	return &BucketState{
		Allowed:   tokens > 0,
		Remaining: tokens,
		Reset:     resetTime,
		Policy:    policy.Name,
	}, nil
}

// Consume deducts tokens from the bucket after the response is known
// The cost depends on the HTTP status code per DICT spec
func (b *Bucket) Consume(ctx context.Context, policy Policy, identifier string, statusCode int) error {
	cost := policy.CostForStatus(statusCode)
	if cost == 0 {
		return nil
	}

	return b.deduct(ctx, policy, identifier, cost)
}

// getTokensWithRefill gets current tokens, applying refill if needed
func (b *Bucket) getTokensWithRefill(ctx context.Context, policy Policy, identifier string) (int, error) {
	tokensKey := b.tokensKey(policy.Name, identifier)
	lastRefillKey := b.lastRefillKey(policy.Name, identifier)

	// Lua script for atomic token bucket with refill
	// This ensures thread-safety across multiple instances
	script := redis.NewScript(`
		local tokens_key = KEYS[1]
		local last_refill_key = KEYS[2]
		local bucket_size = tonumber(ARGV[1])
		local refill_rate = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])

		-- Get current values
		local tokens = tonumber(redis.call('GET', tokens_key) or bucket_size)
		local last_refill = tonumber(redis.call('GET', last_refill_key) or now)

		-- Calculate refill
		local elapsed_minutes = (now - last_refill) / 60
		local refill_amount = math.floor(elapsed_minutes * refill_rate)
		
		if refill_amount > 0 then
			tokens = math.min(bucket_size, tokens + refill_amount)
			redis.call('SET', tokens_key, tokens)
			redis.call('SET', last_refill_key, now)
		end

		-- Set TTL to prevent stale keys (2x refill period)
		local ttl = 120
		redis.call('EXPIRE', tokens_key, ttl)
		redis.call('EXPIRE', last_refill_key, ttl)

		return tokens
	`)

	now := time.Now().Unix()
	result, err := script.Run(ctx, b.client, []string{tokensKey, lastRefillKey},
		policy.BucketSize, policy.RefillRate, now).Int()

	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, err
	}

	return result, nil
}

// deduct removes tokens from the bucket
func (b *Bucket) deduct(ctx context.Context, policy Policy, identifier string, cost int) error {
	tokensKey := b.tokensKey(policy.Name, identifier)

	// Lua script for atomic deduction
	script := redis.NewScript(`
		local tokens_key = KEYS[1]
		local cost = tonumber(ARGV[1])
		local bucket_size = tonumber(ARGV[2])

		local tokens = tonumber(redis.call('GET', tokens_key) or bucket_size)
		tokens = math.max(0, tokens - cost)
		redis.call('SET', tokens_key, tokens)
		redis.call('EXPIRE', tokens_key, 120)

		return tokens
	`)

	_, err := script.Run(ctx, b.client, []string{tokensKey}, cost, policy.BucketSize).Int()
	return err
}

// GetState returns the current bucket state without modifying it
func (b *Bucket) GetState(ctx context.Context, policy Policy, identifier string) (*BucketState, error) {
	return b.Check(ctx, policy, identifier)
}

// Reset resets a bucket to full capacity (useful for testing)
func (b *Bucket) Reset(ctx context.Context, policy Policy, identifier string) error {
	tokensKey := b.tokensKey(policy.Name, identifier)
	lastRefillKey := b.lastRefillKey(policy.Name, identifier)

	pipe := b.client.Pipeline()
	pipe.Set(ctx, tokensKey, strconv.Itoa(policy.BucketSize), 2*time.Minute)
	pipe.Set(ctx, lastRefillKey, strconv.FormatInt(time.Now().Unix(), 10), 2*time.Minute)
	_, err := pipe.Exec(ctx)

	return err
}
