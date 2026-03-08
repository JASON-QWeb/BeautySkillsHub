import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '../../../contexts/AuthContext'
import { useDialog } from '../../../contexts/DialogContext'
import { useI18n } from '../../../i18n/I18nProvider'
import { parseMarkdown } from '../../detail/shared/markdown'
import { fetchSkill, updateResourceFromUpload, uploadContentImage, uploadSkill } from '../../../services/api'
import { addTagItem, MAX_UPLOAD_TAGS, normalizeTagList, serializeTagList } from '../shared/tagInput'

// Constants for cropping
const THUMB_TARGET_WIDTH = 1200
const THUMB_TARGET_HEIGHT = 800
const THUMB_TARGET_RATIO = THUMB_TARGET_WIDTH / THUMB_TARGET_HEIGHT
const THUMB_MIN_WIDTH = 600
const THUMB_MIN_HEIGHT = 400
const THUMB_MAX_SIZE_BYTES = 5 * 1024 * 1024
const THUMB_RATIO_TOLERANCE = 0.03
const THUMB_PREVIEW_WIDTH = 600
const THUMB_PREVIEW_HEIGHT = Math.round((THUMB_PREVIEW_WIDTH * THUMB_TARGET_HEIGHT) / THUMB_TARGET_WIDTH)

interface ImageMeta {
    width: number
    height: number
}

interface ThumbnailCropState {
    file: File
    previewUrl: string
    meta: ImageMeta
    zoom: number
    centerX: number
    centerY: number
}

interface CropArea {
    sx: number
    sy: number
    sw: number
    sh: number
}

function clamp(value: number, min: number, max: number) {
    return Math.min(max, Math.max(min, value))
}

function isThumbnailSizeReasonable(meta: ImageMeta) {
    const ratio = meta.width / meta.height
    const ratioDiff = Math.abs(ratio - THUMB_TARGET_RATIO)
    return ratioDiff <= THUMB_RATIO_TOLERANCE && meta.width >= THUMB_MIN_WIDTH && meta.height >= THUMB_MIN_HEIGHT
}

function computeCropArea(meta: ImageMeta, zoom: number, centerX: number, centerY: number): CropArea {
    const safeZoom = Math.max(1, zoom)
    let baseWidth = meta.width
    let baseHeight = meta.height
    const sourceRatio = meta.width / meta.height
    if (sourceRatio > THUMB_TARGET_RATIO) {
        baseWidth = meta.height * THUMB_TARGET_RATIO
    } else {
        baseHeight = meta.width / THUMB_TARGET_RATIO
    }

    const cropWidth = baseWidth / safeZoom
    const cropHeight = baseHeight / safeZoom
    const centerPxX = clamp(centerX * meta.width, cropWidth / 2, meta.width - cropWidth / 2)
    const centerPxY = clamp(centerY * meta.height, cropHeight / 2, meta.height - cropHeight / 2)

    return {
        sx: centerPxX - cropWidth / 2,
        sy: centerPxY - cropHeight / 2,
        sw: cropWidth,
        sh: cropHeight,
    }
}

function loadImageObject(url: string) {
    return new Promise<HTMLImageElement>((resolve, reject) => {
        const img = new Image()
        img.onload = () => resolve(img)
        img.onerror = () => reject(new Error('load image failed'))
        img.src = url
    })
}

function readImageMeta(file: File) {
    return new Promise<ImageMeta>((resolve, reject) => {
        const url = URL.createObjectURL(file)
        const img = new Image()
        img.onload = () => {
            resolve({ width: img.naturalWidth, height: img.naturalHeight })
            URL.revokeObjectURL(url)
        }
        img.onerror = () => {
            reject(new Error('invalid image'))
            URL.revokeObjectURL(url)
        }
        img.src = url
    })
}

function buildThumbnailFileName(name: string) {
    const base = name.replace(/\.[^.]+$/, '').trim() || 'thumbnail'
    const safeBase = base.replace(/[^a-zA-Z0-9_-]+/g, '-').replace(/-+/g, '-').replace(/^-|-$/g, '') || 'thumbnail'
    return `${safeBase}_thumb.jpg`
}

async function makeCroppedThumbnail(cropState: ThumbnailCropState) {
    const image = await loadImageObject(cropState.previewUrl)
    const area = computeCropArea(cropState.meta, cropState.zoom, cropState.centerX, cropState.centerY)
    const canvas = document.createElement('canvas')
    canvas.width = THUMB_TARGET_WIDTH
    canvas.height = THUMB_TARGET_HEIGHT
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('canvas unavailable')

    ctx.imageSmoothingEnabled = true
    ctx.imageSmoothingQuality = 'high'
    ctx.drawImage(image, area.sx, area.sy, area.sw, area.sh, 0, 0, THUMB_TARGET_WIDTH, THUMB_TARGET_HEIGHT)

    const blob = await new Promise<Blob>((resolve, reject) => {
        canvas.toBlob(
            value => {
                if (!value) {
                    reject(new Error('thumbnail blob generation failed'))
                    return
                }
                resolve(value)
            },
            'image/jpeg',
            0.92,
        )
    })

    return new File([blob], buildThumbnailFileName(cropState.file.name), {
        type: 'image/jpeg',
        lastModified: Date.now(),
    })
}

function drawCropPreview(canvas: HTMLCanvasElement, image: HTMLImageElement, cropState: ThumbnailCropState) {
    const area = computeCropArea(cropState.meta, cropState.zoom, cropState.centerX, cropState.centerY)
    canvas.width = THUMB_PREVIEW_WIDTH
    canvas.height = THUMB_PREVIEW_HEIGHT
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    ctx.clearRect(0, 0, canvas.width, canvas.height)
    ctx.imageSmoothingEnabled = true
    ctx.imageSmoothingQuality = 'high'
    ctx.drawImage(image, area.sx, area.sy, area.sw, area.sh, 0, 0, canvas.width, canvas.height)

    // Optional: Draw a subtle grid or rule of thirds
    ctx.strokeStyle = 'rgba(255, 255, 255, 0.2)'
    ctx.lineWidth = 1
    ctx.strokeRect(0, 0, canvas.width, canvas.height)
}

function ToolsUploadPage() {
    const navigate = useNavigate()
    const [searchParams] = useSearchParams()
    const { user } = useAuth()
    const { showAlert } = useDialog()
    const { t } = useI18n()
    const editId = Number(searchParams.get('edit') || 0)
    const isEditMode = Number.isInteger(editId) && editId > 0

    const [name, setName] = useState('')
    const [tagInput, setTagInput] = useState('')
    const [tagItems, setTagItems] = useState<string[]>([])
    const [description, setDescription] = useState('')
    const [packageFile, setPackageFile] = useState<File | null>(null)
    const [thumbnail, setThumbnail] = useState<File | null>(null)
    const [thumbnailPreviewUrl, setThumbnailPreviewUrl] = useState('')
    const [existingThumbnailUrl, setExistingThumbnailUrl] = useState('')
    const [thumbnailCrop, setThumbnailCrop] = useState<ThumbnailCropState | null>(null)
    const [cropApplying, setCropApplying] = useState(false)
    const [prefillLoading, setPrefillLoading] = useState(false)

    const [uploading, setUploading] = useState(false)
    const [imageUploading, setImageUploading] = useState(false)

    const imageInputRef = useRef<HTMLInputElement>(null)
    const packageInputRef = useRef<HTMLInputElement>(null)
    const thumbnailInputRef = useRef<HTMLInputElement>(null)
    const cropPreviewRef = useRef<HTMLCanvasElement>(null)
    const cropImageRef = useRef<HTMLImageElement | null>(null)

    const previewHtml = useMemo(() => parseMarkdown(description), [description])
    const displayThumbnailPreviewUrl = thumbnailPreviewUrl || existingThumbnailUrl

    useEffect(() => {
        if (!isEditMode) {
            setPrefillLoading(false)
            return
        }

        if (!user) {
            void showAlert('请先登录后再编辑工具资源')
            navigate('/resource/tools', { replace: true })
            return
        }

        const controller = new AbortController()
        setPrefillLoading(true)

        void fetchSkill(editId, 'tools', { signal: controller.signal })
            .then(async skill => {
                const canEdit = (skill.user_id ? skill.user_id === user.id : false)
                    || (skill.author || '').trim().toLowerCase() === user.username.trim().toLowerCase()
                if (!canEdit) {
                    await showAlert('仅上传者本人可以编辑该工具')
                    navigate(`/resource/tools/${editId}`, { replace: true })
                    return
                }

                if (skill.has_pending_revision) {
                    await showAlert('当前已有更新在审核中，请等待本次 Review 完成')
                    navigate(`/resource/tools/${editId}`, { replace: true })
                    return
                }

                setName(skill.name || '')
                setTagInput('')
                setTagItems(normalizeTagList(skill.tags || ''))
                setDescription(skill.description || '')
                setExistingThumbnailUrl(skill.thumbnail_url || '')
            })
            .catch(async err => {
                if ((err as Error).name === 'AbortError') return
                await showAlert(err instanceof Error ? err.message : '加载待编辑资源失败')
                navigate('/resource/tools', { replace: true })
            })
            .finally(() => {
                if (!controller.signal.aborted) {
                    setPrefillLoading(false)
                }
            })

        return () => {
            controller.abort()
        }
    }, [editId, isEditMode, navigate, showAlert, user])

    useEffect(() => {
        if (!thumbnail) {
            setThumbnailPreviewUrl('')
            return
        }
        const url = URL.createObjectURL(thumbnail)
        setThumbnailPreviewUrl(url)
        return () => URL.revokeObjectURL(url)
    }, [thumbnail])

    useEffect(() => {
        if (!thumbnailCrop) {
            cropImageRef.current = null
            return
        }

        let cancelled = false
        void loadImageObject(thumbnailCrop.previewUrl)
            .then(img => {
                if (cancelled) return
                cropImageRef.current = img
                if (cropPreviewRef.current) {
                    drawCropPreview(cropPreviewRef.current, img, thumbnailCrop)
                }
            })
            .catch(() => {
                cropImageRef.current = null
            })

        return () => {
            cancelled = true
        }
    }, [thumbnailCrop?.previewUrl])

    useEffect(() => {
        if (!thumbnailCrop || !cropImageRef.current || !cropPreviewRef.current) return
        drawCropPreview(cropPreviewRef.current, cropImageRef.current, thumbnailCrop)
    }, [thumbnailCrop?.zoom, thumbnailCrop?.centerX, thumbnailCrop?.centerY])

    const commitTagInput = () => {
        setTagItems(prev => addTagItem(prev, tagInput))
        setTagInput('')
    }

    const removeTag = (value: string) => {
        setTagItems(prev => prev.filter(tag => tag !== value))
    }

    const formatFileSize = (bytes: number) => {
        if (bytes === 0) return '0 B'
        const k = 1024
        const sizes = ['B', 'KB', 'MB', 'GB']
        const i = Math.floor(Math.log(bytes) / Math.log(k))
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    }

    const handleThumbnailSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const selected = e.target.files?.[0] || null
        if (!selected) {
            setThumbnail(null)
            return
        }
        if (selected.size > THUMB_MAX_SIZE_BYTES) {
            await showAlert(t('upload.thumbnailTooLarge'))
            e.target.value = ''
            return
        }

        try {
            const meta = await readImageMeta(selected)
            if (isThumbnailSizeReasonable(meta)) {
                setExistingThumbnailUrl('')
                setThumbnail(selected)
                return
            }

            const previewUrl = URL.createObjectURL(selected)
            setExistingThumbnailUrl('')
            setThumbnail(null)
            setThumbnailCrop({
                file: selected,
                previewUrl,
                meta,
                zoom: 1,
                centerX: 0.5,
                centerY: 0.5,
            })
        } catch {
            await showAlert(t('upload.thumbnailInvalid'))
            setThumbnail(null)
            e.target.value = ''
        }
    }

    const handleApplyThumbnailCrop = async () => {
        if (!thumbnailCrop || cropApplying) return
        setCropApplying(true)
        try {
            const cropped = await makeCroppedThumbnail(thumbnailCrop)
            setExistingThumbnailUrl('')
            setThumbnail(cropped)
            setThumbnailCrop(null)
            setCropApplying(false)
        } catch {
            setCropApplying(false)
            await showAlert(t('upload.thumbnailCropFailed'))
        }
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!name.trim() || !description.trim()) {
            await showAlert('请填写标题与正文内容')
            return
        }

        const formData = new FormData()
        formData.append('name', name.trim())
        formData.append('description', description)
        formData.append('resource_type', 'tools')
        formData.append('author', user?.username || 'Anonymous')
        const finalTagItems = addTagItem(tagItems, tagInput)
        formData.append('tags', serializeTagList(finalTagItems))
        formData.append('upload_mode', packageFile ? 'file' : 'metadata')

        if (finalTagItems.length !== tagItems.length || tagInput.trim()) {
            setTagItems(finalTagItems)
            setTagInput('')
        }

        if (packageFile) {
            formData.append('file', packageFile)
        }
        if (thumbnail) {
            formData.append('thumbnail', thumbnail)
        }

        setUploading(true)
        try {
            if (isEditMode) {
                const updated = await updateResourceFromUpload(editId, formData, 'tools')
                navigate(`/resource/tools/${updated.id}`, { replace: true })
            } else {
                const result = await uploadSkill(formData)
                navigate(`/resource/tools/${result.skill.id}`, { replace: true })
            }
        } catch (err) {
            await showAlert(err instanceof Error ? err.message : t('upload.uploadFailed'))
        } finally {
            setUploading(false)
        }
    }

    const handleInsertImage = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const image = e.target.files?.[0]
        if (!image) return

        setImageUploading(true)
        try {
            const url = await uploadContentImage(image)
            setDescription(prev => {
                const trimmed = prev.trimEnd()
                const prefix = trimmed ? `${trimmed}\n\n` : ''
                return `${prefix}![${image.name}](${url})\n`
            })
        } catch (err) {
            await showAlert(err instanceof Error ? err.message : '图片上传失败')
        } finally {
            setImageUploading(false)
            e.target.value = ''
        }
    }

    return (
        <div className="upload-page page-enter">
            <div className="upload-topbar upload-topbar-left">
                <Link to="/resource/tools" className="upload-back-btn">← {t('upload.back')}</Link>
            </div>

            <div className="glass-card" style={{ padding: '32px' }}>
                <header style={{ marginBottom: '32px' }}>
                    <h2 style={{ fontSize: '2rem', marginBottom: '8px' }}>{isEditMode ? '编辑工具 (Tools)' : '发布工具 (Tools)'}</h2>
                    <p style={{ color: 'var(--text-secondary)', fontSize: '1.1rem' }}>
                        {isEditMode ? '已为你预填历史内容，确认修改后提交即可。' : '发布实用的工具插件、脚本或压缩包，提供详细的使用说明。'}
                    </p>
                </header>

                <form className="upload-modern-layout" onSubmit={handleSubmit}>
                    <div className="modern-form-section">
                        <div className="modern-field">
                            <label className="modern-label">{t('upload.fieldTitle')}</label>
                            <input
                                className="modern-input"
                                value={name}
                                onChange={e => setName(e.target.value)}
                                placeholder={t('upload.titlePlaceholder')}
                                disabled={uploading || prefillLoading}
                                required
                            />
                        </div>

                        <div className="modern-field">
                            <label className="modern-label">{t('upload.tags')}</label>
                            <input
                                className="modern-input"
                                value={tagInput}
                                onChange={e => setTagInput(e.target.value)}
                                onKeyDown={e => {
                                    if (e.key !== 'Enter') return
                                    e.preventDefault()
                                    commitTagInput()
                                }}
                                placeholder={t('upload.tagsPlaceholder')}
                                disabled={uploading || prefillLoading || tagItems.length >= MAX_UPLOAD_TAGS}
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
                                            disabled={uploading || prefillLoading}
                                            aria-label={`${t('upload.removeTag')}: ${tag}`}
                                        >
                                            {tag} ×
                                        </button>
                                    ))}
                                </div>
                            )}
                        </div>

                        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
                            <div className="modern-field">
                                <label className="modern-label">工具包附件 (可选)</label>
                                {!packageFile ? (
                                    <div className="modern-dropzone" onClick={() => !prefillLoading && packageInputRef.current?.click()}>
                                        <span className="modern-dropzone-icon">📦</span>
                                        <span style={{ fontSize: '0.9rem' }}>上传 .zip, .rar 等格式</span>
                                        <input
                                            ref={packageInputRef}
                                            type="file"
                                            accept=".zip,.tar,.gz,.tgz,.rar,.7z,.xz,.bz2,.tar.gz"
                                            onChange={e => setPackageFile(e.target.files?.[0] || null)}
                                            disabled={uploading || prefillLoading}
                                            style={{ display: 'none' }}
                                        />
                                    </div>
                                ) : (
                                    <div className="modern-file-preview">
                                        <span className="modern-dropzone-icon" style={{ fontSize: '1.2rem' }}>📄</span>
                                        <div className="modern-file-info">
                                            <span className="modern-file-name">{packageFile.name}</span>
                                            <span className="modern-file-size">{formatFileSize(packageFile.size)}</span>
                                        </div>
                                        <button type="button" className="modern-remove-btn" onClick={() => setPackageFile(null)} disabled={uploading || prefillLoading}>✕</button>
                                    </div>
                                )}
                            </div>

                            <div className="modern-field">
                                <label className="modern-label">封面缩略图</label>
                                {!displayThumbnailPreviewUrl ? (
                                    <div className="modern-dropzone" onClick={() => !prefillLoading && thumbnailInputRef.current?.click()}>
                                        <span className="modern-dropzone-icon">🖼️</span>
                                        <span style={{ fontSize: '0.9rem' }}>点击上传展示图</span>
                                        <input
                                            ref={thumbnailInputRef}
                                            type="file"
                                            accept="image/*"
                                            onChange={handleThumbnailSelect}
                                            disabled={uploading || prefillLoading}
                                            style={{ display: 'none' }}
                                        />
                                    </div>
                                ) : (
                                    <div className="modern-file-preview">
                                        <img 
                                            src={displayThumbnailPreviewUrl}
                                            alt="preview" 
                                            style={{ width: 40, height: 40, borderRadius: 6, objectFit: 'cover' }} 
                                        />
                                        <div className="modern-file-info">
                                            <span className="modern-file-name">{thumbnail?.name || '已预填封面图片'}</span>
                                            <span className="modern-file-size">{thumbnail ? formatFileSize(thumbnail.size) : '可保持不变或重新上传'}</span>
                                        </div>
                                        <button
                                            type="button"
                                            className="modern-remove-btn"
                                            onClick={() => {
                                                setThumbnail(null)
                                                setExistingThumbnailUrl('')
                                            }}
                                            disabled={uploading || prefillLoading}
                                        >
                                            ✕
                                        </button>
                                    </div>
                                )}
                            </div>
                        </div>

                        <div className="modern-field">
                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                <label className="modern-label">详细使用说明 (Markdown)</label>
                                <button
                                    type="button"
                                    className="btn btn-secondary btn-sm"
                                    onClick={() => imageInputRef.current?.click()}
                                    disabled={uploading || imageUploading || prefillLoading}
                                    style={{ borderRadius: '8px' }}
                                >
                                    {imageUploading ? '上传中...' : '🖼️ 插入图片'}
                                </button>
                                <input
                                    ref={imageInputRef}
                                    type="file"
                                    accept="image/*"
                                    onChange={e => { void handleInsertImage(e) }}
                                    style={{ display: 'none' }}
                                />
                            </div>
                            <textarea
                                className="modern-textarea"
                                value={description}
                                onChange={e => setDescription(e.target.value)}
                                placeholder="# 工具说明\n\n在此输入详细的功能说明、安装步骤等..."
                                disabled={uploading || prefillLoading}
                                required
                            />
                        </div>

                        <button className="modern-submit-btn" disabled={uploading || prefillLoading || !name.trim() || !description.trim()}>
                            {uploading ? t('upload.reviewing') : (isEditMode ? '保存工具修改' : '立即发布工具')}
                        </button>
                    </div>

                    <div className="modern-preview-sticky">
                        <div className="modern-preview-header">实时预览</div>
                        <div className="modern-preview-container">
                            <div
                                className="detail-description markdown-body"
                                dangerouslySetInnerHTML={{ __html: previewHtml || '<p style="color:var(--text-muted)">暂无内容预览...</p>' }}
                            />
                        </div>
                    </div>
                </form>
            </div>

            {thumbnailCrop && (
                <div className="modern-crop-overlay">
                    <div className="modern-crop-card">
                        <div className="modern-crop-main">
                            <div className="modern-crop-canvas-container">
                                <canvas ref={cropPreviewRef} className="modern-crop-canvas" />
                            </div>
                        </div>
                        <div className="modern-crop-sidebar">
                            <h3 className="modern-crop-title">裁剪封面图</h3>
                            <p className="modern-crop-desc">
                                请调整缩略图的显示区域，建议将核心内容居中展示。
                            </p>
                            
                            <div className="modern-crop-control-group">
                                <div className="modern-crop-control">
                                    <label>缩放 <span>{thumbnailCrop.zoom.toFixed(2)}x</span></label>
                                    <input 
                                        type="range" 
                                        className="modern-crop-slider" 
                                        min="1" max="2.5" step="0.01" 
                                        value={thumbnailCrop.zoom}
                                        onChange={e => setThumbnailCrop({...thumbnailCrop, zoom: Number(e.target.value)})}
                                    />
                                </div>
                                <div className="modern-crop-control">
                                    <label>水平位置</label>
                                    <input 
                                        type="range" 
                                        className="modern-crop-slider" 
                                        min="0" max="1" step="0.01" 
                                        value={thumbnailCrop.centerX}
                                        onChange={e => setThumbnailCrop({...thumbnailCrop, centerX: Number(e.target.value)})}
                                    />
                                </div>
                                <div className="modern-crop-control">
                                    <label>垂直位置</label>
                                    <input 
                                        type="range" 
                                        className="modern-crop-slider" 
                                        min="0" max="1" step="0.01" 
                                        value={thumbnailCrop.centerY}
                                        onChange={e => setThumbnailCrop({...thumbnailCrop, centerY: Number(e.target.value)})}
                                    />
                                </div>
                            </div>

                            <div className="modern-crop-actions">
                                <button className="modern-btn-confirm" onClick={handleApplyThumbnailCrop} disabled={cropApplying}>
                                    {cropApplying ? '处理中...' : '完成裁剪'}
                                </button>
                                <button className="modern-btn-cancel" onClick={() => setThumbnailCrop(null)} disabled={cropApplying}>
                                    取消
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </div>
    )
}

export default ToolsUploadPage
