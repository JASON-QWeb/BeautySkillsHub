import { useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { uploadSkill, RESOURCE_TYPES } from '../services/api'
import { useDialog } from '../contexts/DialogContext'

const CATEGORIES: Record<string, string[]> = {
    skill: ['自动化', '数据处理', '网络工具', 'AI/ML', '开发工具', '系统管理', '安全', '其他'],
    mcp: ['数据源', '文件系统', 'API集成', '数据库', '搜索', '其他'],
    rules: ['代码规范', '安全策略', 'CI/CD', '测试', '部署', '其他'],
    tools: ['CLI工具', '编辑器插件', '构建工具', '调试', '监控', '其他'],
}

interface UploadModalProps {
    isOpen: boolean
    onClose: () => void
}

function UploadModal({ isOpen, onClose }: UploadModalProps) {
    const navigate = useNavigate()
    const { showAlert } = useDialog()

    const [resourceType, setResourceType] = useState('skill')

    const categoryList = CATEGORIES[resourceType] || CATEGORIES.skill
    const info = RESOURCE_TYPES[resourceType]

    const [name, setName] = useState('')
    const [description, setDescription] = useState('')
    const [category, setCategory] = useState(categoryList[0])
    const [author, setAuthor] = useState('')

    const [uploadMode, setUploadMode] = useState<'file' | 'folder'>('file')
    const [file, setFile] = useState<File | null>(null)
    const [folderFiles, setFolderFiles] = useState<File[]>([])
    const [thumbnail, setThumbnail] = useState<File | null>(null)

    const [uploading, setUploading] = useState(false)
    const [reviewStatus, setReviewStatus] = useState<'idle' | 'pending' | 'approved' | 'rejected'>('idle')
    const [feedback, setFeedback] = useState('')

    const fileInputRef = useRef<HTMLInputElement>(null)
    const folderInputRef = useRef<HTMLInputElement>(null)
    const thumbInputRef = useRef<HTMLInputElement>(null)

    const hasFiles = uploadMode === 'file' ? !!file : folderFiles.length > 0

    const handleFolderSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
        const files = Array.from(e.target.files || [])
        setFolderFiles(files)
        if (files.length > 0 && !name.trim()) {
            const relativePath = files[0].webkitRelativePath || ''
            const folderName = relativePath.split('/')[0]
            if (folderName) setName(folderName)
        }
    }

    const handleTypeChange = (t: string) => {
        setResourceType(t)
        setCategory((CATEGORIES[t] || CATEGORIES.skill)[0])
    }

    const resetForm = () => {
        setName('')
        setDescription('')
        setAuthor('')
        setUploadMode('file')
        setFile(null)
        setFolderFiles([])
        setThumbnail(null)
        setReviewStatus('idle')
        setFeedback('')
    }

    const handleClose = () => {
        if (!uploading) {
            resetForm()
            onClose()
        }
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()

        if (!name.trim()) {
            await showAlert('请输入名称')
            return
        }
        if (!hasFiles) {
            await showAlert(uploadMode === 'file' ? '请上传文件' : '请选择文件夹')
            return
        }

        setUploading(true)
        setReviewStatus('pending')
        setFeedback('')

        const formData = new FormData()
        formData.append('name', name.trim())
        formData.append('description', description.trim())
        formData.append('category', category)
        formData.append('resource_type', resourceType)
        formData.append('author', author.trim() || '匿名')
        formData.append('upload_mode', uploadMode)

        if (uploadMode === 'file' && file) {
            formData.append('file', file)
        } else if (uploadMode === 'folder') {
            for (const f of folderFiles) {
                formData.append('files', f)
                const rel = f.webkitRelativePath || f.name
                const parts = rel.split('/')
                const innerPath = parts.length > 1 ? parts.slice(1).join('/') : parts[0]
                formData.append('file_paths', innerPath)
            }
        }

        if (thumbnail) formData.append('thumbnail', thumbnail)

        try {
            const result = await uploadSkill(formData)
            setReviewStatus(result.approved ? 'approved' : 'rejected')
            setFeedback(result.feedback)

            if (result.approved) {
                setTimeout(() => {
                    handleClose()
                    navigate(`/skill/${result.skill.id}`)
                }, 1500)
            }
        } catch (err) {
            setReviewStatus('rejected')
            setFeedback(err instanceof Error ? err.message : '上传失败')
        } finally {
            setUploading(false)
        }
    }

    return (
        <div className={`upload-modal-overlay ${isOpen ? 'open' : ''}`} onClick={(e) => { if (e.target === e.currentTarget) handleClose() }}>
            <div className="upload-modal-container">
                <div className="upload-modal-header">
                    <h2>发布资源</h2>
                    <button className="upload-modal-close" onClick={handleClose} disabled={uploading}>×</button>
                </div>

                <form className="upload-modal-content" onSubmit={handleSubmit}>
                    {/* Left Column: Info & Files */}
                    <div className="upload-modal-left">
                        <div className="form-group">
                            <label>选择资源类型</label>
                            <div className="resource-type-selector" style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                                {Object.entries(RESOURCE_TYPES).map(([key, val]) => (
                                    <button
                                        key={key}
                                        type="button"
                                        className={`resource-type-btn ${resourceType === key ? 'active' : ''}`}
                                        onClick={() => handleTypeChange(key)}
                                        disabled={uploading}
                                    >
                                        <span>{val.icon}</span>
                                        <span>{val.label}</span>
                                    </button>
                                ))}
                            </div>
                        </div>

                        <div className="form-group">
                            <label>上传方式</label>
                            <div style={{ display: 'flex', gap: 8 }}>
                                <button type="button" className={`resource-type-btn ${uploadMode === 'file' ? 'active' : ''}`}
                                    onClick={() => { setUploadMode('file'); setFolderFiles([]) }} disabled={uploading} style={{ flex: 1 }}>
                                    <span>📄</span><span>单文件</span>
                                </button>
                                <button type="button" className={`resource-type-btn ${uploadMode === 'folder' ? 'active' : ''}`}
                                    onClick={() => { setUploadMode('folder'); setFile(null) }} disabled={uploading} style={{ flex: 1 }}>
                                    <span>📁</span><span>文件夹</span>
                                </button>
                            </div>
                        </div>

                        <div className="form-group" style={{ marginTop: 'auto' }}>
                            <label>上传 {info.label} {uploadMode === 'file' ? '文件' : '文件夹'} *</label>
                            {uploadMode === 'file' ? (
                                <div className="file-upload-area" onClick={() => fileInputRef.current?.click()}>
                                    <input ref={fileInputRef} type="file" onChange={e => setFile(e.target.files?.[0] || null)} disabled={uploading} />
                                    <div className="upload-icon">📄</div>
                                    <p>点击或拖拽上传主文件</p>
                                    {file && <div className="file-name">{file.name}</div>}
                                </div>
                            ) : (
                                <div className="file-upload-area" onClick={() => folderInputRef.current?.click()}>
                                    {/* @ts-ignore */}
                                    <input ref={folderInputRef} type="file" webkitdirectory="" directory="" multiple onChange={handleFolderSelect} disabled={uploading} />
                                    <div className="upload-icon">📁</div>
                                    <p>点击选择文件夹</p>
                                    {folderFiles.length > 0 && <div className="file-name">已选 {folderFiles.length} 个文件</div>}
                                </div>
                            )}
                        </div>

                        <div className="form-group">
                            <label>自定义缩略图 (可选)</label>
                            <div className="file-upload-area" onClick={() => thumbInputRef.current?.click()} style={{ minHeight: 120 }}>
                                <input ref={thumbInputRef} type="file" accept="image/*" onChange={e => setThumbnail(e.target.files?.[0] || null)} disabled={uploading} />
                                <div className="upload-icon" style={{ fontSize: '2rem' }}>🖼️</div>
                                <p>点击上传展示图片</p>
                                {thumbnail && <div className="file-name">{thumbnail.name}</div>}
                            </div>
                        </div>
                    </div>

                    {/* Right Column: Details & Submit */}
                    <div className="upload-modal-right">
                        <div className="form-group">
                            <label>资源名称 *</label>
                            <input type="text" placeholder={`输入 ${info.label} 名称`} value={name} onChange={e => setName(e.target.value)} disabled={uploading} required />
                        </div>

                        <div className="form-group" style={{ flex: 1 }}>
                            <label>详细描述</label>
                            <textarea placeholder={`介绍该 ${info.label} 的用途和使用方式...`} value={description} onChange={e => setDescription(e.target.value)} disabled={uploading} style={{ height: '100%', minHeight: 150 }} />
                        </div>

                        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
                            <div className="form-group">
                                <label>分类</label>
                                <select value={category} onChange={e => setCategory(e.target.value)} disabled={uploading}>
                                    {categoryList.map(cat => <option key={cat} value={cat}>{cat}</option>)}
                                </select>
                            </div>
                            <div className="form-group">
                                <label>作者</label>
                                <input type="text" placeholder="匿名留空" value={author} onChange={e => setAuthor(e.target.value)} disabled={uploading} />
                            </div>
                        </div>

                        {reviewStatus !== 'idle' && (
                            <div className={`ai-review-status ${reviewStatus}`}>
                                <strong>
                                    {reviewStatus === 'pending' && '⏳ AI 正在审核代码...'}
                                    {reviewStatus === 'approved' && '✅ 审核通过！正在发布'}
                                    {reviewStatus === 'rejected' && '❌ 审核未通过'}
                                </strong>
                                <p style={{ fontSize: '0.9rem', marginTop: 4, opacity: 0.9 }}>{feedback || (reviewStatus === 'pending' ? '请稍候...' : '')}</p>
                            </div>
                        )}

                        <div className="upload-modal-actions" style={{ marginTop: 'auto', padding: 0, border: 'none', background: 'transparent' }}>
                            <button type="button" className="btn btn-secondary" onClick={handleClose} disabled={uploading}>取消</button>
                            <button type="submit" className="btn btn-primary" disabled={uploading || !name.trim() || !hasFiles}>
                                {uploading ? '提交审核中...' : '🚀 发布资源'}
                            </button>
                        </div>
                    </div>
                </form>
            </div>
        </div>
    )
}

export default UploadModal
