export interface ReadmeCache {
    get: (skillId: number) => string | undefined
    set: (skillId: number, content: string) => void
    delete: (skillId: number) => void
}

export function createReadmeCache(maxEntries: number): ReadmeCache {
    const cache = new Map<number, string>()

    const touch = (skillId: number, content: string) => {
        cache.delete(skillId)
        cache.set(skillId, content)
    }

    return {
        get(skillId: number) {
            const content = cache.get(skillId)
            if (content === undefined) return undefined
            touch(skillId, content)
            return content
        },
        set(skillId: number, content: string) {
            touch(skillId, content)
            while (cache.size > maxEntries) {
                const oldestKey = cache.keys().next().value
                if (oldestKey === undefined) return
                cache.delete(oldestKey)
            }
        },
        delete(skillId: number) {
            cache.delete(skillId)
        },
    }
}
