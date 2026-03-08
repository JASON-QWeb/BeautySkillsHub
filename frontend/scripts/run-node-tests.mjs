import { spawnSync } from 'node:child_process'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const npxCommand = process.platform === 'win32' ? 'npx.cmd' : 'npx'

const testFiles = [
    'src/services/api/request.test.ts',
    'src/features/skill-detail/readmeCache.test.ts',
    'src/contexts/dialogKeydown.test.ts',
    'src/components/aiMouseTracking.test.ts',
    'src/features/profile/profileActivity.test.ts',
    'src/features/upload/shared/tagInput.test.ts',
    'docker-runtime.test.mjs',
]

const result = spawnSync(npxCommand, ['--no-install', 'tsx', '--test', ...testFiles], {
    cwd: frontendRoot,
    stdio: 'inherit',
})

if (result.error) {
    throw result.error
}

process.exit(result.status ?? 1)
