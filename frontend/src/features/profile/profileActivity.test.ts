import assert from 'node:assert/strict'
import test from 'node:test'
import { buildProfileActivityAction } from './profileActivity.ts'

const messages: Record<string, string> = {
    'profile.activityTypeResource': '资源',
    'profile.activityTypeSkill': 'Skill',
    'profile.activityTypeMcp': 'MCP',
    'profile.activityTypeRule': 'Rule',
    'profile.activityTypeTool': 'Tool',
    'profile.publishedResource': '发布了 {type}',
    'profile.reviewedResource': '人工复核了 {type}',
    'profile.approvedResource': '通过了 {type} 复核',
}

function t(key: string, values?: Record<string, string | number>) {
    const template = messages[key] ?? key
    if (!values) return template

    return template.replace(/\{(\w+)\}/g, (_, token: string) => {
        const value = values[token]
        return value === undefined ? `{${token}}` : String(value)
    })
}

test('uses specific type label for published activities', () => {
    assert.equal(buildProfileActivityAction(t, 'published', 'tools'), '发布了 Tool')
})

test('uses specific type label for approved review activities', () => {
    assert.equal(buildProfileActivityAction(t, 'approved', 'rules'), '通过了 Rule 复核')
})

test('falls back to generic resource label for unknown types', () => {
    assert.equal(buildProfileActivityAction(t, 'reviewed', 'unknown'), '人工复核了 资源')
})
