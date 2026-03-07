import { ReactNode, createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { DEFAULT_LANGUAGE, Language, TranslationValues, getBrowserLanguage, resolveLanguage, translate } from './index'

const LANGUAGE_STORAGE_KEY = 'app-language'

interface I18nContextValue {
    language: Language
    setLanguage: (language: Language) => void
    toggleLanguage: () => void
    t: (key: string, values?: TranslationValues) => string
}

const I18nContext = createContext<I18nContextValue | null>(null)

function getInitialLanguage(): Language {
    if (typeof window === 'undefined') return DEFAULT_LANGUAGE

    const stored = window.localStorage.getItem(LANGUAGE_STORAGE_KEY)
    if (stored) {
        return resolveLanguage(stored)
    }

    return getBrowserLanguage()
}

export function I18nProvider({ children }: { children: ReactNode }) {
    const [language, setLanguageState] = useState<Language>(getInitialLanguage)

    const setLanguage = useCallback((nextLanguage: Language) => {
        setLanguageState(nextLanguage)
    }, [])

    const toggleLanguage = useCallback(() => {
        setLanguageState(prev => (prev === 'en' ? 'zh' : 'en'))
    }, [])

    useEffect(() => {
        document.documentElement.setAttribute('lang', language)
        window.localStorage.setItem(LANGUAGE_STORAGE_KEY, language)
    }, [language])

    const t = useCallback((key: string, values?: TranslationValues) => {
        return translate(language, key, values)
    }, [language])

    const value = useMemo(
        () => ({ language, setLanguage, toggleLanguage, t }),
        [language, setLanguage, toggleLanguage, t],
    )

    return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n() {
    const context = useContext(I18nContext)
    if (!context) {
        throw new Error('useI18n must be used within I18nProvider')
    }
    return context
}
