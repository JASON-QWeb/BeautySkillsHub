import assert from 'node:assert/strict'
import test from 'node:test'
import { existsSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')

function exists(relativePath) {
    return existsSync(resolve(repoRoot, relativePath))
}

function read(relativePath) {
    return readFileSync(resolve(repoRoot, relativePath), 'utf8')
}

test('scripts are consolidated behind scripts/local.sh', () => {
    assert.equal(exists('scripts/local.sh'), true)
    assert.equal(exists('scripts/db-local.sh'), false)
    assert.equal(exists('scripts/dev-all.sh'), false)
    assert.equal(exists('scripts/run-all-migrations.sh'), false)
    assert.equal(exists('scripts/seed-local.sh'), false)
    assert.equal(exists('scripts/clear-db-data.sh'), false)
})

test('root docs point to the unified local script interface', () => {
    const readme = read('README.md')
    const development = read('DEVELOPMENT.md')

    assert.match(readme, /\.\/scripts\/local\.sh/)
    assert.match(development, /\.\/scripts\/local\.sh/)
    assert.doesNotMatch(readme, /\.\/scripts\/db-local\.sh/)
    assert.doesNotMatch(development, /\.\/scripts\/dev-all\.sh/)
})
