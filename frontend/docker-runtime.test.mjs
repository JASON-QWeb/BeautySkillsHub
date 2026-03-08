import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

function read(relativePath) {
    return readFileSync(resolve(process.cwd(), relativePath), 'utf8')
}

test('backend container runs as a non-root user', () => {
    const dockerfile = read('backend/Dockerfile')
    const dockerignore = read('backend/.dockerignore')

    assert.match(dockerfile, /^USER\s+\S+/m)
    assert.match(dockerignore, /^\.env\.local(\.\*)?$/m)
})

test('frontend container runs nginx on a non-privileged port and as a non-root user', () => {
    const dockerfile = read('frontend/Dockerfile')
    const dockerignore = read('frontend/.dockerignore')
    const nginxConfig = read('frontend/nginx.conf')
    const compose = read('docker-compose.yml')

    assert.match(dockerfile, /^FROM nginxinc\/nginx-unprivileged:alpine/m)
    assert.match(dockerignore, /^\.env\.local(\.\*)?$/m)
    assert.match(nginxConfig, /^\s*listen\s+8080;/m)
    assert.match(compose, /FRONTEND_PORT:-5173}:8080/)
})

test('docker compose boots locally without a private backend env file', () => {
    const compose = read('docker-compose.yml')

    assert.doesNotMatch(compose, /^\s*env_file:\s*$/m)
    assert.match(compose, /APP_ENV:\s*"*\$\{APP_ENV:-local\}"*/)
    assert.match(compose, /DATABASE_URL:\s*"*postgres:\/\/\$\{POSTGRES_USER:-skillhub\}:\$\{POSTGRES_PASSWORD:-skillhub\}@postgres:5432\/\$\{POSTGRES_DB:-skillhub_local\}\?sslmode=disable"*/)
    assert.match(compose, /JWT_SECRET:\s*"*\$\{JWT_SECRET:-local-dev-secret\}"*/)
    assert.match(compose, /GITHUB_SYNC_ENABLED:\s*"*\$\{GITHUB_SYNC_ENABLED:-false\}"*/)
})

test('verify workflow checks docker runtime regressions and compose smoke startup', () => {
    const workflow = read('.github/workflows/verify.yml')

    assert.match(workflow, /node --test/)
    assert.match(workflow, /frontend\/docker-runtime\.test\.mjs/)
    assert.match(workflow, /docker compose up -d --build/)
    assert.match(workflow, /curl -sf http:\/\/127\.0\.0\.1:8080\/health/)
    assert.match(workflow, /curl -I http:\/\/127\.0\.0\.1:5173/)
    assert.match(workflow, /docker compose down -v/)
})
