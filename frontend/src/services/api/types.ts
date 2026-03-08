export interface Skill {
    id: number
    user_id?: number
    name: string
    description: string
    category: string
    tags?: string
    resource_type: string
    author: string
    file_name: string
    file_size: number
    thumbnail_url: string
    downloads: number
    likes_count?: number
    user_liked?: boolean
    favorited?: boolean
    ai_approved: boolean
    ai_review_status?: 'queued' | 'running' | 'passed' | 'failed_retryable' | 'failed_terminal'
    ai_review_phase?: 'queued' | 'security' | 'functional' | 'finalizing' | 'done'
    ai_review_attempts?: number
    ai_review_max_attempts?: number
    ai_review_started_at?: string
    ai_review_completed_at?: string
    ai_review_details?: string
    ai_feedback: string
    ai_description: string
    human_review_status?: 'pending' | 'approved' | 'rejected'
    human_reviewer_id?: number
    human_reviewer?: string
    human_review_feedback?: string
    human_reviewed_at?: string
    has_pending_revision?: boolean
    pending_revision_id?: number
    pending_revision_ai?: 'queued' | 'running' | 'passed' | 'failed_retryable' | 'failed_terminal'
    pending_revision_human?: 'pending' | 'approved' | 'rejected'
    pending_revision_updated_at?: string
    published?: boolean
    github_path?: string
    github_files?: string
    github_url?: string
    created_at: string
    updated_at: string
}

export interface SkillListResponse {
    skills: Skill[]
    total: number
    page: number
    page_size: number
}

export interface ProfileActivity {
    kind: 'published' | 'reviewed' | 'approved'
    target: string
    resource_type: string
    occurred_at: string
}

export interface ProfileUploadsStats {
    total_items: number
    total_downloads: number
    total_likes: number
}

export interface ProfileUploadsResponse extends SkillListResponse {
    stats: ProfileUploadsStats
    top_tags: string[]
    activities: ProfileActivity[]
}

export interface UploadResponse {
    skill: Skill
    approved: boolean
    feedback: string
}

export interface SkillSummaryResponse {
    total: number
    yesterday_new: number
}

export interface SkillInstallConfigResponse {
    github_repo: string
    github_base_dir: string
}

export interface SkillReviewStatusResponse {
    status: 'queued' | 'running' | 'passed' | 'failed_retryable' | 'failed_terminal'
    phase: 'queued' | 'security' | 'functional' | 'finalizing' | 'done'
    attempts: number
    max_attempts: number
    retry_remaining: number
    can_retry: boolean
    approved: boolean
    feedback: string
    progress?: SkillReviewProgress
}

export interface SkillReviewProgress {
    total_files: number
    completed_files: number
    current_file?: string
    files: SkillReviewFileProgress[]
}

export interface SkillReviewFileProgress {
    path: string
    kind: string
    status: 'queued' | 'running' | 'passed' | 'failed'
    message?: string
}
