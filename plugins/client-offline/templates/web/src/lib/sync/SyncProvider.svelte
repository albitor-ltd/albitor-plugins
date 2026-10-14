<!-- Managed by Albitor (client-offline): the web.providers entry that creates the sync engine,
     follows whoever is signed in, and syncs on its own. One engine per page load. -->
<script lang="ts">
	import { onMount, setContext } from 'svelte';
	import type { Snippet } from 'svelte';
	import { startAutoSync } from './auto';
	import { collections } from './collections';
	import { SYNC_KEY } from './context';
	import { createSyncEngine } from './engine';
	import { idbStores } from './idb';
	import { memoryStores } from './memory';
	import { onSyncUser } from './session';
	import { apiTransport } from './transport';

	let { children }: { children: Snippet } = $props();

	// Without IndexedDB (a private window in some browsers) the app still works, but only until reload.
	const engine = createSyncEngine({
		stores: typeof indexedDB === 'undefined' ? memoryStores() : idbStores(),
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
				.catch(() => undefined);
		});
		return () => {
			stopUser();
			stopAuto();
			engine.detach();
		};
	});
</script>

{@render children()}
