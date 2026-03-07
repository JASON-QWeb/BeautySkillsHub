package model

import "time"

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"uniqueIndex;not null;size:50"`
	Password  string    `json:"-" gorm:"not null"`
	AvatarURL string    `json:"avatar_url" gorm:"size:512"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
