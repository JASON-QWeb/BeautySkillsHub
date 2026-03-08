package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"skill-hub/internal/testutil"

	"github.com/gin-gonic/gin"
)

func newAuthTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	tdb := testutil.OpenPostgresTestDB(t)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/auth/register", NewAuthHandler(tdb.DB, t.TempDir()).Register)
	return r
}

func TestRegisterRejectsWhitespaceOnlyUsername(t *testing.T) {
	r := newAuthTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(`{"username":"   ","password":"123456"}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for whitespace username, got %d", w.Code)
	}
}

func TestNewAuthHandler_UsesNonDefaultSecretWhenEnvMissing(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	tdb := testutil.OpenPostgresTestDB(t)
	h := NewAuthHandler(tdb.DB, t.TempDir())
	if len(h.secret) == 0 {
		t.Fatal("expected generated secret, got empty")
	}
	if string(h.secret) == "skill-hub-default-secret-change-me" {
		t.Fatal("expected no hardcoded fallback secret")
	}
}

func TestRegisterRejectsTooShortUsernameAfterTrim(t *testing.T) {
	r := newAuthTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(`{"username":" a ","password":"123456"}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for short username after trim, got %d", w.Code)
	}
}

func TestRegisterCreatesUser(t *testing.T) {
	r := newAuthTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(`{"username":"alice","password":"123456"}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["token"]; !ok {
		t.Fatalf("expected token in response body")
	}
}

func TestRegisterRejectsDuplicateUsername(t *testing.T) {
	r := newAuthTestRouter(t)

	firstReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(`{"username":"alice","password":"123456"}`))
	firstReq.Header.Set("Content-Type", "application/json")
	firstResp := httptest.NewRecorder()
	r.ServeHTTP(firstResp, firstReq)
	if firstResp.Code != http.StatusCreated {
		t.Fatalf("setup register failed: code=%d body=%s", firstResp.Code, firstResp.Body.String())
	}

	duplicateReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(`{"username":"alice","password":"654321"}`))
	duplicateReq.Header.Set("Content-Type", "application/json")
	duplicateResp := httptest.NewRecorder()
	r.ServeHTTP(duplicateResp, duplicateReq)

	if duplicateResp.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate username, got %d body=%s", duplicateResp.Code, duplicateResp.Body.String())
	}
}
