package database

import (
	"path/filepath"
	"testing"
)

func TestNewMigrator_RequiresDatabaseURL(t *testing.T) {
	_, err := NewMigrator("", t.TempDir())
	if err == nil {
		t.Fatal("expected error when database url is empty")
	}
}

func TestNewMigrator_RequiresMigrationsDir(t *testing.T) {
	_, err := NewMigrator("postgres://skillhub:test@localhost:5432/skillhub?sslmode=disable", "")
	if err == nil {
		t.Fatal("expected error when migrations dir is empty")
	}
}

func TestNewMigrator_NormalizesFileSourceURL(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "migrations")

	migrator, err := NewMigrator("postgres://skillhub:test@localhost:5432/skillhub?sslmode=disable", dir)
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}

	expectedPrefix := "file://"
	if got := migrator.SourceURL(); got[:len(expectedPrefix)] != expectedPrefix {
		t.Fatalf("expected source url to start with %q, got %q", expectedPrefix, got)
	}

	if !filepath.IsAbs(migrator.MigrationsDir()) {
		t.Fatalf("expected migrations dir to be absolute, got %q", migrator.MigrationsDir())
	}
}
