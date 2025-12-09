import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

const config = {
    kit: {
        adapter: adapter({
            // Build output directory
            pages: 'build',
            assets: 'build',
            fallback: 'index.html', // SPA fallback for client-side routing
            precompress: false,
            strict: true
        }),
        alias: {
            $lib: 'src/lib'
        }
    },
    preprocess: vitePreprocess()
};

export default config;
