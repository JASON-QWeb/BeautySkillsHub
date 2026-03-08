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
	"gorm.io/gorm"
)

func newSkillServiceForSecurityP0Test(t *testing.T) *service.SkillService {
	t.Helper()

	tdb := testutil.OpenPostgresTestDB(t)
	return service.NewSkillService(tdb.DB)
}

func TestSkillHandlerGetSkillResource_RejectsNonSkillType(t *testing.T) {
	svc := newSkillServiceForSecurityP0Test(t)
	h := &SkillHandler{skillSvc: svc}

	nonSkill := &model.Skill{
		Name:              "mcp item",
		ResourceType:      "mcp",
		AIApproved:        true,
		HumanReviewStatus: model.HumanReviewStatusApproved,
	}
	if err := svc.CreateSkill(nonSkill); err != nil {
		t.Fatalf("create non-skill resource: %v", err)
	}

	_, err := h.getSkillResource(nonSkill.ID)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

func TestSkillHandlerGetSkillResource_AllowsSkillType(t *testing.T) {
	svc := newSkillServiceForSecurityP0Test(t)
	h := &SkillHandler{skillSvc: svc}

	skill := &model.Skill{
		Name:              "skill item",
		ResourceType:      "skill",
		AIApproved:        true,
		HumanReviewStatus: model.HumanReviewStatusApproved,
	}
	if err := svc.CreateSkill(skill); err != nil {
		t.Fatalf("create skill resource: %v", err)
	}

	got, err := h.getSkillResource(skill.ID)
	if err != nil {
		t.Fatalf("get skill resource: %v", err)
	}
	if got.ID != skill.ID {
		t.Fatalf("expected skill id %d, got %d", skill.ID, got.ID)
	}
}

func TestSkillHandlerServeThumbnail_RejectsPathTraversal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	thumbDir := t.TempDir()
	outsideDir := t.TempDir()
	secretPath := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("secret-data"), 0644); err != nil {
		t.Fatalf("write secret file: %v", err)
	}

	h := &SkillHandler{
		cfg: &config.Config{
			ThumbnailDir: thumbDir,
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/thumbnails/..%2Fsecret.txt", nil)
	c.Params = gin.Params{{Key: "filename", Value: "../secret.txt"}}

	h.ServeThumbnail(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when path traversal attempted, got %d", w.Code)
	}
	if body := w.Body.String(); body != "" && body != "{\"error\":\"缩略图不存在\"}" {
		t.Fatalf("expected no file content leak, got body %q", body)
	}
}

func TestResourceHandlerServeThumbnail_RejectsPathTraversal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	thumbDir := t.TempDir()
	outsideDir := t.TempDir()
	secretPath := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("secret-data"), 0644); err != nil {
		t.Fatalf("write secret file: %v", err)
	}

	h := &ResourceHandler{
		cfg: &config.Config{
			ThumbnailDir: thumbDir,
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/thumbnails/..%2Fsecret.txt", nil)
	c.Params = gin.Params{{Key: "filename", Value: "../secret.txt"}}

	h.ServeThumbnail(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when path traversal attempted, got %d", w.Code)
	}
	if body := w.Body.String(); body != "" && body != "{\"error\":\"thumbnail not found\"}" {
		t.Fatalf("expected no file content leak, got body %q", body)
	}
}
