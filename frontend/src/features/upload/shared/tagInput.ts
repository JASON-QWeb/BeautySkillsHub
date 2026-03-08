export const MAX_UPLOAD_TAGS = 5

function normalizeSingleTag(rawTag: string) {
    return rawTag.trim().replace(/,+$/g, '').toLowerCase()
}

export function normalizeTagList(rawTags: string, limit = MAX_UPLOAD_TAGS) {
    const seen = new Set<string>()
    const tags: string[] = []

    rawTags
        .split(/[\n\r,]+/)
        .map(normalizeSingleTag)
        .filter(Boolean)
        .forEach(tag => {
            if (seen.has(tag) || tags.length >= limit) return
            seen.add(tag)
            tags.push(tag)
        })

    return tags
}

export function addTagItem(existingTags: string[], rawInput: string, limit = MAX_UPLOAD_TAGS) {
    const next = [...existingTags]
    const seen = new Set(existingTags)

    for (const tag of normalizeTagList(rawInput, limit)) {
        if (seen.has(tag) || next.length >= limit) continue
        seen.add(tag)
        next.push(tag)
    }

    return next
}

export function serializeTagList(tags: string[]) {
    return tags.join(',')
}
