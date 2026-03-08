package service

import (
	"testing"

	"gorm.io/gorm"
)

func TestLikeSkill_UsesTransactionScopedCountReader(t *testing.T) {
	svc := newSkillServiceForLikesTest(t)
	skill := createLikeTestSkill(t, svc, "Skill Like Tx Count")
	user := createLikeTestUser(t, svc, 10)

	original := loadLikesCountTx
	t.Cleanup(func() {
		loadLikesCountTx = original
	})

	var seenTx bool
	loadLikesCountTx = func(tx *gorm.DB, skillID uint) (int, error) {
		if tx != nil {
			seenTx = true
		}
		if skillID != skill.ID {
			t.Fatalf("expected skill id %d, got %d", skill.ID, skillID)
		}
		return 41, nil
	}

	liked, count, err := svc.LikeSkill(skill.ID, user.ID)
	if err != nil {
		t.Fatalf("like failed: %v", err)
	}
	if !liked {
		t.Fatal("expected liked=true")
	}
	if !seenTx {
		t.Fatal("expected transaction-scoped count reader to be used")
	}
	if count != 41 {
		t.Fatalf("expected injected transactional count, got %d", count)
	}
}
