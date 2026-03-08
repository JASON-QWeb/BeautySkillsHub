package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	AppEnv        string
	Port          string
	OpenAIKey     string
	OpenAIBaseURL string
	OpenAIModel   string
	UploadDir     string
	ThumbnailDir  string
	DatabaseURL   string

	GitHubSyncEnabled bool
	GitHubToken       string
	GitHubOwner       string
	GitHubRepo        string
	GitHubBranch      string
	GitHubBaseDir     string

	RedisAddr                 string
	RedisPassword             string
	RedisDB                   int
	AISkillsCacheKey          string
	AISkillsInvalidateChannel string

	CORSAllowedOrigins []string
	CORSAllowedMethods []string
	CORSAllowedHeaders []string
	CORSExposedHeaders []string
	CORSMaxAge         time.Duration

	SecurityCSP           string
	SecurityCSPReportOnly bool
	HSTSEnabled           bool
	HSTSMaxAge            time.Duration
	HSTSIncludeSubdomains bool
	HSTSPreload           bool

	LoginRateLimitCapacity       int
	LoginRateLimitWindow         time.Duration
	RegisterRateLimitCapacity    int
	RegisterRateLimitWindow      time.Duration
	ReviewRetryRateLimitCapacity int
	ReviewRetryRateLimitWindow   time.Duration
	AIChatRateLimitCapacity      int
	AIChatRateLimitWindow        time.Duration
}

func Load() *Config {
	loadDotEnv(".env.local")
	loadDotEnv("backend/.env.local")

	appEnv := getEnv("APP_ENV", "local")

	cfg := &Config{
		AppEnv:        appEnv,
		Port:          getEnv("PORT", "8080"),
		OpenAIKey:     getEnv("OPENAI_API_KEY", ""),
		OpenAIBaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAIModel:   getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
		ThumbnailDir:  getEnv("THUMBNAIL_DIR", "./thumbnails"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://skillhub:skillhub@localhost:5432/skillhub_local?sslmode=disable"),

		GitHubSyncEnabled: getEnvBool("GITHUB_SYNC_ENABLED", false),
		GitHubToken:       getEnv("GITHUB_TOKEN", ""),
		GitHubOwner:       getEnv("GITHUB_OWNER", ""),
		GitHubRepo:        getEnv("GITHUB_REPO", ""),
		GitHubBranch:      getEnv("GITHUB_BRANCH", "main"),
		GitHubBaseDir:     getEnv("GITHUB_BASE_DIR", "skills"),

		RedisAddr:                 getEnv("REDIS_ADDR", ""),
		RedisPassword:             getEnv("REDIS_PASSWORD", ""),
		RedisDB:                   getEnvInt("REDIS_DB", 0),
		AISkillsCacheKey:          getEnv("AI_SKILLS_CACHE_KEY", "ai:skills_context:v1"),
		AISkillsInvalidateChannel: getEnv("AI_SKILLS_INVALIDATE_CHANNEL", "ai:skills_context:invalidate"),

		CORSAllowedOrigins: getEnvList("CORS_ALLOWED_ORIGINS", defaultCORSAllowedOrigins(appEnv)),
		CORSAllowedMethods: getEnvList("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		CORSAllowedHeaders: getEnvList("CORS_ALLOWED_HEADERS", []string{"Content-Type", "Authorization", "Accept"}),
		CORSExposedHeaders: getEnvList("CORS_EXPOSED_HEADERS", []string{"Content-Disposition"}),
		CORSMaxAge:         getEnvDuration("CORS_MAX_AGE", 10*time.Minute),

		SecurityCSP:           getEnv("SECURITY_CSP", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'"),
		SecurityCSPReportOnly: getEnvBool("SECURITY_CSP_REPORT_ONLY", false),
		HSTSEnabled:           getEnvBool("HSTS_ENABLED", appEnv != "local"),
		HSTSMaxAge:            getEnvDuration("HSTS_MAX_AGE", 365*24*time.Hour),
		HSTSIncludeSubdomains: getEnvBool("HSTS_INCLUDE_SUBDOMAINS", true),
		HSTSPreload:           getEnvBool("HSTS_PRELOAD", false),

		LoginRateLimitCapacity:       getEnvInt("RATE_LIMIT_LOGIN_CAPACITY", 5),
		LoginRateLimitWindow:         getEnvDuration("RATE_LIMIT_LOGIN_WINDOW", time.Minute),
		RegisterRateLimitCapacity:    getEnvInt("RATE_LIMIT_REGISTER_CAPACITY", 3),
		RegisterRateLimitWindow:      getEnvDuration("RATE_LIMIT_REGISTER_WINDOW", 30*time.Minute),
		ReviewRetryRateLimitCapacity: getEnvInt("RATE_LIMIT_REVIEW_RETRY_CAPACITY", 3),
		ReviewRetryRateLimitWindow:   getEnvDuration("RATE_LIMIT_REVIEW_RETRY_WINDOW", 10*time.Minute),
		AIChatRateLimitCapacity:      getEnvInt("RATE_LIMIT_AI_CHAT_CAPACITY", 20),
		AIChatRateLimitWindow:        getEnvDuration("RATE_LIMIT_AI_CHAT_WINDOW", time.Minute),
	}
	return cfg
}

func defaultCORSAllowedOrigins(appEnv string) []string {
	if strings.EqualFold(strings.TrimSpace(appEnv), "local") {
		return []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"http://localhost:4173",
			"http://127.0.0.1:4173",
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://localhost:8080",
			"http://127.0.0.1:8080",
		}
	}
	return nil
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}

		value = trimDotEnvValue(value)
		if os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

func trimDotEnvValue(value string) string {
	if len(value) < 2 {
		return value
	}

	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		return strings.Trim(value, "\"")
	}

	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return strings.Trim(value, "'")
	}

	if idx := strings.Index(value, " #"); idx >= 0 {
		return strings.TrimSpace(value[:idx])
	}
	return value
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}

	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func getEnvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}

	var out int
	_, err := fmt.Sscanf(v, "%d", &out)
	if err != nil {
		return fallback
	}
	return out
}

func getEnvList(key string, fallback []string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return append([]string(nil), fallback...)
	}

	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
