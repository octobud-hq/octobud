import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

const API_PROXY_TARGET = process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080';

export default defineConfig({
    plugins: [sveltekit()],
    server: {
        proxy: {
            '/api': {
                target: API_PROXY_TARGET,
                changeOrigin: true,
                secure: false
            }
        }
    },
    test: {
        include: ['src/**/*.{test,spec}.{js,ts}'],
        environment: 'jsdom',
        globals: true,
        setupFiles: ['./src/test/setup.ts']
    }
});

