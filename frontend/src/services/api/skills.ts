import { API_BASE } from './client'
import { Skill, SkillInstallConfigResponse, SkillListResponse, SkillReviewStatusResponse, SkillSummaryResponse, UploadResponse } from './types'

type SkillUpdatePayload = {
    name?: string
    description?: string
}

function getAuthHeaders(): Record<string, string> {
    const token = localStorage.getItem('auth_token')
    const headers: Record<string, string> = {}
    if (token) headers.Authorization = `Bearer ${token}`
    return headers
}

function normalizeTagsForUpload(raw: string): string {
    return raw
        .split(/[\n\r,]+/)
        .map(tag => tag.trim().toLowerCase())
        .filter(Boolean)
        .join(',')
}

/**
 * Maps a resource type to its API route prefix.
 * Skills use /skills, others use their own endpoints.
 */
function getResourcePath(resourceType: string): string {
    switch (resourceType) {
        case 'mcp': return '/mcps'
        case 'tools': return '/tools'
        case 'rules': return '/rules'
        default: return '/skills'
    }
}

export async function fetchSkills(
    search = '', page = 1, pageSize = 20,
    category = '', resourceType = '',
): Promise<SkillListResponse> {
    const params = new URLSearchParams({
        page: String(page),
        page_size: String(pageSize),
    })
    if (search) params.set('search', search)
    if (category) params.set('category', category)

    // If resourceType is specified, use the type-specific endpoint.
    // If empty, use /skills with no filter (returns all types).
    let basePath: string
    if (resourceType) {
        basePath = getResourcePath(resourceType)
    } else {
        basePath = '/skills'
        // No resource_type param needed - returns all types
    }

    const res = await fetch(`${API_BASE}${basePath}?${params}`, {
        headers: getAuthHeaders(),
    })
    if (!res.ok) throw new Error('Failed to fetch list')
    return res.json()
}

export async function fetchCategories(resourceType = ''): Promise<string[]> {
    if (resourceType && resourceType !== 'skill') {
        // Use type-specific endpoint
        const basePath = getResourcePath(resourceType)
        const res = await fetch(`${API_BASE}${basePath}/categories`)
        if (!res.ok) throw new Error('Failed to fetch categories')
        return res.json()
    }

    const params = new URLSearchParams()
    if (resourceType) params.set('resource_type', resourceType)
    const res = await fetch(`${API_BASE}/categories?${params}`)
    if (!res.ok) throw new Error('Failed to fetch categories')
    return res.json()
}

export async function fetchSkill(id: number, resourceType = ''): Promise<Skill> {
    // For get-by-ID, /api/skills/:id works for all types (same DB table).
    // But if we know the type, we can use the type-specific endpoint.
    const basePath = resourceType ? getResourcePath(resourceType) : '/skills'
    const res = await fetch(`${API_BASE}${basePath}/${id}`, {
        headers: getAuthHeaders(),
    })
    if (!res.ok) throw new Error('Failed to fetch details')
    return res.json()
}

export async function fetchSkillReadme(id: number, resourceType = ''): Promise<string> {
    const basePath = resourceType ? getResourcePath(resourceType) : '/skills'
    const res = await fetch(`${API_BASE}${basePath}/${id}/readme`, {
        headers: getAuthHeaders(),
    })
    if (!res.ok) return ''
    return res.text()
}

export async function fetchTrending(limit = 10, resourceType = ''): Promise<Skill[]> {
    const params = new URLSearchParams({ limit: String(limit) })

    let basePath: string
    if (resourceType) {
        basePath = getResourcePath(resourceType)
    } else {
        basePath = '/skills'
        params.set('resource_type', '')
    }

    const res = await fetch(`${API_BASE}${basePath}/trending?${params}`, {
        headers: getAuthHeaders(),
    })
    if (!res.ok) throw new Error('Failed to fetch trending')
    return res.json()
}

export async function fetchSkillSummary(resourceType = ''): Promise<SkillSummaryResponse> {
    if (resourceType && resourceType !== 'skill') {
        const basePath = getResourcePath(resourceType)
        const res = await fetch(`${API_BASE}${basePath}/summary`, {
            headers: getAuthHeaders(),
        })
        if (!res.ok) throw new Error('Failed to fetch summary')
        return res.json()
    }

    const params = new URLSearchParams()
    if (resourceType) params.set('resource_type', resourceType)
    const res = await fetch(`${API_BASE}/skills/summary?${params}`, {
        headers: getAuthHeaders(),
    })
    if (!res.ok) throw new Error('Failed to fetch summary')
    return res.json()
}

export async function fetchSkillInstallConfig(): Promise<SkillInstallConfigResponse> {
    const res = await fetch(`${API_BASE}/skills/install-config`, {
        headers: getAuthHeaders(),
    })
    if (!res.ok) throw new Error('Failed to fetch install config')
    return res.json()
}

export async function uploadSkill(formData: FormData): Promise<UploadResponse> {
    const rawTags = formData.get('tags')
    if (typeof rawTags === 'string') {
        formData.set('tags', normalizeTagsForUpload(rawTags))
    }

    // Determine upload endpoint based on resource_type in the form data
    const resourceType = formData.get('resource_type') as string || 'skill'
    const basePath = getResourcePath(resourceType)

    const res = await fetch(`${API_BASE}${basePath}`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: formData,
    })
    if (!res.ok) {
        let message = `Upload failed (HTTP ${res.status})`
        const raw = (await res.text()).trim()
        if (raw) {
            try {
                const parsed = JSON.parse(raw)
                if (typeof parsed?.error === 'string' && parsed.error.trim()) {
                    message = parsed.error
                } else {
                    message = raw
                }
            } catch {
                message = raw
            }
        }

        if (res.status === 409 && !message.includes('rename')) {
            message = 'Name already exists, please use a different name'
        } else if (res.status === 413) {
            message = 'File too large, please compress and retry'
        }

        throw new Error(message)
    }

    return res.json()
}

export async function fetchSkillReviewStatus(id: number, resourceType = 'skill'): Promise<SkillReviewStatusResponse> {
    if (resourceType && resourceType !== 'skill' && resourceType !== 'rules') {
        return {
            status: 'passed',
            phase: 'done',
            attempts: 0,
            max_attempts: 0,
            retry_remaining: 0,
            can_retry: false,
            approved: true,
            feedback: 'Auto-approved (no AI review for this resource type).',
        }
    }

    const basePath = getResourcePath(resourceType)
    const res = await fetch(`${API_BASE}${basePath}/${id}/review-status`, {
        headers: getAuthHeaders(),
    })
    if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error || 'Failed to get review status')
    }
    return res.json()
}

export async function retrySkillReview(id: number, resourceType = 'skill'): Promise<{ message: string; status: SkillReviewStatusResponse }> {
    const basePath = getResourcePath(resourceType)
    const res = await fetch(`${API_BASE}${basePath}/${id}/review/retry`, {
        method: 'POST',
        headers: getAuthHeaders(),
    })
    if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error || 'Retry failed')
    }
    return res.json()
}

export async function deleteSkill(id: number, resourceType = ''): Promise<{ message: string; github_error?: string }> {
    const basePath = resourceType ? getResourcePath(resourceType) : '/skills'
    const res = await fetch(`${API_BASE}${basePath}/${id}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
    })
    if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error || 'Delete failed')
    }
    return res.json()
}

export type DeleteProgressStage = 'db' | 'github'

interface DeleteStreamDonePayload {
    ok?: boolean
    message?: string
    github_error?: string
    error?: string
}

export async function deleteSkillWithProgress(
    id: number,
    resourceType = '',
    onProgress?: (stage: DeleteProgressStage) => void,
): Promise<{ message: string; github_error?: string }> {
    const basePath = resourceType ? getResourcePath(resourceType) : '/skills'
    const res = await fetch(`${API_BASE}${basePath}/${id}/stream-delete`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
    })
    if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error || 'Delete failed')
    }

    const reader = res.body?.getReader()
    if (!reader) {
        throw new Error('无法读取删除进度')
    }

    const decoder = new TextDecoder()
    let buffer = ''
    let eventType = ''
    let dataLines: string[] = []

    const reset = () => {
        eventType = ''
        dataLines = []
    }

    const parsePayload = (): Record<string, any> | null => {
        if (dataLines.length === 0) return null
        const raw = dataLines.join('\n').trim()
        if (!raw) return null
        try {
            const parsed = JSON.parse(raw)
            if (parsed && typeof parsed === 'object') {
                return parsed
            }
        } catch {
            return { message: raw }
        }
        return null
    }

    const dispatch = (): { done?: { message: string; github_error?: string } } | null => {
        const payload = parsePayload()
        if (!payload) {
            reset()
            return null
        }

        if (eventType === 'progress') {
            const stage = payload.stage
            if (stage === 'db' || stage === 'github') {
                onProgress?.(stage)
            }
            reset()
            return null
        }

        if (eventType === 'error') {
            reset()
            throw new Error(payload.error || payload.message || 'Delete failed')
        }

        if (eventType === 'done') {
            reset()
            const donePayload = payload as DeleteStreamDonePayload
            if (donePayload.ok === false) {
                throw new Error(donePayload.error || donePayload.message || 'Delete failed')
            }
            return {
                done: {
                    message: donePayload.message || 'Deleted',
                    github_error: donePayload.github_error,
                },
            }
        }

        reset()
        return null
    }

    while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
            const normalized = line.replace(/\r$/, '')
            const trimmed = normalized.trim()
            if (!trimmed) {
                const result = dispatch()
                if (result?.done) return result.done
                continue
            }
            if (trimmed.startsWith('event:')) {
                eventType = trimmed.slice(6).trim()
                continue
            }
            if (trimmed.startsWith('data:')) {
                dataLines.push(trimmed.slice(5).trim())
            }
        }
    }

    const final = dispatch()
    if (final?.done) return final.done
    throw new Error('删除进度流意外结束')
}

export async function updateSkill(id: number, payload: SkillUpdatePayload, resourceType = ''): Promise<Skill> {
    const basePath = resourceType ? getResourcePath(resourceType) : '/skills'
    const res = await fetch(`${API_BASE}${basePath}/${id}`, {
        method: 'PUT',
        headers: {
            ...getAuthHeaders(),
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
    })
    if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error || 'Update failed')
    }

    const data = await res.json()
    return data.skill ?? data
}

export async function updateResourceFromUpload(id: number, formData: FormData, resourceType: 'mcp' | 'tools'): Promise<Skill> {
    const rawTags = formData.get('tags')
    if (typeof rawTags === 'string') {
        formData.set('tags', normalizeTagsForUpload(rawTags))
    }

    const basePath = getResourcePath(resourceType)
    const res = await fetch(`${API_BASE}${basePath}/${id}`, {
        method: 'PUT',
        headers: getAuthHeaders(),
        body: formData,
    })
    if (!res.ok) {
        let message = `Update failed (HTTP ${res.status})`
        const raw = (await res.text()).trim()
        if (raw) {
            try {
                const parsed = JSON.parse(raw)
                if (typeof parsed?.error === 'string' && parsed.error.trim()) {
                    message = parsed.error
                } else {
                    message = raw
                }
            } catch {
                message = raw
            }
        }
        throw new Error(message)
    }

    const data = await res.json()
    return data.skill ?? data
}

export async function submitHumanReview(
    id: number,
    resourceType = 'skill',
    approved = true,
    feedback = '',
): Promise<Skill> {
    const basePath = getResourcePath(resourceType)
    const res = await fetch(`${API_BASE}${basePath}/${id}/human-review`, {
        method: 'POST',
        headers: {
            ...getAuthHeaders(),
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ approved, feedback }),
    })

    if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error || 'Human review failed')
    }

    const data = await res.json()
    return data.skill ?? data
}

export async function likeSkill(id: number, resourceType = ''): Promise<{ liked: boolean; likes_count: number }> {
    const basePath = resourceType ? getResourcePath(resourceType) : '/skills'
    const res = await fetch(`${API_BASE}${basePath}/${id}/like`, {
        method: 'POST',
        headers: getAuthHeaders(),
    })

    if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error || 'Like failed')
    }

    return res.json()
}

export async function unlikeSkill(id: number, resourceType = ''): Promise<{ liked: boolean; likes_count: number }> {
    const basePath = resourceType ? getResourcePath(resourceType) : '/skills'
    const res = await fetch(`${API_BASE}${basePath}/${id}/like`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
    })

    if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error || 'Unlike failed')
    }

    return res.json()
}

export async function favoriteSkill(id: number, resourceType = ''): Promise<{ favorited: boolean }> {
    const basePath = resourceType ? getResourcePath(resourceType) : '/skills'
    const res = await fetch(`${API_BASE}${basePath}/${id}/favorite`, {
        method: 'POST',
        headers: getAuthHeaders(),
    })
    if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error || 'Favorite failed')
    }
    return res.json()
}

export async function unfavoriteSkill(id: number, resourceType = ''): Promise<{ favorited: boolean }> {
    const basePath = resourceType ? getResourcePath(resourceType) : '/skills'
    const res = await fetch(`${API_BASE}${basePath}/${id}/favorite`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
    })
    if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error || 'Unfavorite failed')
    }
    return res.json()
}

export async function fetchMyFavorites(resourceType = ''): Promise<Skill[]> {
    const params = new URLSearchParams()
    if (resourceType) params.set('resource_type', resourceType)
    const query = params.toString()

    const res = await fetch(`${API_BASE}/me/favorites${query ? `?${query}` : ''}`, {
        headers: getAuthHeaders(),
    })
    if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error || 'Fetch favorites failed')
    }
    const data = await res.json()
    return data.skills || []
}

export async function trackDownloadHit(id: number, resourceType = ''): Promise<{ downloads: number }> {
    const basePath = resourceType ? getResourcePath(resourceType) : '/skills'
    const res = await fetch(`${API_BASE}${basePath}/${id}/download-hit`, {
        method: 'POST',
    })
    if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error || 'Download tracking failed')
    }
    return res.json()
}

export function getDownloadUrl(id: number, resourceType = ''): string {
    const basePath = resourceType ? getResourcePath(resourceType) : '/skills'
    return `${API_BASE}${basePath}/${id}/download`
}
