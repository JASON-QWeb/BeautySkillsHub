package model

import "time"

type SkillFavorite struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	SkillID   uint      `json:"skill_id" gorm:"not null;index:idx_skill_user_favorite,unique"`
	UserID    uint      `json:"user_id" gorm:"not null;index:idx_skill_user_favorite,unique"`
	CreatedAt time.Time `json:"created_at"`
}

