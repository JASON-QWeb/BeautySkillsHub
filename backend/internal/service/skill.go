package service

import (
	"errors"
	"skill-hub/internal/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SkillService struct {
	db *gorm.DB
}

func NewSkillService(db *gorm.DB) *SkillService {
	return &SkillService{db: db}
}

// ListSkills returns all AI-approved skills that are either pending human review or fully published.
func (s *SkillService) ListSkills(search, category, resourceType string, page, pageSize int) ([]model.Skill, int64, error) {
	var skills []model.Skill
	var total int64

	query := s.db.Model(&model.Skill{}).
		Where("ai_approved = ?", true).
		Where("human_review_status IN ?", []string{model.HumanReviewStatusPending, model.HumanReviewStatusApproved})

	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR description LIKE ? OR category LIKE ? OR tags LIKE ?", like, like, like, like)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&skills).Error; err != nil {
		return nil, 0, err
	}

	return skills, total, nil
}

// GetCategories returns all distinct categories for a given resource type.
func (s *SkillService) GetCategories(resourceType string) ([]string, error) {
	var categories []string
	query := s.db.Model(&model.Skill{}).
		Where("ai_approved = ? AND category != ''", true).
		Where("human_review_status IN ?", []string{model.HumanReviewStatusPending, model.HumanReviewStatusApproved})
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if err := query.Distinct().Pluck("category", &categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

// GetSkill returns a single skill by ID.
func (s *SkillService) GetSkill(id uint) (*model.Skill, error) {
	var skill model.Skill
	if err := s.db.First(&skill, id).Error; err != nil {
		return nil, err
	}
	return &skill, nil
}

// CreateSkill creates a new skill record.
func (s *SkillService) CreateSkill(skill *model.Skill) error {
	return s.db.Create(skill).Error
}

// UpdateSkill updates an existing skill.
func (s *SkillService) UpdateSkill(skill *model.Skill) error {
	return s.db.Save(skill).Error
}

// DeleteSkill deletes a skill record by ID.
func (s *SkillService) DeleteSkill(id uint) error {
	return s.db.Delete(&model.Skill{}, id).Error
}

// IncrementDownload increments the download count for a skill.
func (s *SkillService) IncrementDownload(id uint) error {
	return s.db.Model(&model.Skill{}).Where("id = ?", id).
		UpdateColumn("downloads", gorm.Expr("downloads + 1")).Error
}

// LikeSkill creates one like per (skill,user) and returns the current likes count.
func (s *SkillService) LikeSkill(skillID, userID uint) (bool, int, error) {
	if skillID == 0 || userID == 0 {
		return false, 0, errors.New("invalid skill id or user id")
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return false, 0, tx.Error
	}

	like := model.SkillLike{SkillID: skillID, UserID: userID}
	createResult := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "skill_id"}, {Name: "user_id"}},
		DoNothing: true,
	}).Create(&like)
	if createResult.Error != nil {
		tx.Rollback()
		return false, 0, createResult.Error
	}

	inserted := createResult.RowsAffected > 0
	if inserted {
		if err := tx.Model(&model.Skill{}).Where("id = ?", skillID).
			UpdateColumn("likes_count", gorm.Expr("likes_count + 1")).Error; err != nil {
			tx.Rollback()
			return false, 0, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return false, 0, err
	}

	count, err := s.GetLikesCount(skillID)
	if err != nil {
		return false, 0, err
	}
	return true, count, nil
}

func (s *SkillService) GetLikesCount(skillID uint) (int, error) {
	var skill model.Skill
	if err := s.db.Select("likes_count").First(&skill, skillID).Error; err != nil {
		return 0, err
	}
	return skill.LikesCount, nil
}

func (s *SkillService) HasUserLiked(skillID, userID uint) (bool, error) {
	if skillID == 0 || userID == 0 {
		return false, nil
	}

	var count int64
	if err := s.db.Model(&model.SkillLike{}).
		Where("skill_id = ? AND user_id = ?", skillID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SkillService) AddFavorite(skillID, userID uint) error {
	if skillID == 0 || userID == 0 {
		return errors.New("invalid skill id or user id")
	}

	favorite := model.SkillFavorite{SkillID: skillID, UserID: userID}
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "skill_id"}, {Name: "user_id"}},
		DoNothing: true,
	}).Create(&favorite).Error
}

func (s *SkillService) RemoveFavorite(skillID, userID uint) error {
	if skillID == 0 || userID == 0 {
		return errors.New("invalid skill id or user id")
	}
	return s.db.Where("skill_id = ? AND user_id = ?", skillID, userID).Delete(&model.SkillFavorite{}).Error
}

func (s *SkillService) HasUserFavorited(skillID, userID uint) (bool, error) {
	if skillID == 0 || userID == 0 {
		return false, nil
	}

	var count int64
	if err := s.db.Model(&model.SkillFavorite{}).
		Where("skill_id = ? AND user_id = ?", skillID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// BatchGetUserLikedMap returns a set of skill IDs that the user has liked.
func (s *SkillService) BatchGetUserLikedMap(skillIDs []uint, userID uint) (map[uint]bool, error) {
	result := make(map[uint]bool, len(skillIDs))
	if len(skillIDs) == 0 || userID == 0 {
		return result, nil
	}
	var likedIDs []uint
	if err := s.db.Model(&model.SkillLike{}).
		Where("user_id = ? AND skill_id IN ?", userID, skillIDs).
		Pluck("skill_id", &likedIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range likedIDs {
		result[id] = true
	}
	return result, nil
}

// BatchGetUserFavoritedMap returns a set of skill IDs that the user has favorited.
func (s *SkillService) BatchGetUserFavoritedMap(skillIDs []uint, userID uint) (map[uint]bool, error) {
	result := make(map[uint]bool, len(skillIDs))
	if len(skillIDs) == 0 || userID == 0 {
		return result, nil
	}
	var favIDs []uint
	if err := s.db.Model(&model.SkillFavorite{}).
		Where("user_id = ? AND skill_id IN ?", userID, skillIDs).
		Pluck("skill_id", &favIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range favIDs {
		result[id] = true
	}
	return result, nil
}

func (s *SkillService) GetUserFavorites(userID uint, resourceType string, limit int) ([]model.Skill, error) {
	if userID == 0 {
		return []model.Skill{}, nil
	}

	query := s.db.Model(&model.Skill{}).
		Joins("JOIN skill_favorites ON skill_favorites.skill_id = skills.id").
		Where("skill_favorites.user_id = ?", userID).
		Where("skills.ai_approved = ?", true).
		Where("skills.human_review_status IN ?", []string{model.HumanReviewStatusPending, model.HumanReviewStatusApproved})

	if resourceType != "" {
		query = query.Where("skills.resource_type = ?", resourceType)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	var skills []model.Skill
	if err := query.Order("skill_favorites.created_at DESC").Find(&skills).Error; err != nil {
		return nil, err
	}
	for i := range skills {
		skills[i].Favorited = true
	}
	return skills, nil
}

// GetTrending returns the top N most downloaded approved skills, optionally filtered by resource type.
func (s *SkillService) GetTrending(limit int, resourceType string) ([]model.Skill, error) {
	var skills []model.Skill
	query := s.db.Where("ai_approved = ?", true).
		Where("human_review_status IN ?", []string{model.HumanReviewStatusPending, model.HumanReviewStatusApproved})
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if err := query.Order("downloads DESC").Limit(limit).Find(&skills).Error; err != nil {
		return nil, err
	}
	return skills, nil
}

// GetAllApprovedBrief returns all published skill names and descriptions (for AI context).
func (s *SkillService) GetAllApprovedBrief() ([]map[string]interface{}, error) {
	var skills []model.Skill
	if err := s.db.Select("id, name, description, category, resource_type, downloads").
		Where("published = ?", true).
		Find(&skills).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(skills))
	for i, sk := range skills {
		result[i] = map[string]interface{}{
			"id":            sk.ID,
			"name":          sk.Name,
			"description":   sk.Description,
			"category":      sk.Category,
			"resource_type": sk.ResourceType,
			"downloads":     sk.Downloads,
		}
	}
	return result, nil
}

// GetResourceSummary returns total visible resources and yesterday's new count.
func (s *SkillService) GetResourceSummary(resourceType string) (int64, int64, error) {
	base := s.db.Model(&model.Skill{}).
		Where("ai_approved = ?", true).
		Where("human_review_status IN ?", []string{model.HumanReviewStatusPending, model.HumanReviewStatusApproved})
	if resourceType != "" {
		base = base.Where("resource_type = ?", resourceType)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return 0, 0, err
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterdayStart := todayStart.Add(-24 * time.Hour)

	var yesterdayNew int64
	if err := base.Session(&gorm.Session{}).
		Where("created_at >= ? AND created_at < ?", yesterdayStart, todayStart).
		Count(&yesterdayNew).Error; err != nil {
		return 0, 0, err
	}

	return total, yesterdayNew, nil
}
