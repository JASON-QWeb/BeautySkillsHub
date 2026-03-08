package handler

import (
	"strings"
	"testing"

	"skill-hub/internal/testutil"
)

func TestNewAuthHandler_ProductionRequiresJWTSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "")

	tdb := testutil.OpenPostgresTestDB(t)

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic when JWT_SECRET missing in production")
		}
		if !strings.Contains(strings.ToLower(recovered.(string)), "jwt_secret") {
			t.Fatalf("expected jwt_secret panic, got %v", recovered)
		}
	}()

	_ = NewAuthHandler(tdb.DB, t.TempDir())
}
