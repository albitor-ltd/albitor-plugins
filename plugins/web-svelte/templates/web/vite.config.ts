// Starting point (web-svelte). SvelteKit 3 is configured here, not in svelte.config.js.
//
// A static single-page app: every route renders in the browser and the host rewrites unknown
// paths to index.html (albitor.yaml `site.spa`).
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

// `npm run dev` forwards /api to the API the local run contract starts on :8080.
const apiOrigin = process.env.APP_API_ORIGIN ?? 'http://127.0.0.1:8080';

export default defineConfig({
	plugins: [sveltekit({ adapter: adapter({ fallback: 'index.html', strict: false }) })],
	server: { proxy: { '/api': apiOrigin } },
	test: { include: ['src/**/*.test.ts'] }
});
