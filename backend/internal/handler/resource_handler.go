package handler

import (
	"errors"
	"fmt"
	"io"
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

// ResourceHandler handles CRUD and engagement for non-skill resource types
// (mcp, tools, rules). These resources skip AI review and GitHub sync,
// and are published immediately on upload.
type ResourceHandler struct {
	skillSvc     *service.SkillService
	cfg          *config.Config
	resourceType string // "mcp", "tools", or "rules"
}

func NewResourceHandler(skillSvc *service.SkillService, resourceType string, cfg *config.Config) *ResourceHandler {
	return &ResourceHandler{
		skillSvc:     skillSvc,
		cfg:          cfg,
		resourceType: resourceType,
	}
}

// --------------- Read endpoints ---------------

func (h *ResourceHandler) List(c *gin.Context) {
	search := c.Query("search")
	category := c.Query("category")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	skills, total, err := h.skillSvc.ListSkills(search, category, h.resourceType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range skills {
		skills[i].ThumbnailURL = normalizeThumbnailURL(skills[i].ThumbnailURL)
	}
	userID := optionalCurrentUserID(c)
	if userID != 0 {
		enrichUserEngagement(skills, userID, h.skillSvc)
	}

	c.JSON(http.StatusOK, gin.H{
		"skills":    skills,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *ResourceHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	skill, err := h.skillSvc.GetSkill(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}
	if skill.ResourceType != h.resourceType {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}

	skill.ThumbnailURL = normalizeThumbnailURL(skill.ThumbnailURL)
	userID := optionalCurrentUserID(c)
	if userID != 0 {
		if liked, err := h.skillSvc.HasUserLiked(skill.ID, userID); err == nil {
			skill.UserLiked = liked
		}
		if fav, err := h.skillSvc.HasUserFavorited(skill.ID, userID); err == nil {
			skill.Favorited = fav
		}
	}
	c.JSON(http.StatusOK, skill)
}

func (h *ResourceHandler) GetReadme(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	skill, err := h.skillSvc.GetSkill(uint(id))
	if err != nil || skill.ResourceType != h.resourceType {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}

	var readmePath string
	sessionRoot := uploadSessionRoot(h.cfg.UploadDir, skill.FilePath)
	if sessionRoot != "" {
		if info, err := os.Stat(sessionRoot); err == nil && info.IsDir() {
			candidates := []string{"README.md", "readme.md", "SKILL.md", "skill.md"}
			for _, candidate := range candidates {
				p := filepath.Join(sessionRoot, candidate)
				if _, err := os.Stat(p); err == nil {
					readmePath = p
					break
				}
			}
		}
	}
	if readmePath == "" && strings.HasSuffix(strings.ToLower(skill.FileName), ".md") {
		readmePath = skill.FilePath
	}
	if readmePath == "" {
		c.String(http.StatusOK, "")
		return
	}

	content, err := os.ReadFile(readmePath)
	if err != nil {
		c.String(http.StatusOK, "")
		return
	}
	c.String(http.StatusOK, string(content))
}

func (h *ResourceHandler) Download(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	skill, err := h.skillSvc.GetSkill(uint(id))
	if err != nil || skill.ResourceType != h.resourceType {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}

	h.skillSvc.IncrementDownload(uint(id))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", skill.FileName))
	c.File(skill.FilePath)
}

func (h *ResourceHandler) TrackDownloadHit(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	if skill, err := h.skillSvc.GetSkill(uint(id)); err != nil || skill.ResourceType != h.resourceType {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}

	if err := h.skillSvc.IncrementDownload(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "download count failed"})
		return
	}

	updated, err := h.skillSvc.GetSkill(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read resource failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"downloads": updated.Downloads})
}

func (h *ResourceHandler) GetCategories(c *gin.Context) {
	categories, err := h.skillSvc.GetCategories(h.resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, categories)
}

func (h *ResourceHandler) GetSummary(c *gin.Context) {
	total, yesterdayNew, err := h.skillSvc.GetResourceSummary(h.resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "summary failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"total":         total,
		"yesterday_new": yesterdayNew,
	})
}

func (h *ResourceHandler) GetTrending(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 50 {
		limit = 10
	}

	skills, err := h.skillSvc.GetTrending(limit, h.resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range skills {
		skills[i].ThumbnailURL = normalizeThumbnailURL(skills[i].ThumbnailURL)
	}
	userID := optionalCurrentUserID(c)
	if userID != 0 {
		enrichUserEngagement(skills, userID, h.skillSvc)
	}
	c.JSON(http.StatusOK, skills)
}

// --------------- Upload (no AI review, auto-published) ---------------

func (h *ResourceHandler) Upload(c *gin.Context) {
	userID, username, ok := currentUserIdentity(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	name := c.PostForm("name")
	description := c.PostForm("description")
	tags := normalizeTags(c.PostForm("tags"))
	author := normalizeSkillAuthor(username, c.PostForm("author"))
	uploadMode := c.DefaultPostForm("upload_mode", "file")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	// Upload to type-specific subdirectory
	typeDir := filepath.Join(h.cfg.UploadDir, h.resourceType)
	if err := os.MkdirAll(typeDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create upload dir failed"})
		return
	}
	uploadRoot, err := os.MkdirTemp(typeDir, "upload-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create temp dir failed"})
		return
	}

	var primaryFileName string
	var primaryFilePath string
	var totalFileSize int64

	if uploadMode == "folder" {
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parse form failed"})
			return
		}
		files := form.File["files"]
		filePaths := form.Value["file_paths"]

		if len(files) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no files selected"})
			return
		}

		for i, fh := range files {
			if totalFileSize+fh.Size > maxUploadSize {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "total size exceeds 50MB"})
				return
			}
			relPath := fh.Filename
			if i < len(filePaths) && filePaths[i] != "" {
				relPath = filePaths[i]
			}
			normalizedRel, err := service.NormalizeRepoRelativePath(relPath)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid path %q", relPath)})
				return
			}

			localPath := filepath.Join(uploadRoot, filepath.FromSlash(normalizedRel))
			if !isPathInsideBase(uploadRoot, localPath) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid path %q", relPath)})
				return
			}

			localRelDir := filepath.Dir(localPath)
			if localRelDir != "." {
				if err := os.MkdirAll(localRelDir, 0755); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "create subdir failed"})
					return
				}
			}

			f, err := fh.Open()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("read file failed: %s", relPath)})
				return
			}

			dst, err := os.Create(localPath)
			if err != nil {
				f.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("save file failed: %s", relPath)})
				return
			}

			if _, err := io.Copy(dst, f); err != nil {
				f.Close()
				dst.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("save file failed: %s", relPath)})
				return
			}

			f.Close()
			dst.Close()
			totalFileSize += fh.Size
			if i == 0 {
				primaryFileName = filepath.Base(normalizedRel)
				primaryFilePath = localPath
			}
		}
	} else {
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
			return
		}
		defer file.Close()

		if header.Size > maxUploadSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file size exceeds 50MB"})
			return
		}

		safeFileName := sanitizeLocalFilename(header.Filename)
		filePath := filepath.Join(uploadRoot, safeFileName)

		dst, err := os.Create(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "save file failed"})
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "save file failed"})
			return
		}
		primaryFileName = filepath.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
		if primaryFileName == "" || primaryFileName == "." || primaryFileName == "/" {
			primaryFileName = safeFileName
		}
		primaryFilePath = filePath
		totalFileSize = header.Size
	}

	// Handle thumbnail
	var thumbnailURL string
	thumbnailFile, thumbnailHeader, thumbnailErr := c.Request.FormFile("thumbnail")
	if thumbnailErr == nil && thumbnailFile != nil {
		defer thumbnailFile.Close()
		thumbFileName, err := saveUploadedThumbnail(name, thumbnailFile, thumbnailHeader, h.cfg.ThumbnailDir)
		if err != nil {
			switch {
			case errors.Is(err, errThumbnailTooLarge):
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "thumbnail size exceeds 5MB"})
			case errors.Is(err, errThumbnailExtensionLimit):
				c.JSON(http.StatusBadRequest, gin.H{"error": "thumbnail must be png/jpg/jpeg/webp/gif"})
			default:
				c.JSON(http.StatusBadRequest, gin.H{"error": "save thumbnail failed"})
			}
			return
		}
		thumbnailURL = thumbFileName
	} else {
		thumbFile, err := service.GenerateThumbnail(name, name, h.cfg.ThumbnailDir)
		if err == nil {
			thumbnailURL = thumbFile
		}
	}

	// Create resource: auto-approved and published (no AI review needed)
	resource := &model.Skill{
		UserID:              userID,
		Name:                name,
		Description:         description,
		Tags:                tags,
		ResourceType:        h.resourceType,
		Author:              author,
		FileName:            primaryFileName,
		FilePath:            primaryFilePath,
		FileSize:            totalFileSize,
		ThumbnailURL:        thumbnailURL,
		AIApproved:          true,
		AIReviewStatus:      model.AIReviewStatusPassed,
		AIReviewPhase:       model.AIReviewPhaseDone,
		AIReviewAttempts:    0,
		AIReviewMaxAttempts: 0,
		AIFeedback:          "",
		AIDescription:       "",
		HumanReviewStatus:   model.HumanReviewStatusApproved,
		Published:           true,
		GitHubSyncStatus:    "disabled",
	}

	if err := h.skillSvc.CreateSkill(resource); err != nil {
		_ = os.RemoveAll(uploadRoot)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save record failed"})
		return
	}

	resource.ThumbnailURL = normalizeThumbnailURL(resource.ThumbnailURL)

	c.JSON(http.StatusCreated, gin.H{
		"skill":    resource,
		"approved": true,
		"feedback": "",
	})
}

// --------------- Update ---------------

func (h *ResourceHandler) Update(c *gin.Context) {
	userID, username, ok := currentUserIdentity(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	skill, err := h.skillSvc.GetSkill(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}
	if skill.ResourceType != h.resourceType {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}
	if !canManageSkill(skill, userID, username) {
		c.JSON(http.StatusForbidden, gin.H{"error": "can only edit your own resources"})
		return
	}

	var req updateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Name == nil && req.Description == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one field required"})
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name cannot be empty"})
			return
		}
		skill.Name = name
	}
	if req.Description != nil {
		skill.Description = strings.TrimSpace(*req.Description)
	}

	if err := h.skillSvc.UpdateSkill(skill); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}

	skill.ThumbnailURL = normalizeThumbnailURL(skill.ThumbnailURL)
	c.JSON(http.StatusOK, gin.H{"skill": skill})
}

// --------------- Delete (no GitHub sync) ---------------

func (h *ResourceHandler) Delete(c *gin.Context) {
	userID, username, ok := currentUserIdentity(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	skill, err := h.skillSvc.GetSkill(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}
	if skill.ResourceType != h.resourceType {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}
	if !canManageSkill(skill, userID, username) {
		c.JSON(http.StatusForbidden, gin.H{"error": "can only delete your own resources"})
		return
	}

	// Delete local files
	if skill.FilePath != "" {
		if sessionRoot := uploadSessionRoot(h.cfg.UploadDir, skill.FilePath); sessionRoot != "" &&
			isPathInsideBase(h.cfg.UploadDir, sessionRoot) &&
			filepath.Clean(sessionRoot) != filepath.Clean(h.cfg.UploadDir) {
			_ = os.RemoveAll(sessionRoot)
		} else {
			_ = os.Remove(skill.FilePath)
			dir := filepath.Dir(skill.FilePath)
			if isPathInsideBase(h.cfg.UploadDir, dir) && filepath.Clean(dir) != filepath.Clean(h.cfg.UploadDir) {
				_ = os.RemoveAll(dir)
			}
		}
	}

	if err := h.skillSvc.DeleteSkill(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// --------------- Engagement ---------------

func (h *ResourceHandler) Like(c *gin.Context) {
	userID, _, ok := currentUserIdentity(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	if skill, err := h.skillSvc.GetSkill(uint(id)); err != nil || skill.ResourceType != h.resourceType {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}

	liked, likesCount, err := h.skillSvc.LikeSkill(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "like failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"liked": liked, "likes_count": likesCount})
}

func (h *ResourceHandler) AddFavorite(c *gin.Context) {
	userID, _, ok := currentUserIdentity(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	if skill, err := h.skillSvc.GetSkill(uint(id)); err != nil || skill.ResourceType != h.resourceType {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}

	if err := h.skillSvc.AddFavorite(uint(id), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "favorite failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"favorited": true})
}

func (h *ResourceHandler) RemoveFavorite(c *gin.Context) {
	userID, _, ok := currentUserIdentity(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	if skill, err := h.skillSvc.GetSkill(uint(id)); err != nil || skill.ResourceType != h.resourceType {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}

	if err := h.skillSvc.RemoveFavorite(uint(id), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unfavorite failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"favorited": false})
}

func (h *ResourceHandler) ServeThumbnail(c *gin.Context) {
	filePath, ok := resolveThumbnailPath(h.cfg.ThumbnailDir, c.Param("filename"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "thumbnail not found"})
		return
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "thumbnail not found"})
		return
	}
	c.File(filePath)
}

// registerResourceRoutes is a helper to register all routes for a resource type.
func RegisterResourceRoutes(
	api *gin.RouterGroup,
	h *ResourceHandler,
	authMiddleware gin.HandlerFunc,
	optionalAuthMiddleware gin.HandlerFunc,
	routePrefix string,
) {
	publicReads := api.Group(routePrefix)
	publicReads.Use(optionalAuthMiddleware)
	publicReads.GET("", h.List)
	publicReads.GET("/summary", h.GetSummary)
	publicReads.GET("/trending", h.GetTrending)
	publicReads.GET("/categories", h.GetCategories)
	publicReads.GET("/:id", h.Get)
	publicReads.GET("/:id/readme", h.GetReadme)
	publicReads.GET("/:id/download", h.Download)
	publicReads.POST("/:id/download-hit", h.TrackDownloadHit)

	protected := api.Group(routePrefix)
	protected.Use(authMiddleware)
	protected.POST("", h.Upload)
	protected.PUT("/:id", h.Update)
	protected.DELETE("/:id", h.Delete)
	protected.POST("/:id/like", h.Like)
	protected.POST("/:id/favorite", h.AddFavorite)
	protected.DELETE("/:id/favorite", h.RemoveFavorite)
}
