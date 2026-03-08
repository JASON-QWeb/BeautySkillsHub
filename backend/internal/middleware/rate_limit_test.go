package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMemoryRateLimitStore_DeniesAfterCapacityAndRecoversAfterWindow(t *testing.T) {
	store := NewMemoryRateLimitStore()
	policy := RateLimitPolicy{
		Name:     "login",
		Capacity: 2,
		Window:   time.Minute,
	}
	now := time.Unix(1_700_000_000, 0)

	first, err := store.Allow(context.Background(), "ip:127.0.0.1", now, policy)
	if err != nil {
		t.Fatalf("first allow: %v", err)
	}
	if !first.Allowed {
		t.Fatal("expected first request to be allowed")
	}

	second, err := store.Allow(context.Background(), "ip:127.0.0.1", now, policy)
	if err != nil {
		t.Fatalf("second allow: %v", err)
	}
	if !second.Allowed {
		t.Fatal("expected second request to be allowed")
	}

	third, err := store.Allow(context.Background(), "ip:127.0.0.1", now, policy)
	if err != nil {
		t.Fatalf("third allow: %v", err)
	}
	if third.Allowed {
		t.Fatal("expected third request to be rate limited")
	}
	if third.RetryAfter <= 0 {
		t.Fatalf("expected positive retry after, got %v", third.RetryAfter)
	}

	recovered, err := store.Allow(context.Background(), "ip:127.0.0.1", now.Add(time.Minute), policy)
	if err != nil {
		t.Fatalf("recovered allow: %v", err)
	}
	if !recovered.Allowed {
		t.Fatal("expected limiter to recover after the window")
	}
}

func TestRateLimitMiddleware_UsesUserIdentityAcrossIPs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := NewMemoryRateLimitStore()
	policy := RateLimitPolicy{
		Name:     "review-retry",
		Capacity: 1,
		Window:   time.Minute,
	}
	now := time.Unix(1_700_000_000, 0)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(42))
		c.Next()
	})
	router.POST("/retry",
		NewRateLimitMiddleware(store, policy, UserIDOrClientIPIdentity, func() time.Time { return now }, "rate limit exceeded"),
		func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		},
	)

	req1 := httptest.NewRequest(http.MethodPost, "/retry", nil)
	req1.RemoteAddr = "203.0.113.10:1234"
	resp1 := httptest.NewRecorder()
	router.ServeHTTP(resp1, req1)

	req2 := httptest.NewRequest(http.MethodPost, "/retry", nil)
	req2.RemoteAddr = "198.51.100.20:4567"
	resp2 := httptest.NewRecorder()
	router.ServeHTTP(resp2, req2)

	if resp1.Code != http.StatusNoContent {
		t.Fatalf("expected first user-scoped request to pass, got %d", resp1.Code)
	}
	if resp2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request for same user to be limited, got %d", resp2.Code)
	}
	if got := resp2.Header().Get("Retry-After"); got == "" {
		t.Fatal("expected Retry-After header to be set")
	}
}

func TestRateLimitMiddleware_FallsBackToClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := NewMemoryRateLimitStore()
	policy := RateLimitPolicy{
		Name:     "auth-login",
		Capacity: 1,
		Window:   time.Minute,
	}
	now := time.Unix(1_700_000_000, 0)

	router := gin.New()
	router.POST("/login",
		NewRateLimitMiddleware(store, policy, UserIDOrClientIPIdentity, func() time.Time { return now }, "too many requests"),
		func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		},
	)

	req1 := httptest.NewRequest(http.MethodPost, "/login", nil)
	req1.RemoteAddr = "203.0.113.10:1234"
	resp1 := httptest.NewRecorder()
	router.ServeHTTP(resp1, req1)

	req2 := httptest.NewRequest(http.MethodPost, "/login", nil)
	req2.RemoteAddr = "203.0.113.10:4567"
	resp2 := httptest.NewRecorder()
	router.ServeHTTP(resp2, req2)

	if resp1.Code != http.StatusNoContent {
		t.Fatalf("expected first IP-scoped request to pass, got %d", resp1.Code)
	}
	if resp2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request from same IP to be limited, got %d", resp2.Code)
	}
}

func TestMemoryRateLimitStore_IsolatesPoliciesForSameIdentity(t *testing.T) {
	store := NewMemoryRateLimitStore()
	now := time.Unix(1_700_000_000, 0)

	loginDecision, err := store.Allow(context.Background(), "ip:127.0.0.1", now, RateLimitPolicy{
		Name:     "auth-login",
		Capacity: 1,
		Window:   time.Minute,
	})
	if err != nil {
		t.Fatalf("login allow: %v", err)
	}
	if !loginDecision.Allowed {
		t.Fatal("expected first login request to pass")
	}

	chatDecision, err := store.Allow(context.Background(), "ip:127.0.0.1", now, RateLimitPolicy{
		Name:     "ai-chat",
		Capacity: 1,
		Window:   time.Minute,
	})
	if err != nil {
		t.Fatalf("chat allow: %v", err)
	}
	if !chatDecision.Allowed {
		t.Fatal("expected a different policy to use its own bucket")
	}
}
