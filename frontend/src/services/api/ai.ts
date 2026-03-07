import { API_BASE } from './client'

export async function chatWithAI(
    message: string,
    onChunk: (text: string) => void,
    onDone: () => void,
    onError: (error: string) => void,
): Promise<void> {
    try {
        const res = await fetch(`${API_BASE}/ai/chat`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ message }),
        })

        if (!res.ok) {
            const err = await res.json()
            onError(err.error || 'AI 请求失败')
            return
        }

        const reader = res.body?.getReader()
        if (!reader) {
            onError('无法读取响应流')
            return
        }

        const decoder = new TextDecoder()
        let buffer = ''

        while (true) {
            const { done, value } = await reader.read()
            if (done) break

            buffer += decoder.decode(value, { stream: true })
            const lines = buffer.split('\n')
            buffer = lines.pop() || ''

            for (const line of lines) {
                const trimmed = line.trim()
                if (!trimmed) continue

                if (trimmed.startsWith('data:')) {
                    const data = trimmed.slice(5).trim()
                    if (!data) continue

                    if (data === '[DONE]') {
                        onDone()
                        return
                    }

                    try {
                        const parsed = JSON.parse(data)
                        if (typeof parsed === 'string') {
                            onChunk(parsed)
                        }
                    } catch {
                        onChunk(data)
                    }
                }
            }
        }

        onDone()
    } catch (err) {
        onError(err instanceof Error ? err.message : '网络错误')
    }
}
