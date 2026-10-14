// Managed by Albitor (client-offline): use the durable store when it opens, the in-memory one when
// it cannot (a private window, a blocked or full database). The app keeps working until a reload.
import type { LocalStore, StoreFactory } from './types';

export function withFallback(primary: StoreFactory, fallback: StoreFactory, onFallback?: (err: unknown) => void): StoreFactory {
	return (owner) => {
		const durable = primary(owner);
		let chosen: Promise<LocalStore> | null = null;
		const pick = () =>
			(chosen ??= durable.getCursor('\u0000probe').then(
				() => durable,
				(err) => {
					onFallback?.(err);
					return fallback(owner);
				}
			));
		return {
			getRecord: async (c, id) => (await pick()).getRecord(c, id),
			putRecord: async (c, r) => (await pick()).putRecord(c, r),
			deleteRecord: async (c, id) => (await pick()).deleteRecord(c, id),
			listRecords: async (c) => (await pick()).listRecords(c),
			listOutbox: async () => (await pick()).listOutbox(),
			putOutbox: async (e) => (await pick()).putOutbox(e),
			deleteOutbox: async (id) => (await pick()).deleteOutbox(id),
			getCursor: async (c) => (await pick()).getCursor(c),
			setCursor: async (c, v) => (await pick()).setCursor(c, v),
			clear: async () => (await pick()).clear()
		};
	};
}
