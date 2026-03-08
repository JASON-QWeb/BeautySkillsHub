package main

import (
	"time"

	"skill-hub/internal/config"
	"skill-hub/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type routeRateLimiters struct {
	authRegister gin.HandlerFunc
	authLogin    gin.HandlerFunc
	reviewRetry  gin.HandlerFunc
	aiChat       gin.HandlerFunc
}

func corsConfigFromConfig(cfg *config.Config) middleware.CORSConfig {
	return middleware.CORSConfig{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		AllowedMethods: cfg.CORSAllowedMethods,
		AllowedHeaders: cfg.CORSAllowedHeaders,
		ExposedHeaders: cfg.CORSExposedHeaders,
		MaxAge:         cfg.CORSMaxAge,
	}
}

func securityHeadersConfigFromConfig(cfg *config.Config) middleware.SecurityHeadersConfig {
	hstsMaxAge := time.Duration(0)
	if cfg.HSTSEnabled {
		hstsMaxAge = cfg.HSTSMaxAge
	}

	return middleware.SecurityHeadersConfig{
		ContentSecurityPolicy:           cfg.SecurityCSP,
		ContentSecurityPolicyReportOnly: cfg.SecurityCSPReportOnly,
		HSTSMaxAge:                      hstsMaxAge,
		HSTSIncludeSubdomains:           cfg.HSTSIncludeSubdomains,
		HSTSPreload:                     cfg.HSTSPreload,
	}
}

func newRateLimitStore(redisClient *redis.Client) middleware.RateLimitStore {
	fallback := middleware.NewMemoryRateLimitStore()
	if redisClient == nil {
		return fallback
	}
	return middleware.NewRedisRateLimitStore(redisClient, fallback)
}

func newRouteRateLimiters(cfg *config.Config, store middleware.RateLimitStore, now func() time.Time) routeRateLimiters {
	return routeRateLimiters{
		authRegister: middleware.NewRateLimitMiddleware(store, middleware.RateLimitPolicy{
			Name:     "auth-register",
			Capacity: cfg.RegisterRateLimitCapacity,
			Window:   cfg.RegisterRateLimitWindow,
		}, middleware.ClientIPIdentity, now, "too many registration attempts"),
		authLogin: middleware.NewRateLimitMiddleware(store, middleware.RateLimitPolicy{
			Name:     "auth-login",
			Capacity: cfg.LoginRateLimitCapacity,
			Window:   cfg.LoginRateLimitWindow,
		}, middleware.ClientIPIdentity, now, "too many login attempts"),
		reviewRetry: middleware.NewRateLimitMiddleware(store, middleware.RateLimitPolicy{
			Name:     "review-retry",
			Capacity: cfg.ReviewRetryRateLimitCapacity,
			Window:   cfg.ReviewRetryRateLimitWindow,
		}, middleware.UserIDOrClientIPIdentity, now, "too many review retry requests"),
		aiChat: middleware.NewRateLimitMiddleware(store, middleware.RateLimitPolicy{
			Name:     "ai-chat",
			Capacity: cfg.AIChatRateLimitCapacity,
			Window:   cfg.AIChatRateLimitWindow,
		}, middleware.UserIDOrClientIPIdentity, now, "too many AI chat requests"),
	}
}
