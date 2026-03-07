import { API_BASE } from './client'
import { Skill, SkillListResponse, SkillReviewStatusResponse, SkillSummaryResponse, UploadResponse } from './types'

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

export async function uploadSkill(formData: FormData): Promise<UploadResponse> {
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
    if (resourceType && resourceType !== 'skill') {
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

export async function retrySkillReview(id: number): Promise<{ message: string; status: SkillReviewStatusResponse }> {
    const res = await fetch(`${API_BASE}/skills/${id}/review/retry`, {
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

export async function submitHumanReview(
    id: number,
    approved = true,
    feedback = '',
): Promise<Skill> {
    const res = await fetch(`${API_BASE}/skills/${id}/human-review`, {
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
