package handler

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"skill-hub/internal/model"
	"skill-hub/internal/service"

	"github.com/gin-gonic/gin"
)

// DeleteSkill handles DELETE /api/skills/:id
func (h *SkillHandler) DeleteSkill(c *gin.Context) {
	skill, ok := h.resolveSkillForDeletion(c)
	if !ok {
		return
	}

	githubError, err := h.deleteSkillResource(c.Request.Context(), skill, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除记录失败"})
		return
	}

	resp := gin.H{"message": "删除成功"}
	if githubError != "" {
		resp["github_error"] = githubError
	}
	c.JSON(http.StatusOK, resp)
}

// StreamDeleteSkill handles DELETE /api/skills/:id/stream-delete (SSE progress).
func (h *SkillHandler) StreamDeleteSkill(c *gin.Context) {
	skill, ok := h.resolveSkillForDeletion(c)
	if !ok {
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	emit := func(event string, payload gin.H) {
		c.SSEvent(event, payload)
		c.Writer.Flush()
	}

	githubError, err := h.deleteSkillResource(c.Request.Context(), skill, func(stage string) {
		emit("progress", gin.H{"stage": stage})
	})
	if err != nil {
		emit("error", gin.H{"error": "删除记录失败"})
		emit("done", gin.H{"ok": false})
		return
	}

	done := gin.H{
		"ok":      true,
		"message": "删除成功",
	}
	if githubError != "" {
		done["github_error"] = githubError
	}
	emit("done", done)
}

func (h *SkillHandler) resolveSkillForDeletion(c *gin.Context) (*model.Skill, bool) {
	userID, username, ok := currentUserIdentity(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或登录状态无效"})
		return nil, false
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return nil, false
	}

	skill, err := h.getSkillResource(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "资源不存在"})
		return nil, false
	}

	if !canManageSkill(skill, userID, username) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只能删除自己发布的资源"})
		return nil, false
	}

	return skill, true
}

func (h *SkillHandler) deleteSkillResource(ctx context.Context, skill *model.Skill, report func(stage string)) (string, error) {
	if report != nil {
		report("db")
	}
	removeLocalSkillFiles(h.cfg.UploadDir, skill.FilePath)
	if err := h.skillSvc.DeleteSkill(skill.ID); err != nil {
		return "", err
	}

	if h.skillContextProvider != nil {
		if err := h.skillContextProvider.RefreshSkillsContext(ctx); err != nil {
			log.Printf("refresh skills context after delete failed: %v", err)
		}
	}

	if report != nil {
		report("github")
	}
	githubError := h.deleteSkillFromGitHub(ctx, skill)

	return githubError, nil
}

func (h *SkillHandler) deleteSkillFromGitHub(ctx context.Context, skill *model.Skill) string {
	if h.githubSyncSvc == nil || skill.GitHubSyncStatus != "success" {
		return ""
	}

	githubFiles := service.UnmarshalGitHubFiles(skill.GitHubFiles)
	if len(githubFiles) > 0 {
		if err := h.githubSyncSvc.DeleteSkillFilesFromGitHub(ctx, githubFiles); err != nil {
			log.Printf("delete skill %d github manifest files failed: %v", skill.ID, err)
			return err.Error()
		}
		return ""
	}

	if skill.GitHubPath == "" {
		return ""
	}
	if err := h.githubSyncSvc.DeleteSkillFromGitHub(ctx, skill.GitHubPath); err != nil {
		log.Printf("delete skill %d from github failed: %v", skill.ID, err)
		return err.Error()
	}
	return ""
}

func removeLocalSkillFiles(uploadDir, filePath string) {
	if filePath == "" {
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
