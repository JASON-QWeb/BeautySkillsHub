package service

import (
	"sort"
	"strings"
	"time"

	"skill-hub/internal/model"

	"gorm.io/gorm"
)

type UserUploadStats struct {
	TotalItems     int64 `json:"total_items"`
	TotalDownloads int64 `json:"total_downloads"`
	TotalLikes     int64 `json:"total_likes"`
}

type UserProfileActivity struct {
	Kind         string    `json:"kind"`
	Target       string    `json:"target"`
	ResourceType string    `json:"resource_type"`
	OccurredAt   time.Time `json:"occurred_at"`
}

func (s *SkillService) userOwnedSkillsQuery(userID uint, username string) *gorm.DB {
	query := s.db.Model(&model.Skill{})

	trimmedUsername := strings.TrimSpace(strings.ToLower(username))
	switch {
	case userID != 0 && trimmedUsername != "":
		return query.Where("user_id = ? OR (user_id = 0 AND LOWER(author) = ?)", userID, trimmedUsername)
	case userID != 0:
		return query.Where("user_id = ?", userID)
	case trimmedUsername != "":
		return query.Where("LOWER(author) = ?", trimmedUsername)
	default:
		return query.Where("1 = 0")
	}
}

func (s *SkillService) GetUserUploads(userID uint, username, search, resourceType string, page, pageSize int) ([]model.Skill, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	query := s.userOwnedSkillsQuery(userID, username)
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR description LIKE ? OR tags LIKE ?", like, like, like)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var skills []model.Skill
	offset := (page - 1) * pageSize
	if err := query.Order("downloads DESC").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&skills).Error; err != nil {
		return nil, 0, err
	}
	return skills, total, nil
}

func (s *SkillService) GetUserUploadStats(userID uint, username, resourceType string) (UserUploadStats, error) {
	query := s.userOwnedSkillsQuery(userID, username)
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}

	var stats UserUploadStats
	if err := query.Select(
		"COUNT(*) AS total_items, COALESCE(SUM(downloads), 0) AS total_downloads, COALESCE(SUM(likes_count), 0) AS total_likes",
	).Scan(&stats).Error; err != nil {
		return UserUploadStats{}, err
	}
	return stats, nil
}

func (s *SkillService) GetUserTopTags(userID uint, username, resourceType string, limit int) ([]string, error) {
	if limit <= 0 {
		return []string{}, nil
	}

	query := s.userOwnedSkillsQuery(userID, username)
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}

	var rawTags []string
	if err := query.Pluck("tags", &rawTags).Error; err != nil {
		return nil, err
	}

	type tagStat struct {
		Label string
		Count int
		Index int
	}

	stats := make(map[string]*tagStat)
	nextIndex := 0
	for _, raw := range rawTags {
		for _, part := range strings.Split(raw, ",") {
			tag := strings.TrimSpace(strings.ToLower(part))
			if tag == "" {
				continue
			}
			if existing, ok := stats[tag]; ok {
				existing.Count++
				continue
			}
			stats[tag] = &tagStat{Label: tag, Count: 1, Index: nextIndex}
			nextIndex++
		}
	}

	ordered := make([]tagStat, 0, len(stats))
	for _, stat := range stats {
		ordered = append(ordered, *stat)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Count != ordered[j].Count {
			return ordered[i].Count > ordered[j].Count
		}
		return ordered[i].Index < ordered[j].Index
	})

	if len(ordered) > limit {
		ordered = ordered[:limit]
	}
	result := make([]string, len(ordered))
	for i, stat := range ordered {
		result[i] = stat.Label
	}
	return result, nil
}

func (s *SkillService) GetUserRecentActivities(userID uint, username string, limit int) ([]UserProfileActivity, error) {
	if limit <= 0 {
		return []UserProfileActivity{}, nil
	}

	var activities []UserProfileActivity

	var uploads []model.Skill
	if err := s.userOwnedSkillsQuery(userID, username).
		Select("name, resource_type, created_at").
		Order("created_at DESC").
		Limit(limit).
		Find(&uploads).Error; err != nil {
		return nil, err
	}
	for _, upload := range uploads {
		activities = append(activities, UserProfileActivity{
			Kind:         "published",
			Target:       upload.Name,
			ResourceType: upload.ResourceType,
			OccurredAt:   upload.CreatedAt,
		})
	}

	if userID != 0 {
		var reviewed []model.Skill
		if err := s.db.Model(&model.Skill{}).
			Where("human_reviewer_id = ? AND human_reviewed_at IS NOT NULL", userID).
			Order("human_reviewed_at DESC").
			Limit(limit).
			Find(&reviewed).Error; err != nil {
			return nil, err
		}
		for _, item := range reviewed {
			kind := "reviewed"
			if item.HumanReviewStatus == model.HumanReviewStatusApproved {
				kind = "approved"
			}
			occurredAt := item.UpdatedAt
			if item.HumanReviewedAt != nil {
				occurredAt = *item.HumanReviewedAt
			}
			activities = append(activities, UserProfileActivity{
				Kind:         kind,
				Target:       item.Name,
				ResourceType: item.ResourceType,
				OccurredAt:   occurredAt,
			})
		}
	}

	sort.Slice(activities, func(i, j int) bool {
		return activities[i].OccurredAt.After(activities[j].OccurredAt)
	})
	if len(activities) > limit {
		activities = activities[:limit]
	}
	return activities, nil
}
