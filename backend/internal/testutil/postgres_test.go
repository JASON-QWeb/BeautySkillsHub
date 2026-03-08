package testutil

import (
	"database/sql"
	"fmt"
	"testing"

	"skill-hub/internal/model"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestOpenPostgresTestDB_AppliesMigrations(t *testing.T) {
	tdb := OpenPostgresTestDB(t)

	if !tdb.DB.Migrator().HasTable(&model.User{}) {
		t.Fatal("expected users table from migrations")
	}
	if !tdb.DB.Migrator().HasTable(&model.Skill{}) {
		t.Fatal("expected skills table from migrations")
	}
}

func TestOpenPostgresTestDB_CleansUpSchema(t *testing.T) {
	var schema string
	var adminDSN string

	t.Run("allocates schema", func(t *testing.T) {
		tdb := OpenPostgresTestDB(t)
		schema = tdb.Schema
		adminDSN = tdb.AdminDatabaseURL

		var count int64
		if err := tdb.DB.Model(&model.User{}).Count(&count).Error; err != nil {
			t.Fatalf("count users: %v", err)
		}
	})

	db, err := sql.Open("pgx", adminDSN)
	if err != nil {
		t.Fatalf("open admin sql db: %v", err)
	}
	defer db.Close()

	var exists bool
	if err := db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)`,
		schema,
	).Scan(&exists); err != nil {
		t.Fatalf("query schema existence: %v", err)
	}
	if exists {
		t.Fatalf("expected schema %s to be dropped during cleanup", schema)
	}
}

func TestOpenPostgresTestDB_UsesIsolatedSchemas(t *testing.T) {
	first := OpenPostgresTestDB(t)
	second := OpenPostgresTestDB(t)

	if first.Schema == second.Schema {
		t.Fatalf("expected unique schema names, got %s", first.Schema)
	}

	username := fmt.Sprintf("u_%d", len(first.Schema))
	if err := first.DB.Exec(`INSERT INTO users (username, password) VALUES (?, ?)`, username, "secret").Error; err != nil {
		t.Fatalf("insert user into first schema: %v", err)
	}

	var count int64
	if err := second.DB.Model(&model.User{}).Count(&count).Error; err != nil {
		t.Fatalf("count users in second schema: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected isolated schema data, got %d users in second schema", count)
	}
}
