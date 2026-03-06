package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	pathpkg "path"
	"strings"
	"time"

	"skill-hub/internal/config"
)

const (
	GitHubSyncStatusDisabled = "disabled"
	GitHubSyncStatusPending  = "pending"
	GitHubSyncStatusSuccess  = "success"
	GitHubSyncStatusFailed   = "failed"
)

type GitHubSyncResult struct {
	Status string
	Path   string
	URL    string
	Error  string
}

type GitHubSyncService struct {
	cfg    *config.Config
	client GitHubContentClient
	nowFn  func() time.Time
}

func NewGitHubSyncService(cfg *config.Config, client GitHubContentClient) *GitHubSyncService {
	if client == nil && cfg != nil && cfg.GitHubSyncEnabled {
		client = NewGitHubClient(
			http.DefaultClient,
			"https://api.github.com",
			cfg.GitHubOwner,
			cfg.GitHubRepo,
			cfg.GitHubBranch,
			cfg.GitHubToken,
		)
	}

	return &GitHubSyncService{
		cfg:    cfg,
		client: client,
		nowFn:  time.Now,
	}
}

func (s *GitHubSyncService) SyncUploadedSkill(
	ctx context.Context,
	skillName,
	resourceType,
	originalFilename,
	localFilePath string,
) GitHubSyncResult {
	if s.cfg == nil || !s.cfg.GitHubSyncEnabled {
		return GitHubSyncResult{Status: GitHubSyncStatusDisabled}
	}
	if strings.TrimSpace(s.cfg.GitHubToken) == "" || strings.TrimSpace(s.cfg.GitHubOwner) == "" || strings.TrimSpace(s.cfg.GitHubRepo) == "" {
		return GitHubSyncResult{
			Status: GitHubSyncStatusFailed,
			Error:  "GitHub 同步配置不完整：需要 GITHUB_TOKEN / GITHUB_OWNER / GITHUB_REPO",
		}
	}
	if s.client == nil {
		return GitHubSyncResult{
			Status: GitHubSyncStatusFailed,
			Error:  "GitHub 同步客户端未初始化",
		}
	}

	content, err := os.ReadFile(localFilePath)
	if err != nil {
		return GitHubSyncResult{
			Status: GitHubSyncStatusFailed,
			Error:  fmt.Sprintf("读取本地文件失败: %v", err),
		}
	}

	_, targetPath := BuildSkillRepoPath(s.cfg.GitHubBaseDir, resourceType, skillName, originalFilename)

	_, exists, err := s.client.GetFileSHA(ctx, targetPath)
	if err != nil {
		return GitHubSyncResult{
			Status: GitHubSyncStatusFailed,
			Path:   targetPath,
			Error:  fmt.Sprintf("检查 GitHub 路径失败: %v", err),
		}
	}
	if exists {
		targetPath = appendTimestampSuffix(targetPath, s.nowFn())
	}

	commitMessage := fmt.Sprintf("Add skill: %s", strings.TrimSpace(skillName))
	url, err := s.client.PutFile(ctx, targetPath, commitMessage, content, "")
	if err != nil {
		return GitHubSyncResult{
			Status: GitHubSyncStatusFailed,
			Path:   targetPath,
			Error:  fmt.Sprintf("上传到 GitHub 失败: %v", err),
		}
	}

	return GitHubSyncResult{
		Status: GitHubSyncStatusSuccess,
		Path:   targetPath,
		URL:    url,
	}
}

// DeleteSkillFromGitHub deletes all files under the skill's GitHub path.
// githubPath can be a directory (folder upload) or a file path (single file upload).
func (s *GitHubSyncService) DeleteSkillFromGitHub(ctx context.Context, githubPath string) error {
	if s.cfg == nil || !s.cfg.GitHubSyncEnabled || s.client == nil {
		return nil
	}
	if strings.TrimSpace(githubPath) == "" {
		return nil
	}

	// Try to get SHA directly — if it succeeds, the path is a file
	_, exists, err := s.client.GetFileSHA(ctx, githubPath)
	if err == nil && exists {
		// It's a file path — delete all files in the parent directory
		dirPath := pathpkg.Dir(githubPath)
		return s.deleteAllInDir(ctx, dirPath)
	}

	// It's a directory path — list and delete all files in it
	return s.deleteAllInDir(ctx, githubPath)
}

func (s *GitHubSyncService) deleteAllInDir(ctx context.Context, dirPath string) error {
	files, err := s.client.ListDir(ctx, dirPath)
	if err != nil {
		return fmt.Errorf("列出 GitHub 目录失败: %w", err)
	}
	if len(files) == 0 {
		return nil
	}

	var errors []string
	for _, filePath := range files {
		sha, exists, err := s.client.GetFileSHA(ctx, filePath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filePath, err))
			continue
		}
		if !exists {
			continue
		}
		if err := s.client.DeleteFile(ctx, filePath, fmt.Sprintf("Delete skill file: %s", filePath), sha); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filePath, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("部分文件删除失败: %s", strings.Join(errors, "; "))
	}
	return nil
}

// SyncFileEntry represents a single file in a folder upload.
type SyncFileEntry struct {
	LocalPath    string // absolute path on disk
	RelativePath string // relative path within the skill folder
}

// SyncUploadedFolder syncs multiple files from a folder upload to GitHub.
func (s *GitHubSyncService) SyncUploadedFolder(
	ctx context.Context,
	skillName,
	resourceType string,
	files []SyncFileEntry,
) GitHubSyncResult {
	if s.cfg == nil || !s.cfg.GitHubSyncEnabled {
		return GitHubSyncResult{Status: GitHubSyncStatusDisabled}
	}
	if strings.TrimSpace(s.cfg.GitHubToken) == "" || strings.TrimSpace(s.cfg.GitHubOwner) == "" || strings.TrimSpace(s.cfg.GitHubRepo) == "" {
		return GitHubSyncResult{
			Status: GitHubSyncStatusFailed,
			Error:  "GitHub 同步配置不完整：需要 GITHUB_TOKEN / GITHUB_OWNER / GITHUB_REPO",
		}
	}
	if s.client == nil {
		return GitHubSyncResult{
			Status: GitHubSyncStatusFailed,
			Error:  "GitHub 同步客户端未初始化",
		}
	}

	base := cleanPathSegment(s.cfg.GitHubBaseDir, "skills")
	folder := slugifyTitle(skillName)
	dirPath := pathpkg.Join(base, folder)

	var lastURL string
	var errors []string

	for _, fe := range files {
		content, err := os.ReadFile(fe.LocalPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("读取文件失败 %s: %v", fe.RelativePath, err))
			continue
		}

		targetPath := pathpkg.Join(dirPath, sanitizeFilename(fe.RelativePath))
		// Preserve subdirectory structure for nested files
		if strings.Contains(fe.RelativePath, "/") {
			parts := strings.Split(fe.RelativePath, "/")
			for i, p := range parts {
				parts[i] = sanitizeFilename(p)
			}
			targetPath = pathpkg.Join(dirPath, pathpkg.Join(parts...))
		}

		sha, exists, err := s.client.GetFileSHA(ctx, targetPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("检查 GitHub 路径失败 %s: %v", targetPath, err))
			continue
		}

		commitMessage := fmt.Sprintf("Add skill: %s - %s", strings.TrimSpace(skillName), fe.RelativePath)
		existingSHA := ""
		if exists {
			existingSHA = sha
		}

		url, err := s.client.PutFile(ctx, targetPath, commitMessage, content, existingSHA)
		if err != nil {
			errors = append(errors, fmt.Sprintf("上传失败 %s: %v", fe.RelativePath, err))
			continue
		}
		lastURL = url
	}

	if len(errors) > 0 && len(errors) == len(files) {
		return GitHubSyncResult{
			Status: GitHubSyncStatusFailed,
			Path:   dirPath,
			Error:  strings.Join(errors, "; "),
		}
	}

	result := GitHubSyncResult{
		Status: GitHubSyncStatusSuccess,
		Path:   dirPath,
		URL:    lastURL,
	}
	if len(errors) > 0 {
		result.Error = fmt.Sprintf("部分文件上传失败: %s", strings.Join(errors, "; "))
	}
	return result
}

func appendTimestampSuffix(filePath string, ts time.Time) string {
	ext := pathpkg.Ext(filePath)
	prefix := strings.TrimSuffix(filePath, ext)
	return fmt.Sprintf("%s_%s%s", prefix, ts.Format("20060102_150405"), ext)
}
