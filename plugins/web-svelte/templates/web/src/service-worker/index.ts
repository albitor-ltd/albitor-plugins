// Starting point (web-svelte). Caches the built app so it still opens offline: assets are
// served cache-first, navigations network-first with the cached app shell as the fallback. The
// API is never cached, so offline data calls fail with a NetworkError the page can show.
import { version } from '$app/env';
import { assets, immutable } from '$app/manifest';

const sw = self as unknown as ServiceWorkerGlobalScope;
const CACHE = `shell-${version}`;
const ASSETS = new Set([...immutable, ...assets].map(({ path }) => (path.startsWith('/') ? path : '/' + path)));
const SHELL = '/index.html';

sw.addEventListener('install', (event) => {
	event.waitUntil(
		(async () => {
			const cache = await caches.open(CACHE);
			try {
				await cache.addAll([...ASSETS]);
			} catch (err) {
				console.error('service worker install failed: could not cache the app assets', err);
				throw err;
			}
			// The SPA fallback is written by the adapter, not listed in `build` or `files`.
			await cache.add(SHELL).catch(() => undefined);
			await sw.skipWaiting();
		})()
	);
});

sw.addEventListener('activate', (event) => {
	event.waitUntil(
		(async () => {
			for (const key of await caches.keys()) {
				if (key !== CACHE) await caches.delete(key);
			}
			await sw.clients.claim();
		})()
	);
});

sw.addEventListener('fetch', (event) => {
	const { request } = event;
	if (request.method !== 'GET') return;
	const url = new URL(request.url);
	if (url.origin !== sw.location.origin || url.pathname.startsWith('/api/')) return;

	if (ASSETS.has(url.pathname)) {
		event.respondWith(
			caches.match(request).then((hit) => hit ?? fetch(request))
		);
		return;
	}

	if (request.mode === 'navigate') {
		event.respondWith(
			fetch(request).catch(async () => {
				const shell = await caches.match(SHELL);
				return shell ?? Response.error();
			})
		);
	}
});
