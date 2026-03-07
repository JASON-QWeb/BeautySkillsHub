package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"

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

	var req updateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
		return
	}

	if req.Name == nil && req.Description == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要更新一个字段"})
		return
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "名称不能为空"})
			return
		}
		skill.Name = name
	}
	if req.Description != nil {
		skill.Description = strings.TrimSpace(*req.Description)
	}

	if err := h.skillSvc.UpdateSkill(skill); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新记录失败"})
		return
	}

	if h.skillContextProvider != nil {
		if err := h.skillContextProvider.RefreshSkillsContext(c.Request.Context()); err != nil {
			log.Printf("refresh skills context after update failed: %v", err)
		}
	}

	skill.ThumbnailURL = normalizeThumbnailURL(skill.ThumbnailURL)
	c.JSON(http.StatusOK, gin.H{"skill": skill})
}
