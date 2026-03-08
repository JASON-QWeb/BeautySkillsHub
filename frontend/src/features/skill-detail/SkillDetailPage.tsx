import { useEffect, useMemo, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { useAuth } from '../../contexts/AuthContext'
import { useI18n } from '../../i18n/I18nProvider'
import { DeleteProgressStage, Skill, deleteSkill, deleteSkillWithProgress, favoriteSkill, fetchSkill, fetchSkillInstallConfig, fetchSkillReadme, getDownloadUrl, likeSkill, trackDownloadHit, unfavoriteSkill, unlikeSkill } from '../../services/api'
import { isAbortError } from '../../services/api/request'
import { useDialog } from '../../contexts/DialogContext'
import LoadingBars from '../../components/LoadingBars'
import { formatDate, formatSize } from './formatters'
import { parseMarkdown } from '../detail/shared/markdown'
import { parseReadmeFrontMatter } from '../detail/shared/frontmatter'
import { createReadmeCache } from './readmeCache'

function formatDownloads(downloads: number) {
    if (downloads >= 1000) return `${(downloads / 1000).toFixed(1)}K`
    return `${downloads}`
}

function normalizeSummaryLine(line: string) {
    return line
        .replace(/^(安全性|安全|功能性|功能|security|function)\s*[:：-]?\s*/i, '')
        .trim()
}

function isSecuritySentence(line: string) {
    return /^(安全性|安全|security|safety|risk|风险)\s*[:：-]?/i.test(line)
        || /(未发现风险|高风险|恶意|漏洞|security|risk)/i.test(line)
}

function normalizeInstallSkillName(name: string) {
    const normalized = name
        .trim()
        .toLowerCase()
        .replace(/\s+/g, '_')
        .replace(/[^a-z0-9_-]+/g, '')
    return normalized || 'untitled_skill'
}

function normalizeInstallFlag(baseDir: string) {
    const normalized = baseDir
        .trim()
        .toLowerCase()
        .replace(/[^a-z0-9_-]+/g, '')
    return normalized || 'skills'
}

function buildFunctionalSummaryLines(skill: Skill | null, fallbackDescription: string): string[] {
    if (!skill) return []

    const candidateLines: string[] = []
    const addLine = (line: string) => {
        const trimmed = line.trim()
        if (!trimmed || isSecuritySentence(trimmed)) return
        candidateLines.push(trimmed.replace(/\s+/g, ' '))
    }

    ;(skill.ai_description || '')
        .split('\n')
        .map(line => normalizeSummaryLine(line))
        .forEach(addLine)

    ;(skill.ai_feedback || '')
        .split(/[\n。！？]/)
        .map(line => line.trim())
        .forEach(addLine)

    if (fallbackDescription.trim()) {
        fallbackDescription
            .split(/[\n。！？]/)
            .map(line => line.trim())
            .forEach(addLine)
    }

    const unique = Array.from(new Set(candidateLines))
    if (unique.length >= 2) return unique.slice(0, 2)
    if (unique.length === 1) {
        return [
            unique[0],
            `适用于 ${skill.name} 相关场景，可按项目需求扩展能力与流程。`,
        ]
    }
    return [
        `该资源围绕 ${skill.name} 提供可复用的功能能力与实现路径。`,
        '适合结合 README 的安装与配置步骤，在项目中快速落地使用。',
    ]
}

function extractSafetyChecks(skill: Skill | null): string[] {
    if (!skill) return []
    const source = `${skill.ai_feedback || ''}\n${skill.ai_description || ''}`
    const checks: string[] = []
    const seen = new Set<string>()

    const byEmoji = source.match(/✅\s*[^\n，。,；;]+/g) || []
    byEmoji.forEach(item => {
        const cleaned = item.replace(/\s+/g, ' ').trim()
        if (cleaned && !seen.has(cleaned)) {
            seen.add(cleaned)
            checks.push(cleaned)
        }
    })

    if (checks.length > 0) return checks.slice(0, 6)
    if (!skill.ai_approved) return []

    return [
        '✅ 已通过自动安全审核',
        '✅ 未发现高危恶意行为',
        '✅ 适合社区分发使用',
    ]
}

function FallbackThumb({ name, description }: { name: string; description?: string }) {
    const gradients = [
        'linear-gradient(135deg, #4f83e8 0%, #5d63dc 100%)',
        'linear-gradient(135deg, #36c88f 0%, #0f9467 100%)',
        'linear-gradient(135deg, #38bdf8 0%, #3b82f6 100%)',
        'linear-gradient(135deg, #374151 0%, #1f2937 100%)',
        'linear-gradient(135deg, #f97316 0%, #ef4444 100%)',
        'linear-gradient(135deg, #6C5CE7 0%, #A29BFE 100%)',
    ]
    let hash = 0
    for (const c of name) hash = hash * 31 + c.charCodeAt(0)
    const bg = gradients[Math.abs(hash) % gradients.length]

    return (
        <div className="detail-thumb detail-thumb-fallback" style={{ background: bg }}>
            <div className="detail-thumb-text">
                <span className="detail-thumb-title">{name}</span>
                {description && (
                    <span className="detail-thumb-desc">
                        {description.length > 60 ? description.slice(0, 57) + '...' : description}
                    </span>
                )}
            </div>
        </div>
    )
}

const readmeGlobalCache = createReadmeCache(50)

/** Invalidate readme cache for a specific skill so detail page re-fetches fresh content. */
export function invalidateReadmeCache(skillId: number) {
    readmeGlobalCache.delete(skillId)
}
type DeleteFlowStage = DeleteProgressStage | 'done'
const DELETE_FLOW_ORDER: DeleteFlowStage[] = ['db', 'github', 'done']

function stageRank(stage: DeleteFlowStage) {
    const idx = DELETE_FLOW_ORDER.indexOf(stage)
    return idx >= 0 ? idx : 0
}

interface SkillDetailPageProps {
    resourceTypeOverride?: 'skill' | 'rules' | 'mcp' | 'tools'
}

function SkillDetailPage({ resourceTypeOverride }: SkillDetailPageProps) {
    const { id, type } = useParams<{ id: string; type?: string }>()
    const location = useLocation()
    const navigate = useNavigate()
    const { t } = useI18n()
    const { user } = useAuth()
    const { showAlert, showConfirm } = useDialog()
    const initialResourceType = resourceTypeOverride
        || (location.state as { resourceType?: string } | null)?.resourceType
        || (type || '')

    const [skill, setSkill] = useState<Skill | null>(null)
    const [readmeContent, setReadmeContent] = useState<string>('')
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState('')
    const [deleting, setDeleting] = useState(false)
    const [deleteModalOpen, setDeleteModalOpen] = useState(false)
    const [deleteStage, setDeleteStage] = useState<DeleteFlowStage>('db')
    const [deleteGithubWarning, setDeleteGithubWarning] = useState('')
    const [copied, setCopied] = useState(false)
    const [liking, setLiking] = useState(false)
    const [likesCount, setLikesCount] = useState(0)
    const [userLiked, setUserLiked] = useState(false)
    const [favoriting, setFavoriting] = useState(false)
    const [favorited, setFavorited] = useState(false)
    const [installConfig, setInstallConfig] = useState({
        github_repo: 'https://github.com/skillshub/community',
        github_base_dir: 'skills',
    })

    useEffect(() => {
        if (!id) return

        const controller = new AbortController()

        const load = async () => {
            setLoading(true)
            const numericId = Number(id)

            // Invalidate readme cache if navigating back after a review/update
            const stateAny = location.state as { refreshReadme?: boolean } | null
            if (stateAny?.refreshReadme) {
                readmeGlobalCache.delete(numericId)
            }

            try {
                // If we already have this readme in memory, use it instead of refetching
                let readmePromise: Promise<string>
                const cachedReadme = readmeGlobalCache.get(numericId)
                if (cachedReadme !== undefined) {
                    readmePromise = Promise.resolve(cachedReadme)
                } else {
                    readmePromise = fetchSkillReadme(numericId, initialResourceType, { signal: controller.signal })
                        .then(content => {
                            readmeGlobalCache.set(numericId, content)
                            return content
                        })
                        .catch(() => '')
                }

                const [data, readme] = await Promise.all([
                    fetchSkill(numericId, initialResourceType, { signal: controller.signal }),
                    readmePromise
                ])
                
                setSkill(data)
                setReadmeContent(readme)
                setError('')
            } catch (err) {
                if (isAbortError(err)) return
                setError(err instanceof Error ? err.message : t('detail.loadFailed'))
            } finally {
                if (!controller.signal.aborted) {
                    setLoading(false)
                }
            }
        }

        load()
        return () => controller.abort()
    }, [id, initialResourceType, t])

    useEffect(() => {
        if (!skill) return
        setLikesCount(skill.likes_count || 0)
        setUserLiked(!!skill.user_liked)
        setFavorited(!!skill.favorited)
    }, [skill])

    const canManage = useMemo(() => {
        if (!user || !skill) return false
        if (skill.user_id && skill.user_id > 0) {
            return skill.user_id === user.id
        }
        return (skill.author || '').trim().toLowerCase() === user.username.trim().toLowerCase()
    }, [skill, user])

    const isPublished = useMemo(() => {
        if (!skill) return false
        if (typeof skill.published === 'boolean') return skill.published
        if (skill.human_review_status) return skill.human_review_status === 'approved'
        return skill.ai_approved
    }, [skill])

    const humanReviewStatus = useMemo(() => {
        if (!skill) return 'pending'
        if (skill.human_review_status) return skill.human_review_status
        return isPublished ? 'approved' : 'pending'
    }, [skill, isPublished])

    const hasPendingRevision = useMemo(() => {
        return !!skill?.has_pending_revision
    }, [skill?.has_pending_revision])

    const activeHumanReviewStatus = useMemo(() => {
        if (hasPendingRevision) {
            return skill?.pending_revision_human || 'pending'
        }
        return humanReviewStatus
    }, [hasPendingRevision, humanReviewStatus, skill?.pending_revision_human])


    const canLike = useMemo(() => {
        return !!user && !!skill && skill.ai_approved
    }, [user, skill])

    const canFavorite = useMemo(() => {
        return !!user && !!skill && skill.ai_approved
    }, [user, skill])

    const currentResourceType = useMemo(() => {
        return (skill?.resource_type || initialResourceType || 'skill').toLowerCase()
    }, [skill?.resource_type, initialResourceType])

    const isSkillType = currentResourceType === 'skill'
    const isRulesType = currentResourceType === 'rules'
    const isMcpType = currentResourceType === 'mcp'
    const isToolsType = currentResourceType === 'tools'
    const requiresReview = isSkillType || isRulesType
    const showsHumanReviewPanel = requiresReview || hasPendingRevision

    useEffect(() => {
        if (!isSkillType) return
        const controller = new AbortController()
        void fetchSkillInstallConfig({ signal: controller.signal })
            .then(config => {
                setInstallConfig({
                    github_repo: config.github_repo || 'https://github.com/skillshub/community',
                    github_base_dir: config.github_base_dir || 'skills',
                })
            })
            .catch(err => {
                if (isAbortError(err)) return
                setInstallConfig(prev => ({
                    github_repo: prev.github_repo || 'https://github.com/skillshub/community',
                    github_base_dir: prev.github_base_dir || 'skills',
                }))
            })
        return () => {
            controller.abort()
        }
    }, [isSkillType])

    const installCommand = useMemo(() => {
        if (!skill || !isSkillType) return ''
        const skillKey = normalizeInstallSkillName(skill.name)
        const installFlag = normalizeInstallFlag(installConfig.github_base_dir)
        return `npx skills add ${installConfig.github_repo} --${installFlag} ${skillKey}`
    }, [installConfig.github_base_dir, installConfig.github_repo, isSkillType, skill])

    const breadcrumbPrefix = useMemo(() => {
        if (!skill) return 'skills'
        if (skill.resource_type === 'mcp') return 'mcp'
        if (skill.resource_type === 'rules') return 'rules'
        if (skill.resource_type === 'tools') return 'tools'
        return 'skills'
    }, [skill])

    const downloadPath = useMemo(() => {
        if (!skill) return ''
        return getDownloadUrl(skill.id, skill.resource_type)
    }, [skill])

    const hasDownloadAsset = useMemo(() => {
        if (!skill) return false
        return !!skill.file_name && (skill.file_size || 0) > 0
    }, [skill])

    const parsedReadme = useMemo(() => parseReadmeFrontMatter(readmeContent), [readmeContent])
    const readmeDescription = useMemo(() => {
        return (parsedReadme.frontMatter.description || '').trim()
    }, [parsedReadme.frontMatter.description])
    const displayedReadmeContent = useMemo(() => {
        return parsedReadme.body.trim()
    }, [parsedReadme.body])
    const heroDescription = useMemo(() => {
        if (readmeDescription) return readmeDescription
        if (isMcpType || isToolsType) return ''

        const plain = (skill?.description || '').trim()
        if (!plain) return ''
        if (plain.length > 180) return ''
        if (/[\n#`>*\-]/.test(plain)) return ''
        return plain
    }, [isMcpType, isToolsType, readmeDescription, skill?.description])
    const renderedMarkdownContent = useMemo(() => {
        return displayedReadmeContent
            || ((isMcpType || isToolsType) ? (skill?.description || '').trim() : '')
            || (skill?.description || '').trim()
            || t('detail.descriptionFallback')
    }, [displayedReadmeContent, isMcpType, isToolsType, skill?.description, t])
    const contentHeaderTitle = useMemo(() => {
        if (isMcpType) return '正文内容'
        if (isToolsType) return '正文内容'
        if (isRulesType) return skill?.file_name || 'RULES.md'
        return skill?.file_name || 'SKILLS.md'
    }, [isMcpType, isRulesType, isToolsType, skill?.file_name])
    const aiSummaryLines = useMemo(() => buildFunctionalSummaryLines(skill, heroDescription), [heroDescription, skill])
    const safetyChecks = useMemo(() => extractSafetyChecks(skill), [skill])
    const deleteSteps: Array<{ stage: DeleteFlowStage; label: string }> = useMemo(() => ([
        { stage: 'db', label: t('detail.deleteStepDb') },
        { stage: 'github', label: t('detail.deleteStepGithub') },
        { stage: 'done', label: t('detail.deleteStepDone') },
    ]), [t])

    if (loading) {
        return (
            <div className="detail-page page-enter">
                <div className="empty-state">
                    <div className="loading-spinner" style={{ width: 40, height: 40 }} />
                    <p>{t('detail.loading')}</p>
                </div>
            </div>
        )
    }

    if (!skill || error) {
        return (
            <div className="detail-page page-enter">
                <div className="empty-state">
                    <div className="icon">◐</div>
                    <p>{error || t('detail.resourceNotFound')}</p>
                    <Link to="/resource/skill" className="btn btn-primary">{t('detail.backHome')}</Link>
                </div>
            </div>
        )
    }

    return (
        <div className="detail-page page-enter">
            <header className="detail-topbar">
                <button
                    className="detail-back-btn"
                    onClick={() => navigate(-1)}
                    type="button"
                >
                    ← {t('detail.back')}
                </button>
            </header>

            <div className="detail-breadcrumbs">
                <span>{breadcrumbPrefix}</span>
                <span>/</span>
                <span>{skill.name}</span>
            </div>

            <div className="detail-layout">
                <main className="detail-main">
                    <div className="detail-title-row">
                        <h1 className="detail-title">{skill.name}</h1>
                        {requiresReview && canManage && hasPendingRevision && (
                            <button
                                type="button"
                                className="detail-title-action action-pending"
                                onClick={() => navigate(`/review/${skill.id}`, { state: { resourceType: skill.resource_type } })}
                            >
                                有未Review的更新
                            </button>
                        )}
                        {requiresReview && canManage && !hasPendingRevision && (
                            <button
                                type="button"
                                className="detail-title-action action-update"
                                onClick={() => navigate(`/resource/${currentResourceType}/upload?edit=${skill.id}`)}
                            >
                                {t('detail.update')}
                            </button>
                        )}
                        {requiresReview && !canManage && hasPendingRevision && user && (
                            <button
                                type="button"
                                className="detail-title-action action-review"
                                onClick={() => navigate(`/review/${skill.id}`, { state: { resourceType: skill.resource_type } })}
                            >
                                新版本，待Review
                            </button>
                        )}
                    </div>

                    {skill.tags && (
                        <div className="detail-tags">
                            {skill.tags.split(',').map(tag => (
                                <span key={tag} className="detail-tag">{tag.trim()}</span>
                            ))}
                        </div>
                    )}

                    {heroDescription && <p className="detail-hero-desc">{heroDescription}</p>}

                    {isSkillType && (
                        <div className="detail-terminal">
                            <div className="detail-terminal-text">
                                <span className="detail-terminal-prompt">$</span>
                                <span className="detail-terminal-cmd">{installCommand}</span>
                            </div>
                            <button
                                type="button"
                                className="detail-terminal-copy"
                                onClick={async () => {
                                    try {
                                        await navigator.clipboard.writeText(installCommand)
                                        const result = await trackDownloadHit(skill.id, skill.resource_type)
                                        setSkill(prev => prev ? { ...prev, downloads: result.downloads } : prev)
                                        setCopied(true)
                                        setTimeout(() => setCopied(false), 1200)
                                    } catch {
                                        setCopied(false)
                                    }
                                }}
                            >
                                {copied ? t('detail.copied') : t('detail.copy')}
                            </button>
                        </div>
                    )}

                    {isMcpType && skill.github_url && (
                        <div className="detail-terminal">
                            <div className="detail-terminal-text">
                                <span className="detail-terminal-prompt">↗</span>
                                <span className="detail-terminal-cmd">{skill.github_url}</span>
                            </div>
                            <a
                                href={skill.github_url}
                                target="_blank"
                                rel="noreferrer"
                                className="detail-terminal-copy"
                            >
                                打开
                            </a>
                        </div>
                    )}

                    {requiresReview && (
                        <div className={`detail-ai-review ${skill.ai_approved ? 'approved' : 'rejected'}`}>
                            <span className="detail-ai-icon">◉</span>
                            <div className="detail-ai-text">
                                <div className="detail-ai-label">{t('detail.aiReviewSummary')}</div>
                                <div className="detail-ai-summary-lines">
                                    {aiSummaryLines.map((line, idx) => (
                                        <p key={`${idx}-${line}`}>{line}</p>
                                    ))}
                                </div>
                            </div>
                        </div>
                    )}

                    {/* Content / markdown */}
                    <div className="detail-content-header">
                        <span>{contentHeaderTitle}</span>
                    </div>

                    <div
                        className="detail-description markdown-body"
                        dangerouslySetInnerHTML={{
                            __html: parseMarkdown(renderedMarkdownContent)
                        }}
                    />
                </main>

                <aside className="detail-sidebar">
                    <div className="detail-sidebar-section">
                        <div className="detail-sidebar-label">{t('detail.thumbnail')}</div>
                        {skill.thumbnail_url ? (
                            <img src={skill.thumbnail_url} alt={skill.name} className="detail-thumb" />
                        ) : (
                            <FallbackThumb name={skill.name} description={skill.description} />
                        )}
                    </div>

                    <div className="detail-sidebar-section">
                        <div className="detail-sidebar-label">{t('detail.totalDownloads')}</div>
                        <div className="detail-sidebar-big">{formatDownloads(skill.downloads)}</div>
                    </div>

                    <div className="detail-sidebar-section">
                        <div className="detail-sidebar-label">{t('detail.updateHistory')}</div>
                        <p className="detail-sidebar-value">{t('detail.lastUpdated')}</p>
                        <small className="detail-sidebar-value">{formatDate(skill.updated_at || skill.created_at)}</small>
                        <p className="detail-sidebar-value" style={{ marginTop: 8 }}>{t('detail.firstUploaded')}</p>
                        <small className="detail-sidebar-value">{formatDate(skill.created_at)}</small>
                    </div>

                    <div className="detail-sidebar-section">
                        <div className="detail-sidebar-label">{t('detail.repository')}</div>
                        <p className="detail-sidebar-value">
                            {isMcpType
                                ? (skill.github_url || '未提供 GitHub 链接')
                                : installConfig.github_repo.replace(/^https?:\/\/github\.com\//i, '')}
                        </p>
                    </div>

                    <div className="detail-sidebar-section">
                        <div className="detail-sidebar-label">{t('detail.githubStars')}</div>
                        <div className="detail-like-wrap">
                            <p className="detail-sidebar-value">★ {likesCount}</p>
                            {canLike && (
                                <button
                                    type="button"
                                    className={`btn btn-secondary btn-sm ${userLiked ? 'detail-like-btn-liked' : ''}`}
                                    onClick={async () => {
                                        if (liking) return
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
                                            await showAlert(err instanceof Error ? err.message : t('detail.likeFailed'))
                                        } finally {
                                            setLiking(false)
                                        }
                                    }}
                                    disabled={liking}
                                >
                                    {userLiked ? t('detail.liked') : t('detail.like')}
                                </button>
                            )}
                            {canFavorite && (
                                <button
                                    type="button"
                                    className={`btn btn-secondary btn-sm ${favorited ? 'detail-favorite-btn-active' : ''}`}
                                    onClick={async () => {
                                        if (favoriting) return
                                        const prevFavorited = favorited
                                        const nextFavorited = !prevFavorited
                                        setFavorited(nextFavorited)
                                        setFavoriting(true)
                                        try {
                                            if (prevFavorited) {
                                                await unfavoriteSkill(skill.id, skill.resource_type)
                                            } else {
                                                await favoriteSkill(skill.id, skill.resource_type)
                                            }
                                        } catch (err) {
                                            setFavorited(prevFavorited)
                                            await showAlert(err instanceof Error ? err.message : t('detail.favoriteFailed'))
                                        } finally {
                                            setFavoriting(false)
                                        }
                                    }}
                                    disabled={favoriting}
                                >
                                    {favorited ? t('detail.favorited') : t('detail.favorite')}
                                </button>
                            )}
                        </div>
                    </div>

                    {showsHumanReviewPanel && (
                        <div className="detail-sidebar-section">
                            <div className="detail-sidebar-label">{t('detail.securityChecks')}</div>
                            {safetyChecks.length > 0 ? (
                                <ul className="detail-safety-checks">
                                    {safetyChecks.map((check, idx) => (
                                        <li key={`${idx}-${check}`}>{check}</li>
                                    ))}
                                </ul>
                            ) : (
                                <div className="security-fail">{t('detail.review')}</div>
                            )}
                        </div>
                    )}

                    {requiresReview && (
                        <div className="detail-sidebar-section">
                            <div className="detail-sidebar-label">{t('detail.humanReview')}</div>
                            <div className={`detail-human-review-status ${activeHumanReviewStatus}`}>
                                {activeHumanReviewStatus === 'approved' && t('detail.humanReviewApproved')}
                                {activeHumanReviewStatus === 'pending' && t('detail.humanReviewPending')}
                                {activeHumanReviewStatus === 'rejected' && t('detail.humanReviewRejected')}
                            </div>
                            {hasPendingRevision && (
                                <small className="detail-sidebar-value">当前详情展示的是已发布版本，待审核更新请前往 Review 页面。</small>
                            )}
                            {!hasPendingRevision && skill.human_reviewer && (
                                <small className="detail-sidebar-value">
                                    {t('detail.reviewedBy', { user: skill.human_reviewer })}
                                </small>
                            )}
                            {canManage && activeHumanReviewStatus === 'pending' && (
                                <small className="detail-sidebar-value">{t('detail.humanReviewNeedOtherUser')}</small>
                            )}
                        </div>
                    )}

                    <div className="detail-sidebar-section detail-actions">
                        {hasDownloadAsset && (
                            <a href={downloadPath} className="btn btn-primary">
                                {isPublished ? t('detail.download') : t('detail.downloadPending')}
                            </a>
                        )}
                        {isMcpType && skill.github_url && (
                            <a href={skill.github_url} target="_blank" rel="noreferrer" className="btn btn-primary">
                                打开 GitHub
                            </a>
                        )}
                        {canManage && !requiresReview && hasPendingRevision && (
                            <button className="btn btn-secondary" disabled>
                                更新中 / 待 Review
                            </button>
                        )}
                        {canManage && !requiresReview && !hasPendingRevision && (
                            <button
                                className="btn btn-secondary"
                                onClick={() => navigate(`/resource/${currentResourceType}/upload?edit=${skill.id}`)}
                                disabled={deleting}
                            >
                                {t('detail.edit')}
                            </button>
                        )}
                        {canManage && (
                            <button
                                className="btn btn-secondary"
                                onClick={async () => {
                                    if (!(await showConfirm(t('detail.confirmDelete')))) return
                                    const isSkillResource = (skill.resource_type || 'skill') === 'skill'
                                    if (isSkillResource) {
                                        setDeleteStage('db')
                                        setDeleteGithubWarning('')
                                        setDeleteModalOpen(true)
                                    }
                                    setDeleting(true)
                                    try {
                                        if (isSkillResource) {
                                            const result = await deleteSkillWithProgress(
                                                skill.id,
                                                skill.resource_type,
                                                (stage) => setDeleteStage(stage),
                                            )
                                            if (result.github_error) {
                                                setDeleteGithubWarning(result.github_error)
                                            }
                                            setDeleteStage('done')
                                            await new Promise(resolve => setTimeout(resolve, 420))
                                        } else {
                                            await deleteSkill(skill.id, skill.resource_type)
                                        }
                                        navigate(`/resource/${skill.resource_type || 'skill'}`)
                                    } catch (err) {
                                        if (isSkillResource) {
                                            setDeleteModalOpen(false)
                                        }
                                        await showAlert(err instanceof Error ? err.message : t('detail.deleteFailed'))
                                    } finally {
                                        setDeleting(false)
                                    }
                                }}
                                disabled={deleting}
                            >
                                {deleting ? t('detail.deleting') : t('detail.delete')}
                            </button>
                        )}
                    </div>

                    {(hasDownloadAsset || isMcpType) && (
                        <div className="detail-sidebar-section">
                            <div className="detail-sidebar-label">{t('detail.fileInfo')}</div>
                            {hasDownloadAsset ? (
                                <>
                                    <small className="detail-sidebar-value">{skill.file_name}</small>
                                    <small className="detail-sidebar-value">{formatSize(skill.file_size)}</small>
                                </>
                            ) : (
                                <small className="detail-sidebar-value">无本地附件</small>
                            )}
                        </div>
                    )}
                </aside>
            </div>

            {deleteModalOpen && (
                <div className="delete-progress-overlay">
                    <div className="delete-progress-modal glass-card" role="alertdialog" aria-modal="true" aria-live="polite">
                        <LoadingBars className="delete-progress-loader" />
                        <h3>{t('detail.deleteProgressTitle')}</h3>
                        <p>{t('detail.deleteProgressSubtitle')}</p>
                        <div className="delete-progress-steps">
                            {deleteSteps.map(step => {
                                const done = stageRank(deleteStage) > stageRank(step.stage)
                                const active = stageRank(deleteStage) === stageRank(step.stage)
                                return (
                                    <div
                                        key={step.stage}
                                        className={`delete-progress-step ${done ? 'done' : ''} ${active ? 'active' : ''}`.trim()}
                                    >
                                        <span className="delete-progress-dot">{done ? '✓' : active ? '●' : '○'}</span>
                                        <span>{step.label}</span>
                                    </div>
                                )
                            })}
                        </div>
                        {deleteGithubWarning && (
                            <div className="delete-progress-warning">
                                {t('detail.deleteGithubWarning', { error: deleteGithubWarning })}
                            </div>
                        )}
                    </div>
                </div>
            )}
        </div>
    )
}

export default SkillDetailPage
