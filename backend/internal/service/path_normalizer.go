package service

import (
	"fmt"
	"path"
	"strings"
	"unicode"
)

// BuildSkillRepoPath builds normalized GitHub storage paths for a skill upload.
// Path structure: <baseDir>/<title-slug>/<filename>
func BuildSkillRepoPath(baseDir, resourceType, title, originalFilename string) (dirPath, filePath string) {
	base := cleanPathSegment(baseDir, "skills")
	folder := slugifyTitle(title)
	fileName := sanitizeFilename(originalFilename)
	_ = resourceType // kept for backward-compatible call signature

	dirPath = path.Join(base, folder)
	filePath = path.Join(dirPath, fileName)
	return dirPath, filePath
}

// NormalizeRepoRelativePath validates and normalizes a client-provided relative file path.
// It rejects absolute and traversal paths and preserves Unicode path segments.
func NormalizeRepoRelativePath(raw string) (string, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if normalized == "" {
		return "", fmt.Errorf("path is empty")
	}
	if strings.HasPrefix(normalized, "/") {
		return "", fmt.Errorf("absolute path is not allowed")
	}

	cleaned := path.Clean(normalized)
	if cleaned == "." || cleaned == "" || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("invalid relative path")
	}

	parts := strings.Split(cleaned, "/")
	safeParts := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("invalid path segment")
		}
		if strings.Contains(part, ":") {
			return "", fmt.Errorf("invalid path segment")
		}
		if strings.ContainsRune(part, 0) {
			return "", fmt.Errorf("invalid path segment")
		}
		safeParts = append(safeParts, part)
	}

	return path.Join(safeParts...), nil
}

func normalizeResourceType(resourceType string) string {
	switch strings.ToLower(strings.TrimSpace(resourceType)) {
	case "skill", "mcp", "rules", "tools":
		return strings.ToLower(strings.TrimSpace(resourceType))
	default:
		return "skill"
	}
}

func slugifyTitle(title string) string {
	raw := strings.ToLower(strings.TrimSpace(title))
	if raw == "" {
		return "untitled-skill"
	}

	var b strings.Builder
	lastDash := false
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			lastDash = false
			continue
		}

		if unicode.IsSpace(r) || r == '-' || r == '_' || r == '.' || r == '/' || r == '\\' {
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
			continue
		}

		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "untitled-skill"
	}
	return slug
}

func sanitizeFilename(name string) string {
	normalized := strings.ReplaceAll(name, "\\", "/")
	base := strings.TrimSpace(path.Base(normalized))
	if base == "" || base == "." || base == "/" {
		return "file.bin"
	}

	var b strings.Builder
	lastDash := false
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
			lastDash = false
			continue
		}

		if unicode.IsSpace(r) {
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
			continue
		}

		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}

	safe := strings.Trim(b.String(), "-")
	if safe == "" || safe == "." || safe == ".." {
		return "file.bin"
	}
	return safe
}

func cleanPathSegment(raw, fallback string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}

	parts := strings.Split(strings.ReplaceAll(value, "\\", "/"), "/")
	cleanParts := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "." || p == ".." {
			continue
		}
		cleanParts = append(cleanParts, p)
	}
	if len(cleanParts) == 0 {
		return fallback
	}
	return path.Join(cleanParts...)
}
