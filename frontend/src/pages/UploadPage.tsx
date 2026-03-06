import { useState, useRef } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { uploadSkill, RESOURCE_TYPES } from '../services/api'

const CATEGORIES: Record<string, string[]> = {
    skill: ['Automation', 'Data Processing', 'Network', 'AI/ML', 'Dev Tools', 'System', 'Security', 'Other'],
    mcp: ['Data Source', 'File System', 'API Integration', 'Database', 'Search', 'Other'],
    rules: ['Code Standards', 'Security Policies', 'CI/CD', 'Testing', 'Deployment', 'Other'],
    tools: ['CLI Tools', 'Editor Plugins', 'Build Tools', 'Debug', 'Monitoring', 'Other'],
}

function UploadPage() {
    const navigate = useNavigate()
    const [resourceType, setResourceType] = useState('skill')

    const categoryList = CATEGORIES[resourceType] || CATEGORIES.skill
    const info = RESOURCE_TYPES[resourceType]

    const [name, setName] = useState('')
    const [description, setDescription] = useState('')
    const [category, setCategory] = useState(categoryList[0])

    const [uploadMode, setUploadMode] = useState<'file' | 'folder'>('file')
    const [file, setFile] = useState<File | null>(null)
    const [folderFiles, setFolderFiles] = useState<File[]>([])
    const [thumbnail, setThumbnail] = useState<File | null>(null)

    const [uploadStep, setUploadStep] = useState(1)
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
        // Auto-fill name from folder name if empty
        if (files.length > 0 && !name.trim()) {
            const relativePath = files[0].webkitRelativePath || ''
            const folderName = relativePath.split('/')[0]
            if (folderName) setName(folderName)
        }
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()

        if (!name.trim()) return alert('Please enter a name')
        if (!hasFiles) return alert(uploadMode === 'file' ? 'Please upload a file' : 'Please select a folder')

        setUploading(true)
        setUploadStep(2)
        setReviewStatus('pending')
        setFeedback('')

        const formData = new FormData()
        formData.append('name', name.trim())
        formData.append('description', description.trim())
        formData.append('category', category)
        formData.append('resource_type', resourceType)
        formData.append('author', 'Anonymous')
        formData.append('upload_mode', uploadMode)

        if (uploadMode === 'file' && file) {
            formData.append('file', file)
        } else if (uploadMode === 'folder') {
            for (const f of folderFiles) {
                formData.append('files', f)
                // Send relative path within the folder (strip top-level folder name)
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
                setUploadStep(3)
                setTimeout(() => {
                    navigate(`/skill/${result.skill.id}`)
                }, 1500)
            }
        } catch (err) {
            setReviewStatus('rejected')
            setFeedback(err instanceof Error ? err.message : 'Upload failed')
        } finally {
            setUploading(false)
        }
    }

    return (
        <div className="upload-page page-enter">
            {/* Header with Back Button */}
            <div className="upload-page-header">
                <Link to="/" className="upload-back-btn">
                    ← Back
                </Link>
            </div>

            {/* Step Indicator */}
            <div className="upload-steps">
                <div className={`step ${uploadStep >= 1 ? 'active' : ''}`}>
                    <div className="step-circle">1</div>
                    <span>Configure</span>
                </div>
                <div className="step-connector"></div>
                <div className={`step ${uploadStep >= 2 ? 'active' : ''}`}>
                    <div className="step-circle">2</div>
                    <span>AI Review</span>
                </div>
                <div className="step-connector"></div>
                <div className={`step ${uploadStep >= 3 ? 'active' : ''}`}>
                    <div className="step-circle">3</div>
                    <span>Published</span>
                </div>
            </div>

            <form className="upload-form-container glass-card" onSubmit={handleSubmit}>
                <div className="form-group row-group">
                    <label>Type</label>
                    <div style={{ display: 'flex', gap: 12 }}>
                        <select value={resourceType} onChange={e => {
                            setResourceType(e.target.value);
                            setCategory((CATEGORIES[e.target.value] || CATEGORIES.skill)[0]);
                        }} disabled={uploading}>
                            {Object.entries(RESOURCE_TYPES).map(([k, v]) => (
                                <option key={k} value={k}>{v.label}</option>
                            ))}
                        </select>
                        <select value={category} onChange={e => setCategory(e.target.value)} disabled={uploading} style={{ flex: 1 }}>
                            {categoryList.map(cat => <option key={cat} value={cat}>{cat}</option>)}
                        </select>
                    </div>
                </div>

                <div className="form-group row-group">
                    <label>Name</label>
                    <input type="text" placeholder={`Enter ${info.label} name`} value={name} onChange={e => setName(e.target.value)} disabled={uploading} required />
                </div>

                <div className="form-group row-group">
                    <label>Description</label>
                    <input type="text" placeholder="Brief description..." value={description} onChange={e => setDescription(e.target.value)} disabled={uploading} />
                </div>

                <div className="form-group row-group">
                    <label>Upload Mode</label>
                    <div style={{ display: 'flex', gap: 8 }}>
                        <button type="button" className={`btn ${uploadMode === 'file' ? 'btn-primary' : 'btn-secondary'}`}
                            onClick={() => { setUploadMode('file'); setFolderFiles([]) }} disabled={uploading} style={{ flex: 1 }}>
                            Single File
                        </button>
                        <button type="button" className={`btn ${uploadMode === 'folder' ? 'btn-primary' : 'btn-secondary'}`}
                            onClick={() => { setUploadMode('folder'); setFile(null) }} disabled={uploading} style={{ flex: 1 }}>
                            Folder
                        </button>
                    </div>
                </div>

                <div className="upload-areas">
                    <div className="form-group">
                        <label>Thumbnail</label>
                        <div className="file-upload-box" onClick={() => thumbInputRef.current?.click()}>
                            <input ref={thumbInputRef} type="file" accept="image/*" onChange={e => setThumbnail(e.target.files?.[0] || null)} disabled={uploading} />
                            <div className="upload-icon">🖼️</div>
                        </div>
                        {thumbnail && <div className="file-name">{thumbnail.name}</div>}
                    </div>

                    <div className="form-group">
                        <label>{uploadMode === 'file' ? 'File' : 'Folder'}</label>
                        {uploadMode === 'file' ? (
                            <div className="file-upload-box" onClick={() => fileInputRef.current?.click()}>
                                <input ref={fileInputRef} type="file" onChange={e => setFile(e.target.files?.[0] || null)} disabled={uploading} />
                                <div className="upload-icon">📄</div>
                            </div>
                        ) : (
                            <div className="file-upload-box" onClick={() => folderInputRef.current?.click()}>
                                {/* @ts-ignore webkitdirectory is non-standard */}
                                <input ref={folderInputRef} type="file" webkitdirectory="" directory="" multiple onChange={handleFolderSelect} disabled={uploading} />
                                <div className="upload-icon">📁</div>
                            </div>
                        )}
                        {uploadMode === 'file' && file && <div className="file-name">{file.name}</div>}
                        {uploadMode === 'folder' && folderFiles.length > 0 && (
                            <div className="file-name">{folderFiles.length} files selected</div>
                        )}
                    </div>
                </div>

                {reviewStatus !== 'idle' && (
                    <div className={`ai-review-status ${reviewStatus}`}>
                        <strong>
                            {reviewStatus === 'pending' && '⏳ AI is reviewing...'}
                            {reviewStatus === 'approved' && '✅ Approved! Publishing...'}
                            {reviewStatus === 'rejected' && '❌ Review failed'}
                        </strong>
                        <p style={{ fontSize: '0.9rem', marginTop: 4, opacity: 0.9 }}>{feedback || (reviewStatus === 'pending' ? 'Please wait...' : '')}</p>
                    </div>
                )}

                <div className="upload-actions">
                    <button type="button" className="btn btn-secondary" onClick={() => navigate(-1)} disabled={uploading}>Cancel</button>
                    <button type="submit" className="btn btn-primary" disabled={uploading || !name.trim() || !hasFiles}>
                        {uploading ? 'Submitting...' : 'Publish'}
                    </button>
                </div>
            </form>
        </div>
    )
}

export default UploadPage
