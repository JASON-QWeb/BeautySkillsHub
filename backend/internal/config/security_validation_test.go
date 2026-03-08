package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_NonLocalEnvironmentRejectsInsecureDatabaseURL(t *testing.T) {
	tmp := t.TempDir()

	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://skillhub:skillhub@db:5432/skillhub?sslmode=disable")
	t.Setenv("JWT_SECRET", "prod-secret")

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
	if err := os.WriteFile(filepath.Join(tmp, ".env.local"), []byte(""), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic for insecure database url")
		}
		if !strings.Contains(strings.ToLower(recovered.(string)), "sslmode") {
			t.Fatalf("expected sslmode panic, got %v", recovered)
		}
	}()

	_ = Load()
}
