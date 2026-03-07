package handler

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var readmeCandidateNames = []string{
	"SKILL.md", "SKILLS.md", "README.md",
	"skill.md", "skills.md", "readme.md",
	"SKILL.MD", "SKILLS.MD", "README.MD",
}

var readmeCandidatePriority = map[string]int{
	"skill.md":  0,
	"skills.md": 1,
	"readme.md": 2,
}

func findReadmePathInSession(sessionRoot string) string {
	info, err := os.Stat(sessionRoot)
	if err != nil || !info.IsDir() {
		return ""
	}

	for _, candidate := range readmeCandidateNames {
		p := filepath.Join(sessionRoot, candidate)
		if fileExists(p) {
			return p
		}
	}

	bestPath := ""
	bestDepth := int(^uint(0) >> 1)
	bestPriority := int(^uint(0) >> 1)

	_ = filepath.WalkDir(sessionRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		base := strings.ToLower(strings.TrimSpace(d.Name()))
		priority, ok := readmeCandidatePriority[base]
		if !ok {
			return nil
		}

		rel, relErr := filepath.Rel(sessionRoot, path)
		if relErr != nil {
			return nil
		}
		depth := strings.Count(rel, string(filepath.Separator))

		if bestPath == "" ||
			depth < bestDepth ||
			(depth == bestDepth && priority < bestPriority) ||
			(depth == bestDepth && priority == bestPriority && path < bestPath) {
			bestPath = path
			bestDepth = depth
			bestPriority = priority
		}
		return nil
	})

	return bestPath
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
