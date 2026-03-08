package service

import (
	"context"
	"errors"
	"testing"

	"skill-hub/internal/model"
	"skill-hub/internal/testutil"
)

type fakeSkillContextCache struct {
	data           map[string]string
	getErr         error
	setErr         error
	publishErr     error
	publishCalled  int
	lastPubChannel string
}

func (f *fakeSkillContextCache) Get(_ context.Context, key string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	return f.data[key], nil
}

func (f *fakeSkillContextCache) Set(_ context.Context, key string, value string) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.data[key] = value
	return nil
}

func (f *fakeSkillContextCache) PublishInvalidate(_ context.Context, channel string, _ string) error {
	f.publishCalled++
	f.lastPubChannel = channel
	return f.publishErr
}

func newSkillServiceForContextTest(t *testing.T) *SkillService {
	t.Helper()

	tdb := testutil.OpenPostgresTestDB(t)

	seed := []model.Skill{
		{Name: "approved-a", Description: "ok", Category: "cat", ResourceType: "skill", AIApproved: true},
		{Name: "approved-b", Description: "ok", Category: "cat", ResourceType: "skill", AIApproved: true},
		{Name: "rejected-c", Description: "no", Category: "cat", ResourceType: "skill", AIApproved: false},
	}
	for _, sk := range seed {
		if err := tdb.DB.Create(&sk).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	return NewSkillService(tdb.DB)
}

func TestSkillContextProvider_GetSkillsContextJSON_CacheHit(t *testing.T) {
	cache := &fakeSkillContextCache{
		data: map[string]string{
			"ai:skills_context:v1": `[{"id":1,"name":"cached"}]`,
		},
	}
	p := NewSkillContextProvider(nil, cache, "ai:skills_context:v1", "ai:skills_context:invalidate")

	got, err := p.GetSkillsContextJSON(context.Background())
	if err != nil {
		t.Fatalf("GetSkillsContextJSON error: %v", err)
	}
	if got != `[{"id":1,"name":"cached"}]` {
		t.Fatalf("expected cached payload, got %s", got)
	}
}

func TestSkillContextProvider_GetSkillsContextJSON_FallbackToDBWhenCacheFails(t *testing.T) {
	svc := newSkillServiceForContextTest(t)
	cache := &fakeSkillContextCache{
		data:   map[string]string{},
		getErr: errors.New("redis down"),
	}
	p := NewSkillContextProvider(svc, cache, "ai:skills_context:v1", "ai:skills_context:invalidate")

	got, err := p.GetSkillsContextJSON(context.Background())
	if err != nil {
		t.Fatalf("GetSkillsContextJSON error: %v", err)
	}
	if got == "" {
		t.Fatal("expected db fallback payload")
	}
	if cache.data["ai:skills_context:v1"] == "" {
		t.Fatal("expected fallback rebuild to refresh cache")
	}
}

func TestSkillContextProvider_RefreshSkillsContext_PublishesInvalidate(t *testing.T) {
	svc := newSkillServiceForContextTest(t)
	cache := &fakeSkillContextCache{data: map[string]string{}}
	p := NewSkillContextProvider(svc, cache, "ai:skills_context:v1", "ai:skills_context:invalidate")

	if err := p.RefreshSkillsContext(context.Background()); err != nil {
		t.Fatalf("RefreshSkillsContext error: %v", err)
	}
	if cache.data["ai:skills_context:v1"] == "" {
		t.Fatal("expected cache to be refreshed")
	}
	if cache.publishCalled != 1 {
		t.Fatalf("expected publish called once, got %d", cache.publishCalled)
	}
	if cache.lastPubChannel != "ai:skills_context:invalidate" {
		t.Fatalf("unexpected channel: %s", cache.lastPubChannel)
	}
}
