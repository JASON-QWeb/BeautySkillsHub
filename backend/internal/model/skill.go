package model

import "time"

type Skill struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	Name             string    `json:"name" gorm:"not null;size:255"`
	Description      string    `json:"description" gorm:"type:text"`
	Category         string    `json:"category" gorm:"size:100"`
	ResourceType     string    `json:"resource_type" gorm:"size:50;default:skill"`
	Author           string    `json:"author" gorm:"size:100"`
	FileName         string    `json:"file_name" gorm:"size:255"`
	FilePath         string    `json:"-" gorm:"size:512"`
	FileSize         int64     `json:"file_size"`
	ThumbnailURL     string    `json:"thumbnail_url" gorm:"size:512"`
	Downloads        int       `json:"downloads" gorm:"default:0"`
	AIApproved       bool      `json:"ai_approved" gorm:"default:false"`
	AIFeedback       string    `json:"ai_feedback" gorm:"type:text"`
	GitHubPath       string    `json:"github_path" gorm:"size:1024"`
	GitHubURL        string    `json:"github_url" gorm:"size:1024"`
	GitHubSyncStatus string    `json:"github_sync_status" gorm:"size:32;default:disabled"`
	GitHubSyncError  string    `json:"github_sync_error" gorm:"type:text"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
