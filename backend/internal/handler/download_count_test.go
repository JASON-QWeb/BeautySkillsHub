package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"skill-hub/internal/config"
	"skill-hub/internal/model"
	"skill-hub/internal/service"
	"skill-hub/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestSkillHandlerDownloadSkill_ContinuesWhenDownloadCounterFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tdb := testutil.OpenPostgresTestDB(t)
	svc := service.NewSkillService(tdb.DB)
	filePath := writeDownloadFixture(t, "skill-download.txt", "skill payload")
	skill := model.Skill{
		Name:              "Downloadable Skill",
		ResourceType:      "skill",
		FileName:          "skill-download.txt",
		FilePath:          filePath,
		AIApproved:        true,
		HumanReviewStatus: model.HumanReviewStatusApproved,
		Published:         true,
	}
	if err := tdb.DB.Create(&skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}

	restore := stubIncrementDownload(func(*service.SkillService, uint) error {
		return errors.New("counter unavailable")
	})
	defer restore()

	h := NewSkillHandler(svc, nil, nil, nil, &config.Config{})
	router := gin.New()
	router.GET("/api/skills/:id/download", h.DownloadSkill)

	req := httptest.NewRequest(http.MethodGet, "/api/skills/1/download", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected download to succeed despite counter failure, got %d body=%s", resp.Code, resp.Body.String())
	}
	if body := resp.Body.String(); body != "skill payload" {
		t.Fatalf("expected file body, got %q", body)
	}
}

func TestResourceHandlerDownload_ContinuesWhenDownloadCounterFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tdb := testutil.OpenPostgresTestDB(t)
	svc := service.NewSkillService(tdb.DB)
	filePath := writeDownloadFixture(t, "tool-download.txt", "tool payload")
	resource := model.Skill{
		Name:              "Downloadable Tool",
		ResourceType:      "tools",
		FileName:          "tool-download.txt",
		FilePath:          filePath,
		AIApproved:        true,
		HumanReviewStatus: model.HumanReviewStatusApproved,
		Published:         true,
	}
	if err := tdb.DB.Create(&resource).Error; err != nil {
		t.Fatalf("create resource: %v", err)
	}

	restore := stubIncrementDownload(func(*service.SkillService, uint) error {
		return errors.New("counter unavailable")
	})
	defer restore()

	h := NewResourceHandler(svc, "tools", &config.Config{})
	router := gin.New()
	router.GET("/api/tools/:id/download", h.Download)

	req := httptest.NewRequest(http.MethodGet, "/api/tools/1/download", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected download to succeed despite counter failure, got %d body=%s", resp.Code, resp.Body.String())
	}
	if body := resp.Body.String(); body != "tool payload" {
		t.Fatalf("expected file body, got %q", body)
	}
}

func writeDownloadFixture(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func stubIncrementDownload(fn func(*service.SkillService, uint) error) func() {
	original := incrementDownloadCounter
	incrementDownloadCounter = fn
	return func() {
		incrementDownloadCounter = original
	}
}
