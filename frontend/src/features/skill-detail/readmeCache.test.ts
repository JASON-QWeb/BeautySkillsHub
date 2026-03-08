import assert from 'node:assert/strict'
import test from 'node:test'

import { createReadmeCache } from './readmeCache.ts'

test('readme cache returns stored entries', () => {
    const cache = createReadmeCache(2)
    cache.set(1, '# hello')

    assert.equal(cache.get(1), '# hello')
})

test('readme cache evicts least recently used entry when full', () => {
    const cache = createReadmeCache(2)
    cache.set(1, 'one')
    cache.set(2, 'two')

    assert.equal(cache.get(1), 'one')

    cache.set(3, 'three')

    assert.equal(cache.get(1), 'one')
    assert.equal(cache.get(2), undefined)
    assert.equal(cache.get(3), 'three')
})
