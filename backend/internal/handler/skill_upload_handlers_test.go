package handler

import (
	"net/http/httptest"
	"mime/multipart"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNormalizeTags(t *testing.T) {
	input := "React, FrontEND, hooks, React,  UI , API, extra"
	got := normalizeTags(input)
	want := "react,frontend,hooks,ui,api"
	if got != want {
		t.Fatalf("unexpected tags result, got %q want %q", got, want)
	}
}

func TestNormalizeTags_Empty(t *testing.T) {
	if got := normalizeTags(" , , "); got != "" {
		t.Fatalf("expected empty tags, got %q", got)
	}
}

func TestThumbnailSubtitle(t *testing.T) {
	if got := thumbnailSubtitle("  concise description ", "fallback"); got != "concise description" {
		t.Fatalf("expected description to win, got %q", got)
	}
	if got := thumbnailSubtitle("   ", "  fallback name  "); got != "fallback name" {
		t.Fatalf("expected fallback when description is empty, got %q", got)
	}
}

func TestSanitizeLocalFilename(t *testing.T) {
	if got := sanitizeLocalFilename("../../evil.sh"); got != "evil.sh" {
		t.Fatalf("expected cleaned filename, got %q", got)
	}
	if got := sanitizeLocalFilename("中文 文件.md"); got != "中文-文件.md" {
		t.Fatalf("expected unicode filename preserved, got %q", got)
	}
}

func TestIsPathInsideBase(t *testing.T) {
	base := t.TempDir()
	inside := filepath.Join(base, "a", "b.txt")
	if !isPathInsideBase(base, inside) {
		t.Fatal("expected inside path to be accepted")
	}
	outside := filepath.Join(base, "..", "other.txt")
	if isPathInsideBase(base, outside) {
		t.Fatal("expected outside path to be rejected")
	}
}

func TestUploadSessionRoot(t *testing.T) {
	base := t.TempDir()

	nestedFile := filepath.Join(base, "skill-upload-123", "src", "main.md")
	if got := uploadSessionRoot(base, nestedFile); got != filepath.Join(base, "skill-upload-123") {
		t.Fatalf("unexpected session root for nested file: %q", got)
	}

	typeSubdirFile := filepath.Join(base, "skills", "skill-upload-456", "src", "SKILLS.md")
	if got := uploadSessionRoot(base, typeSubdirFile); got != filepath.Join(base, "skills", "skill-upload-456") {
		t.Fatalf("unexpected session root for type subdir file: %q", got)
	}

	plainFile := filepath.Join(base, "legacy.md")
	if got := uploadSessionRoot(base, plainFile); got != plainFile {
		t.Fatalf("unexpected session root for plain file: %q", got)
	}

	outside := filepath.Join(base, "..", "outside.md")
	if got := uploadSessionRoot(base, outside); got != "" {
		t.Fatalf("expected empty root for outside path, got %q", got)
	}
}

func TestValidateThumbnailHeader(t *testing.T) {
	tests := []struct {
		name        string
		header      *multipart.FileHeader
		wantErr     bool
		expectedExt string
	}{
		{
			name: "accept png within size limit",
			header: &multipart.FileHeader{
				Filename: "cover.PNG",
				Size:     maxThumbnailSize - 1,
			},
			wantErr:     false,
			expectedExt: ".png",
		},
		{
			name: "reject unsupported extension",
			header: &multipart.FileHeader{
				Filename: "cover.svg",
				Size:     1024,
			},
			wantErr: true,
		},
		{
			name: "reject oversized thumbnail",
			header: &multipart.FileHeader{
				Filename: "cover.jpg",
				Size:     maxThumbnailSize + 1,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotExt, err := validateThumbnailHeader(tc.header)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (ext=%q)", gotExt)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if gotExt != tc.expectedExt {
				t.Fatalf("expected ext %q, got %q", tc.expectedExt, gotExt)
			}
		})
	}
}

func TestIsRulesTextExtension(t *testing.T) {
	if !isRulesTextExtension("rule.md") {
		t.Fatal("expected .md to be allowed")
	}
	if !isRulesTextExtension("rule.TXT") {
		t.Fatal("expected .txt to be allowed")
	}
	if isRulesTextExtension("rule.json") {
		t.Fatal("expected .json to be rejected")
	}
}

func TestResolveReviewedUploadResourceType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("uses explicit form value", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/skills", nil)
		c.Request.PostForm = map[string][]string{
			"resource_type": {"rules"},
		}
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		if got := resolveReviewedUploadResourceType(c); got != "rules" {
			t.Fatalf("expected rules, got %q", got)
		}
	})

	t.Run("falls back to route", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/rules", nil)

		if got := resolveReviewedUploadResourceType(c); got != "rules" {
			t.Fatalf("expected rules, got %q", got)
		}
	})

	t.Run("defaults to skill", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/skills", nil)

		if got := resolveReviewedUploadResourceType(c); got != "skill" {
			t.Fatalf("expected skill, got %q", got)
		}
	})
}
