import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useLocation, useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '../../contexts/AuthContext'
import { useI18n } from '../../i18n/I18nProvider'
import { RESOURCE_TYPES, Skill, SkillReviewStatusResponse, fetchSkill, fetchSkillReviewStatus, retrySkillReview, updateResourceFromUpload, uploadSkill, getDownloadUrl } from '../../services/api'
import { useDialog } from '../../contexts/DialogContext'
import FlowStepIcon from '../../components/FlowStepIcon'

const MAX_TAGS = 5
const THUMB_TARGET_WIDTH = 1200
const THUMB_TARGET_HEIGHT = 800
const THUMB_TARGET_RATIO = THUMB_TARGET_WIDTH / THUMB_TARGET_HEIGHT
const THUMB_MIN_WIDTH = 600
const THUMB_MIN_HEIGHT = 400
const THUMB_MAX_SIZE_BYTES = 5 * 1024 * 1024
const THUMB_RATIO_TOLERANCE = 0.03
const THUMB_PREVIEW_WIDTH = 540
const THUMB_PREVIEW_HEIGHT = Math.round((THUMB_PREVIEW_WIDTH * THUMB_TARGET_HEIGHT) / THUMB_TARGET_WIDTH)

type ReviewStatus = 'idle' | 'pending' | 'approved' | 'rejected'

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

interface UploadPrefillState {
    source_skill_id?: number
    name?: string
    description?: string
    tags?: string
    thumbnail_url?: string
}

interface UploadLocationState {
    resourceType?: string
    prefill?: UploadPrefillState
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
    const centerPxX = clamp(centerX * meta.width, cropWidth / 2, meta.width-cropWidth/2)
    const centerPxY = clamp(centerY * meta.height, cropHeight / 2, meta.height-cropHeight/2)

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

function normalizePrefillTags(rawTags: string) {
    const seen = new Set<string>()
    const tags: string[] = []
    rawTags
        .split(/[\n\r,]+/)
        .map(tag => tag.trim().toLowerCase())
        .filter(Boolean)
        .forEach(tag => {
            if (seen.has(tag) || tags.length >= MAX_TAGS) return
            seen.add(tag)
            tags.push(tag)
        })
    return tags
}

async function loadThumbnailFileFromURL(url: string, fallbackName: string) {
    const response = await fetch(url)
    if (!response.ok) {
        throw new Error('load prefill thumbnail failed')
    }
    const blob = await response.blob()
    const guessedName = url.split('/').pop()?.split('?')[0]?.trim() || buildThumbnailFileName(fallbackName)
    const fileName = guessedName || buildThumbnailFileName(fallbackName)
    return new File([blob], fileName, {
        type: blob.type || 'image/jpeg',
        lastModified: Date.now(),
    })
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

    ctx.strokeStyle = 'rgba(255, 255, 255, 0.45)'
    ctx.lineWidth = 1
    ctx.beginPath()
    ctx.moveTo(canvas.width / 3, 0)
    ctx.lineTo(canvas.width / 3, canvas.height)
    ctx.moveTo((canvas.width * 2) / 3, 0)
    ctx.lineTo((canvas.width * 2) / 3, canvas.height)
    ctx.moveTo(0, canvas.height / 3)
    ctx.lineTo(canvas.width, canvas.height / 3)
    ctx.moveTo(0, (canvas.height * 2) / 3)
    ctx.lineTo(canvas.width, (canvas.height * 2) / 3)
    ctx.stroke()
}

function UploadPage() {
    const navigate = useNavigate()
    const location = useLocation()
    const [searchParams] = useSearchParams()
    const { t } = useI18n()
    const { user } = useAuth()
    const { showAlert } = useDialog()

    const resourceType = 'skill'
    const editId = Number(searchParams.get('edit') || 0)
    const isEditMode = Number.isInteger(editId) && editId > 0
    const reviewedUploadTypes: Array<'skill' | 'rules'> = ['skill', 'rules']
    const [name, setName] = useState('')
    const [description, setDescription] = useState('')
    const [tagInput, setTagInput] = useState('')
    const [tagItems, setTagItems] = useState<string[]>([])

    const [uploadMode, setUploadMode] = useState<'file' | 'folder'>('file')
    const [file, setFile] = useState<File | null>(null)
    const [folderFiles, setFolderFiles] = useState<File[]>([])
    const [thumbnail, setThumbnail] = useState<File | null>(null)
    const [thumbnailCrop, setThumbnailCrop] = useState<ThumbnailCropState | null>(null)
    const [thumbnailPreviewUrl, setThumbnailPreviewUrl] = useState('')
    const [prefillThumbnailUrl, setPrefillThumbnailUrl] = useState('')
    const [cropApplying, setCropApplying] = useState(false)

    const [step, setStep] = useState(1)
    const [uploading, setUploading] = useState(false)
    const [prefillLoading, setPrefillLoading] = useState(false)
    const [reviewStatus, setReviewStatus] = useState<ReviewStatus>('idle')
    const [feedback, setFeedback] = useState('')
    const [reviewMeta, setReviewMeta] = useState<SkillReviewStatusResponse | null>(null)
    const [uploadedSkill, setUploadedSkill] = useState<Skill | null>(null)

    const fileInputRef = useRef<HTMLInputElement>(null)
    const folderInputRef = useRef<HTMLInputElement>(null)
    const thumbInputRef = useRef<HTMLInputElement>(null)
    const cropPreviewRef = useRef<HTMLCanvasElement>(null)
    const cropImageRef = useRef<HTMLImageElement | null>(null)
    const reviewPollRef = useRef<ReturnType<typeof setInterval> | null>(null)
    const prefillAppliedRef = useRef(false)

    const prefill = useMemo(
        () => ((location.state as UploadLocationState | null)?.prefill || null),
        [location.state],
    )

    useEffect(() => {
        if (!isEditMode) {
            setPrefillLoading(false)
            return
        }

        if (!user) {
            void showAlert('请先登录后再编辑 Skill 资源')
            navigate('/resource/skill', { replace: true })
            return
        }

        const controller = new AbortController()
        setPrefillLoading(true)

        void fetchSkill(editId, resourceType, { signal: controller.signal })
            .then(async skill => {
                const canEdit = (skill.user_id ? skill.user_id === user.id : false)
                    || (skill.author || '').trim().toLowerCase() === user.username.trim().toLowerCase()
                if (!canEdit) {
                    await showAlert('仅上传者本人可以编辑该 Skill')
                    navigate(`/resource/skill/${editId}`, { replace: true })
                    return
                }

                if (skill.has_pending_revision) {
                    await showAlert('当前已有更新在审核中，请等待本次 Review 完成')
                    navigate(`/resource/skill/${editId}`, { replace: true })
                    return
                }

                setName(skill.name || '')
                setDescription(skill.description || '')
                setTagItems(normalizePrefillTags(skill.tags || ''))
                setTagInput('')
                setPrefillThumbnailUrl(skill.thumbnail_url || '')
                setFile(null)
                setFolderFiles([])
            })
            .catch(async err => {
                if ((err as Error).name === 'AbortError') return
                await showAlert(err instanceof Error ? err.message : '加载待编辑资源失败')
                navigate('/resource/skill', { replace: true })
            })
            .finally(() => {
                if (!controller.signal.aborted) {
                    setPrefillLoading(false)
                }
            })

        return () => {
            controller.abort()
        }
    }, [editId, isEditMode, navigate, resourceType, showAlert, user])

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
        if (isEditMode || !prefill || prefillAppliedRef.current) return
        prefillAppliedRef.current = true

        if (prefill.name?.trim()) {
            setName(prefill.name.trim())
        }
        if (prefill.description?.trim()) {
            setDescription(prefill.description.trim())
        }
        if (prefill.tags?.trim()) {
            setTagItems(normalizePrefillTags(prefill.tags))
        }

        const thumbURL = prefill.thumbnail_url?.trim() || ''
        if (!thumbURL) return
        setPrefillThumbnailUrl(thumbURL)

        void loadThumbnailFileFromURL(thumbURL, prefill.name || 'skill')
            .then(file => setThumbnail(file))
            .catch(() => {
                // Keep preview URL fallback when image fetch is blocked.
            })
    }, [prefill])

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
    }, [thumbnailCrop?.zoom, thumbnailCrop?.centerX, thumbnailCrop?.centerY, thumbnailCrop?.meta.width, thumbnailCrop?.meta.height])

    const selectedFileHint = useMemo(() => {
        if (uploadMode === 'file') return file?.name || ''
        if (folderFiles.length === 0) return ''
        return t('upload.filesSelected', { count: folderFiles.length })
    }, [uploadMode, file, folderFiles, t])

    const hasPayload = isEditMode || (uploadMode === 'file' ? !!file : folderFiles.length > 0)

    const compactTitle = useMemo(() => {
        if (uploadedSkill?.name) return uploadedSkill.name
        return name.trim() || t('upload.titlePlaceholder')
    }, [uploadedSkill, name, t])

    const compactThumb = uploadedSkill?.thumbnail_url || thumbnailPreviewUrl || prefillThumbnailUrl

    const closeThumbnailCropModal = (clearSelection: boolean) => {
        setThumbnailCrop(prev => {
            if (prev) {
                URL.revokeObjectURL(prev.previewUrl)
            }
            return null
        })
        setCropApplying(false)
        cropImageRef.current = null
        if (clearSelection) {
            setThumbnail(null)
            setPrefillThumbnailUrl('')
            if (thumbInputRef.current) {
                thumbInputRef.current.value = ''
            }
        }
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
                setPrefillThumbnailUrl('')
                setThumbnail(selected)
                return
            }

            const previewUrl = URL.createObjectURL(selected)
            setThumbnail(null)
            setThumbnailCrop(prev => {
                if (prev) {
                    URL.revokeObjectURL(prev.previewUrl)
                }
                return {
                    file: selected,
                    previewUrl,
                    meta,
                    zoom: 1,
                    centerX: 0.5,
                    centerY: 0.5,
                }
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
            setThumbnail(cropped)
            closeThumbnailCropModal(false)
        } catch {
            setCropApplying(false)
            await showAlert(t('upload.thumbnailCropFailed'))
        }
    }

    const updateThumbnailCrop = (changes: Partial<ThumbnailCropState>) => {
        setThumbnailCrop(prev => (prev ? { ...prev, ...changes } : prev))
    }

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
        const normalized = rawTag.trim().replace(/,+$/g, '').toLowerCase()
        if (!normalized) return

        if (tagItems.length >= MAX_TAGS) {
            setTagInput('')
            return
        }

        const alreadyExists = tagItems.some(tag => tag === normalized)
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

    const mapAIReviewStatusLabel = (status: SkillReviewStatusResponse['status']) => {
        if (status === 'queued') return t('review.aiStatusQueued')
        if (status === 'running') return t('review.aiStatusRunning')
        if (status === 'passed') return t('review.aiStatusPassed')
        if (status === 'failed_retryable') return t('review.aiStatusFailedRetryable')
        if (status === 'failed_terminal') return t('review.aiStatusFailedTerminal')
        return t('review.unknown')
    }

    const mapAIReviewPhaseLabel = (phase: SkillReviewStatusResponse['phase']) => {
        if (phase === 'queued') return t('review.aiPhaseQueued')
        if (phase === 'security') return t('review.aiPhaseSecurity')
        if (phase === 'functional') return t('review.aiPhaseFunctional')
        if (phase === 'finalizing') return t('review.aiPhaseFinalizing')
        if (phase === 'done') return t('review.aiPhaseDone')
        return t('review.unknown')
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
        closeThumbnailCropModal(true)
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
        setPrefillThumbnailUrl('')
        if (fileInputRef.current) {
            fileInputRef.current.value = ''
        }
        if (folderInputRef.current) {
            folderInputRef.current.value = ''
        }
        if (thumbInputRef.current) {
            thumbInputRef.current.value = ''
        }
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
        if (!isEditMode && prefill?.source_skill_id) {
            formData.append('source_skill_id', String(prefill.source_skill_id))
        }

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
            if (isEditMode) {
                const updated = await updateResourceFromUpload(editId, formData, 'skill')
                navigate(`/resource/skill/${updated.id}`, { replace: true })
            } else {
                const result = await uploadSkill(formData)
                setUploadedSkill(result.skill)
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
            await retrySkillReview(uploadedSkill.id, resourceType)
            startReviewPolling(uploadedSkill.id)
        } catch (err) {
            setReviewStatus('rejected')
            setFeedback(err instanceof Error ? err.message : '重新审核失败')
        }
    }

    const aiProgressTotal = reviewMeta?.progress?.total_files || 0
    const aiProgressPassed = reviewMeta?.progress?.files.filter(item => item.status === 'passed').length || 0
    const aiProgressFailed = reviewMeta?.progress?.files.filter(item => item.status === 'failed').length || 0

    const reviewSecuritySummary = aiProgressFailed > 0
        ? t('review.securitySummaryRisk', { count: aiProgressFailed })
        : t('review.securitySummarySafe')

    const reviewFunctionalSummary = reviewStatus === 'approved'
        ? (aiProgressTotal > 0
            ? t('review.functionalSummaryPassWithFiles', { passed: aiProgressPassed, total: aiProgressTotal })
            : t('review.functionalSummaryPass'))
        : (aiProgressFailed > 0
            ? t('review.functionalSummaryNeedsFix', { count: aiProgressFailed })
            : (feedback || t('upload.reviewRejectedFallback')))

    const showExpandedAIReview = step >= 2

    return (
        <div className="upload-page page-enter">
            <div className="upload-topbar upload-topbar-left">
                <Link to={`/resource/${resourceType}`} className="upload-back-btn">← {t('upload.back')}</Link>
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
                                <h1>{isEditMode ? '更新 Skill' : '上传 Skills'}</h1>
                                <p>{isEditMode ? '可只修改元数据，也可重新上传文件或文件夹；提交后会进入新的 Review。' : '支持上传文件或文件夹内容。'}</p>
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
                                        className={key === 'skill' ? 'active' : ''}
                                        onClick={() => {
                                            if (key === 'skill') return
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
                                    {thumbnail
                                        ? <em>{thumbnail.name}</em>
                                        : (prefillThumbnailUrl && <em>使用当前缩略图</em>)}
                                    <input
                                        ref={thumbInputRef}
                                        type="file"
                                        accept="image/*"
                                        onChange={e => {
                                            void handleThumbnailSelect(e)
                                        }}
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

                            <button className="upload-submit-btn" disabled={prefillLoading || uploading || !name.trim() || !hasPayload}>
                                {uploading ? t('upload.reviewing') : (isEditMode ? '提交更新并进入 Review' : t('upload.submit'))}
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

                <section className={`upload-stage-card ${step < 2 ? 'placeholder' : showExpandedAIReview ? 'expanded' : 'compact'}`}>
                    {step < 2 ? (
                        <div className="upload-placeholder-card glass-card">
                            <strong>{t('upload.waitForStep')}</strong>
                            <p>{t('upload.stepAi')}</p>
                        </div>
                    ) : showExpandedAIReview ? (
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

                            {(reviewStatus === 'approved' || reviewStatus === 'rejected') && (
                                <>
                                    <div className="review-assessment-grid">
                                        <div className="review-assessment-item">
                                            <div className="review-assessment-label">{t('detail.securityAssessment')}</div>
                                            <p>{reviewSecuritySummary}</p>
                                        </div>
                                        <div className="review-assessment-item">
                                            <div className="review-assessment-label">{t('detail.functionalAssessment')}</div>
                                            <p>{reviewFunctionalSummary}</p>
                                        </div>
                                    </div>

                                    {reviewMeta && (
                                        <div className="review-meta-grid">
                                            <div className="review-meta-item">
                                                <span>{t('review.metaStatus')}</span>
                                                <strong>{mapAIReviewStatusLabel(reviewMeta.status)}</strong>
                                            </div>
                                            <div className="review-meta-item">
                                                <span>{t('review.metaPhase')}</span>
                                                <strong>{mapAIReviewPhaseLabel(reviewMeta.phase)}</strong>
                                            </div>
                                            <div className="review-meta-item">
                                                <span>{t('review.metaAttempts')}</span>
                                                <strong>{reviewMeta.attempts}/{reviewMeta.max_attempts}</strong>
                                            </div>
                                            <div className="review-meta-item">
                                                <span>{t('review.metaFiles')}</span>
                                                <strong>
                                                    {reviewMeta.progress
                                                        ? `${reviewMeta.progress.completed_files}/${reviewMeta.progress.total_files}`
                                                        : '--'}
                                                </strong>
                                            </div>
                                        </div>
                                    )}
                                </>
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

                                {reviewStatus !== 'approved' && (
                                    <button type="button" className="btn btn-secondary" onClick={resetState}>
                                        {t('upload.resetForm')}
                                    </button>
                                )}
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
                                <button type="button" className="btn btn-secondary" onClick={resetState}>
                                    {t('upload.uploadAnother')}
                                </button>
                            </div>
                        </div>
                    )}
                </section>
            </div>

            {thumbnailCrop && (
                <div className="thumb-crop-modal-overlay" onClick={() => !cropApplying && closeThumbnailCropModal(true)}>
                    <div className="thumb-crop-modal glass-card" onClick={e => e.stopPropagation()}>
                        <header className="thumb-crop-modal-header">
                            <h3>{t('upload.thumbnailCropTitle')}</h3>
                            <p>
                                {t('upload.thumbnailCropMessage', {
                                    source: `${thumbnailCrop.meta.width}×${thumbnailCrop.meta.height}`,
                                    target: `${THUMB_TARGET_WIDTH}×${THUMB_TARGET_HEIGHT}`,
                                })}
                            </p>
                        </header>
                        <div className="thumb-crop-preview-wrap">
                            <canvas ref={cropPreviewRef} className="thumb-crop-preview" />
                        </div>
                        <div className="thumb-crop-controls">
                            <label className="thumb-crop-control">
                                <span>{t('upload.thumbnailCropZoom')}</span>
                                <input
                                    type="range"
                                    min={1}
                                    max={2.5}
                                    step={0.01}
                                    value={thumbnailCrop.zoom}
                                    onChange={e => updateThumbnailCrop({ zoom: Number(e.target.value) })}
                                    disabled={cropApplying}
                                />
                                <em>{thumbnailCrop.zoom.toFixed(2)}x</em>
                            </label>
                            <label className="thumb-crop-control">
                                <span>{t('upload.thumbnailCropHorizontal')}</span>
                                <input
                                    type="range"
                                    min={0}
                                    max={1}
                                    step={0.01}
                                    value={thumbnailCrop.centerX}
                                    onChange={e => updateThumbnailCrop({ centerX: Number(e.target.value) })}
                                    disabled={cropApplying}
                                />
                            </label>
                            <label className="thumb-crop-control">
                                <span>{t('upload.thumbnailCropVertical')}</span>
                                <input
                                    type="range"
                                    min={0}
                                    max={1}
                                    step={0.01}
                                    value={thumbnailCrop.centerY}
                                    onChange={e => updateThumbnailCrop({ centerY: Number(e.target.value) })}
                                    disabled={cropApplying}
                                />
                            </label>
                        </div>
                        <div className="thumb-crop-actions">
                            <button
                                type="button"
                                className="btn btn-secondary"
                                onClick={() => closeThumbnailCropModal(true)}
                                disabled={cropApplying}
                            >
                                {t('upload.thumbnailCropCancel')}
                            </button>
                            <button
                                type="button"
                                className="btn btn-primary"
                                onClick={() => {
                                    void handleApplyThumbnailCrop()
                                }}
                                disabled={cropApplying}
                            >
                                {cropApplying ? t('upload.reviewing') : t('upload.thumbnailCropApply')}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    )
}

export default UploadPage
