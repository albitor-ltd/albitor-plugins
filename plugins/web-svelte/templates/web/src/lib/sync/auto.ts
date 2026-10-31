// Starting point (web-svelte, offline data layer): when to sync without being asked. After a local write (a
// short delay, so a burst is one run), when the browser comes back online, when the tab becomes
// visible again, and on a timer. Every trigger calls engine.sync(), which shares one guarded run.
import type { SyncEngine } from './engine';

export interface AutoSyncOptions {
	/** Where `online` and `visibilitychange` are heard. Defaults to `window`. */
	events?: EventTarget;
	intervalMs?: number;
	debounceMs?: number;
}

export function startAutoSync(engine: SyncEngine, options: AutoSyncOptions = {}): () => void {
	const events = options.events ?? window;
	const intervalMs = options.intervalMs ?? 30_000;
	const debounceMs = options.debounceMs ?? 400;
	const kick = () => {
		if (engine.owner === null) return;
		// A failure is already reflected in the status; nothing here can act on it.
		void engine.sync().catch(() => undefined);
	};
	let debounce: ReturnType<typeof setTimeout> | undefined;
	const stopWrites = engine.subscribe((kind) => {
		if (kind !== 'write') return;
		clearTimeout(debounce);
		debounce = setTimeout(kick, debounceMs);
	});
	const onVisible = () => {
		if (typeof document === 'undefined' || document.visibilityState === 'visible') kick();
	};
	events.addEventListener('online', kick);
	events.addEventListener('visibilitychange', onVisible);
	const timer = setInterval(kick, intervalMs);
	return () => {
		stopWrites();
		clearTimeout(debounce);
		clearInterval(timer);
		events.removeEventListener('online', kick);
		events.removeEventListener('visibilitychange', onVisible);
	};
}
