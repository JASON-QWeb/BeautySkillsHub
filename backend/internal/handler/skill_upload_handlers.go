package handler

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"skill-hub/internal/model"
	"skill-hub/internal/service"

	"github.com/gin-gonic/gin"
)

const maxUploadSize = 50 << 20 // 50 MB

const maxThumbnailSize = 5 << 20 // 5 MB

var (
	allowedThumbnailExtensions = map[string]struct{}{
		".png":  {},
		".jpg":  {},
		".jpeg": {},
		".webp": {},
		".gif":  {},
	}
	errThumbnailTooLarge       = errors.New("thumbnail file too large")
	errThumbnailExtensionLimit = errors.New("thumbnail extension not allowed")
)

// UploadSkill handles POST /api/skills
func (h *SkillHandler) UploadSkill(c *gin.Context) {
	userID, username, ok := currentUserIdentity(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或登录状态无效"})
		return
	}

	name := c.PostForm("name")
	description := c.PostForm("description")
	tags := normalizeTags(c.PostForm("tags"))
	author := normalizeSkillAuthor(username, c.PostForm("author"))
	uploadMode := c.DefaultPostForm("upload_mode", "file")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "名称不能为空"})
		return
	}

	// Skills always use resource_type=skill; other types use their own routes
	resourceType := "skill"

	// Ensure upload dir exists (skills go into skills/ subdirectory)
	skillsDir := filepath.Join(h.cfg.UploadDir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建上传目录失败"})
		return
	}
	uploadRoot, err := os.MkdirTemp(skillsDir, "skill-upload-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建临时上传目录失败"})
		return
	}

	var primaryFileName string
	var primaryFilePath string
	var totalFileSize int64

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

		for i, fh := range files {
			if totalFileSize+fh.Size > maxUploadSize {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "上传总大小不能超过 50MB"})
				return
			}
			// Determine relative path within folder
			relPath := fh.Filename
			if i < len(filePaths) && filePaths[i] != "" {
				relPath = filePaths[i]
			}
			normalizedRel, err := service.NormalizeRepoRelativePath(relPath)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("非法文件路径 %q", relPath)})
				return
			}

			localPath := filepath.Join(uploadRoot, filepath.FromSlash(normalizedRel))
			if !isPathInsideBase(uploadRoot, localPath) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("非法文件路径 %q", relPath)})
				return
			}

			// Create subdirectories if needed
			localRelDir := filepath.Dir(localPath)
			if localRelDir != "." {
				if err := os.MkdirAll(localRelDir, 0755); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "创建子目录失败"})
					return
				}
			}

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

			if _, err := io.Copy(dst, f); err != nil {
				f.Close()
				dst.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("保存文件失败: %s", relPath)})
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
		// Single file upload
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
			return
		}
		defer file.Close()

		if header.Size > maxUploadSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "上传文件大小不能超过 50MB"})
			return
		}

		safeFileName := sanitizeLocalFilename(header.Filename)
		filePath := filepath.Join(uploadRoot, safeFileName)

		dst, err := os.Create(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
			return
		}
		primaryFileName = filepath.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
		if primaryFileName == "" || primaryFileName == "." || primaryFileName == "/" {
			primaryFileName = safeFileName
		}
		primaryFilePath = filePath
		totalFileSize = header.Size
	}

	// Handle thumbnail (uses AI func_summary as subtitle)
	var thumbnailURL string
	thumbnailFile, thumbnailHeader, thumbnailErr := c.Request.FormFile("thumbnail")
	if thumbnailErr == nil && thumbnailFile != nil {
		defer thumbnailFile.Close()
		thumbFileName, err := saveUploadedThumbnail(name, thumbnailFile, thumbnailHeader, h.cfg.ThumbnailDir)
		if err != nil {
			switch {
			case errors.Is(err, errThumbnailTooLarge):
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "缩略图大小不能超过 5MB"})
			case errors.Is(err, errThumbnailExtensionLimit):
				c.JSON(http.StatusBadRequest, gin.H{"error": "缩略图仅支持 PNG/JPG/JPEG/WEBP/GIF"})
			default:
				c.JSON(http.StatusBadRequest, gin.H{"error": "上传缩略图失败"})
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

	skill := &model.Skill{
		UserID:              userID,
		Name:                name,
		Description:         description,
		Tags:                tags,
		ResourceType:        resourceType,
		Author:              author,
		FileName:            primaryFileName,
		FilePath:            primaryFilePath,
		FileSize:            totalFileSize,
		ThumbnailURL:        thumbnailURL,
		AIApproved:          false,
		AIReviewStatus:      model.AIReviewStatusQueued,
		AIReviewPhase:       model.AIReviewPhaseQueued,
		AIReviewAttempts:    0,
		AIReviewMaxAttempts: 3,
		AIFeedback:          "审核排队中，请稍候...",
		AIDescription:       "",
		HumanReviewStatus:   model.HumanReviewStatusPending,
		Published:           false,
		GitHubPath:          "",
		GitHubURL:           "",
		GitHubSyncStatus:    service.GitHubSyncStatusDisabled,
		GitHubSyncError:     "",
		GitHubFiles:         "",
	}
	if h.cfg.GitHubSyncEnabled && resourceType == "skill" {
		skill.GitHubSyncStatus = service.GitHubSyncStatusNotStarted
	}

	if err := h.skillSvc.CreateSkill(skill); err != nil {
		_ = os.RemoveAll(uploadRoot)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存记录失败"})
		return
	}

	h.dispatchAIReview(skill.ID)

	skill.ThumbnailURL = normalizeThumbnailURL(skill.ThumbnailURL)

	c.JSON(http.StatusCreated, gin.H{
		"skill":    skill,
		"approved": false,
		"feedback": "审核排队中，请稍候...",
	})
}

func normalizeTags(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	})

	result := make([]string, 0, 5)
	seen := make(map[string]struct{}, 5)
	for _, p := range parts {
		tag := strings.TrimSpace(p)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, tag)
		if len(result) >= 5 {
			break
		}
	}

	return strings.Join(result, ",")
}

func thumbnailSubtitle(description, fallback string) string {
	if trimmed := strings.TrimSpace(description); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(fallback)
}

func sanitizeLocalFilename(name string) string {
	base := filepath.Base(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	if base == "" || base == "." || base == "/" {
		return "file.bin"
	}

	var b strings.Builder
	lastDash := false
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if unicode.IsSpace(r) {
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}

	safe := strings.Trim(b.String(), "-")
	if safe == "" || safe == "." || safe == ".." {
		return "file.bin"
	}
	return safe
}

func isPathInsideBase(base, target string) bool {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func uploadSessionRoot(base, filePath string) string {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return ""
	}
	targetAbs, err := filepath.Abs(filePath)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return ""
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}

	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) == 0 || parts[0] == "" || parts[0] == "." || parts[0] == ".." {
		return ""
	}
	if len(parts) == 1 {
		return filepath.Join(baseAbs, parts[0])
	}
	if isUploadSessionDirName(parts[0]) {
		return filepath.Join(baseAbs, parts[0])
	}
	if isUploadSessionDirName(parts[1]) {
		return filepath.Join(baseAbs, parts[0], parts[1])
	}
	return filepath.Join(baseAbs, parts[0])
}

func isUploadSessionDirName(segment string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(segment)), "upload-")
}

func validateThumbnailHeader(header *multipart.FileHeader) (string, error) {
	if header == nil {
		return "", errThumbnailExtensionLimit
	}
	if header.Size > maxThumbnailSize {
		return "", errThumbnailTooLarge
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if _, ok := allowedThumbnailExtensions[ext]; !ok {
		return "", errThumbnailExtensionLimit
	}
	return ext, nil
}

func saveUploadedThumbnail(name string, file multipart.File, header *multipart.FileHeader, thumbnailDir string) (string, error) {
	ext, err := validateThumbnailHeader(header)
	if err != nil {
		return "", err
	}

	safeName := sanitizeLocalFilename(strings.ToLower(name))
	base := strings.TrimSuffix(safeName, filepath.Ext(safeName))
	if base == "" || base == "." || base == ".." {
		base = "thumbnail"
	}

	thumbFileName := base + "_thumb" + ext
	thumbPath := filepath.Join(thumbnailDir, thumbFileName)

	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return "", err
	}
	thumbDst, err := os.Create(thumbPath)
	if err != nil {
		return "", err
	}
	defer thumbDst.Close()

	if _, err := io.Copy(thumbDst, file); err != nil {
		return "", err
	}

	return thumbFileName, nil
}
