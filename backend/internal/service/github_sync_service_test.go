package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"skill-hub/internal/config"
)

type fakeGitHubClient struct {
	existing map[string]bool
	putErr   error
	putPath  string
	putBody  []byte
}

func (f *fakeGitHubClient) GetFileSHA(_ context.Context, path string) (string, bool, error) {
	if f.existing[path] {
		return "exists-sha", true, nil
	}
	return "", false, nil
}

func (f *fakeGitHubClient) PutFile(_ context.Context, path, _ string, content []byte, _ string) (string, error) {
	if f.putErr != nil {
		return "", f.putErr
	}
	f.putPath = path
	f.putBody = append([]byte{}, content...)
	return "https://github.com/owner/repo/blob/main/" + path, nil
}

func TestGitHubSyncService_Disabled(t *testing.T) {
	cfg := &config.Config{GitHubSyncEnabled: false}
	svc := NewGitHubSyncService(cfg, &fakeGitHubClient{})

	tmp := t.TempDir()
	localFile := filepath.Join(tmp, "a.txt")
	if err := os.WriteFile(localFile, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	result := svc.SyncUploadedSkill(context.Background(), "My Skill", "skill", "a.txt", localFile)
	if result.Status != GitHubSyncStatusDisabled {
		t.Fatalf("expected disabled status, got %q", result.Status)
	}
}

func TestGitHubSyncService_Success(t *testing.T) {
	cfg := &config.Config{
		GitHubSyncEnabled: true,
		GitHubBaseDir:     "skills",
		GitHubToken:       "token",
		GitHubOwner:       "owner",
		GitHubRepo:        "repo",
	}
	fake := &fakeGitHubClient{existing: map[string]bool{}}
	svc := NewGitHubSyncService(cfg, fake)

	tmp := t.TempDir()
	localFile := filepath.Join(tmp, "script.sh")
	if err := os.WriteFile(localFile, []byte("echo hi"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	result := svc.SyncUploadedSkill(context.Background(), "My Skill", "tools", "script.sh", localFile)
	if result.Status != GitHubSyncStatusSuccess {
		t.Fatalf("expected success status, got %q (error=%q)", result.Status, result.Error)
	}
	if fake.putPath != "skills/tools/my-skill/script.sh" {
		t.Fatalf("expected normalized github path, got %q", fake.putPath)
	}
	if string(fake.putBody) != "echo hi" {
		t.Fatalf("expected uploaded content echo hi, got %q", string(fake.putBody))
	}
}

func TestGitHubSyncService_ConflictAddsTimestampSuffix(t *testing.T) {
	cfg := &config.Config{
		GitHubSyncEnabled: true,
		GitHubBaseDir:     "skills",
		GitHubToken:       "token",
		GitHubOwner:       "owner",
		GitHubRepo:        "repo",
	}
	initialPath := "skills/skill/my-skill/a.txt"
	fake := &fakeGitHubClient{existing: map[string]bool{initialPath: true}}
	svc := NewGitHubSyncService(cfg, fake)
	svc.nowFn = func() time.Time {
		return time.Date(2026, 3, 5, 22, 30, 0, 0, time.UTC)
	}

	tmp := t.TempDir()
	localFile := filepath.Join(tmp, "a.txt")
	if err := os.WriteFile(localFile, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	result := svc.SyncUploadedSkill(context.Background(), "My Skill", "skill", "a.txt", localFile)
	if result.Status != GitHubSyncStatusSuccess {
		t.Fatalf("expected success status, got %q", result.Status)
	}
	if fake.putPath != "skills/skill/my-skill/a_20260305_223000.txt" {
		t.Fatalf("expected timestamp-suffixed path, got %q", fake.putPath)
	}
}

func TestGitHubSyncService_FailedWhenPutFails(t *testing.T) {
	cfg := &config.Config{
		GitHubSyncEnabled: true,
		GitHubBaseDir:     "skills",
		GitHubToken:       "token",
		GitHubOwner:       "owner",
		GitHubRepo:        "repo",
	}
	fake := &fakeGitHubClient{
		existing: map[string]bool{},
		putErr:   errors.New("github unavailable"),
	}
	svc := NewGitHubSyncService(cfg, fake)

	tmp := t.TempDir()
	localFile := filepath.Join(tmp, "a.txt")
	if err := os.WriteFile(localFile, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	result := svc.SyncUploadedSkill(context.Background(), "My Skill", "skill", "a.txt", localFile)
	if result.Status != GitHubSyncStatusFailed {
		t.Fatalf("expected failed status, got %q", result.Status)
	}
	if result.Error == "" {
		t.Fatal("expected failure reason")
	}
}
