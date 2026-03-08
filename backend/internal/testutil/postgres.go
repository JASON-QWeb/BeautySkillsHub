package testutil

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"skill-hub/internal/database"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const defaultTestDatabaseURL = "postgres://skillhub:skillhub@localhost:5432/skillhub_local?sslmode=disable"

type PostgresTestDB struct {
	DB               *gorm.DB
	Schema           string
	DatabaseURL      string
	AdminDatabaseURL string
}

func OpenPostgresTestDB(t *testing.T) *PostgresTestDB {
	t.Helper()

	repoRoot := mustRepoRoot(t)
	baseURL := resolveDatabaseURL(repoRoot)
	adminURL := clearSearchPath(baseURL)
	schema := uniqueSchemaName(t.Name())
	testURL := withSearchPath(baseURL, schema)

	adminDB, err := gorm.Open(postgres.Open(adminURL), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres admin db: %v", err)
	}

	if err := adminDB.Exec("CREATE SCHEMA " + quoteIdentifier(schema)).Error; err != nil {
		t.Fatalf("create schema %s: %v", schema, err)
	}

	migrator, err := database.NewMigrator(testURL, filepath.Join(repoRoot, "db", "migrations"))
	if err != nil {
		t.Fatalf("init migrator: %v", err)
	}
	if err := migrator.Up(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	testDB, err := gorm.Open(postgres.Open(testURL), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres test db: %v", err)
	}

	t.Cleanup(func() {
		if sqlDB, err := testDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
		if err := adminDB.Exec("DROP SCHEMA IF EXISTS " + quoteIdentifier(schema) + " CASCADE").Error; err != nil {
			t.Fatalf("drop schema %s: %v", schema, err)
		}
		if sqlDB, err := adminDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	return &PostgresTestDB{
		DB:               testDB,
		Schema:           schema,
		DatabaseURL:      testURL,
		AdminDatabaseURL: adminURL,
	}
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func resolveDatabaseURL(repoRoot string) string {
	if value := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL")); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("DATABASE_URL")); value != "" {
		return value
	}
	for _, path := range []string{
		filepath.Join(repoRoot, "backend", ".env.local"),
		filepath.Join(repoRoot, "backend", ".env.example"),
	} {
		if value := readEnvValue(path, "DATABASE_URL"); value != "" {
			return value
		}
	}
	return defaultTestDatabaseURL
}

func readEnvValue(path, key string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) != key {
			continue
		}
		return trimEnvValue(strings.TrimSpace(parts[1]))
	}
	return ""
}

func trimEnvValue(value string) string {
	if len(value) >= 2 {
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			return strings.Trim(value, "\"")
		}
		if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			return strings.Trim(value, "'")
		}
	}
	if idx := strings.Index(value, " #"); idx >= 0 {
		return strings.TrimSpace(value[:idx])
	}
	return value
}

func clearSearchPath(databaseURL string) string {
	parts := strings.SplitN(databaseURL, "?", 2)
	if len(parts) == 1 {
		return databaseURL
	}

	kept := make([]string, 0, 4)
	for _, pair := range strings.Split(parts[1], "&") {
		if pair == "" || strings.HasPrefix(pair, "search_path=") {
			continue
		}
		kept = append(kept, pair)
	}
	if len(kept) == 0 {
		return parts[0]
	}
	return parts[0] + "?" + strings.Join(kept, "&")
}

func withSearchPath(databaseURL, schema string) string {
	base := clearSearchPath(databaseURL)
	separator := "?"
	if strings.Contains(base, "?") {
		separator = "&"
	}
	return base + separator + "search_path=" + schema
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func uniqueSchemaName(testName string) string {
	normalized := strings.ToLower(testName)
	normalized = nonAlphaNum.ReplaceAllString(normalized, "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		normalized = "test"
	}
	return fmt.Sprintf("test_%s_%d", normalized, time.Now().UnixNano())
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
