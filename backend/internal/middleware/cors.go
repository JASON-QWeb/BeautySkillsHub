package middleware

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	ExposedHeaders []string
	MaxAge         time.Duration
}

func CORS(cfg CORSConfig) gin.HandlerFunc {
	allowedOrigins := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		normalized := normalizeOrigin(origin)
		if normalized == "" {
			continue
		}
		allowedOrigins[normalized] = struct{}{}
	}

	allowMethods := joinOrDefault(cfg.AllowedMethods, []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions})
	allowHeaders := joinOrDefault(cfg.AllowedHeaders, []string{"Content-Type", "Authorization", "Accept"})
	exposedHeaders := joinOrDefault(cfg.ExposedHeaders, []string{"Content-Disposition"})

	return func(c *gin.Context) {
		origin := normalizeOrigin(c.GetHeader("Origin"))
		if origin == "" {
			c.Next()
			return
		}

		addVaryHeader(c.Writer.Header(), "Origin")
		if c.Request.Method == http.MethodOptions {
			addVaryHeader(c.Writer.Header(), "Access-Control-Request-Method")
			addVaryHeader(c.Writer.Header(), "Access-Control-Request-Headers")
		}

		if !originAllowed(origin, c.Request, allowedOrigins) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "origin not allowed"})
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Expose-Headers", exposedHeaders)

		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Methods", allowMethods)
			c.Header("Access-Control-Allow-Headers", allowHeaders)
			if cfg.MaxAge > 0 {
				c.Header("Access-Control-Max-Age", strconv.FormatInt(int64(cfg.MaxAge/time.Second), 10))
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func originAllowed(origin string, req *http.Request, allowedOrigins map[string]struct{}) bool {
	if origin == "" {
		return true
	}
	if origin == requestOrigin(req) {
		return true
	}
	_, ok := allowedOrigins[origin]
	return ok
}

func requestOrigin(req *http.Request) string {
	if req == nil || strings.TrimSpace(req.Host) == "" {
		return ""
	}
	scheme := "http"
	if isSecureRequest(req) {
		scheme = "https"
	}
	return scheme + "://" + req.Host
}

func normalizeOrigin(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return ""
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func joinOrDefault(values, fallback []string) string {
	if len(values) == 0 {
		values = fallback
	}
	return strings.Join(values, ", ")
}

func addVaryHeader(header http.Header, value string) {
	existing := header.Values("Vary")
	for _, current := range existing {
		for _, token := range strings.Split(current, ",") {
			if strings.EqualFold(strings.TrimSpace(token), value) {
				return
			}
		}
	}
	header.Add("Vary", value)
}
