package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders_SetsBaselineHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders(SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'none'; frame-ancestors 'none'; base-uri 'none'",
		HSTSMaxAge:            365 * 24 * time.Hour,
		HSTSIncludeSubdomains: true,
	}))
	router.GET("/resource", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if got := resp.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected X-Frame-Options DENY, got %q", got)
	}
	if got := resp.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected nosniff, got %q", got)
	}
	if got := resp.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("expected strict referrer policy, got %q", got)
	}
	if got := resp.Header().Get("Permissions-Policy"); got == "" {
		t.Fatal("expected permissions policy header to be set")
	}
	if got := resp.Header().Get("Cross-Origin-Opener-Policy"); got != "same-origin" {
		t.Fatalf("expected COOP same-origin, got %q", got)
	}
	if got := resp.Header().Get("Cross-Origin-Resource-Policy"); got != "same-site" {
		t.Fatalf("expected CORP same-site, got %q", got)
	}
	if got := resp.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatal("expected CSP header to be set")
	}
	if got := resp.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("expected no HSTS on insecure request, got %q", got)
	}
}

func TestSecurityHeaders_UsesReportOnlyWhenConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders(SecurityHeadersConfig{
		ContentSecurityPolicy:           "default-src 'none'; frame-ancestors 'none'; base-uri 'none'",
		ContentSecurityPolicyReportOnly: true,
	}))
	router.GET("/resource", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if got := resp.Header().Get("Content-Security-Policy"); got != "" {
		t.Fatalf("expected CSP header to be omitted in report-only mode, got %q", got)
	}
	if got := resp.Header().Get("Content-Security-Policy-Report-Only"); got == "" {
		t.Fatal("expected report-only CSP header to be set")
	}
}

func TestSecurityHeaders_AddsHSTSForSecureRequestsOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders(SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'none'; frame-ancestors 'none'; base-uri 'none'",
		HSTSMaxAge:            365 * 24 * time.Hour,
		HSTSIncludeSubdomains: true,
		HSTSPreload:           true,
	}))
	router.GET("/resource", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if got := resp.Header().Get("Strict-Transport-Security"); got != "max-age=31536000; includeSubDomains; preload" {
		t.Fatalf("expected HSTS header for secure request, got %q", got)
	}
}
