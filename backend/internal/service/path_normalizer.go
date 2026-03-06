package service

import (
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

	dirPath = path.Join(base, folder)
	filePath = path.Join(dirPath, fileName)
	return dirPath, filePath
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
