package handler

import "testing"

func TestIsToolsArchiveFilename(t *testing.T) {
	valid := []string{
		"tool.zip",
		"tool.tar.gz",
		"tool.tgz",
		"tool.7z",
		"tool.tar",
	}
	for _, item := range valid {
		if !isToolsArchiveFilename(item) {
			t.Fatalf("expected %q to be accepted", item)
		}
	}

	invalid := []string{
		"",
		"tool.md",
		"tool.exe",
		"tool",
	}
	for _, item := range invalid {
		if isToolsArchiveFilename(item) {
			t.Fatalf("expected %q to be rejected", item)
		}
	}
}

func TestValidateSourceURL(t *testing.T) {
	if err := validateSourceURL(""); err != nil {
		t.Fatalf("expected empty source url to be valid: %v", err)
	}
	if err := validateSourceURL("https://github.com/example/repo"); err != nil {
		t.Fatalf("expected github url to be valid: %v", err)
	}
	if err := validateSourceURL("ftp://github.com/example/repo"); err == nil {
		t.Fatal("expected unsupported scheme to fail")
	}
	if err := validateSourceURL("https://"); err == nil {
		t.Fatal("expected missing host to fail")
	}
}
