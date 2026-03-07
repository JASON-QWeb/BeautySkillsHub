package service

import (
	"testing"

	"skill-hub/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newSkillServiceForLikesTest(t *testing.T) *SkillService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Skill{}, &model.SkillLike{}); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}

	return NewSkillService(db)
}

func createLikeTestSkill(t *testing.T, svc *SkillService, name string) model.Skill {
	t.Helper()

	skill := model.Skill{
		Name:              name,
		AIApproved:        true,
		HumanReviewStatus: model.HumanReviewStatusApproved,
		Published:         true,
	}
	if err := svc.db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}
	return skill
}

func TestLikeAndUnlikeSkill_ToggleAndCount(t *testing.T) {
	svc := newSkillServiceForLikesTest(t)
	skill := createLikeTestSkill(t, svc, "Skill Like A")

	liked, count, err := svc.LikeSkill(skill.ID, 10)
	if err != nil {
		t.Fatalf("first like failed: %v", err)
	}
	if !liked {
		t.Fatalf("expected liked=true after first like")
	}
	if count != 1 {
		t.Fatalf("expected likes_count=1 after first like, got %d", count)
	}

	liked, count, err = svc.LikeSkill(skill.ID, 10)
	if err != nil {
		t.Fatalf("second like failed: %v", err)
	}
	if !liked {
		t.Fatalf("expected liked=true after duplicate like")
	}
	if count != 1 {
		t.Fatalf("expected likes_count stay 1 after duplicate like, got %d", count)
	}

	liked, count, err = svc.UnlikeSkill(skill.ID, 10)
	if err != nil {
		t.Fatalf("first unlike failed: %v", err)
	}
	if liked {
		t.Fatalf("expected liked=false after unlike")
	}
	if count != 0 {
		t.Fatalf("expected likes_count=0 after unlike, got %d", count)
	}

	liked, count, err = svc.UnlikeSkill(skill.ID, 10)
	if err != nil {
		t.Fatalf("second unlike failed: %v", err)
	}
	if liked {
		t.Fatalf("expected liked=false after duplicate unlike")
	}
	if count != 0 {
		t.Fatalf("expected likes_count stay 0 after duplicate unlike, got %d", count)
	}

	hasLiked, err := svc.HasUserLiked(skill.ID, 10)
	if err != nil {
		t.Fatalf("has user liked failed: %v", err)
	}
	if hasLiked {
		t.Fatalf("expected hasLiked=false after unlike")
	}
}

func TestUnlikeSkill_DoesNotGoBelowZero(t *testing.T) {
	svc := newSkillServiceForLikesTest(t)
	skill := createLikeTestSkill(t, svc, "Skill Like B")

	liked, count, err := svc.UnlikeSkill(skill.ID, 99)
	if err != nil {
		t.Fatalf("unlike without like failed: %v", err)
	}
	if liked {
		t.Fatalf("expected liked=false when no like existed")
	}
	if count != 0 {
		t.Fatalf("expected likes_count=0 when no like existed, got %d", count)
	}

	var refreshed model.Skill
	if err := svc.db.First(&refreshed, skill.ID).Error; err != nil {
		t.Fatalf("read refreshed skill: %v", err)
	}
	if refreshed.LikesCount != 0 {
		t.Fatalf("expected stored likes_count stay 0, got %d", refreshed.LikesCount)
	}
}
