// Managed by Albitor (test-playwright).
import { createHash } from "node:crypto";
import { chmodSync, lstatSync, mkdirSync, readFileSync } from "node:fs";
import { tmpdir, userInfo } from "node:os";
import { dirname, join, resolve } from "node:path";
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

// THE SESSION LIVES OUTSIDE THE REPOSITORY. It holds a live session cookie, so it is written to a
// private (0700) directory under the OS temp dir rather than into the tree, where a secrets scan
// or a `git add -A` could reach it. ALBITOR_TESTS_AUTH_DIR overrides the directory. The default
// is named from this checkout's path and the user, so two apps on one machine never share a
// session and every process that needs it computes the same path.
function authDir(): string {
	const override = process.env.ALBITOR_TESTS_AUTH_DIR;
	if (override) return resolve(override);
	const checkout = createHash("sha256").update(TESTS_DIR).digest("hex").slice(0, 12);
	return join(tmpdir(), `albitor-tests-auth-${userInfo().uid}-${checkout}`);
}

export const AUTH_DIR = authDir();

export const DEMO_STORAGE_STATE = join(AUTH_DIR, "demo.json");

// ensureAuthDir creates the session directory, private to the user (0700), and returns it.
// It refuses a path that is not a directory owned by this user: in a shared temp dir a
// pre-made one would let another user read the session.
export function ensureAuthDir(): string {
	mkdirSync(AUTH_DIR, { recursive: true, mode: 0o700 });
	const info = lstatSync(AUTH_DIR);
	if (!info.isDirectory() || (process.platform !== "win32" && info.uid !== userInfo().uid)) {
		throw new Error(`${AUTH_DIR} is not a directory owned by this user; set ALBITOR_TESTS_AUTH_DIR to a private one`);
	}
	chmodSync(AUTH_DIR, 0o700);
	return AUTH_DIR;
}

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
