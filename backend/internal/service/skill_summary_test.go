package service

import (
	"testing"
	"time"

	"skill-hub/internal/model"
	"skill-hub/internal/testutil"
)

func newSkillServiceForSummaryTest(t *testing.T) *SkillService {
	t.Helper()

	tdb := testutil.OpenPostgresTestDB(t)
	return NewSkillService(tdb.DB)
}

func createSummarySkill(t *testing.T, svc *SkillService, sk model.Skill) {
	t.Helper()
	if err := svc.db.Create(&sk).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}
}

func TestGetResourceSummary(t *testing.T) {
	svc := newSkillServiceForSummaryTest(t)

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterdayStart := todayStart.Add(-24 * time.Hour)

	createSummarySkill(t, svc, model.Skill{
		Name:              "skill-yesterday",
		ResourceType:      "skill",
		AIApproved:        true,
		HumanReviewStatus: model.HumanReviewStatusApproved,
		CreatedAt:         yesterdayStart.Add(2 * time.Hour),
	})
	createSummarySkill(t, svc, model.Skill{
		Name:              "skill-today",
		ResourceType:      "skill",
		AIApproved:        true,
		HumanReviewStatus: model.HumanReviewStatusPending,
		CreatedAt:         todayStart.Add(2 * time.Hour),
	})
	createSummarySkill(t, svc, model.Skill{
		Name:              "mcp-yesterday",
		ResourceType:      "mcp",
		AIApproved:        true,
		HumanReviewStatus: model.HumanReviewStatusApproved,
		CreatedAt:         yesterdayStart.Add(3 * time.Hour),
	})
	createSummarySkill(t, svc, model.Skill{
		Name:              "rejected-yesterday",
		ResourceType:      "skill",
		AIApproved:        false,
		HumanReviewStatus: model.HumanReviewStatusRejected,
		CreatedAt:         yesterdayStart.Add(4 * time.Hour),
	})

	totalAll, yesterdayAll, err := svc.GetResourceSummary("")
	if err != nil {
		t.Fatalf("get all summary failed: %v", err)
	}
	if totalAll != 3 {
		t.Fatalf("expected total=3, got %d", totalAll)
	}
	if yesterdayAll != 2 {
		t.Fatalf("expected yesterday_new=2, got %d", yesterdayAll)
	}

	totalSkill, yesterdaySkill, err := svc.GetResourceSummary("skill")
	if err != nil {
		t.Fatalf("get skill summary failed: %v", err)
	}
	if totalSkill != 2 {
		t.Fatalf("expected skill total=2, got %d", totalSkill)
	}
	if yesterdaySkill != 1 {
		t.Fatalf("expected skill yesterday_new=1, got %d", yesterdaySkill)
	}
}
