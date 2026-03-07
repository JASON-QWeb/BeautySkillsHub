import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { useAuth } from '../../contexts/AuthContext'
import { useI18n } from '../../i18n/I18nProvider'
import { RESOURCE_TYPES, Skill, fetchSkill, submitHumanReview, getDownloadUrl } from '../../services/api'
import { useDialog } from '../../contexts/DialogContext'
import FlowStepIcon from '../../components/FlowStepIcon'
import '../../styles/upload.css'

type ReviewFileStatus = 'queued' | 'running' | 'passed' | 'failed'

interface ReviewFileItem {
    path: string
    kind: string
    status: ReviewFileStatus
    message?: string
}

interface ReviewProgress {
    total_files: number
    completed_files: number
    current_file?: string
    files: ReviewFileItem[]
}

type HumanReviewStage = 'idle' | 'confirming' | 'syncing' | 'done'

function normalizeFileStatus(status: string): ReviewFileStatus {
    if (status === 'running' || status === 'passed' || status === 'failed') {
        return status
    }
    return 'queued'
}

function parseReviewProgress(raw?: string): ReviewProgress | null {
    if (!raw) return null

    try {
        const parsed = JSON.parse(raw) as {
            total_files?: number
            completed_files?: number
            current_file?: unknown
            files?: Array<{ path?: unknown; kind?: unknown; status?: unknown; message?: unknown }>
        }

        if (typeof parsed.total_files !== 'number' || !Array.isArray(parsed.files)) {
            return null
        }

        const files = parsed.files
            .filter((item): item is { path: string; kind?: unknown; status?: unknown; message?: unknown } => typeof item?.path === 'string')
            .map(item => ({
                path: item.path,
                kind: typeof item.kind === 'string' ? item.kind : 'file',
                status: normalizeFileStatus(typeof item.status === 'string' ? item.status : ''),
                message: typeof item.message === 'string' ? item.message : undefined,
            }))

        return {
            total_files: parsed.total_files,
            completed_files: typeof parsed.completed_files === 'number' ? parsed.completed_files : 0,
            current_file: typeof parsed.current_file === 'string' ? parsed.current_file : undefined,
            files,
        }
    } catch {
        return null
    }
}

function parseAssessmentFromDescription(description: string): { security: string; functional: string } {
    const lines = description
        .split('\n')
        .map(line => line.trim())
        .filter(Boolean)

    let security = ''
    let functional = ''

    for (const line of lines) {
        if (!security && /^(安全性|安全|security)\s*[:：-]?/i.test(line)) {
            security = line.replace(/^(安全性|安全|security)\s*[:：-]?\s*/i, '').trim()
        }
        if (!functional && /^(功能性|功能|functional|function)\s*[:：-]?/i.test(line)) {
            functional = line.replace(/^(功能性|功能|functional|function)\s*[:：-]?\s*/i, '').trim()
        }
    }

    return { security, functional }
}

function mapReviewFileIcon(status: ReviewFileStatus): string {
    if (status === 'passed') return '✅'
    if (status === 'failed') return '❌'
    if (status === 'running') return '⏳'
    return '⌛'
}

function mapAIReviewStatus(status: Skill['ai_review_status'] | undefined, t: (key: string) => string): string {
    if (status === 'queued') return t('review.aiStatusQueued')
    if (status === 'running') return t('review.aiStatusRunning')
    if (status === 'passed') return t('review.aiStatusPassed')
    if (status === 'failed_retryable') return t('review.aiStatusFailedRetryable')
    if (status === 'failed_terminal') return t('review.aiStatusFailedTerminal')
    return t('review.unknown')
}

function mapAIReviewPhase(phase: Skill['ai_review_phase'] | undefined, t: (key: string) => string): string {
    if (phase === 'queued') return t('review.aiPhaseQueued')
    if (phase === 'security') return t('review.aiPhaseSecurity')
    if (phase === 'functional') return t('review.aiPhaseFunctional')
    if (phase === 'finalizing') return t('review.aiPhaseFinalizing')
    if (phase === 'done') return t('review.aiPhaseDone')
    return t('review.unknown')
}

function formatReviewTime(time?: string): string {
    if (!time) return '--'
    const parsed = new Date(time)
    if (Number.isNaN(parsed.getTime())) return '--'
    return parsed.toLocaleString()
}

function humanReviewStageText(stage: HumanReviewStage, resourceType: string, t: (key: string) => string): string {
    if (stage === 'confirming') return t('review.humanProgressConfirming')
    if (stage === 'syncing') {
        if (resourceType === 'skill') {
            return t('review.humanProgressSyncingGithub')
        }
        return t('review.humanProgressPublishing')
    }
    if (stage === 'done') return t('review.humanProgressDone')
    return ''
}

function isStepDone(current: HumanReviewStage, step: HumanReviewStage): boolean {
    const order: HumanReviewStage[] = ['idle', 'confirming', 'syncing', 'done']
    return order.indexOf(current) >= order.indexOf(step)
}

function ReviewPage() {
    const { id } = useParams()
    const location = useLocation()
    const { t } = useI18n()
    const { user } = useAuth()
    const navigate = useNavigate()
    const { showAlert } = useDialog()
    const initialResourceType = (location.state as { resourceType?: string } | null)?.resourceType || ''

    const [skill, setSkill] = useState<Skill | null>(null)
    const [loading, setLoading] = useState(true)
    const [reviewingHuman, setReviewingHuman] = useState(false)
    const [humanReviewStage, setHumanReviewStage] = useState<HumanReviewStage>('idle')
    const [error, setError] = useState('')

    const humanProgressTimerRef = useRef<number | null>(null)

    useEffect(() => {
        return () => {
            if (humanProgressTimerRef.current !== null) {
                window.clearTimeout(humanProgressTimerRef.current)
                humanProgressTimerRef.current = null
            }
        }
    }, [])

    useEffect(() => {
        const load = async () => {
            if (!id) return
            try {
                const data = await fetchSkill(Number(id), initialResourceType)
                setSkill(data)

                const isPublished = data.published ?? (data.human_review_status
                    ? data.human_review_status === 'approved'
                    : data.ai_approved)
                if (isPublished) {
                    navigate(`/resource/${data.resource_type || 'skill'}`, { replace: true })
                }
            } catch (err) {
                console.error(err)
                setError(t('detail.loadFailed'))
            } finally {
                setLoading(false)
            }
        }
        void load()
    }, [id, navigate, t])

    const reviewProgress = useMemo(() => parseReviewProgress(skill?.ai_review_details), [skill?.ai_review_details])

    const reviewSummary = useMemo(() => {
        if (!skill) {
            return { security: '', functional: '' }
        }

        const fromDescription = parseAssessmentFromDescription(skill.ai_description || '')
        const failedCount = reviewProgress?.files.filter(item => item.status === 'failed').length || 0
        const passedCount = reviewProgress?.files.filter(item => item.status === 'passed').length || 0
        const totalCount = reviewProgress?.total_files || 0

        const security = fromDescription.security || (
            failedCount > 0
                ? t('review.securitySummaryRisk', { count: failedCount })
                : t('review.securitySummarySafe')
        )

        let functional = fromDescription.functional
        if (!functional) {
            if (skill.ai_approved) {
                functional = totalCount > 0
                    ? t('review.functionalSummaryPassWithFiles', { passed: passedCount, total: totalCount })
                    : t('review.functionalSummaryPass')
            } else {
                functional = failedCount > 0
                    ? t('review.functionalSummaryNeedsFix', { count: failedCount })
                    : (skill.ai_feedback || t('upload.reviewRejectedFallback'))
            }
        }

        return { security, functional }
    }, [reviewProgress, skill, t])

    if (loading) {
        return (
            <div className="upload-page page-enter">
                <div className="upload-topbar upload-topbar-left">
                    <Link to="/" className="upload-back-btn">← {t('upload.back')}</Link>
                </div>
                <div className="upload-flow-layout" style={{ justifyContent: 'center', marginTop: '10vh' }}>
                    <div className="upload-placeholder-card glass-card">
                        <p>{t('common.loadingResources')}</p>
                    </div>
                </div>
            </div>
        )
    }

    if (error || !skill) {
        return (
            <div className="upload-page page-enter">
                <div className="upload-topbar upload-topbar-left">
                    <Link to="/" className="upload-back-btn">← {t('upload.back')}</Link>
                </div>
                <div className="upload-flow-layout" style={{ justifyContent: 'center', marginTop: '10vh' }}>
                    <div className="upload-placeholder-card glass-card">
                        <p>{error || t('detail.resourceNotFound')}</p>
                    </div>
                </div>
            </div>
        )
    }

    const canManage = !!user && (
        (skill.user_id ? skill.user_id === user.id : false) ||
        (skill.author || '').trim().toLowerCase() === user.username.trim().toLowerCase()
    )
    const humanReviewStatus = skill.human_review_status || 'pending'
    const canHumanReview = !!user && skill.ai_approved && !canManage && humanReviewStatus === 'pending'
    const step = 3

    return (
        <div className="upload-page page-enter">
            <div className="upload-topbar upload-topbar-left">
                <button type="button" onClick={() => navigate(-1)} className="upload-back-btn">← {t('upload.back')}</button>
            </div>

            <div className="upload-steps single-line">
                <div className={`step ${step >= 1 ? 'active' : ''}`}>
                    <div className="step-circle"><FlowStepIcon kind="upload" /></div>
                    <span className="step-label">{t('upload.stepUser')}</span>
                </div>
                <div className={`step-connector ${step >= 2 ? 'active' : ''}`} />
                <div className={`step ${step >= 2 ? 'active' : ''}`}>
                    <div className="step-circle"><FlowStepIcon kind="ai" /></div>
                    <span className="step-label">{t('upload.stepAi')}</span>
                </div>
                <div className={`step-connector ${step >= 3 ? 'active' : ''}`} />
                <div className={`step ${step >= 3 ? 'active' : ''}`}>
                    <div className="step-circle"><FlowStepIcon kind="human" /></div>
                    <span className="step-label">{t('upload.stepHuman')}</span>
                </div>
            </div>

            <div className="upload-flow-layout">
                <section className="upload-stage-card compact">
                    <div className="upload-compact-card glass-card">
                        <div className="upload-compact-title">{t('upload.userUploadCompactTitle')}</div>
                        <div className="upload-compact-content">
                            {skill.thumbnail_url ? (
                                <img className="upload-compact-thumb" src={skill.thumbnail_url} alt={skill.name} />
                            ) : (
                                <div className="upload-compact-thumb upload-compact-fallback">
                                    {(skill.name || 'S')[0].toUpperCase()}
                                </div>
                            )}
                            <div className="upload-compact-meta" style={{ flex: 1, minWidth: 0 }}>
                                <strong style={{ whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', display: 'block' }}>{skill.name}</strong>
                                <span>{RESOURCE_TYPES[skill.resource_type || 'skill']?.label || skill.resource_type?.toUpperCase() || 'SKILL'}</span>
                                {skill.tags && (
                                    <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginTop: 8 }}>
                                        {skill.tags.split(',').filter(Boolean).map(tag => (
                                            <span key={tag} className="upload-tag-chip" style={{ fontSize: '0.7rem', padding: '2px 6px', margin: 0 }}>
                                                {tag}
                                            </span>
                                        ))}
                                    </div>
                                )}
                                {skill.file_name && (
                                    <div style={{ marginTop: 8 }}>
                                        <a
                                            href={getDownloadUrl(skill.id, skill.resource_type)}
                                            target="_blank"
                                            rel="noreferrer"
                                            className="btn btn-secondary"
                                            style={{ padding: '4px 8px', fontSize: '0.75rem', display: 'inline-flex', alignItems: 'center', gap: 4 }}
                                            onClick={e => e.stopPropagation()}
                                        >
                                            📄 {skill.file_name} (预览)
                                        </a>
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>
                </section>

                <section className="upload-stage-card compact">
                    <div className="upload-compact-card glass-card">
                        <div className="upload-compact-title">{t('upload.aiReviewCompactTitle')}</div>
                        <div className={`review-status ${skill.ai_approved ? 'approved' : 'rejected'}`}>
                            <strong>
                                {skill.ai_approved ? t('upload.reviewApprovedTitle') : t('upload.reviewRejectedTitle')}
                            </strong>
                            <p>{skill.ai_feedback || (skill.ai_approved ? t('upload.reviewApprovedFallback') : t('upload.reviewRejectedFallback'))}</p>
                        </div>

                        <div className="review-assessment-grid">
                            <div className="review-assessment-item">
                                <div className="review-assessment-label">{t('detail.securityAssessment')}</div>
                                <p>{reviewSummary.security}</p>
                            </div>
                            <div className="review-assessment-item">
                                <div className="review-assessment-label">{t('detail.functionalAssessment')}</div>
                                <p>{reviewSummary.functional}</p>
                            </div>
                        </div>

                        <div className="review-meta-grid">
                            <div className="review-meta-item">
                                <span>{t('review.metaStatus')}</span>
                                <strong>{mapAIReviewStatus(skill.ai_review_status, t)}</strong>
                            </div>
                            <div className="review-meta-item">
                                <span>{t('review.metaPhase')}</span>
                                <strong>{mapAIReviewPhase(skill.ai_review_phase, t)}</strong>
                            </div>
                            <div className="review-meta-item">
                                <span>{t('review.metaAttempts')}</span>
                                <strong>{skill.ai_review_attempts ?? 0}/{skill.ai_review_max_attempts ?? 0}</strong>
                            </div>
                            <div className="review-meta-item">
                                <span>{t('review.metaStartedAt')}</span>
                                <strong>{formatReviewTime(skill.ai_review_started_at)}</strong>
                            </div>
                            <div className="review-meta-item">
                                <span>{t('review.metaCompletedAt')}</span>
                                <strong>{formatReviewTime(skill.ai_review_completed_at)}</strong>
                            </div>
                            <div className="review-meta-item">
                                <span>{t('review.metaFiles')}</span>
                                <strong>
                                    {reviewProgress
                                        ? `${reviewProgress.completed_files}/${reviewProgress.total_files}`
                                        : '--'}
                                </strong>
                            </div>
                        </div>

                        {reviewProgress ? (
                            <div className="review-file-progress">
                                <div className="review-file-progress-head">
                                    <span>{t('review.filesChecked', { completed: reviewProgress.completed_files, total: reviewProgress.total_files })}</span>
                                    {reviewProgress.current_file && (
                                        <span>{t('review.currentFile', { file: reviewProgress.current_file })}</span>
                                    )}
                                </div>
                                <ul className="review-file-list">
                                    {reviewProgress.files.map(item => (
                                        <li key={item.path} className={`review-file-item ${item.status}`}>
                                            <span className={`review-file-icon ${item.status}`}>
                                                {mapReviewFileIcon(item.status)}
                                            </span>
                                            <span className="review-file-path">{item.path}</span>
                                            {item.message && (
                                                <span className="review-file-message">{item.message}</span>
                                            )}
                                        </li>
                                    ))}
                                </ul>
                            </div>
                        ) : (
                            <div className="review-raw-details">
                                {skill.ai_review_details?.trim() || t('review.noDetails')}
                            </div>
                        )}
                    </div>
                </section>

                <section className="upload-stage-card expanded">
                    <div className="upload-review-card upload-human-card glass-card">
                        <h2>{t('upload.humanReviewPanelTitle')}</h2>

                        <div className="review-status pending">
                            <strong>{t('upload.stepHuman')}</strong>
                            <p>{t('upload.humanReviewWaiting')}</p>
                        </div>

                        {humanReviewStage !== 'idle' && (
                            <div className="human-review-progress">
                                <div className="human-review-progress-current">{humanReviewStageText(humanReviewStage, skill.resource_type || 'skill', t)}</div>
                                <div className="human-review-progress-steps">
                                    <span className={isStepDone(humanReviewStage, 'confirming') ? 'done' : ''}>
                                        1. {t('review.humanProgressStepConfirm')}
                                    </span>
                                    <span className={isStepDone(humanReviewStage, 'syncing') ? 'done' : ''}>
                                        2. {(skill.resource_type || 'skill') === 'skill'
                                            ? t('review.humanProgressStepSync')
                                            : t('review.humanProgressStepPublish')}
                                    </span>
                                    <span className={isStepDone(humanReviewStage, 'done') ? 'done' : ''}>
                                        3. {t('review.humanProgressStepDone')}
                                    </span>
                                </div>
                            </div>
                        )}

                        {!user ? (
                            <div className="review-actions">
                                <small style={{ color: 'var(--text-muted)' }}>{t('protected.signInRequired')}</small>
                            </div>
                        ) : canManage ? (
                            <div className="review-actions">
                                <small style={{ color: 'var(--warning)' }}>{t('detail.humanReviewNeedOtherUser')}</small>
                            </div>
                        ) : canHumanReview ? (
                            <div className="review-actions">
                                <button
                                    type="button"
                                    className="btn btn-primary"
                                    style={{ width: '100%' }}
                                    onClick={async () => {
                                        setReviewingHuman(true)
                                        setHumanReviewStage('confirming')

                                        if (humanProgressTimerRef.current !== null) {
                                            window.clearTimeout(humanProgressTimerRef.current)
                                        }
                                        humanProgressTimerRef.current = window.setTimeout(() => {
                                            setHumanReviewStage(prev => (prev === 'confirming' ? 'syncing' : prev))
                                            humanProgressTimerRef.current = null
                                        }, 700)

                                        try {
                                            const updated = await submitHumanReview(skill.id, skill.resource_type || 'skill', true)
                                            if (humanProgressTimerRef.current !== null) {
                                                window.clearTimeout(humanProgressTimerRef.current)
                                                humanProgressTimerRef.current = null
                                            }
                                            setSkill(updated)
                                            setHumanReviewStage('done')

                                            await new Promise(resolve => window.setTimeout(resolve, 700))
                                            navigate(`/resource/${updated.resource_type || skill.resource_type || 'skill'}`, { replace: true })
                                        } catch (err) {
                                            if (humanProgressTimerRef.current !== null) {
                                                window.clearTimeout(humanProgressTimerRef.current)
                                                humanProgressTimerRef.current = null
                                            }
                                            setHumanReviewStage('idle')
                                            await showAlert(err instanceof Error ? err.message : t('detail.humanReviewFailed'))
                                        } finally {
                                            setReviewingHuman(false)
                                        }
                                    }}
                                    disabled={reviewingHuman}
                                >
                                    {reviewingHuman
                                        ? humanReviewStageText(humanReviewStage, skill.resource_type || 'skill', t)
                                        : t('detail.confirmHumanReview')}
                                </button>
                            </div>
                        ) : null}
                    </div>
                </section>
            </div>
        </div>
    )
}

export default ReviewPage
