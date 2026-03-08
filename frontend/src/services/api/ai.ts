import { API_BASE } from './client'
import { apiFetch } from './request'

export async function chatWithAI(
    message: string,
    onChunk: (text: string) => void,
    onDone: () => void,
    onError: (error: string) => void,
    signal?: AbortSignal,
): Promise<void> {
    try {
        const res = await apiFetch(`${API_BASE}/ai/chat`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ message }),
            signal,
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
        let eventType = 'message'
        let dataLines: string[] = []

        const resetEvent = () => {
            eventType = 'message'
            dataLines = []
        }

        const parsePayload = (raw: string) => {
            const trimmed = raw.trim()
            if (!trimmed) return ''
            try {
                const parsed = JSON.parse(trimmed)
                if (typeof parsed === 'string') return parsed
                return JSON.stringify(parsed)
            } catch {
                return trimmed
            }
        }

        const dispatchEvent = (): 'continue' | 'done' | 'error' => {
            if (dataLines.length === 0) {
                resetEvent()
                return 'continue'
            }

            const payload = parsePayload(dataLines.join('\n'))
            if (!payload) {
                resetEvent()
                return 'continue'
            }

            if (eventType === 'error') {
                resetEvent()
                onError(payload)
                return 'error'
            }

            if (eventType === 'done' || payload === '[DONE]') {
                resetEvent()
                onDone()
                return 'done'
            }

            onChunk(payload)
            resetEvent()
            return 'continue'
        }

        while (true) {
            const { done, value } = await reader.read()
            if (done) break

            buffer += decoder.decode(value, { stream: true })
            const lines = buffer.split('\n')
            buffer = lines.pop() || ''

            for (const line of lines) {
                const normalized = line.replace(/\r$/, '')
                const trimmed = normalized.trim()
                if (!trimmed) {
                    const status = dispatchEvent()
                    if (status !== 'continue') return
                    continue
                }

                if (trimmed.startsWith(':')) continue
                if (trimmed.startsWith('event:')) {
                    eventType = trimmed.slice(6).trim() || 'message'
                    continue
                }
                if (trimmed.startsWith('data:')) {
                    dataLines.push(trimmed.slice(5).trim())
                    continue
                }
            }
        }

        if (dataLines.length > 0) {
            const status = dispatchEvent()
            if (status !== 'continue') return
        }

        onDone()
    } catch (err) {
        onError(err instanceof Error ? err.message : '网络错误')
    }
}
