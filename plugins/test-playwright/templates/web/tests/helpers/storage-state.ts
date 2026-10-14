// Managed by Albitor (test-playwright).
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

// Where the suite's one signed-in session lives.
//
// IT HAS ONE HOME BECAUSE THREE FILES NEED IT: tests/auth.setup.ts writes it, playwright.config.ts
// hands it to every other project as `storageState`, and a signed-out spec names the empty one.
//
// THE PATH IS ABSOLUTE AND DERIVED FROM THIS FILE, not from the working directory. The suite is
// started from web/tests (`npm test`) and from the repository root (an automated walk, which
// names the journey by a repository-relative path). Playwright resolves a relative
// `storageState` against the working directory, so a relative path would name two different
// files and the walk would browse signed out with no error anywhere.
export const TESTS_DIR = resolve(dirname(fileURLToPath(import.meta.url)), "..");

// web/, the app's front end: the sign-in source scan in auth.setup.ts reads it.
export const WEB_ROOT = resolve(TESTS_DIR, "..");

// The directory is git-ignored: it holds a live session cookie.
export const DEMO_STORAGE_STATE = resolve(TESTS_DIR, ".auth", "demo.json");

// A signed-out session, in the shape Playwright writes and reads. A spec that must NOT inherit
// the signed-in session - the one that tests signing in or registering - opts out with
// `test.use({ storageState: EMPTY_STORAGE_STATE })`.
export const EMPTY_STORAGE_STATE = { cookies: [], origins: [] };

// sessionSaved reports whether a storage state file holds a session: false when it is missing,
// unreadable, or the signed-out shape above.
export function sessionSaved(file: string): boolean {
	try {
		const state = JSON.parse(readFileSync(file, "utf8")) as { cookies?: unknown[]; origins?: unknown[] };
		return (state.cookies?.length ?? 0) > 0 || (state.origins?.length ?? 0) > 0;
	} catch {
		return false;
	}
}

// Whether the setup actually signed in. A spec that needs a session calls this from a beforeEach
// and skips on false. Read at call time, not at import: the setup writes the file after every
// spec file has been loaded.
export function hasDemoSession(): boolean {
	return sessionSaved(DEMO_STORAGE_STATE);
}
