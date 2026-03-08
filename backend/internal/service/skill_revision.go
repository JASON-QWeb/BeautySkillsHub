package service

import (
	"errors"
	"strings"

	"skill-hub/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrActiveRevisionExists = errors.New("active revision already exists")

func (s *SkillService) CreatePendingRevision(skill *model.Skill, revision *model.SkillRevision) (*model.SkillRevision, error) {
	if skill == nil || skill.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if revision == nil {
		return nil, errors.New("revision is nil")
	}

	copyValue := *revision
	copyValue.SkillID = skill.ID
	if copyValue.UserID == 0 {
		copyValue.UserID = skill.UserID
	}
	if strings.TrimSpace(copyValue.ResourceType) == "" {
		copyValue.ResourceType = skill.ResourceType
	}
	if strings.TrimSpace(copyValue.Author) == "" {
		copyValue.Author = skill.Author
	}
	if strings.TrimSpace(copyValue.Status) == "" {
		copyValue.Status = model.SkillRevisionStatusPending
	}
	if copyValue.AIReviewMaxAttempts <= 0 {
		copyValue.AIReviewMaxAttempts = 3
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var existing model.SkillRevision
		if err := tx.Where("skill_id = ? AND status = ?", skill.ID, model.SkillRevisionStatusPending).
			First(&existing).Error; err == nil {
			return ErrActiveRevisionExists
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Create(&copyValue).Error; err != nil {
			if isPendingRevisionConflict(err) {
				return ErrActiveRevisionExists
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &copyValue, nil
}

func (s *SkillService) GetActiveRevision(skillID uint) (*model.SkillRevision, error) {
	var revision model.SkillRevision
	if err := s.db.Where("skill_id = ? AND status = ?", skillID, model.SkillRevisionStatusPending).
		Order("created_at DESC").
		First(&revision).Error; err != nil {
		return nil, err
	}
	return &revision, nil
}

func (s *SkillService) GetSkillRevision(revisionID uint) (*model.SkillRevision, error) {
	var revision model.SkillRevision
	if err := s.db.First(&revision, revisionID).Error; err != nil {
		return nil, err
	}
	return &revision, nil
}

func (s *SkillService) UpdateSkillRevision(revision *model.SkillRevision) error {
	return s.db.Save(revision).Error
}

func (s *SkillService) ApplyApprovedRevision(revisionID uint) (*model.Skill, error) {
	var updated model.Skill

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var revision model.SkillRevision
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&revision, revisionID).Error; err != nil {
			return err
		}
		if revision.Status != model.SkillRevisionStatusPending {
			return errors.New("revision is not pending")
		}

		var skill model.Skill
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&skill, revision.SkillID).Error; err != nil {
			return err
		}

		applyRevisionToSkill(&skill, &revision)
		skill.Published = true
		if err := tx.Save(&skill).Error; err != nil {
			return err
		}

		revision.Status = model.SkillRevisionStatusApplied
		if err := tx.Save(&revision).Error; err != nil {
			return err
		}

		updated = skill
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := s.attachPendingRevisionSummary(&updated); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &updated, nil
}

func BuildRevisionFromSkill(skill *model.Skill) *model.SkillRevision {
	if skill == nil {
		return nil
	}

	return &model.SkillRevision{
		SkillID:             skill.ID,
		UserID:              skill.UserID,
		Name:                skill.Name,
		Description:         skill.Description,
		Category:            skill.Category,
		Tags:                skill.Tags,
		ResourceType:        skill.ResourceType,
		Author:              skill.Author,
		FileName:            skill.FileName,
		FilePath:            skill.FilePath,
		FileSize:            skill.FileSize,
		ThumbnailURL:        skill.ThumbnailURL,
		AIApproved:          skill.AIApproved,
		AIReviewStatus:      skill.AIReviewStatus,
		AIReviewPhase:       skill.AIReviewPhase,
		AIReviewAttempts:    skill.AIReviewAttempts,
		AIReviewMaxAttempts: skill.AIReviewMaxAttempts,
		AIReviewStartedAt:   skill.AIReviewStartedAt,
		AIReviewCompletedAt: skill.AIReviewCompletedAt,
		AIReviewDetails:     skill.AIReviewDetails,
		AIFeedback:          skill.AIFeedback,
		AIDescription:       skill.AIDescription,
		HumanReviewStatus:   skill.HumanReviewStatus,
		HumanReviewerID:     skill.HumanReviewerID,
		HumanReviewer:       skill.HumanReviewer,
		HumanReviewFeedback: skill.HumanReviewFeedback,
		HumanReviewedAt:     skill.HumanReviewedAt,
		GitHubPath:          skill.GitHubPath,
		GitHubURL:           skill.GitHubURL,
		GitHubFiles:         skill.GitHubFiles,
		GitHubSyncStatus:    skill.GitHubSyncStatus,
		GitHubSyncError:     skill.GitHubSyncError,
		Status:              model.SkillRevisionStatusPending,
	}
}

func BuildSkillReviewView(skill *model.Skill, revision *model.SkillRevision) *model.Skill {
	if skill == nil {
		return nil
	}
	if revision == nil {
		copyValue := *skill
		return &copyValue
	}

	copyValue := *skill
	applyRevisionToSkill(&copyValue, revision)
	copyValue.ID = skill.ID
	copyValue.Downloads = skill.Downloads
	copyValue.LikesCount = skill.LikesCount
	copyValue.UserLiked = skill.UserLiked
	copyValue.Favorited = skill.Favorited
	copyValue.Published = false
	copyValue.HasPendingRevision = true
	copyValue.PendingRevisionID = &revision.ID
	copyValue.PendingRevisionAI = revision.AIReviewStatus
	copyValue.PendingRevisionHuman = revision.HumanReviewStatus
	copyValue.PendingRevisionUpdatedAt = &revision.UpdatedAt
	return &copyValue
}

func (s *SkillService) attachPendingRevisionSummary(skill *model.Skill) error {
	if skill == nil || skill.ID == 0 {
		return nil
	}

	revision, err := s.GetActiveRevision(skill.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		skill.HasPendingRevision = false
		skill.PendingRevisionID = nil
		skill.PendingRevisionAI = ""
		skill.PendingRevisionHuman = ""
		skill.PendingRevisionUpdatedAt = nil
		return nil
	}
	if err != nil {
		return err
	}

	skill.HasPendingRevision = true
	skill.PendingRevisionID = &revision.ID
	skill.PendingRevisionAI = revision.AIReviewStatus
	skill.PendingRevisionHuman = revision.HumanReviewStatus
	skill.PendingRevisionUpdatedAt = &revision.UpdatedAt
	return nil
}

func isPendingRevisionConflict(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "idx_skill_revisions_one_pending_per_skill") ||
		strings.Contains(lower, "duplicate key")
}

func applyRevisionToSkill(skill *model.Skill, revision *model.SkillRevision) {
	skill.UserID = revision.UserID
	skill.Name = revision.Name
	skill.Description = revision.Description
	skill.Category = revision.Category
	skill.Tags = revision.Tags
	skill.ResourceType = revision.ResourceType
	skill.Author = revision.Author
	skill.FileName = revision.FileName
	skill.FilePath = revision.FilePath
	skill.FileSize = revision.FileSize
	skill.ThumbnailURL = revision.ThumbnailURL
	skill.AIApproved = revision.AIApproved
	skill.AIReviewStatus = revision.AIReviewStatus
	skill.AIReviewPhase = revision.AIReviewPhase
	skill.AIReviewAttempts = revision.AIReviewAttempts
	skill.AIReviewMaxAttempts = revision.AIReviewMaxAttempts
	skill.AIReviewStartedAt = revision.AIReviewStartedAt
	skill.AIReviewCompletedAt = revision.AIReviewCompletedAt
	skill.AIReviewDetails = revision.AIReviewDetails
	skill.AIFeedback = revision.AIFeedback
	skill.AIDescription = revision.AIDescription
	skill.HumanReviewStatus = revision.HumanReviewStatus
	skill.HumanReviewerID = revision.HumanReviewerID
	skill.HumanReviewer = revision.HumanReviewer
	skill.HumanReviewFeedback = revision.HumanReviewFeedback
	skill.HumanReviewedAt = revision.HumanReviewedAt
	skill.GitHubPath = revision.GitHubPath
	skill.GitHubURL = revision.GitHubURL
	skill.GitHubFiles = revision.GitHubFiles
	skill.GitHubSyncStatus = revision.GitHubSyncStatus
	skill.GitHubSyncError = revision.GitHubSyncError
}
