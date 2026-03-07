package service

import (
	"context"
	"encoding/json"
	"log"
)

// SkillContextCache stores serialized skills context for AI prompts.
type SkillContextCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string) error
	PublishInvalidate(ctx context.Context, channel string, payload string) error
}

// SkillContextProvider provides serialized approved-skill context for AI chat.
type SkillContextProvider struct {
	skillSvc          *SkillService
	cache             SkillContextCache
	cacheKey          string
	invalidateChannel string
}

func NewSkillContextProvider(
	skillSvc *SkillService,
	cache SkillContextCache,
	cacheKey string,
	invalidateChannel string,
) *SkillContextProvider {
	return &SkillContextProvider{
		skillSvc:          skillSvc,
		cache:             cache,
		cacheKey:          cacheKey,
		invalidateChannel: invalidateChannel,
	}
}

// GetSkillsContextJSON returns cached context first and falls back to DB rebuild.
func (p *SkillContextProvider) GetSkillsContextJSON(ctx context.Context) (string, error) {
	if p.cache != nil && p.cacheKey != "" {
		cached, err := p.cache.Get(ctx, p.cacheKey)
		if err == nil && cached != "" {
			return cached, nil
		}
		if err != nil {
			log.Printf("skills context cache get failed, fallback to db: %v", err)
		}
	}

	return p.rebuildFromDB(ctx)
}

// RefreshSkillsContext rebuilds context from DB and updates cache.
func (p *SkillContextProvider) RefreshSkillsContext(ctx context.Context) error {
	data, err := p.rebuildFromDB(ctx)
	if err != nil {
		return err
	}

	if p.cache != nil && p.invalidateChannel != "" {
		if err := p.cache.PublishInvalidate(ctx, p.invalidateChannel, data); err != nil {
			log.Printf("skills context invalidate publish failed: %v", err)
		}
	}
	return nil
}

func (p *SkillContextProvider) rebuildFromDB(ctx context.Context) (string, error) {
	skills, err := p.skillSvc.GetAllApprovedBrief()
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(skills)
	if err != nil {
		return "", err
	}

	data := string(payload)
	if p.cache != nil && p.cacheKey != "" {
		if err := p.cache.Set(ctx, p.cacheKey, data); err != nil {
			log.Printf("skills context cache set failed: %v", err)
		}
	}
	return data, nil
}
