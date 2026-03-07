package handler

import (
	"testing"

	"skill-hub/internal/model"
)

func TestCanManageSkill_UserIDMatch(t *testing.T) {
	skill := &model.Skill{UserID: 12, Author: "alice"}

	if !canManageSkill(skill, 12, "alice") {
		t.Fatalf("expected owner with matching user id to manage skill")
	}
	if canManageSkill(skill, 13, "alice") {
		t.Fatalf("expected different user id to be rejected")
	}
}

func TestCanManageSkill_FallbackToAuthorWhenLegacyData(t *testing.T) {
	skill := &model.Skill{UserID: 0, Author: "Alice"}

	if !canManageSkill(skill, 99, "alice") {
		t.Fatalf("expected case-insensitive author match for legacy data")
	}
	if canManageSkill(skill, 99, "bob") {
		t.Fatalf("expected non-author to be rejected for legacy data")
	}
}

func TestCanManageSkill_RejectsMissingIdentity(t *testing.T) {
	skill := &model.Skill{UserID: 0, Author: "alice"}

	if canManageSkill(nil, 1, "alice") {
		t.Fatalf("expected nil skill to be rejected")
	}
	if canManageSkill(skill, 0, "alice") {
		t.Fatalf("expected missing user id to be rejected")
	}
}

func TestNormalizeSkillAuthor(t *testing.T) {
	if got := normalizeSkillAuthor("alice", "custom"); got != "alice" {
		t.Fatalf("expected authenticated username to win, got %q", got)
	}
	if got := normalizeSkillAuthor("", "  custom  "); got != "custom" {
		t.Fatalf("expected submitted author fallback, got %q", got)
	}
	if got := normalizeSkillAuthor(" ", " "); got != "Anonymous" {
		t.Fatalf("expected anonymous default, got %q", got)
	}
}

func TestCanReviewSkill_RejectOwner(t *testing.T) {
	skill := &model.Skill{
		UserID:            42,
		Author:            "alice",
		AIApproved:        true,
		HumanReviewStatus: model.HumanReviewStatusPending,
	}

	if canReviewSkill(skill, 42, "alice") {
		t.Fatalf("expected owner to be blocked from human review")
	}
}

func TestCanReviewSkill_RejectNonAIApproved(t *testing.T) {
	skill := &model.Skill{
		UserID:            1,
		Author:            "alice",
		AIApproved:        false,
		HumanReviewStatus: model.HumanReviewStatusPending,
	}

	if canReviewSkill(skill, 2, "bob") {
		t.Fatalf("expected non-AI-approved upload to be blocked from review")
	}
}

func TestCanReviewSkill_AllowDifferentReviewer(t *testing.T) {
	skill := &model.Skill{
		UserID:            10,
		Author:            "alice",
		AIApproved:        true,
		HumanReviewStatus: model.HumanReviewStatusPending,
	}

	if !canReviewSkill(skill, 11, "bob") {
		t.Fatalf("expected different authenticated user to review")
	}
}

func TestCanReviewSkill_RejectAlreadyApproved(t *testing.T) {
	skill := &model.Skill{
		UserID:            10,
		Author:            "alice",
		AIApproved:        true,
		HumanReviewStatus: model.HumanReviewStatusApproved,
	}

	if canReviewSkill(skill, 11, "bob") {
		t.Fatalf("expected finalized upload to reject extra reviews")
	}
}
