<!-- Starting point (web-svelte, offline data layer): the web.providers entry that creates the sync engine,
     follows whoever is signed in, and syncs on its own. One engine per page load. -->
<script lang="ts">
	import { onMount, setContext } from 'svelte';
	import type { Snippet } from 'svelte';
	import { startAutoSync } from './auto';
	import { collections } from './collections';
	import { SYNC_KEY } from './context';
	import { createSyncEngine } from './engine';
	import { withFallback } from './fallback';
	import { idbStores } from './idb';
	import { memoryStores } from './memory';
	import { onSyncUser } from './session';
	import { apiTransport } from './transport';

	let { children }: { children: Snippet } = $props();

	// Without a usable IndexedDB (a private window, a blocked database) the app still works, but only
	// until reload.
	const engine = createSyncEngine({
		stores:
			typeof indexedDB === 'undefined'
				? memoryStores()
				: withFallback(idbStores(), memoryStores(), (err) => console.error('sync: the local database cannot be used, keeping data in memory', err)),
		transport: apiTransport(),
		collections
	});
	setContext(SYNC_KEY, engine);

	onMount(() => {
		const stopAuto = startAutoSync(engine);
		const stopUser = onSyncUser((user) => {
			if (user === null) return engine.detach();
			void engine
				.attach(user)
				.then(() => engine.sync())
				.catch((err) => console.error('sync: could not start for this person', err));
		});
		return () => {
			stopUser();
			stopAuto();
			engine.detach();
		};
	});
</script>

{@render children()}
