package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"skill-hub/internal/model"
	"skill-hub/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	reviewFileStatusQueued  = "queued"
	reviewFileStatusRunning = "running"
	reviewFileStatusPassed  = "passed"
	reviewFileStatusFailed  = "failed"

	reviewContentLimitBytes = 12000
)

var reviewScriptExtSet = map[string]struct{}{
	".sh":   {},
	".bash": {},
	".zsh":  {},
	".ps1":  {},
	".bat":  {},
	".cmd":  {},
	".py":   {},
	".js":   {},
	".ts":   {},
	".mjs":  {},
	".cjs":  {},
}

var localRiskRules = []struct {
	pattern *regexp.Regexp
	reason  string
}{
	{regexp.MustCompile(`(?i)\brm\s+-rf\s+/`), "检测到高危删除命令 rm -rf /"},
	{regexp.MustCompile(`(?i)\b(curl|wget)\b[^\n|]*\|\s*(bash|sh|zsh)\b`), "检测到远程下载后直接执行"},
	{regexp.MustCompile(`(?i)\bnc\b[^\n]*\s-e\s+`), "检测到 netcat 执行参数（疑似反弹 shell）"},
	{regexp.MustCompile(`(?i)\bchmod\s+777\s+/`), "检测到对根目录的宽权限变更"},
	{regexp.MustCompile(`(?i)\bmkfs(\.[a-z0-9]+)?\b`), "检测到磁盘格式化命令"},
	{regexp.MustCompile(`(?i)\bdd\s+if=/dev/(zero|random)[^\n]*of=/dev/(sd[a-z]|nvme\d+n\d+)`), "检测到覆盖磁盘设备命令"},
	{regexp.MustCompile(`(?i)\bpowershell\b[^\n]*-enc(odedcommand)?\b`), "检测到 PowerShell 编码命令执行"},
}

type skillReviewStatusResponse struct {
	Status         string                 `json:"status"`
	Phase          string                 `json:"phase"`
	Attempts       int                    `json:"attempts"`
	MaxAttempts    int                    `json:"max_attempts"`
	RetryRemaining int                    `json:"retry_remaining"`
	CanRetry       bool                   `json:"can_retry"`
	Approved       bool                   `json:"approved"`
	Feedback       string                 `json:"feedback"`
	Progress       *reviewProgressDetails `json:"progress,omitempty"`
}

type reviewProgressDetails struct {
	TotalFiles     int                      `json:"total_files"`
	CompletedFiles int                      `json:"completed_files"`
	CurrentFile    string                   `json:"current_file,omitempty"`
	Files          []reviewFileProgressItem `json:"files"`
}

type reviewFileProgressItem struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type reviewTarget struct {
	Path      string
	Kind      string
	LocalPath string
}

// GetSkillReviewStatus handles GET /api/skills/:id/review-status.
func (h *SkillHandler) GetSkillReviewStatus(c *gin.Context) {
	userID, username, ok := currentUserIdentity(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或登录状态无效"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	skill, err := h.getSkillResource(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "资源不存在"})
		return
	}

	if !canManageSkill(skill, userID, username) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只能查看自己上传资源的审核状态"})
		return
	}

	c.JSON(http.StatusOK, buildReviewStatusResponse(skill))
}

// RetrySkillReview handles POST /api/skills/:id/review/retry.
func (h *SkillHandler) RetrySkillReview(c *gin.Context) {
	userID, username, ok := currentUserIdentity(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或登录状态无效"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	skill, err := h.getSkillResource(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "资源不存在"})
		return
	}

	if !canManageSkill(skill, userID, username) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只能重试自己上传资源的审核"})
		return
	}

	if skill.AIReviewStatus == model.AIReviewStatusRunning || skill.AIReviewStatus == model.AIReviewStatusQueued {
		c.JSON(http.StatusConflict, gin.H{"error": "审核正在进行中，请稍候"})
		return
	}
	if skill.AIReviewStatus == model.AIReviewStatusPassed {
		c.JSON(http.StatusConflict, gin.H{"error": "该资源已通过审核，无需重试"})
		return
	}

	maxAttempts := skill.AIReviewMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
		skill.AIReviewMaxAttempts = 3
	}
	if skill.AIReviewAttempts >= maxAttempts {
		skill.AIReviewStatus = model.AIReviewStatusFailedTerminal
		if err := h.skillSvc.UpdateSkill(skill); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新审核状态失败"})
			return
		}
		c.JSON(http.StatusConflict, gin.H{"error": "已达最大重试次数，请重新上传", "status": buildReviewStatusResponse(skill)})
		return
	}

	skill.AIReviewStatus = model.AIReviewStatusQueued
	skill.AIReviewPhase = model.AIReviewPhaseQueued
	skill.AIReviewDetails = ""
	skill.AIFeedback = "重新排队中，请稍候..."
	if err := h.skillSvc.UpdateSkill(skill); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新审核状态失败"})
		return
	}

	h.dispatchAIReview(skill.ID)

	c.JSON(http.StatusOK, gin.H{
		"message": "已重新触发审核",
		"status":  buildReviewStatusResponse(skill),
	})
}

func buildReviewStatusResponse(skill *model.Skill) skillReviewStatusResponse {
	maxAttempts := skill.AIReviewMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	retryRemaining := maxAttempts - skill.AIReviewAttempts
	if retryRemaining < 0 {
		retryRemaining = 0
	}
	canRetry := (skill.AIReviewStatus == model.AIReviewStatusFailedRetry || skill.AIReviewStatus == model.AIReviewStatusFailedTerminal) &&
		skill.AIReviewAttempts < maxAttempts

	return skillReviewStatusResponse{
		Status:         skill.AIReviewStatus,
		Phase:          skill.AIReviewPhase,
		Attempts:       skill.AIReviewAttempts,
		MaxAttempts:    maxAttempts,
		RetryRemaining: retryRemaining,
		CanRetry:       canRetry,
		Approved:       skill.AIApproved,
		Feedback:       skill.AIFeedback,
		Progress:       decodeReviewProgress(skill.AIReviewDetails),
	}
}

func (h *SkillHandler) dispatchAIReview(skillID uint) {
	if h.aiSvc == nil {
		return
	}

	h.reviewMu.Lock()
	if _, exists := h.reviewRunning[skillID]; exists {
		h.reviewMu.Unlock()
		return
	}
	h.reviewRunning[skillID] = struct{}{}
	h.reviewMu.Unlock()

	go h.runAIReview(skillID)
}

func (h *SkillHandler) runAIReview(skillID uint) {
	defer func() {
		h.reviewMu.Lock()
		delete(h.reviewRunning, skillID)
		h.reviewMu.Unlock()
	}()

	skill, err := h.getSkillResource(skillID)
	if err != nil {
		return
	}
	if skill.AIReviewStatus == model.AIReviewStatusPassed {
		return
	}

	maxAttempts := skill.AIReviewMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	if skill.AIReviewAttempts >= maxAttempts {
		skill.AIReviewStatus = model.AIReviewStatusFailedTerminal
		skill.AIFeedback = "已达最大重试次数，请重新上传"
		_ = h.skillSvc.UpdateSkill(skill)
		return
	}

	now := time.Now()
	skill.AIReviewAttempts++
	skill.AIReviewStatus = model.AIReviewStatusRunning
	skill.AIReviewPhase = model.AIReviewPhaseSecurity
	skill.AIReviewStartedAt = &now
	skill.AIReviewCompletedAt = nil
	skill.AIReviewDetails = ""
	skill.AIFeedback = "准备审核文件..."
	skill.AIApproved = false
	if err := h.skillSvc.UpdateSkill(skill); err != nil {
		return
	}

	targets, err := h.collectReviewTargets(skill)
	if err != nil {
		h.finishReviewAsError(skill, maxAttempts, fmt.Sprintf("准备审核文件失败: %v", err))
		return
	}

	progress := newReviewProgress(targets)
	skill.AIReviewDetails = encodeReviewProgress(progress)
	skill.AIFeedback = fmt.Sprintf("共发现 %d 个待审核文件", progress.TotalFiles)
	if err := h.skillSvc.UpdateSkill(skill); err != nil {
		return
	}

	failedFiles := make([]string, 0)
	for i := range targets {
		target := targets[i]
		progress.CurrentFile = target.Path
		progress.Files[i].Status = reviewFileStatusRunning
		progress.Files[i].Message = ""

		skill.AIReviewPhase = model.AIReviewPhaseSecurity
		skill.AIFeedback = fmt.Sprintf("安全性审核中：%s", target.Path)
		skill.AIReviewDetails = encodeReviewProgress(progress)
		if err := h.skillSvc.UpdateSkill(skill); err != nil {
			return
		}

		content, readErr := readReviewContent(target.LocalPath, reviewContentLimitBytes)
		if readErr != nil {
			progress.Files[i].Status = reviewFileStatusFailed
			progress.Files[i].Message = fmt.Sprintf("读取失败: %v", readErr)
			progress.CompletedFiles++
			skill.AIReviewDetails = encodeReviewProgress(progress)
			h.finishReviewAsError(skill, maxAttempts, fmt.Sprintf("读取文件失败（%s）: %v", target.Path, readErr))
			return
		}

		localFindings := detectLocalRiskFindings(content)

		skill.AIReviewPhase = model.AIReviewPhaseFunctional
		skill.AIFeedback = fmt.Sprintf("功能性审核中：%s", target.Path)
		skill.AIReviewDetails = encodeReviewProgress(progress)
		if err := h.skillSvc.UpdateSkill(skill); err != nil {
			return
		}

		reviewResult, aiErr := h.aiSvc.ReviewSkill(
			skill.Name,
			skill.ResourceType,
			fmt.Sprintf("%s\n审查文件: %s\n文件类型: %s", skill.Description, target.Path, target.Kind),
			content,
		)
		if aiErr != nil {
			progress.Files[i].Status = reviewFileStatusFailed
			progress.Files[i].Message = "AI 调用失败"
			progress.CompletedFiles++
			skill.AIReviewDetails = encodeReviewProgress(progress)
			h.finishReviewAsError(skill, maxAttempts, fmt.Sprintf("AI 审核失败（%s）: %v", target.Path, aiErr))
			return
		}

		filePassed := true
		messages := make([]string, 0, 2)
		if len(localFindings) > 0 {
			filePassed = false
			messages = append(messages, "命中风险规则: "+strings.Join(localFindings, "；"))
		}
		if !reviewResult.Approved {
			filePassed = false
			if msg := strings.TrimSpace(reviewResult.Feedback); msg != "" {
				messages = append(messages, truncateReviewMessage(msg, 160))
			} else {
				messages = append(messages, "AI 判定存在安全或功能风险")
			}
		}

		if filePassed {
			progress.Files[i].Status = reviewFileStatusPassed
			if msg := strings.TrimSpace(reviewResult.Feedback); msg != "" {
				progress.Files[i].Message = truncateReviewMessage(msg, 120)
			}
		} else {
			progress.Files[i].Status = reviewFileStatusFailed
			progress.Files[i].Message = truncateReviewMessage(strings.Join(messages, "；"), 220)
			failedFiles = append(failedFiles, target.Path)
		}
		progress.CompletedFiles++
		skill.AIReviewDetails = encodeReviewProgress(progress)
		if err := h.skillSvc.UpdateSkill(skill); err != nil {
			return
		}
	}

	skill.AIReviewPhase = model.AIReviewPhaseFinalizing
	skill.AIFeedback = "结果归档中..."
	skill.AIReviewDetails = encodeReviewProgress(progress)
	_ = h.skillSvc.UpdateSkill(skill)

	doneAt := time.Now()
	progress.CurrentFile = ""
	skill.AIReviewCompletedAt = &doneAt
	skill.AIReviewPhase = model.AIReviewPhaseDone
	skill.AIReviewDetails = encodeReviewProgress(progress)

	if len(failedFiles) == 0 {
		skill.AIApproved = true
		skill.AIReviewStatus = model.AIReviewStatusPassed
		skill.AIFeedback = fmt.Sprintf("AI 审核通过，已检查 %d 个关键文件", len(targets))
		skill.AIDescription = fmt.Sprintf("安全性: 未发现明显风险。\n功能性: 已完成 %d 个文件审查。", len(targets))
	} else {
		skill.AIApproved = false
		if skill.AIReviewAttempts >= maxAttempts {
			skill.AIReviewStatus = model.AIReviewStatusFailedTerminal
		} else {
			skill.AIReviewStatus = model.AIReviewStatusFailedRetry
		}
		skill.AIFeedback = fmt.Sprintf("发现 %d 个风险文件：%s", len(failedFiles), summarizeFileList(failedFiles, 3))
		skill.AIDescription = fmt.Sprintf("安全性: 检测到 %d 个风险文件。\n功能性: 请修复后重新上传。", len(failedFiles))
	}

	_ = h.skillSvc.UpdateSkill(skill)
}

func (h *SkillHandler) finishReviewAsError(skill *model.Skill, maxAttempts int, message string) {
	doneAt := time.Now()
	skill.AIApproved = false
	skill.AIReviewCompletedAt = &doneAt
	skill.AIReviewPhase = model.AIReviewPhaseDone
	if skill.AIReviewAttempts >= maxAttempts {
		skill.AIReviewStatus = model.AIReviewStatusFailedTerminal
	} else {
		skill.AIReviewStatus = model.AIReviewStatusFailedRetry
	}
	skill.AIFeedback = strings.TrimSpace(message)
	if skill.AIFeedback == "" {
		skill.AIFeedback = "AI 审核失败，请稍后重试"
	}
	if strings.TrimSpace(skill.AIDescription) == "" {
		skill.AIDescription = "安全性: 审核未完成。\n功能性: 请重试审核。"
	}
	_ = h.skillSvc.UpdateSkill(skill)
}

func newReviewProgress(targets []reviewTarget) reviewProgressDetails {
	items := make([]reviewFileProgressItem, 0, len(targets))
	for _, target := range targets {
		items = append(items, reviewFileProgressItem{
			Path:   target.Path,
			Kind:   target.Kind,
			Status: reviewFileStatusQueued,
		})
	}
	return reviewProgressDetails{
		TotalFiles:     len(targets),
		CompletedFiles: 0,
		Files:          items,
	}
}

func encodeReviewProgress(progress reviewProgressDetails) string {
	raw, err := json.Marshal(progress)
	if err != nil {
		return ""
	}
	return string(raw)
}

func decodeReviewProgress(raw string) *reviewProgressDetails {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var details reviewProgressDetails
	if err := json.Unmarshal([]byte(trimmed), &details); err != nil {
		return nil
	}
	if details.TotalFiles <= 0 || len(details.Files) == 0 {
		return nil
	}
	if details.CompletedFiles < 0 {
		details.CompletedFiles = 0
	}
	if details.CompletedFiles > details.TotalFiles {
		details.CompletedFiles = details.TotalFiles
	}
	return &details
}

func (h *SkillHandler) collectReviewTargets(skill *model.Skill) ([]reviewTarget, error) {
	if skill == nil || strings.TrimSpace(skill.FilePath) == "" {
		return nil, fmt.Errorf("empty file path")
	}

	root := skill.FilePath
	sessionRoot := uploadSessionRoot(h.cfg.UploadDir, skill.FilePath)
	if sessionRoot != "" {
		if info, err := os.Stat(sessionRoot); err == nil && info.IsDir() {
			root = sessionRoot
		}
	}

	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}

	targets := make([]reviewTarget, 0, 16)
	if info.IsDir() {
		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}

			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			mode := fs.FileMode(0)
			if fileInfo, err := d.Info(); err == nil {
				mode = fileInfo.Mode()
			}
			if kind, ok := classifyReviewTarget(rel, path, mode); ok {
				targets = append(targets, reviewTarget{
					Path:      rel,
					Kind:      kind,
					LocalPath: path,
				})
			}
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	} else {
		rel := sanitizeLocalFilename(filepath.Base(root))
		if kind, ok := classifyReviewTarget(rel, root, info.Mode()); ok {
			targets = append(targets, reviewTarget{Path: rel, Kind: kind, LocalPath: root})
		}
	}

	if len(targets) == 0 {
		fallbackInfo, err := os.Stat(skill.FilePath)
		if err != nil {
			return nil, err
		}
		fallbackPath := sanitizeLocalFilename(filepath.Base(skill.FilePath))
		kind := "primary-file"
		if classified, ok := classifyReviewTarget(fallbackPath, skill.FilePath, fallbackInfo.Mode()); ok {
			kind = classified
		}
		targets = append(targets, reviewTarget{
			Path:      fallbackPath,
			Kind:      kind,
			LocalPath: skill.FilePath,
		})
	}

	targets = dedupeReviewTargets(targets)
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Path < targets[j].Path
	})
	return targets, nil
}

func dedupeReviewTargets(items []reviewTarget) []reviewTarget {
	result := make([]reviewTarget, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.Path == "" {
			continue
		}
		if _, ok := seen[item.Path]; ok {
			continue
		}
		seen[item.Path] = struct{}{}
		result = append(result, item)
	}
	return result
}

func classifyReviewTarget(relPath, localPath string, mode fs.FileMode) (string, bool) {
	rel := strings.TrimPrefix(filepath.ToSlash(relPath), "./")
	if rel == "" || rel == "." {
		return "", false
	}
	lowerRel := strings.ToLower(rel)
	base := filepath.Base(rel)
	lowerBase := strings.ToLower(base)
	ext := strings.ToLower(filepath.Ext(rel))

	if ext == ".md" || ext == ".mdx" {
		return "markdown", true
	}
	if lowerBase == "package.json" {
		return "package-json", true
	}
	if lowerBase == "makefile" {
		return "makefile", true
	}
	if strings.HasPrefix(lowerBase, "dockerfile") {
		return "dockerfile", true
	}
	if (strings.HasPrefix(lowerRel, ".github/workflows/") || strings.Contains(lowerRel, "/.github/workflows/")) && (ext == ".yml" || ext == ".yaml") {
		return "workflow", true
	}
	if _, ok := reviewScriptExtSet[ext]; ok {
		return "script", true
	}
	if mode&0o111 != 0 {
		return "script", true
	}
	if hasShebang(localPath) {
		return "script", true
	}
	return "", false
}

func hasShebang(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 256)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}
	line := string(buf[:n])
	line = strings.TrimSpace(strings.SplitN(line, "\n", 2)[0])
	return strings.HasPrefix(line, "#!")
}

func readReviewContent(path string, limit int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf, err := io.ReadAll(io.LimitReader(f, limit))
	if err != nil {
		return "", err
	}
	if bytes.Contains(buf, []byte{0}) {
		return "", fmt.Errorf("文件可能为二进制")
	}
	return string(buf), nil
}

func detectLocalRiskFindings(content string) []string {
	if strings.TrimSpace(content) == "" {
		return []string{"文件内容为空"}
	}
	findings := make([]string, 0)
	for _, rule := range localRiskRules {
		if rule.pattern.MatchString(content) {
			findings = append(findings, rule.reason)
		}
	}
	return findings
}

func summarizeFileList(files []string, limit int) string {
	if len(files) == 0 {
		return ""
	}
	if limit <= 0 || len(files) <= limit {
		return strings.Join(files, "、")
	}
	head := strings.Join(files[:limit], "、")
	return fmt.Sprintf("%s 等 %d 个文件", head, len(files))
}

func truncateReviewMessage(msg string, max int) string {
	trimmed := strings.TrimSpace(msg)
	if max <= 0 {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= max {
		return trimmed
	}
	return string(runes[:max]) + "..."
}

func (h *SkillHandler) collectSyncEntriesFromLocal(skill *model.Skill) ([]service.SyncFileEntry, error) {
	if skill == nil {
		return nil, fmt.Errorf("skill is nil")
	}
	if strings.TrimSpace(skill.FilePath) == "" {
		return nil, fmt.Errorf("empty file path")
	}

	var entries []service.SyncFileEntry
	sessionRoot := uploadSessionRoot(h.cfg.UploadDir, skill.FilePath)
	if sessionRoot != "" {
		if info, err := os.Stat(sessionRoot); err != nil || !info.IsDir() {
			sessionRoot = ""
		}
	}
	if sessionRoot != "" {
		walkErr := filepath.WalkDir(sessionRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(sessionRoot, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			normalizedRel, err := service.NormalizeRepoRelativePath(rel)
			if err != nil {
				return err
			}
			entries = append(entries, service.SyncFileEntry{
				LocalPath:    path,
				RelativePath: normalizedRel,
			})
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}

	if len(entries) == 0 {
		if _, err := os.Stat(skill.FilePath); err != nil {
			return nil, err
		}
		entries = append(entries, service.SyncFileEntry{
			LocalPath:    skill.FilePath,
			RelativePath: sanitizeLocalFilename(filepath.Base(skill.FilePath)),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].RelativePath < entries[j].RelativePath
	})
	return entries, nil
}

func collectReviewTargetsFromSession(sessionRoot string) ([]reviewTarget, error) {
	info, err := os.Stat(sessionRoot)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("session root is not a directory")
	}

	targets := make([]reviewTarget, 0, 16)
	walkErr := filepath.WalkDir(sessionRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(sessionRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		mode := fs.FileMode(0)
		if fileInfo, err := d.Info(); err == nil {
			mode = fileInfo.Mode()
		}
		if kind, ok := classifyReviewTarget(rel, path, mode); ok {
			targets = append(targets, reviewTarget{Path: rel, Kind: kind, LocalPath: path})
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	targets = dedupeReviewTargets(targets)
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Path < targets[j].Path
	})
	return targets, nil
}
