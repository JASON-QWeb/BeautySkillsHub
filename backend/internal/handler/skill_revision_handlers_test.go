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

func TestReviewedResourceUpdate_CreatesPendingRevisionForRules(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tdb := testutil.OpenPostgresTestDB(t)
	svc := service.NewSkillService(tdb.DB)
	cfg := &config.Config{
		UploadDir:    t.TempDir(),
		ThumbnailDir: t.TempDir(),
	}
	h := NewSkillHandler(svc, nil, nil, nil, cfg)

	base := model.Skill{
		UserID:            7,
		Name:              "Published Rule",
		Description:       "stable rule",
		ResourceType:      "rules",
		Author:            "alice",
		AIApproved:        true,
		AIReviewStatus:    model.AIReviewStatusPassed,
		AIReviewPhase:     model.AIReviewPhaseDone,
		HumanReviewStatus: model.HumanReviewStatusApproved,
		Published:         true,
		Downloads:         11,
		LikesCount:        3,
	}
	if err := tdb.DB.Create(&base).Error; err != nil {
		t.Fatalf("create base rule: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(7))
		c.Set("username", "alice")
		c.Next()
	})
	router.PUT("/rules/:id", h.UpdateSkill)

	body := []byte(`{"name":"Updated Rule","description":"pending revision"}`)
	req := httptest.NewRequest(http.MethodPut, "/rules/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var stored model.Skill
	if err := tdb.DB.First(&stored, base.ID).Error; err != nil {
		t.Fatalf("reload base rule: %v", err)
	}
	if stored.Name != "Published Rule" {
		t.Fatalf("expected published row to stay unchanged, got %q", stored.Name)
	}

	revision, err := svc.GetActiveRevision(base.ID)
	if err != nil {
		t.Fatalf("expected pending revision, got err=%v", err)
	}
	if revision.Name != "Updated Rule" {
		t.Fatalf("expected revision name to be updated, got %q", revision.Name)
	}
	if revision.HumanReviewStatus != model.HumanReviewStatusPending {
		t.Fatalf("expected pending human review, got %q", revision.HumanReviewStatus)
	}
}
