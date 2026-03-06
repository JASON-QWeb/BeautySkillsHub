package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"skill-hub/internal/config"
	"skill-hub/internal/model"
	"skill-hub/internal/service"

	"github.com/gin-gonic/gin"
)

type SkillHandler struct {
	skillSvc      *service.SkillService
	aiSvc         *service.AIService
	githubSyncSvc *service.GitHubSyncService
	cfg           *config.Config
}

func NewSkillHandler(
	skillSvc *service.SkillService,
	aiSvc *service.AIService,
	githubSyncSvc *service.GitHubSyncService,
	cfg *config.Config,
) *SkillHandler {
	return &SkillHandler{
		skillSvc:      skillSvc,
		aiSvc:         aiSvc,
		githubSyncSvc: githubSyncSvc,
		cfg:           cfg,
	}
}

// validResourceTypes defines the allowed resource type values.
var validResourceTypes = map[string]bool{
	"skill": true,
	"mcp":   true,
	"rules": true,
	"tools": true,
}

// ListSkills handles GET /api/skills
func (h *SkillHandler) ListSkills(c *gin.Context) {
	search := c.Query("search")
	category := c.Query("category")
	resourceType := c.DefaultQuery("resource_type", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	skills, total, err := h.skillSvc.ListSkills(search, category, resourceType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add full thumbnail URLs
	for i := range skills {
		if skills[i].ThumbnailURL != "" && !strings.HasPrefix(skills[i].ThumbnailURL, "http") {
			skills[i].ThumbnailURL = "/api/thumbnails/" + skills[i].ThumbnailURL
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"skills":    skills,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetCategories handles GET /api/categories
func (h *SkillHandler) GetCategories(c *gin.Context) {
	resourceType := c.Query("resource_type")
	categories, err := h.skillSvc.GetCategories(resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, categories)
}

// GetSkill handles GET /api/skills/:id
func (h *SkillHandler) GetSkill(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	skill, err := h.skillSvc.GetSkill(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "资源不存在"})
		return
	}

	if skill.ThumbnailURL != "" && !strings.HasPrefix(skill.ThumbnailURL, "http") {
		skill.ThumbnailURL = "/api/thumbnails/" + skill.ThumbnailURL
	}

	c.JSON(http.StatusOK, skill)
}

// UploadSkill handles POST /api/skills
func (h *SkillHandler) UploadSkill(c *gin.Context) {
	name := c.PostForm("name")
	description := c.PostForm("description")
	category := c.PostForm("category")
	author := c.PostForm("author")
	resourceType := c.DefaultPostForm("resource_type", "skill")
	uploadMode := c.DefaultPostForm("upload_mode", "file")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "名称不能为空"})
		return
	}

	if !validResourceTypes[resourceType] {
		resourceType = "skill"
	}

	// Ensure upload dir exists
	if err := os.MkdirAll(h.cfg.UploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建上传目录失败"})
		return
	}

	var primaryFileName string
	var primaryFilePath string
	var totalFileSize int64
	var contentPreview string

	// Collect files to sync to GitHub: [{localPath, repoRelativePath}]
	type fileEntry struct {
		localPath string
		repoPath  string // relative path within the skill folder
	}
	var filesToSync []fileEntry

	if uploadMode == "folder" {
		// Folder upload: multiple files with relative paths
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "解析表单失败"})
			return
		}
		files := form.File["files"]
		filePaths := form.Value["file_paths"]

		if len(files) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请选择文件夹"})
			return
		}

		// Create a subdirectory for this folder upload
		safeName := strings.ReplaceAll(strings.ToLower(name), " ", "_")
		folderDir := filepath.Join(h.cfg.UploadDir, safeName)
		if err := os.MkdirAll(folderDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建上传目录失败"})
			return
		}

		for i, fh := range files {
			// Determine relative path within folder
			relPath := fh.Filename
			if i < len(filePaths) && filePaths[i] != "" {
				relPath = filePaths[i]
			}

			// Create subdirectories if needed
			localRelDir := filepath.Dir(relPath)
			if localRelDir != "." {
				if err := os.MkdirAll(filepath.Join(folderDir, localRelDir), 0755); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "创建子目录失败"})
					return
				}
			}

			localPath := filepath.Join(folderDir, relPath)

			f, err := fh.Open()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("读取文件失败: %s", relPath)})
				return
			}

			dst, err := os.Create(localPath)
			if err != nil {
				f.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("保存文件失败: %s", relPath)})
				return
			}

			// Read first file's content for AI review
			if i == 0 {
				limited := io.LimitReader(f, 2000)
				preview, _ := io.ReadAll(limited)
				contentPreview = string(preview)
				dst.Write(preview)
				io.Copy(dst, f)
			} else {
				io.Copy(dst, f)
			}

			f.Close()
			dst.Close()

			totalFileSize += fh.Size
			filesToSync = append(filesToSync, fileEntry{localPath: localPath, repoPath: relPath})

			if i == 0 {
				primaryFileName = relPath
				primaryFilePath = localPath
			}
		}
	} else {
		// Single file upload
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
			return
		}
		defer file.Close()

		ext := filepath.Ext(header.Filename)
		safeFileName := strings.ReplaceAll(strings.ToLower(name), " ", "_") + "_" + strconv.FormatInt(header.Size, 10) + ext
		filePath := filepath.Join(h.cfg.UploadDir, safeFileName)

		dst, err := os.Create(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
			return
		}
		defer dst.Close()

		var contentBuf strings.Builder
		tee := io.TeeReader(file, dst)
		limited := io.LimitReader(tee, 2000)
		contentBytes, _ := io.ReadAll(limited)
		contentBuf.Write(contentBytes)
		io.Copy(dst, file)

		contentPreview = contentBuf.String()
		primaryFileName = header.Filename
		primaryFilePath = filePath
		totalFileSize = header.Size
		filesToSync = append(filesToSync, fileEntry{localPath: filePath, repoPath: header.Filename})
	}

	// Handle thumbnail
	var thumbnailURL string
	thumbnailFile, thumbnailHeader, thumbnailErr := c.Request.FormFile("thumbnail")
	if thumbnailErr == nil && thumbnailFile != nil {
		defer thumbnailFile.Close()
		thumbExt := filepath.Ext(thumbnailHeader.Filename)
		thumbFileName := strings.ReplaceAll(strings.ToLower(name), " ", "_") + "_thumb" + thumbExt
		thumbPath := filepath.Join(h.cfg.ThumbnailDir, thumbFileName)

		if err := os.MkdirAll(h.cfg.ThumbnailDir, 0755); err == nil {
			thumbDst, err := os.Create(thumbPath)
			if err == nil {
				io.Copy(thumbDst, thumbnailFile)
				thumbDst.Close()
				thumbnailURL = thumbFileName
			}
		}
	} else {
		thumbFile, err := service.GenerateThumbnail(name, h.cfg.ThumbnailDir)
		if err == nil {
			thumbnailURL = thumbFile
		}
	}

	// AI Review
	approved, feedback, aiErr := h.aiSvc.ReviewSkill(name, description, contentPreview)
	if aiErr != nil {
		feedback = fmt.Sprintf("AI 审核出错: %v，自动通过。", aiErr)
		approved = true
	}

	// Create skill record — only "skill" type syncs to GitHub
	initialSyncStatus := service.GitHubSyncStatusDisabled
	if h.cfg.GitHubSyncEnabled && resourceType == "skill" {
		initialSyncStatus = service.GitHubSyncStatusPending
	}

	skill := &model.Skill{
		Name:             name,
		Description:      description,
		Category:         category,
		ResourceType:     resourceType,
		Author:           author,
		FileName:         primaryFileName,
		FilePath:         primaryFilePath,
		FileSize:         totalFileSize,
		ThumbnailURL:     thumbnailURL,
		AIApproved:       approved,
		AIFeedback:       feedback,
		GitHubSyncStatus: initialSyncStatus,
	}

	if err := h.skillSvc.CreateSkill(skill); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存记录失败"})
		return
	}

	// Sync to GitHub (only for "skill" resource type)
	if h.githubSyncSvc != nil && resourceType == "skill" {
		var syncResult service.GitHubSyncResult

		if len(filesToSync) == 1 {
			syncResult = h.githubSyncSvc.SyncUploadedSkill(
				c.Request.Context(),
				name,
				resourceType,
				filesToSync[0].repoPath,
				filesToSync[0].localPath,
			)
		} else {
			entries := make([]service.SyncFileEntry, len(filesToSync))
			for i, fe := range filesToSync {
				entries[i] = service.SyncFileEntry{
					LocalPath:    fe.localPath,
					RelativePath: fe.repoPath,
				}
			}
			syncResult = h.githubSyncSvc.SyncUploadedFolder(
				c.Request.Context(),
				name,
				resourceType,
				entries,
			)
		}

		skill.GitHubSyncStatus = syncResult.Status
		skill.GitHubPath = syncResult.Path
		skill.GitHubURL = syncResult.URL
		skill.GitHubSyncError = syncResult.Error

		if err := h.skillSvc.UpdateSkill(skill); err != nil {
			log.Printf("update github sync result failed for skill %d: %v", skill.ID, err)
		}
	}

	if skill.ThumbnailURL != "" && !strings.HasPrefix(skill.ThumbnailURL, "http") {
		skill.ThumbnailURL = "/api/thumbnails/" + skill.ThumbnailURL
	}

	c.JSON(http.StatusCreated, gin.H{
		"skill":    skill,
		"approved": approved,
		"feedback": feedback,
	})
}

// DeleteSkill handles DELETE /api/skills/:id
func (h *SkillHandler) DeleteSkill(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	skill, err := h.skillSvc.GetSkill(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "资源不存在"})
		return
	}

	// Delete from GitHub if synced
	var githubError string
	if h.githubSyncSvc != nil && skill.GitHubPath != "" && skill.GitHubSyncStatus == "success" {
		if err := h.githubSyncSvc.DeleteSkillFromGitHub(c.Request.Context(), skill.GitHubPath); err != nil {
			githubError = err.Error()
			log.Printf("delete skill %d from github failed: %v", skill.ID, err)
		}
	}

	// Delete local file
	if skill.FilePath != "" {
		os.Remove(skill.FilePath)
		// If it's a folder upload, try removing the parent directory
		dir := filepath.Dir(skill.FilePath)
		if dir != h.cfg.UploadDir {
			os.RemoveAll(dir)
		}
	}

	// Delete DB record
	if err := h.skillSvc.DeleteSkill(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除记录失败"})
		return
	}

	resp := gin.H{"message": "删除成功"}
	if githubError != "" {
		resp["github_error"] = githubError
	}
	c.JSON(http.StatusOK, resp)
}

// DownloadSkill handles GET /api/skills/:id/download
func (h *SkillHandler) DownloadSkill(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	skill, err := h.skillSvc.GetSkill(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "资源不存在"})
		return
	}

	h.skillSvc.IncrementDownload(uint(id))

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", skill.FileName))
	c.File(skill.FilePath)
}

// GetTrending handles GET /api/skills/trending
func (h *SkillHandler) GetTrending(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	resourceType := c.Query("resource_type")
	if limit < 1 || limit > 50 {
		limit = 10
	}

	skills, err := h.skillSvc.GetTrending(limit, resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range skills {
		if skills[i].ThumbnailURL != "" && !strings.HasPrefix(skills[i].ThumbnailURL, "http") {
			skills[i].ThumbnailURL = "/api/thumbnails/" + skills[i].ThumbnailURL
		}
	}

	c.JSON(http.StatusOK, skills)
}

// ServeThumbnail handles GET /api/thumbnails/:filename
func (h *SkillHandler) ServeThumbnail(c *gin.Context) {
	filename := c.Param("filename")
	filePath := filepath.Join(h.cfg.ThumbnailDir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "缩略图不存在"})
		return
	}

	c.File(filePath)
}

// ChatRecommend handles POST /api/ai/chat (SSE streaming)
func (h *SkillHandler) ChatRecommend(c *gin.Context) {
	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入消息"})
		return
	}

	skills, err := h.skillSvc.GetAllApprovedBrief()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取资源列表失败"})
		return
	}

	skillsJSON, _ := json.Marshal(skills)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	c.Stream(func(w io.Writer) bool {
		err := h.aiSvc.ChatRecommendStream(req.Message, string(skillsJSON), func(chunk string) error {
			c.SSEvent("message", chunk)
			c.Writer.Flush()
			return nil
		})

		if err != nil {
			c.SSEvent("error", err.Error())
			c.Writer.Flush()
		}

		c.SSEvent("done", "[DONE]")
		c.Writer.Flush()
		return false
	})
}
