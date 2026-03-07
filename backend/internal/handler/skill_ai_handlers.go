package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ChatRecommend handles POST /api/ai/chat (SSE streaming)
func (h *SkillHandler) ChatRecommend(c *gin.Context) {
	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入消息"})
		return
	}

	if h.skillContextProvider == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 上下文服务未初始化"})
		return
	}

	skillsJSON, err := h.skillContextProvider.GetSkillsContextJSON(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取资源列表失败"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	c.Stream(func(w io.Writer) bool {
		err := h.aiSvc.ChatRecommendStream(c.Request.Context(), req.Message, skillsJSON, func(chunk string) error {
			c.SSEvent("message", chunk)
			c.Writer.Flush()
			return nil
		})

		if err != nil {
			c.SSEvent("error", err.Error())
			c.Writer.Flush()
		}

		c.SSEvent("done", "[DONE]")
		c.Writer.Flush()
		return false
	})
}
