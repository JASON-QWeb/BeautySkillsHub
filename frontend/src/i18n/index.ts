import en from './locales/en'
import zh from './locales/zh'

export type Language = 'en' | 'zh'
export type TranslationValues = Record<string, string | number>

type Messages = Record<string, string>

const messages: Record<Language, Messages> = { en, zh }

export const DEFAULT_LANGUAGE: Language = 'en'

export function resolveLanguage(value?: string | null): Language {
    if (!value) return DEFAULT_LANGUAGE
    const normalized = value.toLowerCase()
    return normalized.startsWith('zh') ? 'zh' : 'en'
}

export function getBrowserLanguage(): Language {
    if (typeof navigator === 'undefined') return DEFAULT_LANGUAGE

    const candidates = [navigator.language, ...(navigator.languages || [])]
    for (const lang of candidates) {
        if (lang && lang.toLowerCase().startsWith('zh')) {
            return 'zh'
        }
    }

    return 'en'
}

export function translate(language: Language, key: string, values?: TranslationValues): string {
    const template = messages[language][key] ?? messages.en[key] ?? key
    if (!values) return template

    return template.replace(/\{(\w+)\}/g, (_, token: string) => {
        const value = values[token]
        return value === undefined ? `{${token}}` : String(value)
    })
}
