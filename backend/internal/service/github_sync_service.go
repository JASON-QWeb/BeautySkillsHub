package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	pathpkg "path"
	"strings"
	"time"

	"skill-hub/internal/config"
)

const (
	GitHubSyncStatusDisabled   = "disabled"
	GitHubSyncStatusNotStarted = "not_started"
	GitHubSyncStatusPending    = "pending"
	GitHubSyncStatusSuccess    = "success"
	GitHubSyncStatusFailed     = "failed"
)

type GitHubSyncResult struct {
	Status string
	Path   string
	URL    string
	Error  string
	Files  []string
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

	dirPath, targetPath := BuildSkillRepoPath(s.cfg.GitHubBaseDir, resourceType, skillName, originalFilename)
	existingPaths, err := s.client.ListDir(ctx, dirPath)
	if err != nil {
		return GitHubSyncResult{
			Status: GitHubSyncStatusFailed,
			Path:   dirPath,
			Error:  fmt.Sprintf("检查 GitHub 路径失败: %v", err),
		}
	}

	sha, exists, err := s.client.GetFileSHA(ctx, targetPath)
	if err != nil {
		return GitHubSyncResult{
			Status: GitHubSyncStatusFailed,
			Path:   dirPath,
			Error:  fmt.Sprintf("检查 GitHub 文件失败: %v", err),
		}
	}

	commitMessage := fmt.Sprintf("Add skill: %s", strings.TrimSpace(skillName))
	commitSHA := ""
	if exists {
		commitMessage = fmt.Sprintf("Update skill: %s", strings.TrimSpace(skillName))
		commitSHA = sha
	}
	url, err := s.client.PutFile(ctx, targetPath, commitMessage, content, commitSHA)
	if err != nil {
		return GitHubSyncResult{
			Status: GitHubSyncStatusFailed,
			Path:   dirPath,
			Error:  fmt.Sprintf("上传到 GitHub 失败: %v", err),
		}
	}

	stalePaths := collectStalePaths(existingPaths, map[string]struct{}{targetPath: {}})
	if len(stalePaths) > 0 {
		if err := s.DeleteSkillFilesFromGitHub(ctx, stalePaths); err != nil {
			return GitHubSyncResult{
				Status: GitHubSyncStatusFailed,
				Path:   dirPath,
				Error:  fmt.Sprintf("清理 GitHub 旧文件失败: %v", err),
			}
		}
	}

	return GitHubSyncResult{
		Status: GitHubSyncStatusSuccess,
		Path:   dirPath,
		URL:    url,
		Files:  []string{targetPath},
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

	// Try to get SHA directly — if it succeeds, delete the file only.
	sha, exists, err := s.client.GetFileSHA(ctx, githubPath)
	if err == nil && exists {
		return s.client.DeleteFile(ctx, githubPath, fmt.Sprintf("Delete skill file: %s", githubPath), sha)
	}
	if err != nil {
		return fmt.Errorf("检查 GitHub 路径失败: %w", err)
	}

	// It's a directory path — list and delete all files in it
	return s.deleteAllInDir(ctx, githubPath)
}

// DeleteSkillFilesFromGitHub deletes files listed in manifest exactly.
func (s *GitHubSyncService) DeleteSkillFilesFromGitHub(ctx context.Context, files []string) error {
	if s.cfg == nil || !s.cfg.GitHubSyncEnabled || s.client == nil {
		return nil
	}
	if len(files) == 0 {
		return nil
	}

	var errs []string
	for _, f := range files {
		clean := strings.TrimSpace(f)
		if clean == "" {
			continue
		}
		if _, err := NormalizeRepoRelativePath(clean); err != nil {
			errs = append(errs, fmt.Sprintf("%s: invalid path", clean))
			continue
		}
		sha, exists, err := s.client.GetFileSHA(ctx, clean)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", clean, err))
			continue
		}
		if !exists {
			continue
		}
		if err := s.client.DeleteFile(ctx, clean, fmt.Sprintf("Delete skill file: %s", clean), sha); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", clean, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("部分文件删除失败: %s", strings.Join(errs, "; "))
	}
	return nil
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
	existingPaths, err := s.client.ListDir(ctx, dirPath)
	if err != nil {
		return GitHubSyncResult{
			Status: GitHubSyncStatusFailed,
			Path:   dirPath,
			Error:  fmt.Sprintf("检查 GitHub 路径失败: %v", err),
		}
	}
	var lastURL string
	var errors []string
	uploadedPaths := make([]string, 0, len(files))
	createdPaths := make([]string, 0, len(files))
	desiredPaths := make(map[string]struct{}, len(files))

	for _, fe := range files {
		content, err := os.ReadFile(fe.LocalPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("读取文件失败 %s: %v", fe.RelativePath, err))
			continue
		}

		normalizedRel, err := NormalizeRepoRelativePath(fe.RelativePath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("非法文件路径 %s: %v", fe.RelativePath, err))
			continue
		}
		targetPath := pathpkg.Join(dirPath, normalizedRel)

		sha, exists, err := s.client.GetFileSHA(ctx, targetPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("检查 GitHub 路径失败 %s: %v", targetPath, err))
			continue
		}

		commitMessage := fmt.Sprintf("Add skill: %s - %s", strings.TrimSpace(skillName), fe.RelativePath)
		commitSHA := ""
		if exists {
			commitMessage = fmt.Sprintf("Update skill: %s - %s", strings.TrimSpace(skillName), fe.RelativePath)
			commitSHA = sha
		}
		url, err := s.client.PutFile(ctx, targetPath, commitMessage, content, commitSHA)
		if err != nil {
			errors = append(errors, fmt.Sprintf("上传失败 %s: %v", fe.RelativePath, err))
			continue
		}
		if !exists {
			createdPaths = append(createdPaths, targetPath)
		}
		lastURL = url
		uploadedPaths = append(uploadedPaths, targetPath)
		desiredPaths[targetPath] = struct{}{}
	}

	if len(errors) > 0 {
		if len(createdPaths) > 0 {
			rollbackErr := s.DeleteSkillFilesFromGitHub(ctx, createdPaths)
			if rollbackErr != nil {
				errors = append(errors, fmt.Sprintf("回滚失败: %v", rollbackErr))
			}
		}
		return GitHubSyncResult{
			Status: GitHubSyncStatusFailed,
			Path:   dirPath,
			Error:  strings.Join(errors, "; "),
		}
	}

	stalePaths := collectStalePaths(existingPaths, desiredPaths)
	if len(stalePaths) > 0 {
		if err := s.DeleteSkillFilesFromGitHub(ctx, stalePaths); err != nil {
			return GitHubSyncResult{
				Status: GitHubSyncStatusFailed,
				Path:   dirPath,
				Error:  fmt.Sprintf("清理 GitHub 旧文件失败: %v", err),
			}
		}
	}

	result := GitHubSyncResult{
		Status: GitHubSyncStatusSuccess,
		Path:   dirPath,
		URL:    lastURL,
		Files:  uploadedPaths,
	}
	return result
}

func appendTimestampSuffix(filePath string, ts time.Time) string {
	ext := pathpkg.Ext(filePath)
	prefix := strings.TrimSuffix(filePath, ext)
	return fmt.Sprintf("%s_%s%s", prefix, ts.Format("20060102_150405"), ext)
}

func collectStalePaths(existingPaths []string, keep map[string]struct{}) []string {
	if len(existingPaths) == 0 {
		return nil
	}
	stale := make([]string, 0, len(existingPaths))
	for _, filePath := range existingPaths {
		if _, ok := keep[filePath]; ok {
			continue
		}
		stale = append(stale, filePath)
	}
	return stale
}

func MarshalGitHubFiles(files []string) string {
	if len(files) == 0 {
		return ""
	}
	b, err := json.Marshal(files)
	if err != nil {
		return ""
	}
	return string(b)
}

func UnmarshalGitHubFiles(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var files []string
	if err := json.Unmarshal([]byte(raw), &files); err != nil {
		return nil
	}
	return files
}
