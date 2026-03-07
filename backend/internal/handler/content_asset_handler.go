package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const maxContentImageSize = 8 << 20 // 8MB

var allowedContentImageExtensions = map[string]struct{}{
	".png":  {},
	".jpg":  {},
	".jpeg": {},
	".webp": {},
	".gif":  {},
}

type ContentAssetHandler struct {
	assetDir string
}

func NewContentAssetHandler(uploadDir string) *ContentAssetHandler {
	return &ContentAssetHandler{
		assetDir: filepath.Join(uploadDir, "content-assets"),
	}
}

func (h *ContentAssetHandler) UploadImage(c *gin.Context) {
	if _, _, ok := currentUserIdentity(c); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image required"})
		return
	}
	defer file.Close()

	if header.Size > maxContentImageSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "image size exceeds 8MB"})
		return
	}

	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(header.Filename)))
	if _, ok := allowedContentImageExtensions[ext]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported image type"})
		return
	}

	if err := os.MkdirAll(h.assetDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create asset dir failed"})
		return
	}

	token, err := randomHex(8)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate filename failed"})
		return
	}
	fileName := time.Now().Format("20060102-150405") + "-" + token + ext
	filePath := filepath.Join(h.assetDir, fileName)

	dst, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save image failed"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save image failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"url": "/api/content-assets/" + fileName,
	})
}

func (h *ContentAssetHandler) ServeImage(c *gin.Context) {
	fileName := strings.TrimSpace(c.Param("filename"))
	if fileName == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}
	safeName := filepath.Base(fileName)
	if safeName != fileName {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}

	filePath := filepath.Join(h.assetDir, safeName)
	if !isPathInsideBase(h.assetDir, filePath) {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}
	c.File(filePath)
}

func randomHex(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
