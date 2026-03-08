package handler

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"skill-hub/internal/model"
	"skill-hub/internal/service"

	"gorm.io/gorm"
)

func buildReviewStatusResponseValues(
	status string,
	phase string,
	attempts int,
	maxAttempts int,
	approved bool,
	feedback string,
	details string,
) skillReviewStatusResponse {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	retryRemaining := maxAttempts - attempts
	if retryRemaining < 0 {
		retryRemaining = 0
	}
	canRetry := (status == model.AIReviewStatusFailedRetry || status == model.AIReviewStatusFailedTerminal) &&
		attempts < maxAttempts

	return skillReviewStatusResponse{
		Status:         status,
		Phase:          phase,
		Attempts:       attempts,
		MaxAttempts:    maxAttempts,
		RetryRemaining: retryRemaining,
		CanRetry:       canRetry,
		Approved:       approved,
		Feedback:       feedback,
		Progress:       decodeReviewProgress(details),
	}
}

func buildRevisionReviewStatusResponse(revision *model.SkillRevision) skillReviewStatusResponse {
	return buildReviewStatusResponseValues(
		revision.AIReviewStatus,
		revision.AIReviewPhase,
		revision.AIReviewAttempts,
		revision.AIReviewMaxAttempts,
		revision.AIApproved,
		revision.AIFeedback,
		revision.AIReviewDetails,
	)
}

func (h *SkillHandler) loadSkillWithActiveRevision(id uint) (*model.Skill, *model.SkillRevision, error) {
	skill, err := h.skillSvc.GetSkill(id)
	if err != nil {
		return nil, nil, err
	}

	revision, err := h.skillSvc.GetActiveRevision(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return skill, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return skill, revision, nil
}

func (h *SkillHandler) collectReviewTargetsForFile(filePath string) ([]reviewTarget, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("empty file path")
	}

	root := filePath
	sessionRoot := uploadSessionRoot(h.cfg.UploadDir, filePath)
	if sessionRoot != "" {
		if info, err := os.Stat(sessionRoot); err == nil && info.IsDir() {
			root = sessionRoot
		}
	}

	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}

	targets := make([]reviewTarget, 0, 16)
	if info.IsDir() {
		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}

			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			mode := fs.FileMode(0)
			if fileInfo, err := d.Info(); err == nil {
				mode = fileInfo.Mode()
			}
			if kind, ok := classifyReviewTarget(rel, path, mode); ok {
				targets = append(targets, reviewTarget{
					Path:      rel,
					Kind:      kind,
					LocalPath: path,
				})
			}
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	} else {
		rel := sanitizeLocalFilename(filepath.Base(root))
		if kind, ok := classifyReviewTarget(rel, root, info.Mode()); ok {
			targets = append(targets, reviewTarget{Path: rel, Kind: kind, LocalPath: root})
		}
	}

	if len(targets) == 0 {
		fallbackInfo, err := os.Stat(filePath)
		if err != nil {
			return nil, err
		}
		fallbackPath := sanitizeLocalFilename(filepath.Base(filePath))
		kind := "primary-file"
		if classified, ok := classifyReviewTarget(fallbackPath, filePath, fallbackInfo.Mode()); ok {
			kind = classified
		}
		targets = append(targets, reviewTarget{
			Path:      fallbackPath,
			Kind:      kind,
			LocalPath: filePath,
		})
	}

	targets = dedupeReviewTargets(targets)
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Path < targets[j].Path
	})
	return targets, nil
}

func (h *SkillHandler) collectSyncEntriesForFile(filePath string) ([]service.SyncFileEntry, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("empty file path")
	}

	var entries []service.SyncFileEntry
	sessionRoot := uploadSessionRoot(h.cfg.UploadDir, filePath)
	if sessionRoot != "" {
		if info, err := os.Stat(sessionRoot); err != nil || !info.IsDir() {
			sessionRoot = ""
		}
	}
	if sessionRoot != "" {
		walkErr := filepath.WalkDir(sessionRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(sessionRoot, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			normalizedRel, err := service.NormalizeRepoRelativePath(rel)
			if err != nil {
				return err
			}
			entries = append(entries, service.SyncFileEntry{
				LocalPath:    path,
				RelativePath: normalizedRel,
			})
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}

	if len(entries) == 0 {
		if _, err := os.Stat(filePath); err != nil {
			return nil, err
		}
		entries = append(entries, service.SyncFileEntry{
			LocalPath:    filePath,
			RelativePath: sanitizeLocalFilename(filepath.Base(filePath)),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].RelativePath < entries[j].RelativePath
	})
	return entries, nil
}
