package handler

import (
	"mime/multipart"
	"path/filepath"
	"testing"
)

func TestNormalizeTags(t *testing.T) {
	input := "react, frontend, hooks, React,  ui , api, extra"
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
