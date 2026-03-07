package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AddFavoriteSkill handles POST /api/skills/:id/favorite.
func (h *SkillHandler) AddFavoriteSkill(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI 审核未通过，无法收藏"})
		return
	}

	if err := h.skillSvc.AddFavorite(uint(id), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "收藏失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"favorited": true})
}

// RemoveFavoriteSkill handles DELETE /api/skills/:id/favorite.
func (h *SkillHandler) RemoveFavoriteSkill(c *gin.Context) {
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

	if _, err := h.getSkillResource(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "资源不存在"})
		return
	}

	if err := h.skillSvc.RemoveFavorite(uint(id), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消收藏失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"favorited": false})
}

// ListMyFavorites handles GET /api/me/favorites.
func (h *SkillHandler) ListMyFavorites(c *gin.Context) {
	userID, _, ok := currentUserIdentity(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或登录状态无效"})
		return
	}

	resourceType := c.Query("resource_type")
	skills, err := h.skillSvc.GetUserFavorites(userID, resourceType, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取收藏失败"})
		return
	}

	for i := range skills {
		skills[i].ThumbnailURL = normalizeThumbnailURL(skills[i].ThumbnailURL)
		liked, err := h.skillSvc.HasUserLiked(skills[i].ID, userID)
		if err == nil {
			skills[i].UserLiked = liked
		}
	}

	c.JSON(http.StatusOK, gin.H{"skills": skills})
}
