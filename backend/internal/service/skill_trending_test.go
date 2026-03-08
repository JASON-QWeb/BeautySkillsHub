package service

import (
	"testing"

	"skill-hub/internal/model"
	"skill-hub/internal/testutil"
)

func newSkillServiceForTrendingTest(t *testing.T) *SkillService {
	t.Helper()

	tdb := testutil.OpenPostgresTestDB(t)
	return NewSkillService(tdb.DB)
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
