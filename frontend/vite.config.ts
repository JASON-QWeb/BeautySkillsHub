import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
    const env = loadEnv(mode, process.cwd(), '')
    const apiBase = (env.VITE_API_BASE_URL || 'http://localhost:8080').trim()
    const proxyTarget = apiBase.replace(/\/api\/?$/, '')

    return {
        plugins: [react()],
        server: {
            port: 5173,
            proxy: {
                '/api': {
                    target: proxyTarget,
                    changeOrigin: true,
                },
            },
        },
    }
})
