package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

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

// GetSkillInstallConfig handles GET /api/skills/install-config.
func (h *SkillHandler) GetSkillInstallConfig(c *gin.Context) {
	repoURL := resolveInstallRepoURL(h.cfg.GitHubOwner, h.cfg.GitHubRepo)
	baseDir := sanitizeInstallBaseDir(h.cfg.GitHubBaseDir)
	c.JSON(http.StatusOK, gin.H{
		"github_repo":     repoURL,
		"github_base_dir": baseDir,
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

	skill, err := h.getSkillResource(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "资源不存在"})
		return
	}

	skill.ThumbnailURL = normalizeThumbnailURL(skill.ThumbnailURL)
	userID := optionalCurrentUserID(c)
	if userID != 0 {
		liked, err := h.skillSvc.HasUserLiked(skill.ID, userID)
		if err == nil {
			skill.UserLiked = liked
		}
		favorited, err := h.skillSvc.HasUserFavorited(skill.ID, userID)
		if err == nil {
			skill.Favorited = favorited
		}
	}
	c.JSON(http.StatusOK, skill)
}

// DownloadSkill handles GET /api/skills/:id/download
func (h *SkillHandler) DownloadSkill(c *gin.Context) {
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

	h.skillSvc.IncrementDownload(uint(id))

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", skill.FileName))
	c.File(skill.FilePath)
}

// GetSkillReadme handles GET /api/skills/:id/readme
func (h *SkillHandler) GetSkillReadme(c *gin.Context) {
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

	// Try to locate README.md or SKILL.md
	var readmePath string
	sessionRoot := uploadSessionRoot(h.cfg.UploadDir, skill.FilePath)

	if sessionRoot != "" {
		readmePath = findReadmePathInSession(sessionRoot)
	}

	if readmePath == "" {
		// Fallback to primary file if it's a markdown file
		if strings.HasSuffix(strings.ToLower(skill.FileName), ".md") {
			readmePath = skill.FilePath
		}
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
		skills[i].ThumbnailURL = normalizeThumbnailURL(skills[i].ThumbnailURL)
	}
	userID := optionalCurrentUserID(c)
	if userID != 0 {
		enrichUserEngagement(skills, userID, h.skillSvc)
	}

	c.JSON(http.StatusOK, skills)
}

// ServeThumbnail handles GET /api/thumbnails/:filename
func (h *SkillHandler) ServeThumbnail(c *gin.Context) {
	filePath, ok := resolveThumbnailPath(h.cfg.ThumbnailDir, c.Param("filename"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "缩略图不存在"})
		return
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "缩略图不存在"})
		return
	}

	c.File(filePath)
}

func resolveInstallRepoURL(owner, repo string) string {
	const fallbackRepo = "https://github.com/skillshub/community"

	rawRepo := strings.TrimSpace(repo)
	rawOwner := strings.TrimSpace(owner)
	if rawRepo == "" {
		if rawOwner == "" {
			return fallbackRepo
		}
		return fallbackRepo
	}

	if strings.HasPrefix(rawRepo, "http://") || strings.HasPrefix(rawRepo, "https://") {
		u, err := url.Parse(rawRepo)
		if err != nil {
			return fallbackRepo
		}
		parts := strings.Split(strings.Trim(strings.TrimSuffix(u.Path, ".git"), "/"), "/")
		if len(parts) >= 2 {
			return fmt.Sprintf("https://github.com/%s/%s", parts[0], parts[1])
		}
		return fallbackRepo
	}

	normalizedRepo := strings.Trim(strings.TrimPrefix(rawRepo, "github.com/"), "/")
	parts := strings.Split(normalizedRepo, "/")
	if len(parts) >= 2 {
		return fmt.Sprintf("https://github.com/%s/%s", parts[0], parts[1])
	}

	if rawOwner == "" {
		return fallbackRepo
	}
	return fmt.Sprintf("https://github.com/%s/%s", rawOwner, parts[0])
}

func sanitizeInstallBaseDir(baseDir string) string {
	raw := strings.ToLower(strings.TrimSpace(baseDir))
	if raw == "" {
		return "skills"
	}
	var b strings.Builder
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	value := strings.Trim(b.String(), "-_")
	if value == "" {
		return "skills"
	}
	return value
}
