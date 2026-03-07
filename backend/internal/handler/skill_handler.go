package handler

import (
	"context"
	"errors"
	"strings"
	"sync"

	"skill-hub/internal/config"
	"skill-hub/internal/model"
	"skill-hub/internal/service"
	"skill-hub/internal/service/ai"

	"gorm.io/gorm"
)

type SkillHandler struct {
	skillSvc             *service.SkillService
	aiSvc                *ai.Service
	githubSyncSvc        *service.GitHubSyncService
	skillContextProvider skillContextProvider
	cfg                  *config.Config
	reviewMu             sync.Mutex
	reviewRunning        map[uint]struct{}
}

type skillContextProvider interface {
	GetSkillsContextJSON(ctx context.Context) (string, error)
	RefreshSkillsContext(ctx context.Context) error
}

func NewSkillHandler(
	skillSvc *service.SkillService,
	aiSvc *ai.Service,
	githubSyncSvc *service.GitHubSyncService,
	skillContextProvider skillContextProvider,
	cfg *config.Config,
) *SkillHandler {
	return &SkillHandler{
		skillSvc:             skillSvc,
		aiSvc:                aiSvc,
		githubSyncSvc:        githubSyncSvc,
		skillContextProvider: skillContextProvider,
		cfg:                  cfg,
		reviewRunning:        make(map[uint]struct{}),
	}
}

// validResourceTypes defines the allowed resource type values.
var validResourceTypes = map[string]bool{
	"skill": true,
	"mcp":   true,
	"rules": true,
	"tools": true,
}

func normalizeThumbnailURL(url string) string {
	if url == "" || strings.HasPrefix(url, "http") {
		return url
	}
	return "/api/thumbnails/" + url
}

func isReviewedResourceType(resourceType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(resourceType))
	return normalized == "skill" || normalized == "rules"
}

func (h *SkillHandler) getSkillResource(id uint) (*model.Skill, error) {
	skill, err := h.skillSvc.GetSkill(id)
	if err != nil {
		return nil, err
	}
	if !isReviewedResourceType(skill.ResourceType) {
		return nil, gorm.ErrRecordNotFound
	}
	return skill, nil
}

func isNotFoundError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// enrichUserEngagement batch-loads liked/favorited flags for a list of skills.
func enrichUserEngagement(skills []model.Skill, userID uint, svc *service.SkillService) {
	if len(skills) == 0 || userID == 0 {
		return
	}
	ids := make([]uint, len(skills))
	for i := range skills {
		ids[i] = skills[i].ID
	}
	likedMap, _ := svc.BatchGetUserLikedMap(ids, userID)
	favMap, _ := svc.BatchGetUserFavoritedMap(ids, userID)
	for i := range skills {
		skills[i].UserLiked = likedMap[skills[i].ID]
		skills[i].Favorited = favMap[skills[i].ID]
	}
}
