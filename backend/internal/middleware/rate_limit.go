package middleware

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimitPolicy struct {
	Name     string
	Capacity int
	Window   time.Duration
}

type RateLimitDecision struct {
	Allowed    bool
	Remaining  int
	RetryAfter time.Duration
}

type RateLimitStore interface {
	Allow(ctx context.Context, key string, now time.Time, policy RateLimitPolicy) (RateLimitDecision, error)
}

type IdentityFunc func(*gin.Context) (string, bool)

type MemoryRateLimitStore struct {
	mu     sync.Mutex
	states map[string]tokenBucketState
}

type tokenBucketState struct {
	Tokens    float64
	UpdatedAt time.Time
}

type RedisRateLimitStore struct {
	client   *redis.Client
	fallback RateLimitStore
}

func NewMemoryRateLimitStore() *MemoryRateLimitStore {
	return &MemoryRateLimitStore{
		states: make(map[string]tokenBucketState),
	}
}

func (s *MemoryRateLimitStore) Allow(_ context.Context, key string, now time.Time, policy RateLimitPolicy) (RateLimitDecision, error) {
	if limiterDisabled(policy) {
		return RateLimitDecision{Allowed: true}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key = memoryRateLimitKey(policy.Name, key)
	state := s.states[key]
	state = refillBucket(state, now, policy)

	decision := consumeToken(state, now, policy)
	s.states[key] = decision.state

	return decision.result, nil
}

func NewRedisRateLimitStore(client *redis.Client, fallback RateLimitStore) *RedisRateLimitStore {
	return &RedisRateLimitStore{
		client:   client,
		fallback: fallback,
	}
}

func (s *RedisRateLimitStore) Allow(ctx context.Context, key string, now time.Time, policy RateLimitPolicy) (RateLimitDecision, error) {
	if limiterDisabled(policy) || s == nil || s.client == nil {
		if s != nil && s.fallback != nil {
			return s.fallback.Allow(ctx, key, now, policy)
		}
		return RateLimitDecision{Allowed: true}, nil
	}

	resp, err := s.client.Eval(ctx, redisTokenBucketScript, []string{redisRateLimitKey(policy.Name, key)}, policy.Capacity, policy.Window.Milliseconds(), now.UnixMilli()).Result()
	if err != nil {
		if s.fallback != nil {
			return s.fallback.Allow(ctx, key, now, policy)
		}
		return RateLimitDecision{}, err
	}

	values, ok := resp.([]interface{})
	if !ok || len(values) != 3 {
		if s.fallback != nil {
			return s.fallback.Allow(ctx, key, now, policy)
		}
		return RateLimitDecision{}, fmt.Errorf("unexpected redis rate limit response: %T", resp)
	}

	allowed, err := redisInt64(values[0])
	if err != nil {
		return RateLimitDecision{}, err
	}
	remaining, err := redisInt64(values[1])
	if err != nil {
		return RateLimitDecision{}, err
	}
	retryAfterMs, err := redisInt64(values[2])
	if err != nil {
		return RateLimitDecision{}, err
	}

	return RateLimitDecision{
		Allowed:    allowed == 1,
		Remaining:  int(remaining),
		RetryAfter: time.Duration(retryAfterMs) * time.Millisecond,
	}, nil
}

func NewRateLimitMiddleware(store RateLimitStore, policy RateLimitPolicy, identity IdentityFunc, now func() time.Time, message string) gin.HandlerFunc {
	if strings.TrimSpace(message) == "" {
		message = "rate limit exceeded"
	}
	if now == nil {
		now = time.Now
	}
	if identity == nil {
		identity = ClientIPIdentity
	}

	return func(c *gin.Context) {
		if store == nil || limiterDisabled(policy) {
			c.Next()
			return
		}

		identityValue, ok := identity(c)
		if !ok {
			c.Next()
			return
		}

		decision, err := store.Allow(c.Request.Context(), identityValue, now(), policy)
		if err != nil {
			log.Printf("rate limiter fallback: %v", err)
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(policy.Capacity))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(maxInt(decision.Remaining, 0)))

		if !decision.Allowed {
			retryAfterSeconds := retryAfterSeconds(decision.RetryAfter)
			c.Header("Retry-After", strconv.FormatInt(retryAfterSeconds, 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       message,
				"retry_after": retryAfterSeconds,
			})
			return
		}

		c.Next()
	}
}

func ClientIPIdentity(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		return "", false
	}
	return "ip:" + ip, true
}

func UserIDOrClientIPIdentity(c *gin.Context) (string, bool) {
	if c != nil {
		if rawUserID, ok := c.Get("userID"); ok {
			switch userID := rawUserID.(type) {
			case uint:
				if userID > 0 {
					return fmt.Sprintf("user:%d", userID), true
				}
			case int:
				if userID > 0 {
					return fmt.Sprintf("user:%d", userID), true
				}
			case int64:
				if userID > 0 {
					return fmt.Sprintf("user:%d", userID), true
				}
			}
		}
	}
	return ClientIPIdentity(c)
}

func limiterDisabled(policy RateLimitPolicy) bool {
	return strings.TrimSpace(policy.Name) == "" || policy.Capacity <= 0 || policy.Window <= 0
}

func refillBucket(state tokenBucketState, now time.Time, policy RateLimitPolicy) tokenBucketState {
	if state.UpdatedAt.IsZero() {
		return tokenBucketState{
			Tokens:    float64(policy.Capacity),
			UpdatedAt: now,
		}
	}

	if now.Before(state.UpdatedAt) {
		now = state.UpdatedAt
	}

	refill := (float64(now.Sub(state.UpdatedAt)) / float64(policy.Window)) * float64(policy.Capacity)
	state.Tokens = math.Min(float64(policy.Capacity), state.Tokens+refill)
	state.UpdatedAt = now
	return state
}

type tokenBucketDecision struct {
	state  tokenBucketState
	result RateLimitDecision
}

func consumeToken(state tokenBucketState, now time.Time, policy RateLimitPolicy) tokenBucketDecision {
	ratePerNanosecond := float64(policy.Capacity) / float64(policy.Window)
	if state.Tokens < 1 {
		retryAfter := time.Duration(math.Ceil((1-state.Tokens)/ratePerNanosecond)) * time.Nanosecond
		state.UpdatedAt = now
		return tokenBucketDecision{
			state: state,
			result: RateLimitDecision{
				Allowed:    false,
				Remaining:  0,
				RetryAfter: retryAfter,
			},
		}
	}

	state.Tokens -= 1
	state.UpdatedAt = now
	return tokenBucketDecision{
		state: state,
		result: RateLimitDecision{
			Allowed:   true,
			Remaining: int(math.Floor(state.Tokens)),
		},
	}
}

func retryAfterSeconds(d time.Duration) int64 {
	if d <= 0 {
		return 1
	}
	return int64(math.Ceil(d.Seconds()))
}

func redisInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case []byte:
		return strconv.ParseInt(string(v), 10, 64)
	default:
		return 0, fmt.Errorf("unexpected redis numeric type %T", value)
	}
}

func redisRateLimitKey(policyName, identity string) string {
	return "ratelimit:" + strings.TrimSpace(policyName) + ":" + strings.TrimSpace(identity)
}

func memoryRateLimitKey(policyName, identity string) string {
	return strings.TrimSpace(policyName) + ":" + strings.TrimSpace(identity)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

const redisTokenBucketScript = `
local capacity = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local key = KEYS[1]

local data = redis.call("HMGET", key, "tokens", "ts")
local tokens = tonumber(data[1])
local ts = tonumber(data[2])

if tokens == nil then
  tokens = capacity
  ts = now_ms
end

if now_ms > ts then
  local refill = ((now_ms - ts) / window_ms) * capacity
  tokens = math.min(capacity, tokens + refill)
  ts = now_ms
end

if tokens < 1 then
  local rate = capacity / window_ms
  local retry_ms = math.ceil((1 - tokens) / rate)
  redis.call("HMSET", key, "tokens", tokens, "ts", ts)
  redis.call("PEXPIRE", key, math.max(window_ms * 2, retry_ms))
  return {0, math.floor(tokens), retry_ms}
end

tokens = tokens - 1
redis.call("HMSET", key, "tokens", tokens, "ts", ts)
redis.call("PEXPIRE", key, window_ms * 2)
return {1, math.floor(tokens), 0}
`
