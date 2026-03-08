package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEngagementForeignKeyMigration_AddsCascadeConstraintsAfterCleanup(t *testing.T) {
	path := filepath.Join("..", "..", "..", "db", "migrations", "0003_add_engagement_foreign_keys.up.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	sql := string(content)
	requiredSnippets := []string{
		"DELETE FROM skill_likes",
		"DELETE FROM skill_favorites",
		"FOREIGN KEY (skill_id) REFERENCES skills(id) ON DELETE CASCADE",
		"FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(sql, snippet) {
			t.Fatalf("expected migration to contain %q", snippet)
		}
	}
}
