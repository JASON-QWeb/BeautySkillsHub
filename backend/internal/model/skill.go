package model

import "time"

type Skill struct {
	ID                  uint       `json:"id" gorm:"primaryKey"`
	UserID              uint       `json:"user_id" gorm:"index"`
	Name                string     `json:"name" gorm:"not null;size:255"`
	Description         string     `json:"description" gorm:"type:text"`
	Category            string     `json:"category" gorm:"size:100"`
	Tags                string     `json:"tags" gorm:"type:text"`
	ResourceType        string     `json:"resource_type" gorm:"size:50;default:skill;index"`
	Author              string     `json:"author" gorm:"size:100"`
	FileName            string     `json:"file_name" gorm:"size:255"`
	FilePath            string     `json:"-" gorm:"size:512"`
	FileSize            int64      `json:"file_size"`
	ThumbnailURL        string     `json:"thumbnail_url" gorm:"size:512"`
	Downloads           int        `json:"downloads" gorm:"default:0"`
	LikesCount          int        `json:"likes_count" gorm:"default:0"`
	UserLiked           bool       `json:"user_liked" gorm:"-"`
	Favorited           bool       `json:"favorited" gorm:"-"`
	AIApproved          bool       `json:"ai_approved" gorm:"default:false"`
	AIReviewStatus      string     `json:"ai_review_status" gorm:"size:32;default:queued;index"`
	AIReviewPhase       string     `json:"ai_review_phase" gorm:"size:32;default:queued"`
	AIReviewAttempts    int        `json:"ai_review_attempts" gorm:"default:0"`
	AIReviewMaxAttempts int        `json:"ai_review_max_attempts" gorm:"default:3"`
	AIReviewStartedAt   *time.Time `json:"ai_review_started_at"`
	AIReviewCompletedAt *time.Time `json:"ai_review_completed_at"`
	AIReviewDetails     string     `json:"ai_review_details" gorm:"type:text"`
	AIFeedback          string     `json:"ai_feedback" gorm:"type:text"`
	AIDescription       string     `json:"ai_description" gorm:"column:ai_description;type:text"`
	HumanReviewStatus   string     `json:"human_review_status" gorm:"size:32;default:pending;index"`
	HumanReviewerID     *uint      `json:"human_reviewer_id" gorm:"index"`
	HumanReviewer       string     `json:"human_reviewer" gorm:"size:100"`
	HumanReviewFeedback string     `json:"human_review_feedback" gorm:"type:text"`
	HumanReviewedAt     *time.Time `json:"human_reviewed_at"`
	Published           bool       `json:"published" gorm:"default:false;index"`
	GitHubPath          string     `json:"github_path" gorm:"column:github_path;size:1024"`
	GitHubURL           string     `json:"github_url" gorm:"column:github_url;size:1024"`
	GitHubFiles         string     `json:"github_files" gorm:"column:github_files;type:text"`
	GitHubSyncStatus    string     `json:"github_sync_status" gorm:"column:github_sync_status;size:32;default:disabled"`
	GitHubSyncError     string     `json:"github_sync_error" gorm:"column:github_sync_error;type:text"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

const (
	HumanReviewStatusPending  = "pending"
	HumanReviewStatusApproved = "approved"
	HumanReviewStatusRejected = "rejected"
)

const (
	AIReviewStatusQueued         = "queued"
	AIReviewStatusRunning        = "running"
	AIReviewStatusPassed         = "passed"
	AIReviewStatusFailedRetry    = "failed_retryable"
	AIReviewStatusFailedTerminal = "failed_terminal"
)

const (
	AIReviewPhaseQueued     = "queued"
	AIReviewPhaseSecurity   = "security"
	AIReviewPhaseFunctional = "functional"
	AIReviewPhaseFinalizing = "finalizing"
	AIReviewPhaseDone       = "done"
)
