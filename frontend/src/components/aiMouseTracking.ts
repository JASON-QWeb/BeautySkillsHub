type RAFRequest = (callback: FrameRequestCallback) => number
type RAFCancel = (id: number) => void
type MouseUpdate = (x: number, y: number) => void

export type MouseMoveScheduler = ((x: number, y: number) => void) & { cancel: () => void }

export function createMouseMoveScheduler(
    requestFrame: RAFRequest,
    cancelFrame: RAFCancel,
    applyUpdate: MouseUpdate,
): MouseMoveScheduler {
    let frameId: number | null = null
    let nextPosition = { x: 0, y: 0 }

    const flush = () => {
        frameId = null
        applyUpdate(nextPosition.x, nextPosition.y)
    }

    const schedule = ((x: number, y: number) => {
        nextPosition = { x, y }
        if (frameId !== null) return
        frameId = requestFrame(() => flush())
    }) as MouseMoveScheduler

    schedule.cancel = () => {
        if (frameId === null) return
        cancelFrame(frameId)
        frameId = null
    }

    return schedule
}
