import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '../../../contexts/AuthContext'
import { useDialog } from '../../../contexts/DialogContext'
import { useI18n } from '../../../i18n/I18nProvider'
import { RESOURCE_TYPES } from '../../../services/api'
import { Skill, SkillReviewStatusResponse, fetchSkill, fetchSkillReadme, fetchSkillReviewStatus, retrySkillReview, updateResourceFromUpload, uploadSkill } from '../../../services/api'
import FlowStepIcon from '../../../components/FlowStepIcon'
import { addTagItem, MAX_UPLOAD_TAGS, normalizeTagList, serializeTagList } from '../shared/tagInput'

type RulesInputMode = 'file' | 'paste'
type ReviewStatus = 'idle' | 'pending' | 'approved' | 'rejected'

function RulesUploadPage() {
    const navigate = useNavigate()
    const [searchParams] = useSearchParams()
    const { t } = useI18n()
    const { user } = useAuth()
    const { showAlert } = useDialog()
    const editId = Number(searchParams.get('edit') || 0)
    const isEditMode = Number.isInteger(editId) && editId > 0

    const [name, setName] = useState('')
    const [description, setDescription] = useState('')
    const [tagInput, setTagInput] = useState('')
    const [tagItems, setTagItems] = useState<string[]>([])
    const [mode, setMode] = useState<RulesInputMode>('file')
    const [file, setFile] = useState<File | null>(null)
    const [markdownContent, setMarkdownContent] = useState('')
    const [uploading, setUploading] = useState(false)
    const [prefillLoading, setPrefillLoading] = useState(false)
    const [step, setStep] = useState(1)
    const [reviewStatus, setReviewStatus] = useState<ReviewStatus>('idle')
    const [feedback, setFeedback] = useState('')
    const [reviewMeta, setReviewMeta] = useState<SkillReviewStatusResponse | null>(null)
    const [uploadedSkill, setUploadedSkill] = useState<Skill | null>(null)

    const fileInputRef = useRef<HTMLInputElement>(null)
    const reviewPollRef = useRef<ReturnType<typeof setInterval> | null>(null)
    const reviewedUploadTypes: Array<'skill' | 'rules'> = ['skill', 'rules']

    const canSubmit = useMemo(() => {
        if (!name.trim()) return false
        if (isEditMode) return true
        if (mode === 'file') return !!file
        return markdownContent.trim().length > 0
    }, [isEditMode, name, mode, file, markdownContent])

    const clearReviewPolling = () => {
        if (reviewPollRef.current) {
            clearInterval(reviewPollRef.current)
            reviewPollRef.current = null
        }
    }

    useEffect(() => {
        return () => clearReviewPolling()
    }, [])

    useEffect(() => {
        if (!isEditMode) {
            setPrefillLoading(false)
            return
        }

        if (!user) {
            void showAlert('请先登录后再编辑 Rules 资源')
            navigate('/resource/rules', { replace: true })
            return
        }

        let cancelled = false
        setPrefillLoading(true)

        void Promise.all([
            fetchSkill(editId, 'rules'),
            fetchSkillReadme(editId, 'rules').catch(() => ''),
        ])
            .then(async ([skill, readme]) => {
                if (cancelled) return

                const canEdit = (skill.user_id ? skill.user_id === user.id : false)
                    || (skill.author || '').trim().toLowerCase() === user.username.trim().toLowerCase()
                if (!canEdit) {
                    await showAlert('仅上传者本人可以编辑该 Rules 资源')
                    if (!cancelled) {
                        navigate(`/resource/rules/${editId}`, { replace: true })
                    }
                    return
                }

                if (skill.has_pending_revision) {
                    await showAlert('当前已有更新在审核中，请等待本次 Review 完成')
                    if (!cancelled) {
                        navigate(`/resource/rules/${editId}`, { replace: true })
                    }
                    return
                }

                setName(skill.name || '')
                setDescription(skill.description || '')
                setTagInput('')
                setTagItems(normalizeTagList(skill.tags || ''))
                if (readme.trim()) {
                    setMode('paste')
                    setMarkdownContent(readme)
                }
                setFile(null)
            })
            .catch(async err => {
                if (cancelled) return
                await showAlert(err instanceof Error ? err.message : '加载待编辑资源失败')
                if (!cancelled) {
                    navigate('/resource/rules', { replace: true })
                }
            })
            .finally(() => {
                if (!cancelled) {
                    setPrefillLoading(false)
                }
            })

        return () => {
            cancelled = true
        }
    }, [editId, isEditMode, navigate, showAlert, user])

    const commitTagInput = () => {
        setTagItems(prev => addTagItem(prev, tagInput))
        setTagInput('')
    }

    const removeTag = (value: string) => {
        setTagItems(prev => prev.filter(tag => tag !== value))
    }
    const applyReviewStatus = (status: SkillReviewStatusResponse) => {
        setReviewMeta(status)
        if (status.status === 'queued' || status.status === 'running') {
            setReviewStatus('pending')
            setFeedback(status.feedback || t('upload.reviewPendingText'))
            return
        }
        if (status.status === 'passed') {
            setReviewStatus('approved')
            setFeedback(status.feedback || t('upload.reviewApprovedFallback'))
            setStep(3)
            clearReviewPolling()
            return
        }
        setReviewStatus('rejected')
        setFeedback(status.feedback || t('upload.reviewRejectedFallback'))
        clearReviewPolling()
    }

    const startReviewPolling = (resourceID: number) => {
        clearReviewPolling()
        const tick = async () => {
            try {
                const status = await fetchSkillReviewStatus(resourceID, 'rules')
                applyReviewStatus(status)
            } catch (err) {
                setReviewStatus('rejected')
                setFeedback(err instanceof Error ? err.message : t('upload.uploadFailed'))
                clearReviewPolling()
            }
        }

        void tick()
        reviewPollRef.current = setInterval(() => {
            void tick()
        }, 2000)
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!canSubmit) return

        if (mode === 'file' && file) {
            const lower = file.name.toLowerCase()
            if (!lower.endsWith('.md') && !lower.endsWith('.txt')) {
                await showAlert('Rules 仅支持 .md 或 .txt 文件')
                return
            }
        }

        const formData = new FormData()
        formData.append('name', name.trim())
        formData.append('description', description.trim())
        formData.append('resource_type', 'rules')
        formData.append('author', user?.username || 'Anonymous')
        const finalTagItems = addTagItem(tagItems, tagInput)
        formData.append('tags', serializeTagList(finalTagItems))
        formData.append('upload_mode', mode === 'paste' ? 'paste' : 'file')

        if (finalTagItems.length !== tagItems.length || tagInput.trim()) {
            setTagItems(finalTagItems)
            setTagInput('')
        }

        if (mode === 'file' && file) {
            formData.append('file', file)
        }
        if (mode === 'paste') {
            formData.append('markdown_content', markdownContent)
            formData.append('file_name', `${name.trim() || 'RULES'}.md`)
        }

        setUploading(true)
        setStep(2)
        setReviewStatus('pending')
        setFeedback('')
        setReviewMeta(null)

        try {
            if (isEditMode) {
                const updated = await updateResourceFromUpload(editId, formData, 'rules')
                navigate(`/resource/rules/${updated.id}`, { replace: true })
            } else {
                const result = await uploadSkill(formData)
                setUploadedSkill(result.skill)
                setFeedback(result.feedback || t('upload.reviewPendingText'))
                if (result.skill?.id) {
                    startReviewPolling(result.skill.id)
                }
            }
        } catch (err) {
            setReviewStatus('rejected')
            setFeedback(err instanceof Error ? err.message : t('upload.uploadFailed'))
            clearReviewPolling()
        } finally {
            setUploading(false)
        }
    }

    const handleRetryReview = async () => {
        if (!uploadedSkill) return
        try {
            setReviewStatus('pending')
            setFeedback('重新触发审核中...')
            setReviewMeta(null)
            await retrySkillReview(uploadedSkill.id, 'rules')
            startReviewPolling(uploadedSkill.id)
        } catch (err) {
            setReviewStatus('rejected')
            setFeedback(err instanceof Error ? err.message : '重新审核失败')
        }
    }

    const resetForm = () => {
        clearReviewPolling()
        setName('')
        setDescription('')
        setTagInput('')
        setTagItems([])
        setFile(null)
        setMarkdownContent('')
        setMode('file')
        setStep(1)
        setReviewStatus('idle')
        setFeedback('')
        setReviewMeta(null)
        setUploadedSkill(null)
        if (fileInputRef.current) {
            fileInputRef.current.value = ''
        }
    }

    return (
        <div className="upload-page page-enter">
            <div className="upload-topbar upload-topbar-left">
                <Link to="/resource/rules" className="upload-back-btn">← {t('upload.back')}</Link>
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
                <section className={`upload-stage-card ${step === 1 ? 'expanded' : 'compact'}`}>
                    {step === 1 ? (
                        <form className="upload-form-card glass-card" onSubmit={handleSubmit}>
                            <header>
                                <h1>{isEditMode ? '更新 Rules' : '上传 Rules'}</h1>
                                <p>{isEditMode ? '可只修改元数据，也可重新上传规则文件或直接修改 Markdown；提交后会进入新的 Review。' : '支持上传 .md/.txt 或直接粘贴 Markdown 内容。'}</p>
                            </header>

                            {prefillLoading && (
                                <div className="upload-placeholder-card" style={{ marginBottom: 16 }}>
                                    <p>{t('common.loadingResources')}</p>
                                </div>
                            )}

                            <div className="upload-type-tabs">
                                {reviewedUploadTypes.map((key) => (
                                    <button
                                        key={key}
                                        type="button"
                                        className={key === 'rules' ? 'active' : ''}
                                        onClick={() => {
                                            if (key === 'rules') return
                                            navigate(`/resource/${key}/upload`)
                                        }}
                                        disabled={uploading}
                                    >
                                        {RESOURCE_TYPES[key].label}
                                    </button>
                                ))}
                            </div>

                            <label className="upload-field">
                                <span>{t('upload.fieldTitle')}</span>
                                <input
                                    value={name}
                                    onChange={e => setName(e.target.value)}
                                    placeholder={t('upload.titlePlaceholder')}
                                    disabled={uploading}
                                    required
                                />
                            </label>

                            <label className="upload-field">
                                <span>{t('upload.description')}</span>
                                <textarea
                                    value={description}
                                    onChange={e => setDescription(e.target.value)}
                                    placeholder={t('upload.descriptionPlaceholder')}
                                    disabled={uploading}
                                />
                            </label>

                            <label className="upload-field">
                                <span>{t('upload.tags')}</span>
                                <input
                                    value={tagInput}
                                    onChange={e => setTagInput(e.target.value)}
                                    onKeyDown={e => {
                                        if (e.key !== 'Enter') return
                                        e.preventDefault()
                                        commitTagInput()
                                    }}
                                    placeholder={t('upload.tagsPlaceholder')}
                                    disabled={uploading || tagItems.length >= MAX_UPLOAD_TAGS}
                                />
                                <small className="upload-tag-counter">
                                    {t('upload.tagHint')} {tagItems.length}/{MAX_UPLOAD_TAGS}
                                </small>
                                {tagItems.length >= MAX_UPLOAD_TAGS && (
                                    <small className="upload-tag-limit">{t('upload.tagLimit')}</small>
                                )}
                                {tagItems.length > 0 && (
                                    <div className="upload-tag-list">
                                        {tagItems.map(tag => (
                                            <button
                                                key={tag}
                                                type="button"
                                                className="upload-tag-chip"
                                                onClick={() => removeTag(tag)}
                                                disabled={uploading}
                                                aria-label={`${t('upload.removeTag')}: ${tag}`}
                                            >
                                                {tag} ×
                                            </button>
                                        ))}
                                    </div>
                                )}
                            </label>

                            <div className="upload-mode-toggle">
                                <button
                                    type="button"
                                    className={mode === 'file' ? 'active' : ''}
                                    onClick={() => {
                                        setMode('file')
                                        setMarkdownContent('')
                                    }}
                                    disabled={uploading}
                                >
                                    上传文件
                                </button>
                                <button
                                    type="button"
                                    className={mode === 'paste' ? 'active' : ''}
                                    onClick={() => {
                                        setMode('paste')
                                        setFile(null)
                                    }}
                                    disabled={uploading}
                                >
                                    粘贴 Markdown
                                </button>
                            </div>

                            {mode === 'file' ? (
                                <div className="upload-dropzones">
                                    <button type="button" className="upload-dropzone upload-dropzone-highlight" onClick={() => fileInputRef.current?.click()}>
                                        <strong>上传规则文件</strong>
                                        <span>仅支持 .md / .txt</span>
                                        {file && <em>{file.name}</em>}
                                        <input
                                            ref={fileInputRef}
                                            type="file"
                                            accept=".md,.txt,text/plain,text/markdown"
                                            onChange={e => setFile(e.target.files?.[0] || null)}
                                            disabled={uploading}
                                        />
                                    </button>
                                </div>
                            ) : (
                                <label className="upload-field">
                                    <span>规则内容（Markdown）</span>
                                    <textarea
                                        value={markdownContent}
                                        onChange={e => setMarkdownContent(e.target.value)}
                                        placeholder="# Rules\n\n- rule 1\n- rule 2"
                                        disabled={uploading}
                                        style={{ minHeight: 260 }}
                                    />
                                </label>
                            )}

                            <button className="upload-submit-btn" disabled={prefillLoading || uploading || !canSubmit}>
                                {uploading ? t('upload.reviewing') : (isEditMode ? '提交更新并进入 Review' : t('upload.submit'))}
                            </button>
                        </form>
                    ) : (
                        <div className="upload-compact-card glass-card">
                            <div className="upload-compact-title">{t('upload.userUploadCompactTitle')}</div>
                            <div className="upload-compact-content">
                                <div className="upload-compact-thumb upload-compact-fallback">R</div>
                                <div className="upload-compact-meta" style={{ flex: 1, minWidth: 0 }}>
                                    <strong>{uploadedSkill?.name || name}</strong>
                                    <span>Rules</span>
                                </div>
                            </div>
                        </div>
                    )}
                </section>

                <section className={`upload-stage-card ${step < 2 ? 'placeholder' : 'expanded'}`}>
                    {step < 2 ? (
                        <div className="upload-placeholder-card glass-card">
                            <strong>{t('upload.waitForStep')}</strong>
                            <p>{t('upload.stepAi')}</p>
                        </div>
                    ) : (
                        <div className="upload-review-card glass-card">
                            <h2>{t('upload.aiReviewPanelTitle')}</h2>
                            <div className={`review-status ${reviewStatus === 'approved' ? 'approved' : reviewStatus === 'rejected' ? 'rejected' : 'pending'}`}>
                                <strong>
                                    {reviewStatus === 'approved'
                                        ? t('upload.reviewApprovedTitle')
                                        : reviewStatus === 'rejected'
                                            ? t('upload.reviewRejectedTitle')
                                            : t('upload.reviewPendingTitle')}
                                </strong>
                                <p>{feedback || t('upload.reviewPendingText')}</p>
                            </div>

                            {reviewMeta?.progress && (
                                <div className="review-file-progress">
                                    <div className="review-file-progress-head">
                                        <span>已检查 {reviewMeta.progress.completed_files}/{reviewMeta.progress.total_files}</span>
                                    </div>
                                </div>
                            )}

                            <div className="review-actions">
                                {reviewStatus === 'rejected' && reviewMeta?.can_retry && (
                                    <button type="button" className="btn btn-primary" onClick={handleRetryReview}>
                                        重新触发审核（剩余 {reviewMeta.retry_remaining} 次）
                                    </button>
                                )}
                                {reviewStatus !== 'approved' && (
                                    <button type="button" className="btn btn-secondary" onClick={resetForm}>
                                        {t('upload.resetForm')}
                                    </button>
                                )}
                            </div>
                        </div>
                    )}
                </section>

                <section className={`upload-stage-card ${step === 3 ? 'expanded' : 'placeholder'}`}>
                    {step < 3 ? (
                        <div className="upload-placeholder-card glass-card">
                            <strong>{t('upload.waitForStep')}</strong>
                            <p>{t('upload.stepHuman')}</p>
                        </div>
                    ) : (
                        <div className="upload-review-card upload-human-card glass-card">
                            <h2>{t('upload.humanReviewPanelTitle')}</h2>
                            <div className="review-status pending">
                                <strong>{t('upload.stepHuman')}</strong>
                                <p>{t('upload.humanReviewWaiting')}</p>
                            </div>
                            <small style={{ color: 'var(--warning)' }}>{t('detail.humanReviewNeedOtherUser')}</small>
                            <div className="review-actions">
                                <button type="button" className="btn btn-secondary" onClick={resetForm}>
                                    {t('upload.uploadAnother')}
                                </button>
                            </div>
                        </div>
                    )}
                </section>
            </div>
        </div>
    )
}

export default RulesUploadPage
