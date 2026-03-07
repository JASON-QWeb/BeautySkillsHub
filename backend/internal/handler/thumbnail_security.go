package handler

import (
	"path/filepath"
	"strings"
)

func resolveThumbnailPath(thumbnailDir, requested string) (string, bool) {
	normalized := strings.ReplaceAll(strings.TrimSpace(requested), "\\", "/")
	fileName := filepath.Base(normalized)
	if fileName == "" || fileName == "." || fileName == "/" {
		return "", false
	}

	thumbnailPath := filepath.Join(thumbnailDir, fileName)
	if !isPathInsideBase(thumbnailDir, thumbnailPath) {
		return "", false
	}
	return thumbnailPath, true
}
