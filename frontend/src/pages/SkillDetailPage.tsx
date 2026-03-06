import { useState, useEffect } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { Skill, fetchSkill, getDownloadUrl, deleteSkill } from '../services/api'

function SkillDetailPage() {
    const { id } = useParams<{ id: string }>()
    const navigate = useNavigate()
    const [skill, setSkill] = useState<Skill | null>(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState('')
    const [deleting, setDeleting] = useState(false)

    useEffect(() => {
        if (!id) return
        const load = async () => {
            try {
                const data = await fetchSkill(Number(id))
                setSkill(data)
            } catch (err) {
                setError(err instanceof Error ? err.message : 'Failed to load')
            } finally {
                setLoading(false)
            }
        }
        load()
    }, [id])

    const formatSize = (bytes: number): string => {
        if (bytes < 1024) return bytes + ' B'
        if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
        return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
    }

    const formatDate = (dateStr: string): string => {
        return new Date(dateStr).toLocaleString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
        })
    }

    if (loading) {
        return (
            <div className="detail-page page-enter">
                <div className="empty-state">
                    <div className="loading-spinner" style={{ width: 40, height: 40 }}></div>
                    <p style={{ marginTop: 16 }}>Loading...</p>
                </div>
            </div>
        )
    }

    if (error || !skill) {
        return (
            <div className="detail-page page-enter">
                <div className="empty-state">
                    <div className="icon">😕</div>
                    <p>{error || 'Skill not found'}</p>
                    <Link to="/" className="btn btn-primary" style={{ marginTop: 16 }}>
                        Back to Home
                    </Link>
                </div>
            </div>
        )
    }

    const thumbnailSrc = skill.thumbnail_url || 'data:image/svg+xml,' + encodeURIComponent(
        `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="400" viewBox="0 0 400 400">
      <defs><linearGradient id="g" x1="0%" y1="0%" x2="100%" y2="100%"><stop offset="0%" style="stop-color:#3b82f6"/><stop offset="100%" style="stop-color:#8b5cf6"/></linearGradient></defs>
      <rect fill="url(#g)" width="400" height="400"/>
      <text x="200" y="220" text-anchor="middle" fill="white" font-size="96" font-family="sans-serif">${skill.name.charAt(0).toUpperCase()}</text>
    </svg>`
    )

    return (
        <div className="detail-page page-enter">
            <Link to="/" className="detail-back-btn">
                ← Back to Home
            </Link>

            <div className="detail-header">
                <img
                    className="detail-thumb"
                    src={thumbnailSrc}
                    alt={skill.name}
                />

                <div className="detail-info-wrapper">
                    <div className="detail-info-card glass-card">
                        <h3>ℹ️ Information</h3>
                        <h1>{skill.name}</h1>
                        <div className="detail-meta">
                            {skill.category && (
                                <div className="detail-meta-item">
                                    <span className="tag">{skill.category}</span>
                                </div>
                            )}
                            <div className="detail-meta-item">
                                <span>👤</span>
                                <span>{skill.author || 'Anonymous'}</span>
                            </div>
                            <div className="detail-meta-item">
                                <span>📄</span>
                                <span>{skill.file_name} ({formatSize(skill.file_size)})</span>
                            </div>
                        </div>

                        <div style={{ display: 'flex', gap: 8, marginTop: 'auto', alignSelf: 'flex-start' }}>
                            <a
                                href={getDownloadUrl(skill.id)}
                                className="btn btn-success btn-lg"
                                onClick={() => {
                                    setSkill(prev => prev ? { ...prev, downloads: prev.downloads + 1 } : prev)
                                }}
                            >
                                Download
                            </a>
                            <button
                                className="btn btn-lg"
                                style={{ background: '#ef4444', color: '#fff', border: 'none' }}
                                disabled={deleting}
                                onClick={async () => {
                                    if (!confirm('Are you sure you want to delete this skill? This will also remove it from GitHub.')) return
                                    setDeleting(true)
                                    try {
                                        await deleteSkill(skill.id)
                                        navigate('/')
                                    } catch (err) {
                                        alert(err instanceof Error ? err.message : 'Delete failed')
                                    } finally {
                                        setDeleting(false)
                                    }
                                }}
                            >
                                {deleting ? 'Deleting...' : 'Delete'}
                            </button>
                        </div>
                    </div>

                    <div className="detail-info-card glass-card">
                        <h3>📊 Stats</h3>
                        <div className="detail-meta-vertical">
                            <div className="detail-meta-item">
                                <span>📥</span>
                                <span>Total Downloads: <strong style={{ color: 'var(--text-primary)' }}>{skill.downloads}</strong></span>
                            </div>
                            <div className="detail-meta-item">
                                <span>🕐</span>
                                <span>Updated: {formatDate(skill.created_at)}</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div className="detail-content-sections">
                {skill.ai_feedback && (
                    <div className="detail-section glass-card">
                        <h3 className="section-title">🤖 AI Description</h3>
                        <div
                            className={`ai-review-status ${skill.ai_approved ? 'approved' : 'rejected'}`}
                            style={{ marginTop: 0 }}
                        >
                            <h4>{skill.ai_approved ? '✅ Approved' : '❌ Risk Detected'}</h4>
                            <p>{skill.ai_feedback}</p>
                        </div>
                    </div>
                )}

                {skill.description && (
                    <div className="detail-section glass-card">
                        <h3 className="section-title">📝 Description</h3>
                        <div className="readme-content">
                            <p>{skill.description}</p>
                        </div>
                    </div>
                )}
            </div>
        </div>
    )
}

export default SkillDetailPage
