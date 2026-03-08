package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"skill-hub/internal/config"
	"skill-hub/internal/model"
	"skill-hub/internal/service"
	"skill-hub/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestResourceUpdate_OverwritesPublishedRowForTools(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tdb := testutil.OpenPostgresTestDB(t)
	svc := service.NewSkillService(tdb.DB)
	cfg := &config.Config{
		UploadDir:    t.TempDir(),
		ThumbnailDir: t.TempDir(),
	}
	h := NewResourceHandler(svc, "tools", cfg)

	base := model.Skill{
		UserID:            7,
		Name:              "Published Tool",
		Description:       "stable version",
		ResourceType:      "tools",
		Author:            "alice",
		AIApproved:        true,
		AIReviewStatus:    model.AIReviewStatusPassed,
		AIReviewPhase:     model.AIReviewPhaseDone,
		HumanReviewStatus: model.HumanReviewStatusApproved,
		Published:         true,
		Downloads:         9,
		LikesCount:        2,
	}
	if err := tdb.DB.Create(&base).Error; err != nil {
		t.Fatalf("create base skill: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(7))
		c.Set("username", "alice")
		c.Next()
	})
	router.PUT("/tools/:id", h.Update)

	body := []byte(`{"name":"Updated Tool","description":"pending revision"}`)
	req := httptest.NewRequest(http.MethodPut, "/tools/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var stored model.Skill
	if err := tdb.DB.First(&stored, base.ID).Error; err != nil {
		t.Fatalf("reload base skill: %v", err)
	}
	if stored.Name != "Updated Tool" {
		t.Fatalf("expected published row to be updated in place, got %q", stored.Name)
	}
	if stored.Description != "pending revision" {
		t.Fatalf("expected description to be updated in place, got %q", stored.Description)
	}
	if stored.Downloads != 9 || stored.LikesCount != 2 {
		t.Fatalf("expected counters to be preserved, downloads=%d likes=%d", stored.Downloads, stored.LikesCount)
	}
	if _, err := svc.GetActiveRevision(base.ID); err == nil {
		t.Fatalf("expected no pending revision for tools update")
	}
}
