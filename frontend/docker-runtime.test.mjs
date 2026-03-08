import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

function read(relativePath) {
    return readFileSync(resolve(process.cwd(), relativePath), 'utf8')
}

test('backend container runs as a non-root user', () => {
    const dockerfile = read('backend/Dockerfile')
    assert.match(dockerfile, /^USER\s+\S+/m)
})

test('frontend container runs nginx on a non-privileged port and as a non-root user', () => {
    const dockerfile = read('frontend/Dockerfile')
    const nginxConfig = read('frontend/nginx.conf')
    const compose = read('docker-compose.yml')

    assert.match(dockerfile, /^USER\s+\S+/m)
    assert.match(nginxConfig, /^\s*listen\s+8080;/m)
    assert.match(compose, /FRONTEND_PORT:-5173}:8080/)
})
