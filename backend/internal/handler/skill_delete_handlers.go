package handler

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"skill-hub/internal/service"

	"github.com/gin-gonic/gin"
)

// DeleteSkill handles DELETE /api/skills/:id
func (h *SkillHandler) DeleteSkill(c *gin.Context) {
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
		c.JSON(http.StatusForbidden, gin.H{"error": "只能删除自己发布的资源"})
		return
	}

	// Delete from GitHub if synced
	var githubError string
	if h.githubSyncSvc != nil && skill.GitHubSyncStatus == "success" {
		githubFiles := service.UnmarshalGitHubFiles(skill.GitHubFiles)
		if len(githubFiles) > 0 {
			if err := h.githubSyncSvc.DeleteSkillFilesFromGitHub(c.Request.Context(), githubFiles); err != nil {
				githubError = err.Error()
				log.Printf("delete skill %d github manifest files failed: %v", skill.ID, err)
			}
		} else if skill.GitHubPath != "" {
			if err := h.githubSyncSvc.DeleteSkillFromGitHub(c.Request.Context(), skill.GitHubPath); err != nil {
				githubError = err.Error()
				log.Printf("delete skill %d from github failed: %v", skill.ID, err)
			}
		}
	}

	// Delete local file
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

	// Delete DB record
	if err := h.skillSvc.DeleteSkill(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除记录失败"})
		return
	}

	if h.skillContextProvider != nil {
		if err := h.skillContextProvider.RefreshSkillsContext(c.Request.Context()); err != nil {
			log.Printf("refresh skills context after delete failed: %v", err)
		}
	}

	resp := gin.H{"message": "删除成功"}
	if githubError != "" {
		resp["github_error"] = githubError
	}
	c.JSON(http.StatusOK, resp)
}
