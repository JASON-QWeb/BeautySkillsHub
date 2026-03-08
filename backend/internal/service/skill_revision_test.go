package service

import (
	"errors"
	"testing"

	"skill-hub/internal/model"
	"skill-hub/internal/testutil"
)

func newSkillServiceForRevisionTest(t *testing.T) *SkillService {
	t.Helper()

	tdb := testutil.OpenPostgresTestDB(t)
	return NewSkillService(tdb.DB)
}

func createPublishedRevisionTestSkill(t *testing.T, svc *SkillService, resourceType string) model.Skill {
	t.Helper()

	skill := model.Skill{
		UserID:            7,
		Name:              "Published Resource",
		Description:       "published description",
		ResourceType:      resourceType,
		Author:            "alice",
		FileName:          "old.md",
		FilePath:          "/tmp/old.md",
		FileSize:          12,
		ThumbnailURL:      "old-thumb.png",
		Downloads:         42,
		LikesCount:        5,
		AIApproved:        true,
		AIReviewStatus:    model.AIReviewStatusPassed,
		AIReviewPhase:     model.AIReviewPhaseDone,
		HumanReviewStatus: model.HumanReviewStatusApproved,
		Published:         true,
	}
	if err := svc.db.Create(&skill).Error; err != nil {
		t.Fatalf("create published skill: %v", err)
	}
	return skill
}

func TestCreatePendingRevision_DoesNotCreateSecondPublishedSkill(t *testing.T) {
	svc := newSkillServiceForRevisionTest(t)
	base := createPublishedRevisionTestSkill(t, svc, "skill")

	revision, err := svc.CreatePendingRevision(&base, &model.SkillRevision{
		UserID:         base.UserID,
		ResourceType:   base.ResourceType,
		Name:           "Published Resource v2",
		Description:    "updated description",
		Author:         base.Author,
		FileName:       "new.md",
		FilePath:       "/tmp/new.md",
		FileSize:       99,
		ThumbnailURL:   "new-thumb.png",
		AIReviewStatus: model.AIReviewStatusQueued,
		AIReviewPhase:  model.AIReviewPhaseQueued,
		Status:         model.SkillRevisionStatusPending,
	})
	if err != nil {
		t.Fatalf("create pending revision: %v", err)
	}

	var skillCount int64
	if err := svc.db.Model(&model.Skill{}).Count(&skillCount).Error; err != nil {
		t.Fatalf("count skills: %v", err)
	}
	if skillCount != 1 {
		t.Fatalf("expected 1 published skill row, got %d", skillCount)
	}

	if revision.SkillID != base.ID {
		t.Fatalf("expected revision to reference base skill %d, got %d", base.ID, revision.SkillID)
	}
	if revision.Name != "Published Resource v2" {
		t.Fatalf("expected revision name to be updated, got %q", revision.Name)
	}
}

func TestCreatePendingRevision_RejectsSecondActiveRevision(t *testing.T) {
	svc := newSkillServiceForRevisionTest(t)
	base := createPublishedRevisionTestSkill(t, svc, "rules")

	_, err := svc.CreatePendingRevision(&base, &model.SkillRevision{
		UserID:         base.UserID,
		ResourceType:   base.ResourceType,
		Name:           "Rules v2",
		Description:    "first draft",
		Author:         base.Author,
		FileName:       "rules.md",
		FilePath:       "/tmp/rules.md",
		FileSize:       10,
		AIReviewStatus: model.AIReviewStatusQueued,
		AIReviewPhase:  model.AIReviewPhaseQueued,
		Status:         model.SkillRevisionStatusPending,
	})
	if err != nil {
		t.Fatalf("create first revision: %v", err)
	}

	_, err = svc.CreatePendingRevision(&base, &model.SkillRevision{
		UserID:         base.UserID,
		ResourceType:   base.ResourceType,
		Name:           "Rules v3",
		Description:    "second draft",
		Author:         base.Author,
		FileName:       "rules-v3.md",
		FilePath:       "/tmp/rules-v3.md",
		FileSize:       11,
		AIReviewStatus: model.AIReviewStatusQueued,
		AIReviewPhase:  model.AIReviewPhaseQueued,
		Status:         model.SkillRevisionStatusPending,
	})
	if !errors.Is(err, ErrActiveRevisionExists) {
		t.Fatalf("expected ErrActiveRevisionExists, got %v", err)
	}
}

func TestApplyApprovedRevision_OverwritesPublishedContentButPreservesCounters(t *testing.T) {
	svc := newSkillServiceForRevisionTest(t)
	base := createPublishedRevisionTestSkill(t, svc, "tools")

	revision, err := svc.CreatePendingRevision(&base, &model.SkillRevision{
		UserID:              base.UserID,
		ResourceType:        base.ResourceType,
		Name:                "Tools v2",
		Description:         "updated tool package",
		Author:              base.Author,
		FileName:            "tool.zip",
		FilePath:            "/tmp/tool.zip",
		FileSize:            256,
		ThumbnailURL:        "tool-thumb.png",
		AIApproved:          true,
		AIReviewStatus:      model.AIReviewStatusPassed,
		AIReviewPhase:       model.AIReviewPhaseDone,
		HumanReviewStatus:   model.HumanReviewStatusApproved,
		HumanReviewer:       "bob",
		HumanReviewFeedback: "looks good",
		Status:              model.SkillRevisionStatusPending,
	})
	if err != nil {
		t.Fatalf("create revision: %v", err)
	}

	updated, err := svc.ApplyApprovedRevision(revision.ID)
	if err != nil {
		t.Fatalf("apply revision: %v", err)
	}

	if updated.ID != base.ID {
		t.Fatalf("expected base skill id %d to be preserved, got %d", base.ID, updated.ID)
	}
	if updated.Name != "Tools v2" {
		t.Fatalf("expected updated name, got %q", updated.Name)
	}
	if updated.Downloads != 42 {
		t.Fatalf("expected downloads to be preserved, got %d", updated.Downloads)
	}
	if updated.LikesCount != 5 {
		t.Fatalf("expected likes_count to be preserved, got %d", updated.LikesCount)
	}
	if !updated.Published {
		t.Fatalf("expected published skill to stay published after apply")
	}

	storedRevision, err := svc.GetSkillRevision(revision.ID)
	if err != nil {
		t.Fatalf("reload revision: %v", err)
	}
	if storedRevision.Status != model.SkillRevisionStatusApplied {
		t.Fatalf("expected revision to be marked applied, got %q", storedRevision.Status)
	}
}
