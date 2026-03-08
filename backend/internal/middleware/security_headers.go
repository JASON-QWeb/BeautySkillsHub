package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type SecurityHeadersConfig struct {
	ContentSecurityPolicy           string
	ContentSecurityPolicyReportOnly bool
	HSTSMaxAge                      time.Duration
	HSTSIncludeSubdomains           bool
	HSTSPreload                     bool
}

func SecurityHeaders(cfg SecurityHeadersConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()")
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Header("Cross-Origin-Resource-Policy", "same-site")

		csp := strings.TrimSpace(cfg.ContentSecurityPolicy)
		if csp != "" {
			if cfg.ContentSecurityPolicyReportOnly {
				c.Header("Content-Security-Policy-Report-Only", csp)
			} else {
				c.Header("Content-Security-Policy", csp)
			}
		}

		if cfg.HSTSMaxAge > 0 && isSecureRequest(c.Request) {
			c.Header("Strict-Transport-Security", buildHSTSValue(cfg))
		}

		c.Next()
	}
}

func buildHSTSValue(cfg SecurityHeadersConfig) string {
	parts := []string{"max-age=" + strconv.FormatInt(int64(cfg.HSTSMaxAge/time.Second), 10)}
	if cfg.HSTSIncludeSubdomains {
		parts = append(parts, "includeSubDomains")
	}
	if cfg.HSTSPreload {
		parts = append(parts, "preload")
	}
	return strings.Join(parts, "; ")
}

func isSecureRequest(req *http.Request) bool {
	if req == nil {
		return false
	}
	if req.TLS != nil {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(req.Header.Get("X-Forwarded-Proto")), "https") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(req.Header.Get("X-Forwarded-SSL")), "on")
}
