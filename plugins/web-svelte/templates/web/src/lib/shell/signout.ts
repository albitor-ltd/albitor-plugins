// Work that must finish before a person is signed out: wiping the data layer's local copy
// (data-offline: `sync.signOut()`; data-online: `queryClient.clear()`). The Sign out control
// (AccountBar) runs these first, then posts the form. Auth never imports a data layer, so the
// app registers its own here. With no script the form still posts: the wipe is skipped, which is
// why the app should also clear on the sign-in screen.
export type BeforeSignOut = () => unknown | Promise<unknown>;

const hooks = new Set<BeforeSignOut>();

/** Registers a hook; returns a function that removes it. Return `false` (or throw) to stop the sign-out. */
export function onBeforeSignOut(hook: BeforeSignOut): () => void {
	hooks.add(hook);
	return () => hooks.delete(hook);
}

/** Awaits every hook in turn. False when one asked to stop (returned false or threw). */
export async function runBeforeSignOut(): Promise<boolean> {
	for (const hook of [...hooks]) {
		try {
			if ((await hook()) === false) return false;
		} catch {
			return false;
		}
	}
	return true;
}
