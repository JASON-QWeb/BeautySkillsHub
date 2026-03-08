import assert from 'node:assert/strict'
import test from 'node:test'

import { getDialogEscapeResult } from './dialogKeyboard.ts'

test('escape closes confirm dialogs with false result', () => {
    assert.equal(getDialogEscapeResult('Escape', 'confirm'), false)
})

test('escape closes alert dialogs with true result', () => {
    assert.equal(getDialogEscapeResult('Escape', 'alert'), true)
})

test('other keys do not close dialogs', () => {
    assert.equal(getDialogEscapeResult('Enter', 'confirm'), null)
})
