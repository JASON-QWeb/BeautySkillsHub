package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *SkillHandler) ListMyUploads(c *gin.Context) {
	userID, username, ok := currentUserIdentity(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或登录状态无效"})
		return
	}

	search := strings.TrimSpace(c.Query("search"))
	resourceType := strings.TrimSpace(c.Query("resource_type"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	skills, total, err := h.skillSvc.GetUserUploads(userID, username, search, resourceType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取上传资源失败"})
		return
	}
	stats, err := h.skillSvc.GetUserUploadStats(userID, username, resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取上传统计失败"})
		return
	}
	topTags, err := h.skillSvc.GetUserTopTags(userID, username, resourceType, 3)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取标签统计失败"})
		return
	}
	activities, err := h.skillSvc.GetUserRecentActivities(userID, username, 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取活动失败"})
		return
	}

	for i := range skills {
		skills[i].ThumbnailURL = normalizeThumbnailURL(skills[i].ThumbnailURL)
	}
	enrichUserEngagement(skills, userID, h.skillSvc)

	c.JSON(http.StatusOK, gin.H{
		"skills":     skills,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"stats":      stats,
		"top_tags":   topTags,
		"activities": activities,
	})
}
