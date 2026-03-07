import { useEffect, useMemo, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { useAuth } from '../../contexts/AuthContext'
import { useI18n } from '../../i18n/I18nProvider'
import { Skill, deleteSkill, favoriteSkill, fetchSkill, fetchSkillReadme, getDownloadUrl, likeSkill, submitHumanReview, trackDownloadHit, unfavoriteSkill, updateSkill } from '../../services/api'
import { useDialog } from '../../contexts/DialogContext'
import { formatDate, formatSize } from './formatters'

function formatDownloads(downloads: number) {
    if (downloads >= 1000) return `${(downloads / 1000).toFixed(1)}K`
    return `${downloads}`
}

function escapeHtml(str: string): string {
    return str
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;')
}

function parseMarkdown(text: string) {
    if (!text) return ''

    // Temporarily replace code blocks to protect their contents
    const codeBlocks: string[] = []
    let raw = text.replace(/```([\s\S]*?)```/g, (_match, code) => {
        codeBlocks.push(code)
        return `__CODE_BLOCK_${codeBlocks.length - 1}__`
    })

    // Extract inline code before escaping
    const inlineCodes: string[] = []
    raw = raw.replace(/`([^`]+)`/g, (_match, code) => {
        inlineCodes.push(code)
        return `__INLINE_CODE_${inlineCodes.length - 1}__`
    })

    // Extract markdown links before escaping
    const links: { text: string; url: string }[] = []
    raw = raw.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_match, linkText, url) => {
        links.push({ text: linkText, url })
        return `__LINK_${links.length - 1}__`
    })

    // Escape all HTML in the remaining text to prevent XSS
    let html = escapeHtml(raw)

    // Restore inline code (escaped)
    html = html.replace(/__INLINE_CODE_(\d+)__/g, (_match, index) => {
        return `<code>${escapeHtml(inlineCodes[Number(index)])}</code>`
    })

    // Restore links (sanitize href to only allow http/https)
    html = html.replace(/__LINK_(\d+)__/g, (_match, index) => {
        const link = links[Number(index)]
        const safeUrl = /^https?:\/\//i.test(link.url) ? escapeHtml(link.url) : '#'
        return `<a href="${safeUrl}" target="_blank" rel="noopener noreferrer">${escapeHtml(link.text)}</a>`
    })

    // Bold
    html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')

    // Headers
    html = html.replace(/^### (.+)$/gm, '<h3>$1</h3>')
    html = html.replace(/^## (.+)$/gm, '<h2>$1</h2>')
    html = html.replace(/^# (.+)$/gm, '<h2>$1</h2>')

    // Lists
    html = html.replace(/^- (.+)$/gm, '<li>$1</li>')
    html = html.replace(/^\* (.+)$/gm, '<li>$1</li>')
    html = html.replace(/((?:<li>.*<\/li>\n?)+)/g, '<ul>$1</ul>')

    // Paragraphs
    html = html.replace(/\n{2,}/g, '</p><p>')
    html = '<p>' + html + '</p>'
    html = html.replace(/<p>\s*<(h[23]|ul)/g, '<$1')
    html = html.replace(/<\/(h[23]|ul)>\s*<\/p>/g, '</$1>')
    html = html.replace(/<p>\s*<\/p>/g, '')

    // Restore code blocks (escaped)
    html = html.replace(/__CODE_BLOCK_(\d+)__/g, (_match, index) => {
        return `<pre><code>${escapeHtml(codeBlocks[Number(index)])}</code></pre>`
    })

    return html
}

function normalizeSummaryLine(line: string) {
    return line
        .replace(/^(安全性|安全|功能性|功能|security|function)\s*[:：-]?\s*/i, '')
        .trim()
}

function extractAISummaryLines(skill: Skill | null): string[] {
    if (!skill) return []

    const lines = (skill.ai_description || '')
        .split('\n')
        .map(line => normalizeSummaryLine(line))
        .filter(Boolean)

    if (lines.length > 0) return lines.slice(0, 3)

    const fallback = (skill.ai_feedback || '')
        .split(/[\n。！？]/)
        .map(line => line.trim())
        .filter(Boolean)

    return fallback.slice(0, 3)
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

const readmeGlobalCache = new Map<number, string>()

function SkillDetailPage() {
    const { id } = useParams<{ id: string }>()
    const location = useLocation()
    const navigate = useNavigate()
    const { t } = useI18n()
    const { user } = useAuth()
    const { showAlert, showConfirm } = useDialog()
    const initialResourceType = (location.state as { resourceType?: string } | null)?.resourceType || ''

    const [skill, setSkill] = useState<Skill | null>(null)
    const [readmeContent, setReadmeContent] = useState<string>('')
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState('')
    const [editing, setEditing] = useState(false)
    const [savingEdit, setSavingEdit] = useState(false)
    const [editName, setEditName] = useState('')
    const [editDescription, setEditDescription] = useState('')
    const [deleting, setDeleting] = useState(false)
    const [copied, setCopied] = useState(false)
    const [reviewingHuman, setReviewingHuman] = useState(false)
    const [liking, setLiking] = useState(false)
    const [likesCount, setLikesCount] = useState(0)
    const [userLiked, setUserLiked] = useState(false)
    const [favoriting, setFavoriting] = useState(false)
    const [favorited, setFavorited] = useState(false)

    useEffect(() => {
        if (!id) return

        const load = async () => {
            setLoading(true)
            const numericId = Number(id)
            try {
                // If we already have this readme in memory, use it instead of refetching
                let readmePromise: Promise<string>
                if (readmeGlobalCache.has(numericId)) {
                    readmePromise = Promise.resolve(readmeGlobalCache.get(numericId)!)
                } else {
                    readmePromise = fetchSkillReadme(numericId, initialResourceType)
                        .then(content => {
                            readmeGlobalCache.set(numericId, content)
                            return content
                        })
                        .catch(() => '')
                }

                const [data, readme] = await Promise.all([
                    fetchSkill(numericId, initialResourceType),
                    readmePromise
                ])
                
                setSkill(data)
                setReadmeContent(readme)
                setEditing(false)
                setError('')
            } catch (err) {
                setError(err instanceof Error ? err.message : t('detail.loadFailed'))
            } finally {
                setLoading(false)
            }
        }

        load()
    }, [id, initialResourceType, t])

    useEffect(() => {
        if (!skill) return
        setLikesCount(skill.likes_count || 0)
        setUserLiked(!!skill.user_liked)
        setFavorited(!!skill.favorited)
    }, [skill])

    const installCommand = useMemo(() => {
        if (!skill) return ''
        const slug = skill.name.toLowerCase().replace(/\s+/g, '-')
        return `npx skills add https://github.com/skillshub/community --skill ${slug}`
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

    const canHumanReview = useMemo(() => {
        return !!user && !!skill && skill.ai_approved && !canManage && humanReviewStatus === 'pending'
    }, [user, skill, canManage, humanReviewStatus])

    const canLike = useMemo(() => {
        if (!user || !skill) return false
        if (!skill.ai_approved) return false
        if (skill.user_id && skill.user_id > 0) return skill.user_id !== user.id
        return (skill.author || '').trim().toLowerCase() !== user.username.trim().toLowerCase()
    }, [user, skill])

    const canFavorite = useMemo(() => {
        return !!user && !!skill && skill.ai_approved
    }, [user, skill])

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

    const aiSummaryLines = useMemo(() => extractAISummaryLines(skill), [skill])
    const safetyChecks = useMemo(() => extractSafetyChecks(skill), [skill])

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
                    <h1 className="detail-title">{skill.name}</h1>

                    {skill.tags && (
                        <div className="detail-tags">
                            {skill.tags.split(',').map(tag => (
                                <span key={tag} className="detail-tag">{tag.trim()}</span>
                            ))}
                        </div>
                    )}

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

                    {/* AI Review summary (2-3 lines) */}
                    <div className={`detail-ai-review ${skill.ai_approved ? 'approved' : 'rejected'}`}>
                        <span className="detail-ai-icon">◉</span>
                        <div className="detail-ai-text">
                            <div className="detail-ai-label">{t('detail.aiReviewSummary')}</div>
                            <div className="detail-ai-summary-lines">
                                {aiSummaryLines.length > 0 ? aiSummaryLines.map((line, idx) => (
                                    <p key={`${idx}-${line}`}>{line}</p>
                                )) : (
                                    <p>{t('detail.aiReviewFallback')}</p>
                                )}
                            </div>
                        </div>
                    </div>

                    {/* Content / markdown */}
                    <div className="detail-content-header">
                        <span>{skill.resource_type?.toUpperCase() || 'SKILL'}.md</span>
                    </div>

                    <div
                        className="detail-description markdown-body"
                        dangerouslySetInnerHTML={{
                            __html: parseMarkdown(readmeContent || skill.description || t('detail.descriptionFallback'))
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
                        <p className="detail-sidebar-value">skillshub/community</p>
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
                                        setLiking(true)
                                        try {
                                            const result = await likeSkill(skill.id, skill.resource_type)
                                            setLikesCount(result.likes_count || likesCount)
                                            setUserLiked(!!result.liked)
                                        } catch (err) {
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
                                        setFavoriting(true)
                                        try {
                                            if (favorited) {
                                                await unfavoriteSkill(skill.id, skill.resource_type)
                                                setFavorited(false)
                                            } else {
                                                await favoriteSkill(skill.id, skill.resource_type)
                                                setFavorited(true)
                                            }
                                        } catch (err) {
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

                    <div className="detail-sidebar-section">
                        <div className="detail-sidebar-label">{t('detail.humanReview')}</div>
                        <div className={`detail-human-review-status ${humanReviewStatus}`}>
                            {humanReviewStatus === 'approved' && t('detail.humanReviewApproved')}
                            {humanReviewStatus === 'pending' && t('detail.humanReviewPending')}
                            {humanReviewStatus === 'rejected' && t('detail.humanReviewRejected')}
                        </div>
                        {skill.human_reviewer && (
                            <small className="detail-sidebar-value">
                                {t('detail.reviewedBy', { user: skill.human_reviewer })}
                            </small>
                        )}
                        {canManage && humanReviewStatus === 'pending' && (
                            <small className="detail-sidebar-value">{t('detail.humanReviewNeedOtherUser')}</small>
                        )}
                        {canHumanReview && (
                            <button
                                type="button"
                                className="btn btn-primary"
                                onClick={async () => {
                                    setReviewingHuman(true)
                                    try {
                                        const updated = await submitHumanReview(skill.id, true)
                                        setSkill(updated)
                                    } catch (err) {
                                        await showAlert(err instanceof Error ? err.message : t('detail.humanReviewFailed'))
                                    } finally {
                                        setReviewingHuman(false)
                                    }
                                }}
                                disabled={reviewingHuman}
                            >
                                {reviewingHuman ? t('detail.humanReviewSubmitting') : t('detail.confirmHumanReview')}
                            </button>
                        )}
                    </div>

                    <div className="detail-sidebar-section detail-actions">
                        <a href={downloadPath} className="btn btn-primary">
                            {isPublished ? t('detail.download') : t('detail.downloadPending')}
                        </a>
                        {canManage && (
                            <>
                                <button
                                    className="btn btn-secondary"
                                    onClick={() => {
                                        if (!editing) {
                                            setEditName(skill.name)
                                            setEditDescription(skill.description || '')
                                        }
                                        setEditing(prev => !prev)
                                    }}
                                    disabled={savingEdit || deleting}
                                >
                                    {editing ? t('detail.cancelEdit') : t('detail.edit')}
                                </button>
                                <button
                                    className="btn btn-secondary"
                                    onClick={async () => {
                                        if (!(await showConfirm(t('detail.confirmDelete')))) return
                                        setDeleting(true)
                                        try {
                                            await deleteSkill(skill.id, skill.resource_type)
                                            navigate(`/resource/${skill.resource_type || 'skill'}`)
                                        } catch (err) {
                                            await showAlert(err instanceof Error ? err.message : t('detail.deleteFailed'))
                                        } finally {
                                            setDeleting(false)
                                        }
                                    }}
                                    disabled={deleting || savingEdit}
                                >
                                    {deleting ? t('detail.deleting') : t('detail.delete')}
                                </button>
                            </>
                        )}
                    </div>

                    {canManage && editing && (
                        <div className="detail-sidebar-section detail-owner-editor">
                            <h4>{t('detail.editResource')}</h4>
                            <label>
                                <span>{t('detail.editName')}</span>
                                <input
                                    value={editName}
                                    onChange={e => setEditName(e.target.value)}
                                    disabled={savingEdit}
                                />
                            </label>
                            <label>
                                <span>{t('detail.editDescription')}</span>
                                <textarea
                                    value={editDescription}
                                    onChange={e => setEditDescription(e.target.value)}
                                    disabled={savingEdit}
                                />
                            </label>

                            <div className="detail-owner-editor-actions">
                                <button
                                    className="btn btn-primary"
                                    onClick={async () => {
                                        const name = editName.trim()
                                        if (!name) {
                                            await showAlert(t('upload.enterTitle'))
                                            return
                                        }

                                        setSavingEdit(true)
                                        try {
                                            const updated = await updateSkill(skill.id, {
                                                name,
                                                description: editDescription,
                                            }, skill.resource_type)
                                            setSkill(updated)
                                            setEditing(false)
                                        } catch (err) {
                                            await showAlert(err instanceof Error ? err.message : t('detail.updateFailed'))
                                        } finally {
                                            setSavingEdit(false)
                                        }
                                    }}
                                    disabled={savingEdit}
                                >
                                    {savingEdit ? t('detail.saving') : t('detail.save')}
                                </button>
                                <button
                                    className="btn btn-secondary"
                                    onClick={() => setEditing(false)}
                                    disabled={savingEdit}
                                >
                                    {t('detail.cancelEdit')}
                                </button>
                            </div>
                        </div>
                    )}

                    <div className="detail-sidebar-section">
                        <div className="detail-sidebar-label">{t('detail.fileInfo')}</div>
                        <small className="detail-sidebar-value">{skill.file_name}</small>
                        <small className="detail-sidebar-value">{formatSize(skill.file_size)}</small>
                    </div>
                </aside>
            </div>
        </div>
    )
}

export default SkillDetailPage
