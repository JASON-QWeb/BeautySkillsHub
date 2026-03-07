package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetSkillSummary handles GET /api/skills/summary.
func (h *SkillHandler) GetSkillSummary(c *gin.Context) {
	resourceType := strings.TrimSpace(c.Query("resource_type"))
	if resourceType != "" && !validResourceTypes[resourceType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 resource_type"})
		return
	}

	total, yesterdayNew, err := h.skillSvc.GetResourceSummary(resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取汇总失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":         total,
		"yesterday_new": yesterdayNew,
	})
}

// LikeSkill handles POST /api/skills/:id/like.
func (h *SkillHandler) LikeSkill(c *gin.Context) {
	userID, _, ok := currentUserIdentity(c)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI 审核未通过，无法点赞"})
		return
	}

	liked, likesCount, err := h.skillSvc.LikeSkill(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "点赞失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"liked":       liked,
		"likes_count": likesCount,
	})
}

// UnlikeSkill handles DELETE /api/skills/:id/like.
func (h *SkillHandler) UnlikeSkill(c *gin.Context) {
	userID, _, ok := currentUserIdentity(c)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI 审核未通过，无法取消点赞"})
		return
	}

	liked, likesCount, err := h.skillSvc.UnlikeSkill(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消点赞失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"liked":       liked,
		"likes_count": likesCount,
	})
}

// TrackDownloadHit handles POST /api/skills/:id/download-hit.
func (h *SkillHandler) TrackDownloadHit(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	if _, err := h.getSkillResource(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "资源不存在"})
		return
	}

	if err := h.skillSvc.IncrementDownload(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "下载计数失败"})
		return
	}

	updated, err := h.getSkillResource(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取资源失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"downloads": updated.Downloads,
	})
}
