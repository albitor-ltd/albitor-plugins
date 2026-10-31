// Starting point (web-svelte, offline data layer): reactive views of the engine, for use while a component is
// being created. They re-read whenever the local data or the sync state changes (see watch.ts).
import type { SyncEngine } from './engine';
import type { SyncRecord, SyncStatus } from './types';
import { watchEngine, watchOnline } from './watch';

/** `const todos = liveCollection<Todo>(getSync(), 'todos')`; read `todos.items` in the template. */
export function liveCollection<T extends SyncRecord = SyncRecord>(engine: SyncEngine, collection: string) {
	let items = $state<T[]>([]);
	let loaded = $state(false);
	$effect(() =>
		watchEngine(
			engine,
			async () => (engine.owner === null ? [] : ((await engine.list(collection)) as T[])),
			(next) => {
				items = next;
				loaded = true;
			}
		)
	);
	return {
		get items() {
			return items;
		},
		/** False until the first read of the local store finishes. */
		get loaded() {
			return loaded;
		}
	};
}

const idle: SyncStatus = { owner: null, pending: 0, failed: 0, syncing: false, needsSignIn: false };

export function liveStatus(engine: SyncEngine) {
	let status = $state<SyncStatus>(idle);
	let online = $state(true);
	$effect(() => watchEngine(engine, () => engine.status(), (next) => (status = next)));
	$effect(() => watchOnline(window, navigator.onLine, (now) => (online = now)));
	return {
		get status() {
			return status;
		},
		get online() {
			return online;
		}
	};
}
