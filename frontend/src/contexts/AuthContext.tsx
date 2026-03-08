import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react'
import { API_BASE } from '../services/api/client'
import { apiFetch, isAbortError, isTokenExpired, setUnauthorizedHandler } from '../services/api/request'

export interface User {
    id: number
    username: string
    avatar_url?: string
    created_at: string
}

interface AuthContextType {
    user: User | null
    token: string | null
    loading: boolean
    login: (username: string, password: string) => Promise<void>
    register: (username: string, password: string) => Promise<void>
    logout: () => void
}

const AuthContext = createContext<AuthContextType | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
    const [user, setUser] = useState<User | null>(null)
    const [token, setToken] = useState<string | null>(() => localStorage.getItem('auth_token'))
    const [loading, setLoading] = useState(true)

    const logout = useCallback(() => {
        setUser(null)
        setToken(null)
        localStorage.removeItem('auth_token')
    }, [])

    useEffect(() => {
        setUnauthorizedHandler(() => logout())
        return () => setUnauthorizedHandler(null)
    }, [logout])

    // Verify token on mount
    useEffect(() => {
        if (!token) {
            setLoading(false)
            return
        }

        if (isTokenExpired(token)) {
            logout()
            setLoading(false)
            return
        }

        const controller = new AbortController()

        apiFetch(`${API_BASE}/auth/me`, {
            auth: true,
            signal: controller.signal,
        })
            .then(res => {
                if (!res.ok) throw new Error('Invalid token')
                return res.json()
            })
            .then(setUser)
            .catch(err => {
                if (isAbortError(err)) return
                logout()
            })
            .finally(() => {
                if (!controller.signal.aborted) {
                    setLoading(false)
                }
            })

        return () => controller.abort()
    }, [token, logout])

    const login = async (username: string, password: string) => {
        const res = await apiFetch(`${API_BASE}/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password }),
        })
        if (!res.ok) {
            const err = await res.json()
            throw new Error(err.error || 'Login failed')
        }
        const data = await res.json()
        setToken(data.token)
        setUser(data.user)
        localStorage.setItem('auth_token', data.token)
    }

    const register = async (username: string, password: string) => {
        const res = await apiFetch(`${API_BASE}/auth/register`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password }),
        })
        if (!res.ok) {
            const err = await res.json()
            throw new Error(err.error || 'Registration failed')
        }
        const data = await res.json()
        setToken(data.token)
        setUser(data.user)
        localStorage.setItem('auth_token', data.token)
    }

    return (
        <AuthContext.Provider value={{ user, token, loading, login, register, logout }}>
            {children}
        </AuthContext.Provider>
    )
}

export function useAuth() {
    const ctx = useContext(AuthContext)
    if (!ctx) throw new Error('useAuth must be used within AuthProvider')
    return ctx
}
