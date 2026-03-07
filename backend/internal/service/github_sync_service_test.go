package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

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

func TestGitHubSyncService_OverwritesExistingDirectory(t *testing.T) {
	cfg := &config.Config{
		GitHubSyncEnabled: true,
		GitHubBaseDir:     "skills",
		GitHubToken:       "token",
		GitHubOwner:       "owner",
		GitHubRepo:        "repo",
	}
	fake := &fakeGitHubClient{
		existing: map[string]bool{
			"skills/my-skill/existing.md": true,
		},
		dirs: map[string][]string{
			"skills/my-skill": {"skills/my-skill/existing.md"},
		},
	}
	svc := NewGitHubSyncService(cfg, fake)

	tmp := t.TempDir()
	localFile := filepath.Join(tmp, "a.txt")
	if err := os.WriteFile(localFile, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	result := svc.SyncUploadedSkill(context.Background(), "My Skill", "skill", "a.txt", localFile)
	if result.Status != GitHubSyncStatusSuccess {
		t.Fatalf("expected success status, got %q (error=%q)", result.Status, result.Error)
	}
	if fake.putPath != "skills/my-skill/a.txt" {
		t.Fatalf("expected upload to target file, got %q", fake.putPath)
	}
	if len(fake.deleted) != 1 || fake.deleted[0] != "skills/my-skill/existing.md" {
		t.Fatalf("expected stale file deleted, got %#v", fake.deleted)
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

func TestGitHubSyncService_SyncUploadedFolderReplacesStaleFiles(t *testing.T) {
	cfg := &config.Config{
		GitHubSyncEnabled: true,
		GitHubBaseDir:     "skills",
		GitHubToken:       "token",
		GitHubOwner:       "owner",
		GitHubRepo:        "repo",
	}
	fake := &fakeGitHubClient{
		existing: map[string]bool{
			"skills/my-skill/a.txt": true,
			"skills/my-skill/b.txt": true,
			"skills/my-skill/c.txt": true,
		},
		dirs: map[string][]string{
			"skills/my-skill": {
				"skills/my-skill/a.txt",
				"skills/my-skill/b.txt",
				"skills/my-skill/c.txt",
			},
		},
	}
	svc := NewGitHubSyncService(cfg, fake)

	tmp := t.TempDir()
	localA := filepath.Join(tmp, "a.txt")
	localD := filepath.Join(tmp, "d.txt")
	if err := os.WriteFile(localA, []byte("updated"), 0o600); err != nil {
		t.Fatalf("write a.txt: %v", err)
	}
	if err := os.WriteFile(localD, []byte("new"), 0o600); err != nil {
		t.Fatalf("write d.txt: %v", err)
	}

	result := svc.SyncUploadedFolder(context.Background(), "My Skill", "skill", []SyncFileEntry{
		{LocalPath: localA, RelativePath: "a.txt"},
		{LocalPath: localD, RelativePath: "d.txt"},
	})
	if result.Status != GitHubSyncStatusSuccess {
		t.Fatalf("expected success status, got %q (error=%q)", result.Status, result.Error)
	}

	if len(result.Files) != 2 {
		t.Fatalf("expected two synced files, got %#v", result.Files)
	}

	deleted := map[string]bool{}
	for _, p := range fake.deleted {
		deleted[p] = true
	}
	if !deleted["skills/my-skill/b.txt"] || !deleted["skills/my-skill/c.txt"] {
		t.Fatalf("expected stale files b/c deleted, got %#v", fake.deleted)
	}
}
