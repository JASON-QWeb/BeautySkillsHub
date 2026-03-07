package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
// (mcp, tools, rules).
type ResourceHandler struct {
	skillSvc     *service.SkillService
	cfg          *config.Config
	resourceType string // "mcp", "tools", or "rules"
}

var toolsArchiveSuffixes = []string{
	".zip", ".tar", ".tar.gz", ".tgz", ".rar", ".7z", ".xz", ".bz2", ".gz",
}

func isToolsArchiveFilename(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return false
	}
	for _, suffix := range toolsArchiveSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func validateSourceURL(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported source url scheme")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return fmt.Errorf("missing source url host")
	}
	return nil
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
		readmePath = findReadmePathInSession(sessionRoot)
	}
	if readmePath == "" && strings.HasSuffix(strings.ToLower(skill.FileName), ".md") {
		readmePath = skill.FilePath
	}
	if readmePath == "" {
		c.String(http.StatusOK, skill.Description)
		return
	}

	content, err := os.ReadFile(readmePath)
	if err != nil {
		c.String(http.StatusOK, skill.Description)
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
	if strings.TrimSpace(skill.FilePath) == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "download file not found"})
		return
	}
	if _, statErr := os.Stat(skill.FilePath); statErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "download file not found"})
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
	uploadMode := strings.ToLower(strings.TrimSpace(c.DefaultPostForm("upload_mode", "file")))
	sourceURL := strings.TrimSpace(c.PostForm("source_url"))

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if err := validateSourceURL(sourceURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source_url"})
		return
	}
	if uploadMode == "" {
		uploadMode = "file"
	}
	switch h.resourceType {
	case "mcp":
		if uploadMode != "metadata" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mcp only supports metadata upload mode"})
			return
		}
	case "tools":
		if uploadMode != "metadata" && uploadMode != "file" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tools upload_mode must be metadata or file"})
			return
		}
	default:
		if uploadMode == "metadata" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "metadata upload mode is not supported"})
			return
		}
	}
	if uploadMode == "folder" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "folder upload is not supported for this resource type"})
		return
	}

	var uploadRoot string
	keepUploadRoot := false
	defer func() {
		if keepUploadRoot || uploadRoot == "" {
			return
		}
		_ = os.RemoveAll(uploadRoot)
	}()

	if uploadMode == "file" {
		typeDir := filepath.Join(h.cfg.UploadDir, h.resourceType)
		if err := os.MkdirAll(typeDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create upload dir failed"})
			return
		}
		var err error
		uploadRoot, err = os.MkdirTemp(typeDir, "upload-*")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create temp dir failed"})
			return
		}
	}

	var primaryFileName string
	var primaryFilePath string
	var totalFileSize int64

	if uploadMode == "file" {
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
		if h.resourceType == "tools" && !isToolsArchiveFilename(header.Filename) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tools attachment must be an archive file"})
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
		thumbFile, err := service.GenerateThumbnail(name, thumbnailSubtitle(description, name), h.cfg.ThumbnailDir)
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
		GitHubURL:           sourceURL,
		GitHubSyncStatus:    "disabled",
	}

	if err := h.skillSvc.CreateSkill(resource); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save record failed"})
		return
	}
	keepUploadRoot = true

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

	if c.ContentType() == "multipart/form-data" {
		h.updateFromMultipart(c, skill)
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

func (h *ResourceHandler) updateFromMultipart(c *gin.Context, skill *model.Skill) {
	nameRaw, hasName := c.GetPostForm("name")
	descriptionRaw, hasDescription := c.GetPostForm("description")
	tagsRaw, hasTags := c.GetPostForm("tags")
	sourceRaw, hasSourceURL := c.GetPostForm("source_url")
	uploadMode := strings.ToLower(strings.TrimSpace(c.DefaultPostForm("upload_mode", "")))
	if uploadMode == "" {
		uploadMode = "metadata"
	}
	if h.resourceType == "mcp" && uploadMode != "metadata" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mcp only supports metadata upload mode"})
		return
	}
	if h.resourceType == "tools" && uploadMode != "metadata" && uploadMode != "file" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tools upload_mode must be metadata or file"})
		return
	}

	changed := false
	oldFilePath := skill.FilePath
	oldThumbnail := skill.ThumbnailURL

	if hasName {
		name := strings.TrimSpace(nameRaw)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name cannot be empty"})
			return
		}
		if skill.Name != name {
			changed = true
		}
		skill.Name = name
	}
	if hasDescription {
		description := strings.TrimSpace(descriptionRaw)
		if skill.Description != description {
			changed = true
		}
		skill.Description = description
	}
	if hasTags {
		tags := normalizeTags(tagsRaw)
		if skill.Tags != tags {
			changed = true
		}
		skill.Tags = tags
	}
	if hasSourceURL {
		sourceURL := strings.TrimSpace(sourceRaw)
		if err := validateSourceURL(sourceURL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source_url"})
			return
		}
		if skill.GitHubURL != sourceURL {
			changed = true
		}
		skill.GitHubURL = sourceURL
	}

	var newUploadRoot string
	keepNewUploadRoot := false
	defer func() {
		if keepNewUploadRoot || newUploadRoot == "" {
			return
		}
		_ = os.RemoveAll(newUploadRoot)
	}()

	uploadedFile := false
	file, header, fileErr := c.Request.FormFile("file")
	if fileErr == nil && file != nil {
		defer file.Close()
		uploadedFile = true
		if header.Size > maxUploadSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file size exceeds 50MB"})
			return
		}
		if h.resourceType == "tools" && !isToolsArchiveFilename(header.Filename) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tools attachment must be an archive file"})
			return
		}

		typeDir := filepath.Join(h.cfg.UploadDir, h.resourceType)
		if err := os.MkdirAll(typeDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create upload dir failed"})
			return
		}
		tempRoot, err := os.MkdirTemp(typeDir, "upload-*")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create temp dir failed"})
			return
		}
		newUploadRoot = tempRoot

		safeFileName := sanitizeLocalFilename(header.Filename)
		nextFilePath := filepath.Join(newUploadRoot, safeFileName)
		dst, err := os.Create(nextFilePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "save file failed"})
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "save file failed"})
			return
		}

		fileName := filepath.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
		if fileName == "" || fileName == "." || fileName == "/" {
			fileName = safeFileName
		}

		skill.FileName = fileName
		skill.FilePath = nextFilePath
		skill.FileSize = header.Size
		changed = true
	} else if fileErr != nil && !errors.Is(fileErr, http.ErrMissingFile) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file"})
		return
	}

	if h.resourceType == "tools" && uploadMode == "file" && !uploadedFile {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required for upload_mode=file"})
		return
	}

	var newThumbnailFile string
	keepNewThumbnail := false
	defer func() {
		if keepNewThumbnail || newThumbnailFile == "" {
			return
		}
		if thumbPath, ok := resolveThumbnailPath(h.cfg.ThumbnailDir, newThumbnailFile); ok {
			_ = os.Remove(thumbPath)
		}
	}()

	thumbnailFile, thumbnailHeader, thumbnailErr := c.Request.FormFile("thumbnail")
	if thumbnailErr == nil && thumbnailFile != nil {
		defer thumbnailFile.Close()
		thumbFileName, err := saveUploadedThumbnail(skill.Name, thumbnailFile, thumbnailHeader, h.cfg.ThumbnailDir)
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
		newThumbnailFile = thumbFileName
		skill.ThumbnailURL = thumbFileName
		changed = true
	} else if thumbnailErr != nil && !errors.Is(thumbnailErr, http.ErrMissingFile) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid thumbnail"})
		return
	}

	if !changed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one field required"})
		return
	}

	if err := h.skillSvc.UpdateSkill(skill); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}

	keepNewUploadRoot = true
	keepNewThumbnail = true

	if uploadedFile && oldFilePath != "" && oldFilePath != skill.FilePath {
		removeResourceUploadAsset(h.cfg.UploadDir, oldFilePath)
	}
	if newThumbnailFile != "" && oldThumbnail != "" && oldThumbnail != newThumbnailFile {
		removeStoredThumbnail(h.cfg.ThumbnailDir, oldThumbnail)
	}

	skill.ThumbnailURL = normalizeThumbnailURL(skill.ThumbnailURL)
	c.JSON(http.StatusOK, gin.H{"skill": skill})
}

func removeResourceUploadAsset(uploadDir, filePath string) {
	if strings.TrimSpace(filePath) == "" {
		return
	}
	if sessionRoot := uploadSessionRoot(uploadDir, filePath); sessionRoot != "" &&
		isPathInsideBase(uploadDir, sessionRoot) &&
		filepath.Clean(sessionRoot) != filepath.Clean(uploadDir) {
		_ = os.RemoveAll(sessionRoot)
		return
	}

	_ = os.Remove(filePath)
	dir := filepath.Dir(filePath)
	if isPathInsideBase(uploadDir, dir) && filepath.Clean(dir) != filepath.Clean(uploadDir) {
		_ = os.RemoveAll(dir)
	}
}

func removeStoredThumbnail(thumbnailDir, storedThumbnail string) {
	cleaned := strings.TrimSpace(storedThumbnail)
	if cleaned == "" || strings.HasPrefix(cleaned, "http://") || strings.HasPrefix(cleaned, "https://") {
		return
	}
	cleaned = strings.TrimPrefix(cleaned, "/api/thumbnails/")
	if thumbPath, ok := resolveThumbnailPath(thumbnailDir, cleaned); ok {
		_ = os.Remove(thumbPath)
	}
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

func (h *ResourceHandler) Unlike(c *gin.Context) {
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

	liked, likesCount, err := h.skillSvc.UnlikeSkill(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unlike failed"})
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
	protected.DELETE("/:id/like", h.Unlike)
	protected.POST("/:id/favorite", h.AddFavorite)
	protected.DELETE("/:id/favorite", h.RemoveFavorite)
}
