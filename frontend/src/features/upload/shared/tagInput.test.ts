import assert from 'node:assert/strict'
import test from 'node:test'

import { addTagItem, normalizeTagList, serializeTagList } from './tagInput.ts'

test('normalizeTagList lowercases, trims, deduplicates, and enforces the upload limit', () => {
    assert.deepEqual(
        normalizeTagList('Frontend, angular\nFRONTEND, dev, tools, extra'),
        ['frontend', 'angular', 'dev', 'tools', 'extra'],
    )
})

test('addTagItem appends a normalized tag for Enter-style submission without duplicating existing items', () => {
    assert.deepEqual(addTagItem(['frontend'], ' Angular,'), ['frontend', 'angular'])
    assert.deepEqual(addTagItem(['frontend'], 'frontend'), ['frontend'])
})

test('serializeTagList persists the exact chip order expected by upload APIs', () => {
    assert.equal(serializeTagList(['frontend', 'angular', 'dev']), 'frontend,angular,dev')
})
