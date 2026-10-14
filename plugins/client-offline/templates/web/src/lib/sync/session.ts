// Managed by Albitor (client-offline): who is signed in, as far as the sync layer is concerned.
//
// The auth plugin's provider calls `setSyncUser(id)` when a session starts or its person changes,
// and `setSyncUser(null)` when it ends. The id must be the stable user id (the token's subject),
// never an email or a display name, because it names the person's local database.
let user: string | null = null;
const listeners = new Set<(user: string | null) => void>();

export function setSyncUser(id: string | null): void {
	if (id !== null && (typeof id !== 'string' || id === '')) {
		throw new Error('the sync user id must be a non-empty string or null');
	}
	if (id === user) return;
	user = id;
	listeners.forEach((l) => l(user));
}

export function getSyncUser(): string | null {
	return user;
}

/** Calls `fn` now with the current user and again on every change. Returns the unsubscribe. */
export function onSyncUser(fn: (user: string | null) => void): () => void {
	listeners.add(fn);
	fn(user);
	return () => listeners.delete(fn);
}
