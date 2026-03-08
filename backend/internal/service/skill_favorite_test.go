package service

import (
	"testing"
	"time"

	"skill-hub/internal/model"
	"skill-hub/internal/testutil"
)

func newSkillServiceForFavoritesTest(t *testing.T) *SkillService {
	t.Helper()

	tdb := testutil.OpenPostgresTestDB(t)
	return NewSkillService(tdb.DB)
}

func createFavoriteTestSkill(t *testing.T, svc *SkillService, name string, status string) model.Skill {
	t.Helper()

	skill := model.Skill{
		Name:              name,
		AIApproved:        true,
		HumanReviewStatus: status,
		Published:         status == model.HumanReviewStatusApproved,
	}
	if err := svc.db.Create(&skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}
	return skill
}

func TestAddFavorite_IdempotentAndRemovable(t *testing.T) {
	svc := newSkillServiceForFavoritesTest(t)
	skill := createFavoriteTestSkill(t, svc, "Skill A", model.HumanReviewStatusApproved)

	if err := svc.AddFavorite(skill.ID, 9); err != nil {
		t.Fatalf("first add favorite failed: %v", err)
	}
	if err := svc.AddFavorite(skill.ID, 9); err != nil {
		t.Fatalf("second add favorite should be idempotent, got: %v", err)
	}

	var count int64
	if err := svc.db.Model(&model.SkillFavorite{}).
		Where("skill_id = ? AND user_id = ?", skill.ID, 9).
		Count(&count).Error; err != nil {
		t.Fatalf("count favorites: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one favorite record, got %d", count)
	}

	favorited, err := svc.HasUserFavorited(skill.ID, 9)
	if err != nil {
		t.Fatalf("has user favorited failed: %v", err)
	}
	if !favorited {
		t.Fatalf("expected skill to be favorited")
	}

	if err := svc.RemoveFavorite(skill.ID, 9); err != nil {
		t.Fatalf("remove favorite failed: %v", err)
	}
	favorited, err = svc.HasUserFavorited(skill.ID, 9)
	if err != nil {
		t.Fatalf("has user favorited after remove failed: %v", err)
	}
	if favorited {
		t.Fatalf("expected skill not favorited after remove")
	}
}

func TestGetUserFavorites_OrderedAndFiltered(t *testing.T) {
	svc := newSkillServiceForFavoritesTest(t)
	approved := createFavoriteTestSkill(t, svc, "Approved", model.HumanReviewStatusApproved)
	pending := createFavoriteTestSkill(t, svc, "Pending", model.HumanReviewStatusPending)
	rejected := createFavoriteTestSkill(t, svc, "Rejected", model.HumanReviewStatusRejected)

	if err := svc.AddFavorite(approved.ID, 11); err != nil {
		t.Fatalf("add approved favorite: %v", err)
	}
	time.Sleep(15 * time.Millisecond)
	if err := svc.AddFavorite(pending.ID, 11); err != nil {
		t.Fatalf("add pending favorite: %v", err)
	}
	if err := svc.AddFavorite(rejected.ID, 11); err != nil {
		t.Fatalf("add rejected favorite: %v", err)
	}

	result, err := svc.GetUserFavorites(11, "", 0)
	if err != nil {
		t.Fatalf("get user favorites: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 visible favorites (approved+pending), got %d", len(result))
	}
	if result[0].ID != pending.ID || result[1].ID != approved.ID {
		t.Fatalf("expected newest favorite first, got ids %d then %d", result[0].ID, result[1].ID)
	}
	if !result[0].Favorited || !result[1].Favorited {
		t.Fatalf("expected returned favorites to mark favorited=true")
	}
}
