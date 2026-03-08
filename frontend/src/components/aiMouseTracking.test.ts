import assert from 'node:assert/strict'
import test from 'node:test'

import { createMouseMoveScheduler } from './aiMouseTracking.ts'

test('mouse scheduler coalesces multiple moves into one frame update', () => {
    const queue: FrameRequestCallback[] = []
    const updates: Array<{ x: number; y: number }> = []
    const schedule = createMouseMoveScheduler(
        (cb) => {
            queue.push(cb)
            return queue.length
        },
        (_id) => {},
        (x, y) => {
            updates.push({ x, y })
        },
    )

    schedule(10, 20)
    schedule(15, 25)
    schedule(30, 40)

    assert.equal(queue.length, 1)

    queue[0](16)

    assert.deepEqual(updates, [{ x: 30, y: 40 }])
})
