import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
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

function SkillCard({ skill, onFavoriteChange }: Props) {
    const navigate = useNavigate()
    const { t } = useI18n()
    const { user } = useAuth()

    const isPublished = skill.published ?? (skill.human_review_status
        ? skill.human_review_status === 'approved'
        : skill.ai_approved)
    const isPendingReview = skill.ai_approved && !isPublished
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

    return (
        <article
            className={`skill-card glass-card ${isPendingReview ? 'pending-review' : ''}`}
            onClick={() => {
                if (isPendingReview) {
                    navigate(`/review/${skill.id}`, { state: { resourceType: skill.resource_type } })
                } else {
                    navigate(`/skill/${skill.id}`, { state: { resourceType: skill.resource_type } })
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
                    <span className="skill-card-edit">✎</span>
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
                        <span className="metric metric-download">↧ {formatDownload(skill.downloads || 0)}</span>
                        <button
                            type="button"
                            className={`metric metric-favorite-btn ${userLiked ? 'favorited' : ''}`}
                            disabled={liking || !canLike}
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
                                    console.error('Failed to like skill:', err)
                                } finally {
                                    setLiking(false)
                                }
                            }}
                            aria-label={t('skillCard.like')}
                        >
                            {userLiked ? '♥' : '♡'} {formatDownload(likesCount)}
                        </button>
                        <button
                            type="button"
                            className={`metric metric-star-btn ${favorited ? 'liked' : ''}`}
                            disabled={favoriting || !canFavorite}
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
                                    console.error('Failed to favorite skill:', err)
                                } finally {
                                    setFavoriting(false)
                                }
                            }}
                            aria-label={t('skillCard.favorite')}
                        >
                            {favorited ? '★' : '☆'}
                        </button>
                    </div>
                </div>
            </div>
        </article>
    )
}

export default SkillCard
