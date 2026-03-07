package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"skill-hub/internal/config"
)

type fakeGitHubClient struct {
	existing map[string]bool
	dirs     map[string][]string
	putErr   error
	putPath  string
	putBody  []byte
	deleted  []string
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
	f.existing[path] = true
	return "https://github.com/owner/repo/blob/main/" + path, nil
}

func (f *fakeGitHubClient) DeleteFile(_ context.Context, path, _, _ string) error {
	f.deleted = append(f.deleted, path)
	delete(f.existing, path)
	return nil
}

func (f *fakeGitHubClient) ListDir(_ context.Context, dirPath string) ([]string, error) {
	if f.dirs != nil {
		if entries, ok := f.dirs[dirPath]; ok {
			return append([]string{}, entries...), nil
		}
	}
	return nil, nil
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
	if fake.putPath != "skills/my-skill/script.sh" {
		t.Fatalf("expected normalized github path, got %q", fake.putPath)
	}
	if string(fake.putBody) != "echo hi" {
		t.Fatalf("expected uploaded content echo hi, got %q", string(fake.putBody))
	}
}

func TestGitHubSyncService_ConflictRequiresRename(t *testing.T) {
	cfg := &config.Config{
		GitHubSyncEnabled: true,
		GitHubBaseDir:     "skills",
		GitHubToken:       "token",
		GitHubOwner:       "owner",
		GitHubRepo:        "repo",
	}
	fake := &fakeGitHubClient{
		existing: map[string]bool{},
		dirs: map[string][]string{
			"skills/my-skill": {"skills/my-skill/existing.md"},
		},
	}
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
	if result.Status != GitHubSyncStatusFailed {
		t.Fatalf("expected failed status, got %q", result.Status)
	}
	if !strings.Contains(result.Error, "已存在") {
		t.Fatalf("expected conflict error, got %q", result.Error)
	}
	if fake.putPath != "" {
		t.Fatalf("expected no upload when conflict, got %q", fake.putPath)
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

func TestDeleteSkillFromGitHub_FilePathDeletesOnlyThatFile(t *testing.T) {
	cfg := &config.Config{
		GitHubSyncEnabled: true,
		GitHubToken:       "token",
		GitHubOwner:       "owner",
		GitHubRepo:        "repo",
	}
	fake := &fakeGitHubClient{
		existing: map[string]bool{
			"skills/my-skill/a.txt": true,
			"skills/my-skill/b.txt": true,
		},
	}
	svc := NewGitHubSyncService(cfg, fake)

	if err := svc.DeleteSkillFromGitHub(context.Background(), "skills/my-skill/a.txt"); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if len(fake.deleted) != 1 || fake.deleted[0] != "skills/my-skill/a.txt" {
		t.Fatalf("expected only a.txt deleted, got %#v", fake.deleted)
	}
	if !fake.existing["skills/my-skill/b.txt"] {
		t.Fatal("expected sibling file to remain")
	}
}

func TestDeleteSkillFilesFromGitHub_DeletesManifestEntries(t *testing.T) {
	cfg := &config.Config{
		GitHubSyncEnabled: true,
		GitHubToken:       "token",
		GitHubOwner:       "owner",
		GitHubRepo:        "repo",
	}
	fake := &fakeGitHubClient{
		existing: map[string]bool{
			"skills/my-skill/a.txt": true,
			"skills/my-skill/b.txt": true,
		},
	}
	svc := NewGitHubSyncService(cfg, fake)

	err := svc.DeleteSkillFilesFromGitHub(context.Background(), []string{
		"skills/my-skill/a.txt",
		"skills/my-skill/b.txt",
		"skills/my-skill/missing.txt",
	})
	if err != nil {
		t.Fatalf("unexpected delete manifest error: %v", err)
	}
	if len(fake.deleted) != 2 {
		t.Fatalf("expected two deleted files, got %#v", fake.deleted)
	}
}
