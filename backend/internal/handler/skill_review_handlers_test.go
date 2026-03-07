package handler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"skill-hub/internal/model"
)

func TestBuildReviewStatusResponse_CanRetry(t *testing.T) {
	skill := &model.Skill{
		AIReviewStatus:      model.AIReviewStatusFailedRetry,
		AIReviewAttempts:    1,
		AIReviewMaxAttempts: 3,
		AIApproved:          false,
		AIFeedback:          "failed",
	}

	resp := buildReviewStatusResponse(skill)
	if !resp.CanRetry {
		t.Fatal("expected can_retry to be true")
	}
	if resp.RetryRemaining != 2 {
		t.Fatalf("expected retry_remaining=2, got %d", resp.RetryRemaining)
	}
}

func TestBuildReviewStatusResponse_NoRetryAfterMax(t *testing.T) {
	skill := &model.Skill{
		AIReviewStatus:      model.AIReviewStatusFailedTerminal,
		AIReviewAttempts:    3,
		AIReviewMaxAttempts: 3,
		AIApproved:          false,
		AIFeedback:          "failed",
	}

	resp := buildReviewStatusResponse(skill)
	if resp.CanRetry {
		t.Fatal("expected can_retry to be false")
	}
	if resp.RetryRemaining != 0 {
		t.Fatalf("expected retry_remaining=0, got %d", resp.RetryRemaining)
	}
}

func TestBuildReviewStatusResponse_Progress(t *testing.T) {
	details := reviewProgressDetails{
		TotalFiles:     2,
		CompletedFiles: 1,
		CurrentFile:    "scripts/deploy.sh",
		Files: []reviewFileProgressItem{
			{Path: "README.md", Status: reviewFileStatusPassed},
			{Path: "scripts/deploy.sh", Status: reviewFileStatusRunning},
		},
	}
	raw, err := json.Marshal(details)
	if err != nil {
		t.Fatalf("marshal details: %v", err)
	}

	skill := &model.Skill{AIReviewDetails: string(raw)}
	resp := buildReviewStatusResponse(skill)

	if resp.Progress == nil {
		t.Fatal("expected progress to be present")
	}
	if resp.Progress.TotalFiles != 2 || resp.Progress.CompletedFiles != 1 {
		t.Fatalf("unexpected progress counts: %+v", resp.Progress)
	}
	if resp.Progress.CurrentFile != "scripts/deploy.sh" {
		t.Fatalf("unexpected current file: %s", resp.Progress.CurrentFile)
	}
}

func TestCollectReviewTargetsFromSession(t *testing.T) {
	sessionRoot := t.TempDir()
	mustWriteFile(t, filepath.Join(sessionRoot, "README.md"), "# title", 0644)
	mustWriteFile(t, filepath.Join(sessionRoot, "docs", "guide.mdx"), "# guide", 0644)
	mustWriteFile(t, filepath.Join(sessionRoot, "scripts", "run.sh"), "echo hi", 0644)
	mustWriteFile(t, filepath.Join(sessionRoot, "tools", "build.ts"), "console.log('ok')", 0644)
	mustWriteFile(t, filepath.Join(sessionRoot, ".github", "workflows", "ci.yml"), "name: ci", 0644)
	mustWriteFile(t, filepath.Join(sessionRoot, "Dockerfile"), "FROM alpine", 0644)
	mustWriteFile(t, filepath.Join(sessionRoot, "Makefile"), "all:\n\techo ok", 0644)
	mustWriteFile(t, filepath.Join(sessionRoot, "package.json"), `{"scripts":{"build":"node build.js"}}`, 0644)
	mustWriteFile(t, filepath.Join(sessionRoot, "bin", "deploy"), "#!/usr/bin/env bash\necho deploy", 0755)
	mustWriteFile(t, filepath.Join(sessionRoot, "notes.txt"), "plain text", 0644)

	targets, err := collectReviewTargetsFromSession(sessionRoot)
	if err != nil {
		t.Fatalf("collect targets: %v", err)
	}

	got := make([]string, 0, len(targets))
	for _, item := range targets {
		got = append(got, item.Path)
	}

	want := []string{
		".github/workflows/ci.yml",
		"Dockerfile",
		"Makefile",
		"README.md",
		"bin/deploy",
		"docs/guide.mdx",
		"package.json",
		"scripts/run.sh",
		"tools/build.ts",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected targets\nwant: %#v\n got: %#v", want, got)
	}
}

func mustWriteFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
