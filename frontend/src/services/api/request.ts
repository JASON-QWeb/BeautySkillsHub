type UnauthorizedHandler = (() => void) | null

let unauthorizedHandler: UnauthorizedHandler = null

function decodeBase64URL(segment: string): string {
    const normalized = segment.replace(/-/g, '+').replace(/_/g, '/')
    const padding = normalized.length % 4 === 0 ? '' : '='.repeat(4 - (normalized.length % 4))
    return atob(normalized + padding)
}

function getStoredAuthToken(): string | null {
    if (typeof localStorage === 'undefined') return null
    return localStorage.getItem('auth_token')
}

function mergeHeaders(headers?: HeadersInit): Record<string, string> {
    if (!headers) return {}
    if (headers instanceof Headers) {
        return Object.fromEntries(headers.entries())
    }
    if (Array.isArray(headers)) {
        return Object.fromEntries(headers)
    }
    return { ...headers }
}

export function setUnauthorizedHandler(handler: UnauthorizedHandler) {
    unauthorizedHandler = handler
}

export function getAuthHeaders(headers?: HeadersInit): Record<string, string> {
    const merged = mergeHeaders(headers)
    const token = getStoredAuthToken()
    if (token) {
        merged.Authorization = `Bearer ${token}`
    }
    return merged
}

export function isTokenExpired(token: string, nowMs = Date.now()): boolean {
    try {
        const [, payloadSegment] = token.split('.')
        if (!payloadSegment) return false

        const payload = JSON.parse(decodeBase64URL(payloadSegment)) as { exp?: number }
        if (typeof payload.exp !== 'number') return false
        return payload.exp * 1000 <= nowMs
    } catch {
        return false
    }
}

export function isAbortError(error: unknown): boolean {
    return error instanceof DOMException && error.name === 'AbortError'
}

export async function apiFetch(
    input: RequestInfo | URL,
    init: RequestInit & { auth?: boolean } = {},
): Promise<Response> {
    const { auth = false, headers, ...rest } = init
    const requestHeaders = auth ? getAuthHeaders(headers) : mergeHeaders(headers)
    const hasToken = !!requestHeaders.Authorization

    const response = await fetch(input, {
        ...rest,
        headers: requestHeaders,
    })

    if (auth && hasToken && response.status === 401 && unauthorizedHandler) {
        unauthorizedHandler()
    }

    return response
}
