package model

import "time"

type SkillLike struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	SkillID   uint      `json:"skill_id" gorm:"not null;index:idx_skill_user_like,unique"`
	UserID    uint      `json:"user_id" gorm:"not null;index:idx_skill_user_like,unique"`
	CreatedAt time.Time `json:"created_at"`
}
