import { ReactNode, createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'

export type Theme = 'light' | 'dark'

const THEME_STORAGE_KEY = 'app-theme'

interface ThemeContextValue {
    theme: Theme
    setTheme: (theme: Theme) => void
    toggleTheme: () => void
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

function resolveTheme(value?: string | null): Theme | null {
    if (value === 'light' || value === 'dark') {
        return value
    }
    return null
}

function getBrowserTheme(): Theme {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
        return 'light'
    }
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function getInitialTheme(): Theme {
    if (typeof window === 'undefined') return 'light'

    const storedTheme = resolveTheme(window.localStorage.getItem(THEME_STORAGE_KEY))
    if (storedTheme) {
        return storedTheme
    }

    return getBrowserTheme()
}

export function ThemeProvider({ children }: { children: ReactNode }) {
    const [theme, setThemeState] = useState<Theme>(getInitialTheme)

    const setTheme = useCallback((nextTheme: Theme) => {
        setThemeState(nextTheme)
    }, [])

    const toggleTheme = useCallback(() => {
        setThemeState(prev => (prev === 'light' ? 'dark' : 'light'))
    }, [])

    useEffect(() => {
        document.documentElement.setAttribute('data-theme', theme)
        document.documentElement.style.colorScheme = theme
        window.localStorage.setItem(THEME_STORAGE_KEY, theme)
    }, [theme])

    const value = useMemo(() => ({ theme, setTheme, toggleTheme }), [theme, setTheme, toggleTheme])

    return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useTheme() {
    const context = useContext(ThemeContext)
    if (!context) {
        throw new Error('useTheme must be used within ThemeProvider')
    }
    return context
}
