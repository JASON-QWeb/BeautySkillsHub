import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useAuth } from '../../contexts/AuthContext'
import { useI18n } from '../../i18n/I18nProvider'
import { RESOURCE_TYPES, Skill, SkillReviewStatusResponse, fetchSkillReviewStatus, retrySkillReview, uploadSkill, getDownloadUrl } from '../../services/api'
import { useDialog } from '../../contexts/DialogContext'

const MAX_TAGS = 5

type ReviewStatus = 'idle' | 'pending' | 'approved' | 'rejected'

function UploadPage() {
    const [searchParams] = useSearchParams()
    const { t } = useI18n()
    const { user } = useAuth()
    const { showAlert } = useDialog()

    const initialType = searchParams.get('type') || 'skill'
    const [resourceType, setResourceType] = useState(initialType in RESOURCE_TYPES ? initialType : 'skill')
    const [name, setName] = useState('')
    const [description, setDescription] = useState('')
    const [tagInput, setTagInput] = useState('')
    const [tagItems, setTagItems] = useState<string[]>([])

    const [uploadMode, setUploadMode] = useState<'file' | 'folder'>('file')
    const [file, setFile] = useState<File | null>(null)
    const [folderFiles, setFolderFiles] = useState<File[]>([])
    const [thumbnail, setThumbnail] = useState<File | null>(null)
    const [thumbnailPreviewUrl, setThumbnailPreviewUrl] = useState('')

    const [step, setStep] = useState(1)
    const [uploading, setUploading] = useState(false)
    const [reviewStatus, setReviewStatus] = useState<ReviewStatus>('idle')
    const [feedback, setFeedback] = useState('')
    const [reviewMeta, setReviewMeta] = useState<SkillReviewStatusResponse | null>(null)
    const [uploadedSkill, setUploadedSkill] = useState<Skill | null>(null)

    const fileInputRef = useRef<HTMLInputElement>(null)
    const folderInputRef = useRef<HTMLInputElement>(null)
    const thumbInputRef = useRef<HTMLInputElement>(null)
    const reviewPollRef = useRef<ReturnType<typeof setInterval> | null>(null)

    useEffect(() => {
        if (!thumbnail) {
            setThumbnailPreviewUrl('')
            return
        }

        const url = URL.createObjectURL(thumbnail)
        setThumbnailPreviewUrl(url)
        return () => URL.revokeObjectURL(url)
    }, [thumbnail])

    const selectedFileHint = useMemo(() => {
        if (uploadMode === 'file') return file?.name || ''
        if (folderFiles.length === 0) return ''
        return t('upload.filesSelected', { count: folderFiles.length })
    }, [uploadMode, file, folderFiles, t])

    const hasPayload = uploadMode === 'file' ? !!file : folderFiles.length > 0

    const compactTitle = useMemo(() => {
        if (uploadedSkill?.name) return uploadedSkill.name
        return name.trim() || t('upload.titlePlaceholder')
    }, [uploadedSkill, name, t])

    const compactThumb = uploadedSkill?.thumbnail_url || thumbnailPreviewUrl

    const handleFolderSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
        const files = Array.from(e.target.files || [])
        setFolderFiles(files)

        if (files.length > 0 && !name.trim()) {
            const relativePath = files[0].webkitRelativePath || ''
            const folderName = relativePath.split('/')[0]
            if (folderName) setName(folderName)
        }
    }

    const addTag = (rawTag: string) => {
        const normalized = rawTag.trim().replace(/,+$/g, '')
        if (!normalized) return

        if (tagItems.length >= MAX_TAGS) {
            setTagInput('')
            return
        }

        const alreadyExists = tagItems.some(tag => tag.toLowerCase() === normalized.toLowerCase())
        if (alreadyExists) {
            setTagInput('')
            return
        }

        setTagItems(prev => [...prev, normalized])
        setTagInput('')
    }

    const removeTag = (value: string) => {
        setTagItems(prev => prev.filter(tag => tag !== value))
    }

    const clearReviewPolling = () => {
        if (reviewPollRef.current) {
            clearInterval(reviewPollRef.current)
            reviewPollRef.current = null
        }
    }

    const mapReviewPhaseText = (phase: string) => {
        if (phase === 'security') return '安全性审核中...'
        if (phase === 'functional') return '功能性审核中...'
        if (phase === 'finalizing') return '结果归档中...'
        return '审核排队中，请稍候...'
    }

    const mapReviewFileIcon = (status: string) => {
        if (status === 'passed') return '✅'
        if (status === 'failed') return '❌'
        if (status === 'running') return '⏳'
        return '⌛'
    }

    const applyReviewStatus = (status: SkillReviewStatusResponse) => {
        setReviewMeta(status)
        if (status.status === 'queued' || status.status === 'running') {
            setReviewStatus('pending')
            if (status.progress?.current_file) {
                setFeedback(`${mapReviewPhaseText(status.phase)} ${status.progress.current_file}`)
            } else {
                setFeedback(status.feedback || mapReviewPhaseText(status.phase))
            }
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

    const startReviewPolling = (skillID: number) => {
        clearReviewPolling()
        const tick = async () => {
            try {
                const status = await fetchSkillReviewStatus(skillID, resourceType)
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

    const resetState = () => {
        clearReviewPolling()
        setStep(1)
        setUploading(false)
        setReviewStatus('idle')
        setFeedback('')
        setReviewMeta(null)
        setUploadedSkill(null)
        setName('')
        setDescription('')
        setTagInput('')
        setTagItems([])
        setFile(null)
        setFolderFiles([])
        setThumbnail(null)
    }

    useEffect(() => {
        return () => {
            clearReviewPolling()
        }
    }, [])

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()

        if (!name.trim()) {
            await showAlert(t('upload.enterTitle'))
            return
        }

        if (!hasPayload) {
            await showAlert(uploadMode === 'file' ? t('upload.chooseFile') : t('upload.chooseFolder'))
            return
        }

        const formData = new FormData()
        formData.append('name', name.trim())
        formData.append('description', description.trim())
        formData.append('resource_type', resourceType)
        formData.append('author', user?.username || 'Anonymous')
        formData.append('upload_mode', uploadMode)
        formData.append('tags', tagItems.join(','))

        if (uploadMode === 'file' && file) {
            formData.append('file', file)
        }

        if (uploadMode === 'folder') {
            for (const f of folderFiles) {
                formData.append('files', f)
                const rel = f.webkitRelativePath || f.name
                const parts = rel.split('/')
                const innerPath = parts.length > 1 ? parts.slice(1).join('/') : parts[0]
                formData.append('file_paths', innerPath)
            }
        }

        if (thumbnail) {
            formData.append('thumbnail', thumbnail)
        }

        setUploading(true)
        setStep(2)
        setReviewStatus('pending')
        setFeedback('')

        try {
            const result = await uploadSkill(formData)
            setUploadedSkill(result.skill)

            // Non-skill types are auto-published (no AI review needed)
            if (resourceType !== 'skill') {
                setReviewStatus('approved')
                setFeedback(t('upload.reviewApprovedFallback'))
                setStep(3)
                clearReviewPolling()
            } else {
                setReviewStatus('pending')
                setFeedback(result.feedback || '审核排队中，请稍候...')
                setReviewMeta(null)
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
            await retrySkillReview(uploadedSkill.id)
            startReviewPolling(uploadedSkill.id)
        } catch (err) {
            setReviewStatus('rejected')
            setFeedback(err instanceof Error ? err.message : '重新审核失败')
        }
    }

    return (
        <div className="upload-page page-enter">
            <div className="upload-topbar upload-topbar-left">
                <Link to={`/resource/${resourceType}`} className="upload-back-btn">← {t('upload.back')}</Link>
            </div>

            <div className="upload-steps single-line">
                <div className={`step ${step >= 1 ? 'active' : ''}`}>
                    <div className="step-circle">↥</div>
                    <span className="step-label">{t('upload.stepUser')}</span>
                </div>
                <div className={`step-connector ${step >= 2 ? 'active' : ''}`} />
                <div className={`step ${step >= 2 ? 'active' : ''}`}>
                    <div className="step-circle">◎</div>
                    <span className="step-label">{t('upload.stepAi')}</span>
                </div>
                <div className={`step-connector ${step >= 3 ? 'active' : ''}`} />
                <div className={`step ${step >= 3 ? 'active' : ''}`}>
                    <div className="step-circle">◍</div>
                    <span className="step-label">{t('upload.stepHuman')}</span>
                </div>
            </div>

            <div className="upload-flow-layout">
                <section className={`upload-stage-card ${step === 1 ? 'expanded' : 'compact'}`}>
                    {step === 1 ? (
                        <form className="upload-form-card glass-card" onSubmit={handleSubmit}>
                            <header>
                                <h1>{t('upload.title')}</h1>
                                <p>{t('upload.subtitle')}</p>
                            </header>

                            <div className="upload-type-tabs">
                                {Object.entries(RESOURCE_TYPES).map(([key, value]) => (
                                    <button
                                        key={key}
                                        type="button"
                                        className={resourceType === key ? 'active' : ''}
                                        onClick={() => setResourceType(key)}
                                        disabled={uploading}
                                    >
                                        {value.label}
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
                                        addTag(tagInput)
                                    }}
                                    placeholder={t('upload.tagsPlaceholder')}
                                    disabled={uploading || tagItems.length >= MAX_TAGS}
                                />
                                <small className="upload-tag-counter">
                                    {t('upload.tagHint')} {tagItems.length}/{MAX_TAGS}
                                </small>
                                {tagItems.length >= MAX_TAGS && (
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
                                    className={uploadMode === 'file' ? 'active' : ''}
                                    onClick={() => {
                                        setUploadMode('file')
                                        setFolderFiles([])
                                    }}
                                    disabled={uploading}
                                >
                                    {t('upload.modeFile')}
                                </button>
                                <button
                                    type="button"
                                    className={uploadMode === 'folder' ? 'active' : ''}
                                    onClick={() => {
                                        setUploadMode('folder')
                                        setFile(null)
                                    }}
                                    disabled={uploading}
                                >
                                    {t('upload.modeFolder')}
                                </button>
                            </div>

                            <div className="upload-dropzones">
                                <button type="button" className="upload-dropzone" onClick={() => thumbInputRef.current?.click()}>
                                    <strong>{t('upload.uploadThumbnail')}</strong>
                                    <span>{t('upload.thumbnailHint')}</span>
                                    {thumbnail && <em>{thumbnail.name}</em>}
                                    <input
                                        ref={thumbInputRef}
                                        type="file"
                                        accept="image/*"
                                        onChange={e => setThumbnail(e.target.files?.[0] || null)}
                                        disabled={uploading}
                                    />
                                </button>

                                {uploadMode === 'file' ? (
                                    <button type="button" className="upload-dropzone upload-dropzone-highlight" onClick={() => fileInputRef.current?.click()}>
                                        <strong>{t('upload.uploadFile')}</strong>
                                        <span>{t('upload.fileHint')}</span>
                                        {selectedFileHint && <em>{selectedFileHint}</em>}
                                        <input
                                            ref={fileInputRef}
                                            type="file"
                                            onChange={e => setFile(e.target.files?.[0] || null)}
                                            disabled={uploading}
                                        />
                                    </button>
                                ) : (
                                    <button type="button" className="upload-dropzone upload-dropzone-highlight" onClick={() => folderInputRef.current?.click()}>
                                        <strong>{t('upload.uploadFolder')}</strong>
                                        <span>{t('upload.folderHint')}</span>
                                        {selectedFileHint && <em>{selectedFileHint}</em>}
                                        <input
                                            ref={folderInputRef}
                                            type="file"
                                            webkitdirectory=""
                                            directory=""
                                            multiple
                                            onChange={handleFolderSelect}
                                            disabled={uploading}
                                        />
                                    </button>
                                )}
                            </div>

                            <button className="upload-submit-btn" disabled={uploading || !name.trim() || !hasPayload}>
                                {uploading ? t('upload.reviewing') : t('upload.submit')}
                            </button>
                        </form>
                    ) : (
                        <div className="upload-compact-card glass-card">
                            <div className="upload-compact-title">{t('upload.userUploadCompactTitle')}</div>
                            <div className="upload-compact-content">
                                {compactThumb ? (
                                    <img className="upload-compact-thumb" src={compactThumb} alt={compactTitle} />
                                ) : (
                                    <div className="upload-compact-thumb upload-compact-fallback">S</div>
                                )}
                                <div className="upload-compact-meta" style={{ flex: 1, minWidth: 0 }}>
                                    <strong style={{ whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', display: 'block' }}>{compactTitle}</strong>
                                    <span>{RESOURCE_TYPES[resourceType]?.label || resourceType}</span>
                                    {uploadedSkill?.tags && (
                                        <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginTop: 8 }}>
                                            {uploadedSkill.tags.split(',').filter(Boolean).map(tag => (
                                                <span key={tag} className="upload-tag-chip" style={{ fontSize: '0.7rem', padding: '2px 6px', margin: 0 }}>
                                                    {tag}
                                                </span>
                                            ))}
                                        </div>
                                    )}
                                    {uploadedSkill?.file_name && (
                                        <div style={{ marginTop: 8 }}>
                                            <a
                                                href={getDownloadUrl(uploadedSkill.id, resourceType)}
                                                target="_blank"
                                                rel="noreferrer"
                                                className="btn btn-secondary"
                                                style={{ padding: '4px 8px', fontSize: '0.75rem', display: 'inline-flex', alignItems: 'center', gap: 4 }}
                                                onClick={e => e.stopPropagation()}
                                            >
                                                📄 {uploadedSkill.file_name} (预览)
                                            </a>
                                        </div>
                                    )}
                                </div>
                            </div>
                        </div>
                    )}
                </section>

                <section className={`upload-stage-card ${step === 2 ? 'expanded' : step > 2 ? 'compact' : 'placeholder'}`}>
                    {step < 2 ? (
                        <div className="upload-placeholder-card glass-card">
                            <strong>{t('upload.waitForStep')}</strong>
                            <p>{t('upload.stepAi')}</p>
                        </div>
                    ) : step === 2 ? (
                        <div className="upload-review-card glass-card">
                            <h2>{t('upload.aiReviewPanelTitle')}</h2>

                            {reviewStatus === 'pending' && (
                                <div className="review-status pending">
                                    <strong>{t('upload.reviewPendingTitle')}</strong>
                                    <p>{feedback || t('upload.reviewPendingText')}</p>
                                </div>
                            )}

                            {reviewStatus === 'approved' && (
                                <div className="review-status approved">
                                    <strong>{t('upload.reviewApprovedTitle')}</strong>
                                    <p>{feedback || t('upload.reviewApprovedFallback')}</p>
                                </div>
                            )}

                            {reviewStatus === 'rejected' && (
                                <div className="review-status rejected">
                                    <strong>{t('upload.reviewRejectedTitle')}</strong>
                                    <p>{feedback || t('upload.reviewRejectedFallback')}</p>
                                </div>
                            )}

                            {reviewMeta?.progress && (
                                <div className="review-file-progress">
                                    <div className="review-file-progress-head">
                                        <span>已检查 {reviewMeta.progress.completed_files}/{reviewMeta.progress.total_files}</span>
                                        {reviewMeta.progress.current_file && (
                                            <span>当前：{reviewMeta.progress.current_file}</span>
                                        )}
                                    </div>
                                    <ul className="review-file-list">
                                        {reviewMeta.progress.files.map(item => (
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
                            )}

                            <div className="review-actions">
                                {reviewStatus === 'rejected' && (
                                    <>
                                        {reviewMeta?.can_retry ? (
                                            <button type="button" className="btn btn-primary" onClick={handleRetryReview}>
                                                重新触发审核（剩余 {reviewMeta.retry_remaining} 次）
                                            </button>
                                        ) : (
                                            <button type="button" className="btn btn-secondary" onClick={() => setStep(1)}>
                                                {t('upload.backToEdit')}
                                            </button>
                                        )}
                                    </>
                                )}

                                <button type="button" className="btn btn-secondary" onClick={resetState}>
                                    {t('upload.resetForm')}
                                </button>
                            </div>
                        </div>
                    ) : (
                        <div className="upload-compact-card glass-card">
                            <div className="upload-compact-title">{t('upload.aiReviewCompactTitle')}</div>
                            <div className={`review-status ${reviewStatus === 'approved' ? 'approved' : 'rejected'}`}>
                                <strong>
                                    {reviewStatus === 'approved' ? t('upload.reviewApprovedTitle') : t('upload.reviewRejectedTitle')}
                                </strong>
                                <p>{feedback || (reviewStatus === 'approved' ? t('upload.reviewApprovedFallback') : t('upload.reviewRejectedFallback'))}</p>
                            </div>
                        </div>
                    )}
                </section>

                <section className={`upload-stage-card ${step === 3 ? 'expanded' : 'placeholder'}`}>
                    {step < 3 ? (
                        <div className="upload-placeholder-card glass-card">
                            <strong>{t('upload.waitForStep')}</strong>
                            <p>{resourceType === 'skill' ? t('upload.stepHuman') : t('upload.stepPublish')}</p>
                        </div>
                    ) : resourceType !== 'skill' ? (
                        <div className="upload-review-card upload-human-card glass-card">
                            <h2>{t('upload.publishedTitle')}</h2>
                            <div className="review-status approved">
                                <strong>{t('upload.publishedSuccess')}</strong>
                                <p>{t('upload.publishedDescription')}</p>
                            </div>

                            <div className="review-actions">
                                {uploadedSkill && (
                                    <Link
                                        to={`/resource/${resourceType}`}
                                        className="btn btn-primary"
                                    >
                                        {t('upload.viewResource')}
                                    </Link>
                                )}
                                <button type="button" className="btn btn-secondary" onClick={resetState}>
                                    {t('upload.uploadAnother')}
                                </button>
                            </div>
                        </div>
                    ) : (
                        <div className="upload-review-card upload-human-card glass-card">
                            <h2>{t('upload.humanReviewPanelTitle')}</h2>
                            <div className="review-status pending">
                                <strong>{t('upload.stepHuman')}</strong>
                                <p>{t('upload.humanReviewWaiting')}</p>
                            </div>

                            <div className="review-actions">
                                {uploadedSkill && (
                                    <Link
                                        to={`/review/${uploadedSkill.id}`}
                                        state={{ resourceType }}
                                        className="btn btn-primary"
                                    >
                                        {t('upload.humanReviewConfirm')}
                                    </Link>
                                )}
                                <button type="button" className="btn btn-secondary" onClick={resetState}>
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

export default UploadPage
