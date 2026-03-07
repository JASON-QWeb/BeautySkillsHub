package service

import (
	"path/filepath"
	"testing"

	"skill-hub/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newSkillServiceForTrendingTest(t *testing.T) *SkillService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "trending.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Skill{}); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	return NewSkillService(db)
}

func TestGetTrending_UsesSameVisibilityRulesAsList(t *testing.T) {
	svc := newSkillServiceForTrendingTest(t)

	seeds := []model.Skill{
		{
			Name:              "visible-approved",
			ResourceType:      "skill",
			AIApproved:        true,
			HumanReviewStatus: model.HumanReviewStatusApproved,
			Downloads:         20,
		},
		{
			Name:              "visible-pending",
			ResourceType:      "skill",
			AIApproved:        true,
			HumanReviewStatus: model.HumanReviewStatusPending,
			Downloads:         10,
		},
		{
			Name:              "hidden-ai-rejected",
			ResourceType:      "skill",
			AIApproved:        false,
			HumanReviewStatus: model.HumanReviewStatusApproved,
			Downloads:         999,
		},
		{
			Name:              "hidden-human-rejected",
			ResourceType:      "skill",
			AIApproved:        true,
			HumanReviewStatus: model.HumanReviewStatusRejected,
			Downloads:         998,
		},
	}
	for i := range seeds {
		if err := svc.db.Create(&seeds[i]).Error; err != nil {
			t.Fatalf("create seed %d: %v", i, err)
		}
	}

	trending, err := svc.GetTrending(10, "skill")
	if err != nil {
		t.Fatalf("get trending failed: %v", err)
	}
	if len(trending) != 2 {
		t.Fatalf("expected 2 visible items, got %d", len(trending))
	}
	if trending[0].Name != "visible-approved" || trending[1].Name != "visible-pending" {
		t.Fatalf("unexpected trending order/contents: %+v", []string{trending[0].Name, trending[1].Name})
	}
}
