package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_UsesDotEnvLocalValuesWhenProcessEnvIsUnset(t *testing.T) {
	tmp := t.TempDir()
	envPath := filepath.Join(tmp, ".env.local")
	content := "APP_ENV=local\nPORT=9090\nOPENAI_API_KEY=test-key\nOPENAI_BASE_URL=http://localhost:11434/v1\nOPENAI_MODEL=gpt-4.1-mini\nDATABASE_URL=postgres://tester:secret@localhost:5432/skillhub?sslmode=disable\nUPLOAD_DIR=./tmp-uploads\nTHUMBNAIL_DIR=./tmp-thumbs\n"
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env.local: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".env"), []byte("PORT=9999\nOPENAI_MODEL=ignored-model\n"), 0o600); err != nil {
		t.Fatalf("write ignored .env: %v", err)
	}

	t.Setenv("APP_ENV", "")
	t.Setenv("PORT", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "")
	t.Setenv("OPENAI_MODEL", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("UPLOAD_DIR", "")
	t.Setenv("THUMBNAIL_DIR", "")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cfg := Load()
	if cfg.AppEnv != "local" {
		t.Fatalf("expected app env local, got %q", cfg.AppEnv)
	}
	if cfg.Port != "9090" {
		t.Fatalf("expected port 9090, got %q", cfg.Port)
	}
	if cfg.OpenAIKey != "test-key" {
		t.Fatalf("expected OpenAI key from .env.local, got %q", cfg.OpenAIKey)
	}
	if cfg.OpenAIBaseURL != "http://localhost:11434/v1" {
		t.Fatalf("expected base url from .env.local, got %q", cfg.OpenAIBaseURL)
	}
	if cfg.OpenAIModel != "gpt-4.1-mini" {
		t.Fatalf("expected model from .env.local, got %q", cfg.OpenAIModel)
	}
	if cfg.DatabaseURL != "postgres://tester:secret@localhost:5432/skillhub?sslmode=disable" {
		t.Fatalf("expected database url from .env.local, got %q", cfg.DatabaseURL)
	}
	if cfg.UploadDir != "./tmp-uploads" {
		t.Fatalf("expected upload dir from .env.local, got %q", cfg.UploadDir)
	}
	if cfg.ThumbnailDir != "./tmp-thumbs" {
		t.Fatalf("expected thumbnail dir from .env.local, got %q", cfg.ThumbnailDir)
	}
}

func TestLoad_ProcessEnvOverridesDotEnvLocal(t *testing.T) {
	tmp := t.TempDir()
	envPath := filepath.Join(tmp, ".env.local")
	content := "OPENAI_MODEL=file-model\nOPENAI_API_KEY=file-key\n"
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env.local: %v", err)
	}

	t.Setenv("OPENAI_MODEL", "env-model")
	t.Setenv("OPENAI_API_KEY", "env-key")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cfg := Load()
	if cfg.OpenAIModel != "env-model" {
		t.Fatalf("expected process env model to win, got %q", cfg.OpenAIModel)
	}
	if cfg.OpenAIKey != "env-key" {
		t.Fatalf("expected process env key to win, got %q", cfg.OpenAIKey)
	}
}

func TestLoad_DefaultModelWhenUnset(t *testing.T) {
	tmp := t.TempDir()

	t.Setenv("APP_ENV", "")
	t.Setenv("OPENAI_MODEL", "")
	t.Setenv("DATABASE_URL", "")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cfg := Load()
	if cfg.AppEnv != "local" {
		t.Fatalf("expected default app env local, got %q", cfg.AppEnv)
	}
	if cfg.OpenAIModel != "gpt-4o-mini" {
		t.Fatalf("expected default model gpt-4o-mini, got %q", cfg.OpenAIModel)
	}
	if cfg.DatabaseURL != "postgres://skillhub:skillhub@localhost:5432/skillhub_local?sslmode=disable" {
		t.Fatalf("expected default local database url, got %q", cfg.DatabaseURL)
	}
}

func TestLoad_GitHubDefaults(t *testing.T) {
	tmp := t.TempDir()

	t.Setenv("GITHUB_SYNC_ENABLED", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GITHUB_OWNER", "")
	t.Setenv("GITHUB_REPO", "")
	t.Setenv("GITHUB_BRANCH", "")
	t.Setenv("GITHUB_BASE_DIR", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DB", "")
	t.Setenv("AI_SKILLS_CACHE_KEY", "")
	t.Setenv("AI_SKILLS_INVALIDATE_CHANNEL", "")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cfg := Load()
	if cfg.GitHubSyncEnabled {
		t.Fatal("expected github sync disabled by default")
	}
	if cfg.GitHubBranch != "main" {
		t.Fatalf("expected default branch main, got %q", cfg.GitHubBranch)
	}
	if cfg.GitHubBaseDir != "skills" {
		t.Fatalf("expected default base dir skills, got %q", cfg.GitHubBaseDir)
	}
	if cfg.RedisDB != 0 {
		t.Fatalf("expected default redis db 0, got %d", cfg.RedisDB)
	}
	if cfg.AISkillsCacheKey != "ai:skills_context:v1" {
		t.Fatalf("expected default cache key, got %q", cfg.AISkillsCacheKey)
	}
	if cfg.AISkillsInvalidateChannel != "ai:skills_context:invalidate" {
		t.Fatalf("expected default invalidate channel, got %q", cfg.AISkillsInvalidateChannel)
	}
	if len(cfg.CORSAllowedOrigins) == 0 {
		t.Fatal("expected local development CORS origins to be populated by default")
	}
	if cfg.CORSMaxAge != 10*time.Minute {
		t.Fatalf("expected default CORS max age 10m, got %v", cfg.CORSMaxAge)
	}
	if cfg.SecurityCSP == "" {
		t.Fatal("expected default CSP to be set")
	}
	if cfg.HSTSMaxAge != 365*24*time.Hour {
		t.Fatalf("expected default HSTS max age 365 days, got %v", cfg.HSTSMaxAge)
	}
	if cfg.LoginRateLimitCapacity != 5 {
		t.Fatalf("expected default login rate limit capacity 5, got %d", cfg.LoginRateLimitCapacity)
	}
	if cfg.LoginRateLimitWindow != time.Minute {
		t.Fatalf("expected default login rate limit window 1m, got %v", cfg.LoginRateLimitWindow)
	}
}

func TestLoad_SecurityOverrides(t *testing.T) {
	tmp := t.TempDir()

	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://skillhub:secret@db:5432/skillhub?sslmode=require")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com, https://admin.example.com")
	t.Setenv("CORS_ALLOWED_METHODS", "GET,POST,OPTIONS")
	t.Setenv("CORS_ALLOWED_HEADERS", "Content-Type,Authorization")
	t.Setenv("CORS_EXPOSED_HEADERS", "Content-Disposition,X-Request-Id")
	t.Setenv("CORS_MAX_AGE", "15m")
	t.Setenv("SECURITY_CSP", "default-src 'none'; frame-ancestors 'none'")
	t.Setenv("SECURITY_CSP_REPORT_ONLY", "true")
	t.Setenv("HSTS_ENABLED", "true")
	t.Setenv("HSTS_MAX_AGE", "720h")
	t.Setenv("HSTS_INCLUDE_SUBDOMAINS", "true")
	t.Setenv("HSTS_PRELOAD", "true")
	t.Setenv("RATE_LIMIT_LOGIN_CAPACITY", "7")
	t.Setenv("RATE_LIMIT_LOGIN_WINDOW", "2m")
	t.Setenv("RATE_LIMIT_REGISTER_CAPACITY", "3")
	t.Setenv("RATE_LIMIT_REGISTER_WINDOW", "30m")
	t.Setenv("RATE_LIMIT_REVIEW_RETRY_CAPACITY", "4")
	t.Setenv("RATE_LIMIT_REVIEW_RETRY_WINDOW", "10m")
	t.Setenv("RATE_LIMIT_AI_CHAT_CAPACITY", "25")
	t.Setenv("RATE_LIMIT_AI_CHAT_WINDOW", "1m")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cfg := Load()
	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Fatalf("expected 2 cors origins, got %d", len(cfg.CORSAllowedOrigins))
	}
	if cfg.CORSAllowedOrigins[0] != "https://app.example.com" {
		t.Fatalf("unexpected first origin %q", cfg.CORSAllowedOrigins[0])
	}
	if cfg.CORSAllowedMethods[2] != "OPTIONS" {
		t.Fatalf("expected methods to be parsed, got %v", cfg.CORSAllowedMethods)
	}
	if cfg.CORSExposedHeaders[1] != "X-Request-Id" {
		t.Fatalf("expected exposed headers to be parsed, got %v", cfg.CORSExposedHeaders)
	}
	if cfg.CORSMaxAge != 15*time.Minute {
		t.Fatalf("expected CORS max age 15m, got %v", cfg.CORSMaxAge)
	}
	if !cfg.SecurityCSPReportOnly {
		t.Fatal("expected CSP report-only to be enabled")
	}
	if !cfg.HSTSEnabled {
		t.Fatal("expected HSTS to be enabled")
	}
	if cfg.HSTSMaxAge != 720*time.Hour {
		t.Fatalf("expected HSTS max age 720h, got %v", cfg.HSTSMaxAge)
	}
	if !cfg.HSTSIncludeSubdomains {
		t.Fatal("expected HSTS includeSubDomains to be enabled")
	}
	if !cfg.HSTSPreload {
		t.Fatal("expected HSTS preload to be enabled")
	}
	if cfg.LoginRateLimitCapacity != 7 {
		t.Fatalf("expected login capacity 7, got %d", cfg.LoginRateLimitCapacity)
	}
	if cfg.RegisterRateLimitCapacity != 3 {
		t.Fatalf("expected register capacity 3, got %d", cfg.RegisterRateLimitCapacity)
	}
	if cfg.ReviewRetryRateLimitCapacity != 4 {
		t.Fatalf("expected review retry capacity 4, got %d", cfg.ReviewRetryRateLimitCapacity)
	}
	if cfg.AIChatRateLimitCapacity != 25 {
		t.Fatalf("expected AI chat capacity 25, got %d", cfg.AIChatRateLimitCapacity)
	}
}

func TestLoad_IgnoresDotEnvFilesWithoutLocalSuffix(t *testing.T) {
	tmp := t.TempDir()
	backendDir := filepath.Join(tmp, "backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("mkdir backend: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(tmp, ".env"),
		[]byte("OPENAI_MODEL=file-model\nDATABASE_URL=postgres://ignored-root-dot-env\n"),
		0o600,
	); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(backendDir, ".env"),
		[]byte("APP_ENV=stg\nPORT=9999\n"),
		0o600,
	); err != nil {
		t.Fatalf("write backend/.env: %v", err)
	}

	t.Setenv("APP_ENV", "")
	t.Setenv("PORT", "")
	t.Setenv("OPENAI_MODEL", "")
	t.Setenv("DATABASE_URL", "")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cfg := Load()
	if cfg.AppEnv != "local" {
		t.Fatalf("expected default app env when only .env files exist, got %q", cfg.AppEnv)
	}
	if cfg.Port != "8080" {
		t.Fatalf("expected default port when only .env files exist, got %q", cfg.Port)
	}
	if cfg.OpenAIModel != "gpt-4o-mini" {
		t.Fatalf("expected default model when only .env files exist, got %q", cfg.OpenAIModel)
	}
	if cfg.DatabaseURL != "postgres://skillhub:skillhub@localhost:5432/skillhub_local?sslmode=disable" {
		t.Fatalf("expected default database url when only .env files exist, got %q", cfg.DatabaseURL)
	}
}

func TestLoad_UsesBackendDotEnvLocalAndIgnoresBackendDotEnv(t *testing.T) {
	tmp := t.TempDir()
	backendDir := filepath.Join(tmp, "backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("mkdir backend: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(backendDir, ".env"),
		[]byte("DATABASE_URL=postgres://from-dot-env\nOPENAI_MODEL=file-model\n"),
		0o600,
	); err != nil {
		t.Fatalf("write backend/.env: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(backendDir, ".env.local"),
		[]byte("DATABASE_URL=postgres://from-dot-env-local\nAPP_ENV=stg\n"),
		0o600,
	); err != nil {
		t.Fatalf("write backend/.env.local: %v", err)
	}

	t.Setenv("APP_ENV", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("OPENAI_MODEL", "")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cfg := Load()
	if cfg.DatabaseURL != "postgres://from-dot-env-local" {
		t.Fatalf("expected backend/.env.local to win for database url, got %q", cfg.DatabaseURL)
	}
	if cfg.AppEnv != "stg" {
		t.Fatalf("expected backend/.env.local to set app env, got %q", cfg.AppEnv)
	}
	if cfg.OpenAIModel != "gpt-4o-mini" {
		t.Fatalf("expected backend/.env to be ignored, got %q", cfg.OpenAIModel)
	}
}
