package handler

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"skill-hub/internal/model"
	"skill-hub/internal/service"

	"github.com/gin-gonic/gin"
)

type updateSkillRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// UpdateSkill handles PUT /api/skills/:id
func (h *SkillHandler) UpdateSkill(c *gin.Context) {
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
		c.JSON(http.StatusForbidden, gin.H{"error": "只能编辑自己发布的资源"})
		return
	}
	if skill.HasPendingRevision {
		c.JSON(http.StatusConflict, gin.H{"error": "当前已有待审核更新，请等待审核完成"})
		return
	}

	if c.ContentType() == "multipart/form-data" {
		h.updateReviewedResourceFromMultipart(c, skill)
		return
	}

	var req updateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
		return
	}

	if req.Name == nil && req.Description == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要更新一个字段"})
		return
	}

	revision := service.BuildRevisionFromSkill(skill)
	if revision == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建更新修订失败"})
		return
	}

	changed := false
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "名称不能为空"})
			return
		}
		if err := validateContentTextFields(contentTextFields{Name: name}); err != nil {
			var fieldErr *fieldLengthError
			if errors.As(err, &fieldErr) {
				c.JSON(http.StatusBadRequest, gin.H{"error": formatSkillFieldLengthError(fieldErr)})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "输入内容过长"})
			return
		}
		if revision.Name != name {
			changed = true
		}
		revision.Name = name
	}
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		if err := validateContentTextFields(contentTextFields{Description: description}); err != nil {
			var fieldErr *fieldLengthError
			if errors.As(err, &fieldErr) {
				c.JSON(http.StatusBadRequest, gin.H{"error": formatSkillFieldLengthError(fieldErr)})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "输入内容过长"})
			return
		}
		if revision.Description != description {
			changed = true
		}
		revision.Description = description
	}

	if !changed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有检测到实际更新内容"})
		return
	}

	preparePendingRevisionForResourceUpdate(revision, skill.ResourceType)
	created, err := h.skillSvc.CreatePendingRevision(skill, revision)
	if errors.Is(err, service.ErrActiveRevisionExists) {
		c.JSON(http.StatusConflict, gin.H{"error": "当前已有待审核更新，请等待审核完成"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建更新修订失败"})
		return
	}

	h.dispatchAIReviewRevision(created.ID)

	view := service.BuildSkillReviewView(skill, created)
	view.ThumbnailURL = normalizeThumbnailURL(view.ThumbnailURL)
	c.JSON(http.StatusOK, gin.H{
		"skill":    view,
		"approved": false,
		"feedback": created.AIFeedback,
	})
}

func (h *SkillHandler) updateReviewedResourceFromMultipart(c *gin.Context, skill *model.Skill) {
	revision := service.BuildRevisionFromSkill(skill)
	if revision == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建更新修订失败"})
		return
	}

	changed := false

	if nameRaw, hasName := c.GetPostForm("name"); hasName {
		name := strings.TrimSpace(nameRaw)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "名称不能为空"})
			return
		}
		if err := validateContentTextFields(contentTextFields{Name: name}); err != nil {
			var fieldErr *fieldLengthError
			if errors.As(err, &fieldErr) {
				c.JSON(http.StatusBadRequest, gin.H{"error": formatSkillFieldLengthError(fieldErr)})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "输入内容过长"})
			return
		}
		if revision.Name != name {
			changed = true
		}
		revision.Name = name
	}

	if descriptionRaw, hasDescription := c.GetPostForm("description"); hasDescription {
		description := strings.TrimSpace(descriptionRaw)
		if err := validateContentTextFields(contentTextFields{Description: description}); err != nil {
			var fieldErr *fieldLengthError
			if errors.As(err, &fieldErr) {
				c.JSON(http.StatusBadRequest, gin.H{"error": formatSkillFieldLengthError(fieldErr)})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "输入内容过长"})
			return
		}
		if revision.Description != description {
			changed = true
		}
		revision.Description = description
	}

	if tagsRaw, hasTags := c.GetPostForm("tags"); hasTags {
		tags := normalizeTags(tagsRaw)
		if err := validateContentTextFields(contentTextFields{Tags: tags}); err != nil {
			var fieldErr *fieldLengthError
			if errors.As(err, &fieldErr) {
				c.JSON(http.StatusBadRequest, gin.H{"error": formatSkillFieldLengthError(fieldErr)})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "输入内容过长"})
			return
		}
		if revision.Tags != tags {
			changed = true
		}
		revision.Tags = tags
	}

	resourceType := skill.ResourceType
	uploadMode := strings.ToLower(strings.TrimSpace(c.DefaultPostForm("upload_mode", "")))
	if uploadMode == "" {
		if resourceType == "rules" && strings.TrimSpace(c.PostForm("markdown_content")) != "" {
			uploadMode = "paste"
		} else {
			uploadMode = "file"
		}
	}

	if resourceType == "rules" && uploadMode == "folder" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rules 仅支持单文件上传或粘贴 Markdown"})
		return
	}

	uploadSubdir := "skills"
	if resourceType == "rules" {
		uploadSubdir = "rules"
	}
	uploadTypeDir := filepath.Join(h.cfg.UploadDir, uploadSubdir)
	if err := os.MkdirAll(uploadTypeDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建上传目录失败"})
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

	createUploadRoot := func() (string, error) {
		if uploadRoot != "" {
			return uploadRoot, nil
		}
		root, err := os.MkdirTemp(uploadTypeDir, "skill-update-*")
		if err != nil {
			return "", err
		}
		uploadRoot = root
		return uploadRoot, nil
	}

	if uploadMode == "paste" && resourceType == "rules" {
		rawMarkdown := strings.TrimSpace(c.PostForm("markdown_content"))
		if rawMarkdown != "" {
			root, err := createUploadRoot()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "创建临时上传目录失败"})
				return
			}

			requestedName := strings.TrimSpace(c.PostForm("file_name"))
			if requestedName == "" {
				requestedName = "RULES.md"
			}
			if !isRulesTextExtension(requestedName) {
				requestedName = "RULES.md"
			}
			safeFileName := sanitizeLocalFilename(requestedName)
			if !isRulesTextExtension(safeFileName) {
				safeFileName = "RULES.md"
			}

			filePath := filepath.Join(root, safeFileName)
			contentBytes := []byte(rawMarkdown)
			if int64(len(contentBytes)) > maxUploadSize {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "上传文件大小不能超过 50MB"})
				return
			}
			if err := os.WriteFile(filePath, contentBytes, 0644); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "保存规则内容失败"})
				return
			}
			revision.FileName = safeFileName
			revision.FilePath = filePath
			revision.FileSize = int64(len(contentBytes))
			changed = true
		}
	} else if uploadMode == "folder" {
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "解析表单失败"})
			return
		}
		files := form.File["files"]
		filePaths := form.Value["file_paths"]
		if len(files) > 0 {
			root, err := createUploadRoot()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "创建临时上传目录失败"})
				return
			}

			var totalFileSize int64
			for i, fh := range files {
				if totalFileSize+fh.Size > maxUploadSize {
					c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "上传总大小不能超过 50MB"})
					return
				}
				relPath := fh.Filename
				if i < len(filePaths) && filePaths[i] != "" {
					relPath = filePaths[i]
				}
				normalizedRel, err := service.NormalizeRepoRelativePath(relPath)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "非法文件路径"})
					return
				}
				localPath := filepath.Join(root, filepath.FromSlash(normalizedRel))
				if !isPathInsideBase(root, localPath) {
					c.JSON(http.StatusBadRequest, gin.H{"error": "非法文件路径"})
					return
				}
				if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "创建子目录失败"})
					return
				}

				src, err := fh.Open()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
					return
				}
				dst, err := os.Create(localPath)
				if err != nil {
					src.Close()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
					return
				}
				if _, err := io.Copy(dst, src); err != nil {
					src.Close()
					dst.Close()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
					return
				}
				src.Close()
				dst.Close()

				totalFileSize += fh.Size
				if i == 0 {
					revision.FileName = filepath.Base(normalizedRel)
					revision.FilePath = localPath
					revision.FileSize = fh.Size
				}
			}
			changed = true
		}
	} else {
		file, header, err := c.Request.FormFile("file")
		if err == nil {
			defer file.Close()
			if header.Size > maxUploadSize {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "上传文件大小不能超过 50MB"})
				return
			}
			if resourceType == "rules" && !isRulesTextExtension(header.Filename) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Rules 仅支持 .md 或 .txt 文件"})
				return
			}

			root, err := createUploadRoot()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "创建临时上传目录失败"})
				return
			}
			safeFileName := sanitizeLocalFilename(header.Filename)
			filePath := filepath.Join(root, safeFileName)
			dst, err := os.Create(filePath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
				return
			}
			if _, err := io.Copy(dst, file); err != nil {
				dst.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
				return
			}
			dst.Close()

			revision.FileName = filepath.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
			if revision.FileName == "" || revision.FileName == "." || revision.FileName == "/" {
				revision.FileName = safeFileName
			}
			revision.FilePath = filePath
			revision.FileSize = header.Size
			changed = true
		} else if !errors.Is(err, http.ErrMissingFile) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
			return
		}
	}

	var newThumbnail string
	keepThumbnail := false
	defer func() {
		if keepThumbnail || newThumbnail == "" {
			return
		}
		if thumbPath, ok := resolveThumbnailPath(h.cfg.ThumbnailDir, newThumbnail); ok {
			_ = os.Remove(thumbPath)
		}
	}()

	thumbnailFile, thumbnailHeader, thumbnailErr := c.Request.FormFile("thumbnail")
	if thumbnailErr == nil && thumbnailFile != nil {
		defer thumbnailFile.Close()
		thumbFileName, err := saveUploadedThumbnail(revision.Name, thumbnailFile, thumbnailHeader, h.cfg.ThumbnailDir)
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
		newThumbnail = thumbFileName
		revision.ThumbnailURL = thumbFileName
		changed = true
	} else if thumbnailErr != nil && !errors.Is(thumbnailErr, http.ErrMissingFile) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid thumbnail"})
		return
	}

	if !changed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有检测到实际更新内容"})
		return
	}

	preparePendingRevisionForResourceUpdate(revision, resourceType)
	created, err := h.skillSvc.CreatePendingRevision(skill, revision)
	if errors.Is(err, service.ErrActiveRevisionExists) {
		c.JSON(http.StatusConflict, gin.H{"error": "当前已有待审核更新，请等待审核完成"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建更新修订失败"})
		return
	}

	keepUploadRoot = true
	keepThumbnail = true
	h.dispatchAIReviewRevision(created.ID)

	view := service.BuildSkillReviewView(skill, created)
	view.ThumbnailURL = normalizeThumbnailURL(view.ThumbnailURL)
	c.JSON(http.StatusOK, gin.H{
		"skill":    view,
		"approved": false,
		"feedback": created.AIFeedback,
	})
}
