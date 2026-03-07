import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import { useDialog } from '../contexts/DialogContext'
import { useI18n } from '../i18n/I18nProvider'
import { Skill, favoriteSkill, likeSkill, unfavoriteSkill, unlikeSkill } from '../services/api'

interface Props {
    skill: Skill
    onFavoriteChange?: (skillId: number, favorited: boolean) => void
}

const THUMB_GRADIENTS = [
    'linear-gradient(135deg, #4f83e8 0%, #5d63dc 100%)',
    'linear-gradient(135deg, #36c88f 0%, #0f9467 100%)',
    'linear-gradient(135deg, #38bdf8 0%, #3b82f6 100%)',
    'linear-gradient(135deg, #374151 0%, #1f2937 100%)',
    'linear-gradient(135deg, #f97316 0%, #ef4444 100%)',
]

function formatDownload(num: number) {
    if (num >= 1000) {
        return `${(num / 1000).toFixed(1)}k`
    }
    return `${num}`
}

function normalizeType(resourceType: string) {
    if (!resourceType) return 'SKILLS'
    return resourceType.toUpperCase()
}

function DownloadIcon() {
    return (
        <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <path d="M12 5v8" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
            <path d="m8.7 10.6 3.3 3.4 3.3-3.4" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
            <rect x="5" y="16" width="14" height="3.5" rx="1.4" stroke="currentColor" strokeWidth="1.8" />
        </svg>
    )
}

function HeartIcon({ filled }: { filled: boolean }) {
    if (filled) {
        return (
            <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M12 20.2 4.9 13.8a4.7 4.7 0 0 1 6.6-6.7L12 7.6l.5-.5a4.7 4.7 0 1 1 6.6 6.7L12 20.2Z" />
            </svg>
        )
    }
    return (
        <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <path d="M12 20.2 4.9 13.8a4.7 4.7 0 0 1 6.6-6.7L12 7.6l.5-.5a4.7 4.7 0 1 1 6.6 6.7L12 20.2Z" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" />
        </svg>
    )
}

function StarIcon({ filled }: { filled: boolean }) {
    if (filled) {
        return (
            <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="m12 3.8 2.5 5 5.5.8-4 3.9.9 5.5L12 16.3 7.1 19l1-5.5-4-3.9 5.4-.8L12 3.8Z" />
            </svg>
        )
    }
    return (
        <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <path d="m12 3.8 2.5 5 5.5.8-4 3.9.9 5.5L12 16.3 7.1 19l1-5.5-4-3.9 5.4-.8L12 3.8Z" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" />
        </svg>
    )
}

function EditIcon() {
    return (
        <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <path d="m15.6 5.2 3.2 3.2-8.2 8.2-3.9.6.6-3.9 8.3-8.1Z" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" />
            <path d="m13.7 7.1 3.2 3.2" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
        </svg>
    )
}

function SkillCard({ skill, onFavoriteChange }: Props) {
    const navigate = useNavigate()
    const { t } = useI18n()
    const { user } = useAuth()
    const { showAlert } = useDialog()

    const isPublished = skill.published ?? (skill.human_review_status
        ? skill.human_review_status === 'approved'
        : skill.ai_approved)
    const isPendingReview = skill.ai_approved && !isPublished
    const isPendingReviewedResource = isPendingReview && (skill.resource_type === 'skill' || skill.resource_type === 'rules')
    const [likesCount, setLikesCount] = useState(skill.likes_count || 0)
    const [userLiked, setUserLiked] = useState(!!skill.user_liked)
    const [liking, setLiking] = useState(false)
    const [favorited, setFavorited] = useState(!!skill.favorited)
    const [favoriting, setFavoriting] = useState(false)

    const fallbackGradient = useMemo(() => {
        return THUMB_GRADIENTS[skill.id % THUMB_GRADIENTS.length]
    }, [skill.id])

    useEffect(() => {
        setLikesCount(skill.likes_count || 0)
        setUserLiked(!!skill.user_liked)
        setFavorited(!!skill.favorited)
    }, [skill.id, skill.likes_count, skill.user_liked, skill.favorited])

    const canFavorite = !!user && skill.ai_approved
    const canLike = !!user && skill.ai_approved
    const canOpenUploadEditor = !!user
        && (skill.resource_type === 'mcp' || skill.resource_type === 'tools')
        && ((skill.user_id ? skill.user_id === user.id : false)
            || (skill.author || '').trim().toLowerCase() === user.username.trim().toLowerCase())

    return (
        <article
            className={`skill-card glass-card ${isPendingReview ? 'pending-review' : ''}`}
            onClick={(e) => {
                const target = e.target as HTMLElement
                if (target.closest('button')) return
                if (isPendingReviewedResource) {
                    navigate(`/review/${skill.id}`, { state: { resourceType: skill.resource_type } })
                } else {
                    navigate(`/resource/${skill.resource_type || 'skill'}/${skill.id}`, { state: { resourceType: skill.resource_type } })
                }
            }}
            id={`skill-card-${skill.id}`}
        >
            {skill.thumbnail_url ? (
                <img className="skill-card-thumb" src={skill.thumbnail_url} alt={skill.name} />
            ) : (
                <div className="skill-card-thumb skill-card-thumb-fallback" style={{ background: fallbackGradient }}>
                    <div className="skill-card-fallback-content">
                        <span className="skill-card-fallback-title">{skill.name}</span>
                        {skill.description && (
                            <span className="skill-card-fallback-desc">
                                {skill.description.length > 50 ? skill.description.slice(0, 47) + '...' : skill.description}
                            </span>
                        )}
                    </div>
                </div>
            )}

            <div className="skill-card-publish-tag">
                {isPublished ? t('skillCard.published') : t('skillCard.pendingReview')}
            </div>

            <div className="skill-card-body">
                <div className="skill-card-row">
                    <span className="skill-card-type">{normalizeType(skill.resource_type)}</span>
                    {canOpenUploadEditor ? (
                        <button
                            type="button"
                            className="skill-card-edit-btn"
                            aria-label="编辑资源"
                            onClick={e => {
                                e.stopPropagation()
                                navigate(`/resource/${skill.resource_type}/upload?edit=${skill.id}`)
                            }}
                        >
                            <EditIcon />
                        </button>
                    ) : (
                        <span className="skill-card-edit-placeholder" />
                    )}
                </div>

                <h3 className="skill-card-title">{skill.name}</h3>
                
                {skill.tags && (
                    <div className="skill-card-tags">
                        {skill.tags.split(',').slice(0, 3).map(tag => (
                            <span key={tag} className="skill-card-tag">{tag.trim()}</span>
                        ))}
                        {skill.tags.split(',').length > 3 && (
                            <span className="skill-card-tag">+{skill.tags.split(',').length - 3}</span>
                        )}
                    </div>
                )}

                <div className="skill-card-footer">
                    <div className="skill-card-author-wrap">
                        <span className="skill-card-author-avatar">
                            {(skill.author || 'A')[0].toUpperCase()}
                        </span>
                        <span className="skill-card-author">{skill.author || t('common.anonymous')}</span>
                    </div>

                    <div className="skill-card-metrics">
                        <span className="metric metric-download">
                            <DownloadIcon />
                            <span>{formatDownload(skill.downloads || 0)}</span>
                        </span>
                        <button
                            type="button"
                            className={`metric metric-favorite-btn ${userLiked ? 'favorited' : ''}`}
                            disabled={liking || !canLike}
                            onMouseDown={e => e.stopPropagation()}
                            onClick={async e => {
                                e.stopPropagation()
                                if (!canLike || liking) return
                                const prevLiked = userLiked
                                const prevLikesCount = likesCount
                                const nextLiked = !prevLiked
                                setUserLiked(nextLiked)
                                setLikesCount(prev => Math.max(0, prev + (nextLiked ? 1 : -1)))
                                setLiking(true)
                                try {
                                    const result = nextLiked
                                        ? await likeSkill(skill.id, skill.resource_type)
                                        : await unlikeSkill(skill.id, skill.resource_type)
                                    setLikesCount(result.likes_count ?? prevLikesCount)
                                    setUserLiked(!!result.liked)
                                } catch (err) {
                                    setUserLiked(prevLiked)
                                    setLikesCount(prevLikesCount)
                                    await showAlert(err instanceof Error ? err.message : '点赞操作失败，请重试')
                                    console.error('Failed to like skill:', err)
                                } finally {
                                    setLiking(false)
                                }
                            }}
                            aria-label={t('skillCard.like')}
                        >
                            <HeartIcon filled={userLiked} />
                            <span>{formatDownload(likesCount)}</span>
                        </button>
                        <button
                            type="button"
                            className={`metric metric-star-btn ${favorited ? 'liked' : ''}`}
                            disabled={favoriting || !canFavorite}
                            onMouseDown={e => e.stopPropagation()}
                            onClick={async e => {
                                e.stopPropagation()
                                if (!canFavorite || favoriting) return
                                const prevFavorited = favorited
                                const nextFavorited = !prevFavorited
                                setFavorited(nextFavorited)
                                onFavoriteChange?.(skill.id, nextFavorited)
                                setFavoriting(true)
                                try {
                                    if (prevFavorited) {
                                        await unfavoriteSkill(skill.id, skill.resource_type)
                                    } else {
                                        await favoriteSkill(skill.id, skill.resource_type)
                                    }
                                } catch (err) {
                                    setFavorited(prevFavorited)
                                    onFavoriteChange?.(skill.id, prevFavorited)
                                    await showAlert(err instanceof Error ? err.message : '收藏操作失败，请重试')
                                    console.error('Failed to favorite skill:', err)
                                } finally {
                                    setFavoriting(false)
                                }
                            }}
                            aria-label={t('skillCard.favorite')}
                        >
                            <StarIcon filled={favorited} />
                        </button>
                    </div>
                </div>
            </div>
        </article>
    )
}

export default SkillCard
