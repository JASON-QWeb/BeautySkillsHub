package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_UsesDotEnvValuesWhenProcessEnvIsUnset(t *testing.T) {
	tmp := t.TempDir()
	envPath := filepath.Join(tmp, ".env")
	content := "APP_ENV=local\nPORT=9090\nOPENAI_API_KEY=test-key\nOPENAI_BASE_URL=http://localhost:11434/v1\nOPENAI_MODEL=gpt-4.1-mini\nDATABASE_URL=postgres://tester:secret@localhost:5432/skillhub?sslmode=disable\nUPLOAD_DIR=./tmp-uploads\nTHUMBNAIL_DIR=./tmp-thumbs\n"
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
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
		t.Fatalf("expected OpenAI key from .env, got %q", cfg.OpenAIKey)
	}
	if cfg.OpenAIBaseURL != "http://localhost:11434/v1" {
		t.Fatalf("expected base url from .env, got %q", cfg.OpenAIBaseURL)
	}
	if cfg.OpenAIModel != "gpt-4.1-mini" {
		t.Fatalf("expected model from .env, got %q", cfg.OpenAIModel)
	}
	if cfg.DatabaseURL != "postgres://tester:secret@localhost:5432/skillhub?sslmode=disable" {
		t.Fatalf("expected database url from .env, got %q", cfg.DatabaseURL)
	}
	if cfg.UploadDir != "./tmp-uploads" {
		t.Fatalf("expected upload dir from .env, got %q", cfg.UploadDir)
	}
	if cfg.ThumbnailDir != "./tmp-thumbs" {
		t.Fatalf("expected thumbnail dir from .env, got %q", cfg.ThumbnailDir)
	}
}

func TestLoad_ProcessEnvOverridesDotEnv(t *testing.T) {
	tmp := t.TempDir()
	envPath := filepath.Join(tmp, ".env")
	content := "OPENAI_MODEL=file-model\nOPENAI_API_KEY=file-key\n"
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
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
}

func TestLoad_PrefersBackendDotEnvLocalOverBackendDotEnv(t *testing.T) {
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
	if cfg.OpenAIModel != "file-model" {
		t.Fatalf("expected backend/.env fallback to remain available, got %q", cfg.OpenAIModel)
	}
}
