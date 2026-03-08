package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"skill-hub/internal/config"
	"skill-hub/internal/model"
	"skill-hub/internal/service"
	"skill-hub/internal/testutil"

	"github.com/gin-gonic/gin"
)

func TestValidateContentTextFields_AcceptsBroadLimits(t *testing.T) {
	err := validateContentTextFields(contentTextFields{
		Name:        strings.Repeat("n", 255),
		Description: strings.Repeat("d", 5000),
		Tags:        strings.Repeat("t", 1000),
		Author:      strings.Repeat("a", 100),
		SourceURL:   "https://example.com/" + strings.Repeat("p", 900),
	})
	if err != nil {
		t.Fatalf("expected values within broad limits to pass, got %v", err)
	}
}

func TestValidateContentTextFields_RejectsOverlongFields(t *testing.T) {
	tests := []struct {
		name      string
		fields    contentTextFields
		wantField string
		wantLimit int
	}{
		{
			name:      "name too long",
			fields:    contentTextFields{Name: strings.Repeat("n", 256)},
			wantField: "name",
			wantLimit: 255,
		},
		{
			name:      "description too long",
			fields:    contentTextFields{Description: strings.Repeat("d", 5001)},
			wantField: "description",
			wantLimit: 5000,
		},
		{
			name:      "tags too long",
			fields:    contentTextFields{Tags: strings.Repeat("t", 1001)},
			wantField: "tags",
			wantLimit: 1000,
		},
		{
			name:      "author too long",
			fields:    contentTextFields{Author: strings.Repeat("a", 101)},
			wantField: "author",
			wantLimit: 100,
		},
		{
			name:      "source url too long",
			fields:    contentTextFields{SourceURL: "https://example.com/" + strings.Repeat("p", 1100)},
			wantField: "source_url",
			wantLimit: 1024,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateContentTextFields(tc.fields)
			if err == nil {
				t.Fatal("expected validation error")
			}
			var fieldErr *fieldLengthError
			if !errors.As(err, &fieldErr) {
				t.Fatalf("expected fieldLengthError, got %T", err)
			}
			if fieldErr.Field != tc.wantField {
				t.Fatalf("expected field %q, got %q", tc.wantField, fieldErr.Field)
			}
			if fieldErr.Limit != tc.wantLimit {
				t.Fatalf("expected limit %d, got %d", tc.wantLimit, fieldErr.Limit)
			}
		})
	}
}

func TestUploadSkill_RejectsOverlongNameBeforeFileValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tdb := testutil.OpenPostgresTestDB(t)
	svc := service.NewSkillService(tdb.DB)
	h := NewSkillHandler(svc, nil, nil, nil, &config.Config{
		UploadDir:    t.TempDir(),
		ThumbnailDir: t.TempDir(),
	})

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(7))
		c.Set("username", "alice")
		c.Next()
	})
	router.POST("/api/skills", h.UploadSkill)

	req := newMultipartFormRequest(t, "/api/skills", map[string]string{
		"name": strings.Repeat("n", 256),
	})
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for overlong skill name, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "名称长度不能超过 255 个字符") {
		t.Fatalf("expected length validation message, got %s", resp.Body.String())
	}
}

func TestResourceUpload_RejectsOverlongDescription(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tdb := testutil.OpenPostgresTestDB(t)
	svc := service.NewSkillService(tdb.DB)
	h := NewResourceHandler(svc, "mcp", &config.Config{
		UploadDir:    t.TempDir(),
		ThumbnailDir: t.TempDir(),
	})

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(7))
		c.Set("username", "alice")
		c.Next()
	})
	router.POST("/api/mcps", h.Upload)

	req := newMultipartFormRequest(t, "/api/mcps", map[string]string{
		"name":        "valid name",
		"description": strings.Repeat("d", 5001),
		"upload_mode": "metadata",
	})
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for overlong resource description, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "description exceeds 5000 characters") {
		t.Fatalf("expected length validation message, got %s", resp.Body.String())
	}
}

func TestReviewedResourceUpdate_RejectsOverlongDescription(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tdb := testutil.OpenPostgresTestDB(t)
	svc := service.NewSkillService(tdb.DB)
	h := NewSkillHandler(svc, nil, nil, nil, &config.Config{
		UploadDir:    t.TempDir(),
		ThumbnailDir: t.TempDir(),
	})

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

	body, err := json.Marshal(map[string]string{
		"description": strings.Repeat("d", 5001),
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/rules/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for overlong reviewed update description, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "描述长度不能超过 5000 个字符") {
		t.Fatalf("expected length validation message, got %s", resp.Body.String())
	}
}

func TestResourceUpdate_RejectsOverlongName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tdb := testutil.OpenPostgresTestDB(t)
	svc := service.NewSkillService(tdb.DB)
	h := NewResourceHandler(svc, "tools", &config.Config{
		UploadDir:    t.TempDir(),
		ThumbnailDir: t.TempDir(),
	})

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
	}
	if err := tdb.DB.Create(&base).Error; err != nil {
		t.Fatalf("create base tool: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(7))
		c.Set("username", "alice")
		c.Next()
	})
	router.PUT("/tools/:id", h.Update)

	body, err := json.Marshal(map[string]string{
		"name": strings.Repeat("n", 256),
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/tools/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for overlong resource update name, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "name exceeds 255 characters") {
		t.Fatalf("expected length validation message, got %s", resp.Body.String())
	}
}

func newMultipartFormRequest(t *testing.T, target string, fields map[string]string) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field %s: %v", key, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
