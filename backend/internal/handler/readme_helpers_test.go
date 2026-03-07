package handler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindReadmePathInSession_PrefersRootCandidate(t *testing.T) {
	root := t.TempDir()
	rootReadme := filepath.Join(root, "SKILLS.md")
	nestedReadme := filepath.Join(root, "docs", "README.md")

	if err := os.WriteFile(rootReadme, []byte("# root"), 0644); err != nil {
		t.Fatalf("write root readme: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(nestedReadme), 0755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(nestedReadme, []byte("# nested"), 0644); err != nil {
		t.Fatalf("write nested readme: %v", err)
	}

	got := findReadmePathInSession(root)
	if got != rootReadme {
		t.Fatalf("expected root readme %q, got %q", rootReadme, got)
	}
}

func TestFindReadmePathInSession_FindsNestedCandidate(t *testing.T) {
	root := t.TempDir()
	nestedReadme := filepath.Join(root, "packages", "ui", "SKILLS.md")

	if err := os.MkdirAll(filepath.Dir(nestedReadme), 0755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(nestedReadme, []byte("# nested"), 0644); err != nil {
		t.Fatalf("write nested readme: %v", err)
	}

	got := findReadmePathInSession(root)
	if got != nestedReadme {
		t.Fatalf("expected nested readme %q, got %q", nestedReadme, got)
	}
}

func TestFindReadmePathInSession_ReturnsEmptyWhenMissing(t *testing.T) {
	root := t.TempDir()
	got := findReadmePathInSession(root)
	if got != "" {
		t.Fatalf("expected empty path, got %q", got)
	}
}
