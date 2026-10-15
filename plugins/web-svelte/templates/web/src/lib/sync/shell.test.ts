// The offline promise needs the shell itself to open with no network. SvelteKit emits a service
// worker only when src/service-worker exists, and the worker only helps if it caches the app shell
// and stays away from the API. These checks read the worker the shell ships (web-svelte), so a
// build that drops or guts it fails here, and the compose check confirms the built output.
import { readFileSync, existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const worker = resolve(process.cwd(), 'src/service-worker/index.ts');

describe('the service worker the shell ships', () => {
	it('exists, so the build emits service-worker.js into the deploy output', () => {
		expect(existsSync(worker)).toBe(true);
	});

	it('precaches the built assets and the SPA shell at install', () => {
		const src = readFileSync(worker, 'utf8');
		expect(src).toMatch(/addEventListener\('install'/);
		expect(src).toMatch(/cache\.addAll\(/);
		expect(src).toContain("'/index.html'");
		expect(src).toMatch(/\$app\/manifest/);
	});

	it('falls back to the cached shell for a navigation made offline', () => {
		const src = readFileSync(worker, 'utf8');
		expect(src).toMatch(/request\.mode === 'navigate'/);
		expect(src).toMatch(/caches\.match\(SHELL\)/);
	});

	it('never caches the API: stale data would hide the outbox from the person', () => {
		const src = readFileSync(worker, 'utf8');
		expect(src).toContain("startsWith('/api/')");
		expect(src).toMatch(/request\.method !== 'GET'\) return/);
	});
});
