import { API_BASE } from './client'

function getAuthHeaders(): Record<string, string> {
    const token = localStorage.getItem('auth_token')
    return token ? { Authorization: `Bearer ${token}` } : {}
}

export async function uploadContentImage(file: File): Promise<string> {
    const formData = new FormData()
    formData.append('image', file)

    const res = await fetch(`${API_BASE}/content-assets/images`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: formData,
    })

    if (!res.ok) {
        let message = '图片上传失败'
        try {
            const data = await res.json()
            if (typeof data?.error === 'string' && data.error.trim()) {
                message = data.error
            }
        } catch {
            // ignore and keep fallback message
        }
        throw new Error(message)
    }

    const data = await res.json() as { url?: string }
    if (!data.url) {
        throw new Error('图片地址返回异常')
    }
    return data.url
}
