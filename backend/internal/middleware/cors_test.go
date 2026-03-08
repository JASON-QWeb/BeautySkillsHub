package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestCORS_AllowsConfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS(CORSConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		ExposedHeaders: []string{"Content-Disposition"},
		MaxAge:         10 * time.Minute,
	}))
	router.GET("/resource", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Host = "api.example.com"
	req.Header.Set("Origin", "https://app.example.com")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected request to pass, got %d", resp.Code)
	}
	if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("expected reflected origin, got %q", got)
	}
	if got := resp.Header().Get("Access-Control-Expose-Headers"); got != "Content-Disposition" {
		t.Fatalf("expected exposed headers to be set, got %q", got)
	}
	if !headerContainsToken(resp.Header().Values("Vary"), "Origin") {
		t.Fatalf("expected Vary to include Origin, got %v", resp.Header().Values("Vary"))
	}
}

func TestCORS_RejectsDisallowedCrossOriginRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reached := false
	router := gin.New()
	router.Use(CORS(CORSConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		AllowedMethods: []string{http.MethodGet, http.MethodOptions},
		AllowedHeaders: []string{"Content-Type"},
	}))
	router.GET("/resource", func(c *gin.Context) {
		reached = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Host = "api.example.com"
	req.Header.Set("Origin", "https://evil.example.com")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if reached {
		t.Fatal("expected disallowed cross-origin request to be blocked before the handler")
	}
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed origin, got %d", resp.Code)
	}
}

func TestCORS_AllowsSameOriginRequestsWithoutExplicitAllowlist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS(CORSConfig{
		AllowedMethods: []string{http.MethodGet, http.MethodOptions},
		AllowedHeaders: []string{"Content-Type"},
	}))
	router.GET("/resource", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Host = "api.example.com"
	req.Header.Set("Origin", "https://api.example.com")
	req.Header.Set("X-Forwarded-Proto", "https")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected same-origin request to pass, got %d", resp.Code)
	}
	if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "https://api.example.com" {
		t.Fatalf("expected same-origin request to be reflected, got %q", got)
	}
}

func TestCORS_HandlesAllowedPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS(CORSConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		ExposedHeaders: []string{"Content-Disposition"},
		MaxAge:         10 * time.Minute,
	}))
	router.OPTIONS("/resource", func(c *gin.Context) {
		t.Fatal("preflight should be handled by middleware")
	})

	req := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	req.Host = "api.example.com"
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", resp.Code)
	}
	if got := resp.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Fatalf("expected allow methods header, got %q", got)
	}
	if got := resp.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization" {
		t.Fatalf("expected allow headers header, got %q", got)
	}
	if got := resp.Header().Get("Access-Control-Max-Age"); got != "600" {
		t.Fatalf("expected max age of 600 seconds, got %q", got)
	}
	if !headerContainsToken(resp.Header().Values("Vary"), "Access-Control-Request-Method") {
		t.Fatalf("expected Vary to include Access-Control-Request-Method, got %v", resp.Header().Values("Vary"))
	}
}

func headerContainsToken(values []string, want string) bool {
	for _, value := range values {
		for _, token := range strings.Split(value, ",") {
			if strings.TrimSpace(token) == want {
				return true
			}
		}
	}
	return false
}
