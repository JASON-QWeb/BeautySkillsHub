package service

import (
	"skill-hub/internal/model"

	"gorm.io/gorm"
)

type SkillService struct {
	db *gorm.DB
}

func NewSkillService(db *gorm.DB) *SkillService {
	return &SkillService{db: db}
}

// ListSkills returns all approved skills, with optional search, category, resource_type and pagination.
func (s *SkillService) ListSkills(search, category, resourceType string, page, pageSize int) ([]model.Skill, int64, error) {
	var skills []model.Skill
	var total int64

	query := s.db.Model(&model.Skill{}).Where("ai_approved = ?", true)

	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR description LIKE ? OR category LIKE ?", like, like, like)
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
	query := s.db.Model(&model.Skill{}).Where("ai_approved = ? AND category != ''", true)
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

// GetTrending returns the top N most downloaded approved skills, optionally filtered by resource type.
func (s *SkillService) GetTrending(limit int, resourceType string) ([]model.Skill, error) {
	var skills []model.Skill
	query := s.db.Where("ai_approved = ?", true)
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if err := query.Order("downloads DESC").Limit(limit).Find(&skills).Error; err != nil {
		return nil, err
	}
	return skills, nil
}

// GetAllApprovedBrief returns all approved skill names and descriptions (for AI context).
func (s *SkillService) GetAllApprovedBrief() ([]map[string]interface{}, error) {
	var skills []model.Skill
	if err := s.db.Select("id, name, description, category, resource_type, downloads").
		Where("ai_approved = ?", true).
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
