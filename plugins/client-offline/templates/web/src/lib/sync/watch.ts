// Managed by Albitor (client-offline): the framework-free core of the reactive views. It reads once,
// then again on every engine change, and drops a read that a newer one has overtaken.
import type { SyncEngine } from './engine';

export function watchEngine<T>(engine: SyncEngine, read: () => Promise<T>, apply: (value: T) => void): () => void {
	let stopped = false;
	let latest = 0;
	const refresh = async () => {
		const mine = ++latest;
		let value: T;
		try {
			value = await read();
		} catch {
			// The person detached between the change and the read; the next change reads again.
			return;
		}
		if (!stopped && mine === latest) apply(value);
	};
	void refresh();
	const unsubscribe = engine.subscribe(() => void refresh());
	return () => {
		stopped = true;
		unsubscribe();
	};
}

/** Reports whether the browser is online now and whenever that changes. */
export function watchOnline(target: EventTarget, initial: boolean, apply: (online: boolean) => void): () => void {
	const up = () => apply(true);
	const down = () => apply(false);
	apply(initial);
	target.addEventListener('online', up);
	target.addEventListener('offline', down);
	return () => {
		target.removeEventListener('online', up);
		target.removeEventListener('offline', down);
	};
}
