// Starting point (web-svelte, offline data layer): how a page finds the engine the provider created.
import { getContext } from 'svelte';
import type { SyncEngine } from './engine';

export const SYNC_KEY = Symbol('app.sync');

export function getSync(): SyncEngine {
	const engine = getContext<SyncEngine | undefined>(SYNC_KEY);
	if (!engine) throw new Error('getSync() needs the SyncProvider: was the app composed with the offline data layer?');
	return engine;
}
