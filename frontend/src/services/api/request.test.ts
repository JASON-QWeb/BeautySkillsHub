import assert from 'node:assert/strict'
import test from 'node:test'

import { apiFetch, getAuthHeaders, isTokenExpired, setUnauthorizedHandler } from './request.ts'

function makeToken(payload: Record<string, unknown>) {
    const encoded = Buffer.from(JSON.stringify(payload)).toString('base64url')
    return `header.${encoded}.signature`
}

function installLocalStorage(token: string | null) {
    const storage = new Map<string, string>()
    if (token) storage.set('auth_token', token)

    Object.defineProperty(globalThis, 'localStorage', {
        configurable: true,
        value: {
            getItem(key: string) {
                return storage.has(key) ? storage.get(key)! : null
            },
            setItem(key: string, value: string) {
                storage.set(key, value)
            },
            removeItem(key: string) {
                storage.delete(key)
            },
        },
    })
}

test('getAuthHeaders includes bearer token when available', () => {
    installLocalStorage('abc123')

    const headers = getAuthHeaders()

    assert.equal(headers.Authorization, 'Bearer abc123')
})

test('isTokenExpired returns true for past exp claim', () => {
    const token = makeToken({ exp: Math.floor(Date.now() / 1000) - 10 })
    assert.equal(isTokenExpired(token), true)
})

test('isTokenExpired returns false for future exp claim', () => {
    const token = makeToken({ exp: Math.floor(Date.now() / 1000) + 60 })
    assert.equal(isTokenExpired(token), false)
})

test('apiFetch triggers unauthorized handler for authenticated 401 responses', async () => {
    installLocalStorage('abc123')

    let unauthorizedCount = 0
    setUnauthorizedHandler(() => {
        unauthorizedCount += 1
    })

    const originalFetch = globalThis.fetch
    globalThis.fetch = async (_input: RequestInfo | URL, init?: RequestInit) => {
        const headers = init?.headers as Record<string, string> | undefined
        assert.equal(headers?.Authorization, 'Bearer abc123')
        return new Response('{}', {
            status: 401,
            headers: { 'Content-Type': 'application/json' },
        })
    }

    try {
        const response = await apiFetch('https://example.com/api/skills', { auth: true })
        assert.equal(response.status, 401)
        assert.equal(unauthorizedCount, 1)
    } finally {
        globalThis.fetch = originalFetch
        setUnauthorizedHandler(null)
    }
})

test('apiFetch does not trigger unauthorized handler for anonymous requests', async () => {
    installLocalStorage('abc123')

    let unauthorizedCount = 0
    setUnauthorizedHandler(() => {
        unauthorizedCount += 1
    })

    const originalFetch = globalThis.fetch
    globalThis.fetch = async () => new Response('{}', {
        status: 401,
        headers: { 'Content-Type': 'application/json' },
    })

    try {
        const response = await apiFetch('https://example.com/api/auth/login')
        assert.equal(response.status, 401)
        assert.equal(unauthorizedCount, 0)
    } finally {
        globalThis.fetch = originalFetch
        setUnauthorizedHandler(null)
    }
})
