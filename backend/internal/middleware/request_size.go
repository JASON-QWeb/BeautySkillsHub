package middleware

import (
	"errors"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func LimitMultipartBody(maxBytes, maxMemory int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request == nil || c.Request.Body == nil {
			c.Next()
			return
		}
		if c.Request.ContentLength > maxBytes && c.Request.ContentLength != -1 {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		contentType := strings.ToLower(strings.TrimSpace(c.GetHeader("Content-Type")))
		if !strings.HasPrefix(contentType, "multipart/form-data") {
			c.Next()
			return
		}

		if err := c.Request.ParseMultipartForm(maxMemory); err != nil {
			if isRequestBodyTooLarge(err) {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
				return
			}
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
			return
		}
		defer func() {
			if c.Request.MultipartForm != nil {
				_ = c.Request.MultipartForm.RemoveAll()
			}
		}()

		c.Next()
	}
}

func isRequestBodyTooLarge(err error) bool {
	if err == nil {
		return false
	}

	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return true
	}
	if errors.Is(err, multipart.ErrMessageTooLarge) {
		return true
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "request body too large") || strings.Contains(text, "too large")
}
