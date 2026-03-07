export interface ReadmeFrontMatter {
    name?: string
    description?: string
    [key: string]: string | undefined
}

export interface ParsedReadme {
    body: string
    frontMatter: ReadmeFrontMatter
}

const FRONTMATTER_RE = /^\uFEFF?\s*---\s*\r?\n([\s\S]*?)\r?\n---\s*(?:\r?\n)?/

function stripWrappingQuotes(value: string): string {
    const trimmed = value.trim()
    if (trimmed.length < 2) return trimmed
    const first = trimmed[0]
    const last = trimmed[trimmed.length - 1]
    if ((first === '"' && last === '"') || (first === '\'' && last === '\'')) {
        return trimmed.slice(1, -1).trim()
    }
    return trimmed
}

export function parseReadmeFrontMatter(raw: string): ParsedReadme {
    const source = raw || ''
    const match = source.match(FRONTMATTER_RE)
    if (!match) {
        return {
            body: source,
            frontMatter: {},
        }
    }

    const metadata: ReadmeFrontMatter = {}
    const lines = match[1].split(/\r?\n/)
    for (const line of lines) {
        const trimmed = line.trim()
        if (!trimmed || trimmed.startsWith('#')) continue

        const kv = trimmed.match(/^([A-Za-z0-9_-]+)\s*:\s*(.*)$/)
        if (!kv) continue

        const key = kv[1].trim().toLowerCase()
        const value = stripWrappingQuotes(kv[2])
        if (!key || !value) continue
        metadata[key] = value
    }

    return {
        body: source.slice(match[0].length),
        frontMatter: metadata,
    }
}
