package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"skill-hub/internal/config"
	"skill-hub/internal/middleware"

	"github.com/gin-gonic/gin"
)

func TestRouteRateLimiters_ProtectSensitiveRoutesWithoutLimitingPublicReads(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		CORSAllowedOrigins:           []string{"https://app.example.com"},
		CORSAllowedMethods:           []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		CORSAllowedHeaders:           []string{"Content-Type", "Authorization"},
		CORSExposedHeaders:           []string{"Content-Disposition"},
		CORSMaxAge:                   10 * time.Minute,
		SecurityCSP:                  "default-src 'none'; frame-ancestors 'none'; base-uri 'none'",
		HSTSEnabled:                  true,
		HSTSMaxAge:                   365 * 24 * time.Hour,
		HSTSIncludeSubdomains:        true,
		LoginRateLimitCapacity:       1,
		LoginRateLimitWindow:         time.Minute,
		ReviewRetryRateLimitCapacity: 1,
		ReviewRetryRateLimitWindow:   time.Minute,
	}
	now := time.Unix(1_700_000_000, 0)
	limiters := newRouteRateLimiters(cfg, middleware.NewMemoryRateLimitStore(), func() time.Time { return now })

	router := gin.New()
	router.Use(middleware.SecurityHeaders(securityHeadersConfigFromConfig(cfg)))
	router.Use(middleware.CORS(corsConfigFromConfig(cfg)))

	api := router.Group("/api")
	api.POST("/auth/login", limiters.authLogin, func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	api.POST("/skills/:id/review/retry",
		func(c *gin.Context) {
			c.Set("userID", uint(7))
			c.Next()
		},
		limiters.reviewRetry,
		func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		},
	)
	api.GET("/skills", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	login1 := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	login1.Header.Set("Origin", "https://app.example.com")
	login1.RemoteAddr = "203.0.113.10:1234"
	login1.Host = "api.example.com"
	login1.Header.Set("X-Forwarded-Proto", "https")
	loginResp1 := httptest.NewRecorder()
	router.ServeHTTP(loginResp1, login1)

	login2 := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	login2.Header.Set("Origin", "https://app.example.com")
	login2.RemoteAddr = "203.0.113.10:1235"
	login2.Host = "api.example.com"
	login2.Header.Set("X-Forwarded-Proto", "https")
	loginResp2 := httptest.NewRecorder()
	router.ServeHTTP(loginResp2, login2)

	retry1 := httptest.NewRequest(http.MethodPost, "/api/skills/1/review/retry", nil)
	retry1.Header.Set("Origin", "https://app.example.com")
	retry1.RemoteAddr = "198.51.100.20:2222"
	retry1.Host = "api.example.com"
	retry1.Header.Set("X-Forwarded-Proto", "https")
	retryResp1 := httptest.NewRecorder()
	router.ServeHTTP(retryResp1, retry1)

	retry2 := httptest.NewRequest(http.MethodPost, "/api/skills/1/review/retry", nil)
	retry2.Header.Set("Origin", "https://app.example.com")
	retry2.RemoteAddr = "198.51.100.21:3333"
	retry2.Host = "api.example.com"
	retry2.Header.Set("X-Forwarded-Proto", "https")
	retryResp2 := httptest.NewRecorder()
	router.ServeHTTP(retryResp2, retry2)

	publicReq := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	publicReq.Header.Set("Origin", "https://app.example.com")
	publicReq.Host = "api.example.com"
	publicReq.Header.Set("X-Forwarded-Proto", "https")
	publicResp := httptest.NewRecorder()
	router.ServeHTTP(publicResp, publicReq)

	if loginResp1.Code != http.StatusNoContent {
		t.Fatalf("expected first login request to pass, got %d", loginResp1.Code)
	}
	if loginResp2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second login request to be limited, got %d", loginResp2.Code)
	}
	if retryResp1.Code != http.StatusNoContent {
		t.Fatalf("expected first review retry to pass, got %d", retryResp1.Code)
	}
	if retryResp2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second review retry for same user to be limited, got %d", retryResp2.Code)
	}
	if publicResp.Code != http.StatusNoContent {
		t.Fatalf("expected public GET route to remain available, got %d", publicResp.Code)
	}
	if got := loginResp2.Header().Get("Retry-After"); got == "" {
		t.Fatal("expected limited login response to include Retry-After")
	}
	if got := loginResp1.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("expected CORS allow origin on guarded route, got %q", got)
	}
	if got := loginResp1.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected security headers on guarded route, got %q", got)
	}
	if got := loginResp1.Header().Get("Strict-Transport-Security"); got == "" {
		t.Fatal("expected HSTS on secure guarded route")
	}
}
