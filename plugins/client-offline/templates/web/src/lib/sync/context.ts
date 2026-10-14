// Managed by Albitor (client-offline): how a page finds the engine the provider created.
import { getContext } from 'svelte';
import type { SyncEngine } from './engine';

export const SYNC_KEY = Symbol('albitor.sync');

export function getSync(): SyncEngine {
	const engine = getContext<SyncEngine | undefined>(SYNC_KEY);
	if (!engine) throw new Error('getSync() needs the SyncProvider: is client-offline composed?');
	return engine;
}
