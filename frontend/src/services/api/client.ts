function normalizeApiBase(rawBase?: string) {
    const trimmed = rawBase?.trim()
    if (!trimmed) {
        return '/api'
    }

    const withoutTrailingSlash = trimmed.replace(/\/+$/, '')
    if (withoutTrailingSlash.endsWith('/api')) {
        return withoutTrailingSlash
    }

    return `${withoutTrailingSlash}/api`
}

export const APP_ENV = import.meta.env.VITE_APP_ENV || 'local'
export const API_BASE = normalizeApiBase(import.meta.env.VITE_API_BASE_URL)
