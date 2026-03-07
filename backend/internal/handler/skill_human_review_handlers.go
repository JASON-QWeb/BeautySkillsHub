package handler

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"skill-hub/internal/model"
	"skill-hub/internal/service"

	"github.com/gin-gonic/gin"
)

type humanReviewRequest struct {
	Approved *bool  `json:"approved"`
	Feedback string `json:"feedback"`
}

// HumanReviewSkill handles POST /api/skills/:id/human-review.
func (h *SkillHandler) HumanReviewSkill(c *gin.Context) {
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

	if !skill.AIApproved {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI 审核未通过，无法人工复核"})
		return
	}

	if skill.HumanReviewStatus == model.HumanReviewStatusApproved {
		c.JSON(http.StatusConflict, gin.H{"error": "该资源已完成人工复核"})
		return
	}

	if canManageSkill(skill, userID, username) {
		c.JSON(http.StatusForbidden, gin.H{"error": "不能复核自己上传的资源"})
		return
	}

	if !canReviewSkill(skill, userID, username) {
		c.JSON(http.StatusForbidden, gin.H{"error": "当前用户无权执行复核"})
		return
	}

	req := humanReviewRequest{}
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
		return
	}

	approved := true
	if req.Approved != nil {
		approved = *req.Approved
	}

	now := time.Now()
	reviewerID := userID

	skill.HumanReviewerID = &reviewerID
	skill.HumanReviewer = strings.TrimSpace(username)
	skill.HumanReviewFeedback = strings.TrimSpace(req.Feedback)
	skill.HumanReviewedAt = &now
	if approved {
		skill.HumanReviewStatus = model.HumanReviewStatusApproved
		skill.Published = false
	} else {
		skill.HumanReviewStatus = model.HumanReviewStatusRejected
		skill.Published = false
	}

	// Reject path: save and finish
	if !approved {
		if err := h.skillSvc.UpdateSkill(skill); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存复核结果失败"})
			return
		}
		if h.skillContextProvider != nil {
			if err := h.skillContextProvider.RefreshSkillsContext(c.Request.Context()); err != nil {
				log.Printf("refresh skills context after human review failed: %v", err)
			}
		}
		skill.ThumbnailURL = normalizeThumbnailURL(skill.ThumbnailURL)
		c.JSON(http.StatusOK, gin.H{"skill": skill})
		return
	}

	// Non-skill resources do not need GitHub sync.
	if skill.ResourceType != "skill" || !h.cfg.GitHubSyncEnabled || h.githubSyncSvc == nil {
		skill.GitHubSyncStatus = service.GitHubSyncStatusDisabled
		skill.GitHubSyncError = ""
		skill.Published = true
		if err := h.skillSvc.UpdateSkill(skill); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存复核结果失败"})
			return
		}
		if h.skillContextProvider != nil {
			if err := h.skillContextProvider.RefreshSkillsContext(c.Request.Context()); err != nil {
				log.Printf("refresh skills context after human review failed: %v", err)
			}
		}
		skill.ThumbnailURL = normalizeThumbnailURL(skill.ThumbnailURL)
		c.JSON(http.StatusOK, gin.H{"skill": skill})
		return
	}

	// Skill resources: sync to GitHub after human review approval.
	skill.GitHubSyncStatus = service.GitHubSyncStatusPending
	skill.GitHubSyncError = ""
	skill.Published = false
	if err := h.skillSvc.UpdateSkill(skill); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存复核结果失败"})
		return
	}

	entries, err := h.collectSyncEntriesFromLocal(skill)
	if err != nil {
		skill.GitHubSyncStatus = service.GitHubSyncStatusFailed
		skill.GitHubSyncError = "收集待同步文件失败: " + err.Error()
		skill.Published = false
		_ = h.skillSvc.UpdateSkill(skill)
		c.JSON(http.StatusBadGateway, gin.H{"error": skill.GitHubSyncError, "skill": skill})
		return
	}

	syncResult := h.githubSyncSvc.SyncUploadedFolder(c.Request.Context(), skill.Name, skill.ResourceType, entries)
	if syncResult.Status != service.GitHubSyncStatusSuccess {
		skill.GitHubSyncStatus = service.GitHubSyncStatusFailed
		skill.GitHubSyncError = strings.TrimSpace(syncResult.Error)
		skill.Published = false
		_ = h.skillSvc.UpdateSkill(skill)
		if skill.GitHubSyncError == "" {
			skill.GitHubSyncError = "GitHub 同步失败"
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": skill.GitHubSyncError, "skill": skill})
		return
	}

	skill.GitHubSyncStatus = syncResult.Status
	skill.GitHubPath = syncResult.Path
	skill.GitHubURL = syncResult.URL
	skill.GitHubFiles = service.MarshalGitHubFiles(syncResult.Files)
	skill.GitHubSyncError = ""
	skill.Published = true
	if err := h.skillSvc.UpdateSkill(skill); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存复核结果失败"})
		return
	}

	if h.skillContextProvider != nil {
		if err := h.skillContextProvider.RefreshSkillsContext(c.Request.Context()); err != nil {
			log.Printf("refresh skills context after human review failed: %v", err)
		}
	}
	skill.ThumbnailURL = normalizeThumbnailURL(skill.ThumbnailURL)
	c.JSON(http.StatusOK, gin.H{
		"skill": skill,
	})
}
