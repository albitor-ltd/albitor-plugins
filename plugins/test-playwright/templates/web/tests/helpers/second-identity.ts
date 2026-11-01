// Starting point (test-playwright). The second verifier identity's sign-in, which never throws.
import { rmSync, writeFileSync } from "node:fs";
import { sessionSaved } from "./storage-state.ts";

// saveSecondSession runs `signIn` and saves the browser state it returns at `file`. It NEVER
// throws: a failed sign-in, a state that holds no session, or a write that fails leaves no file at
// `file` and returns why, so the caller can log it and the specs that need a second person skip by
// hasVerifierBSession(). The second account is the platform's to seed, not the app's, so its
// failure must not fail the setup project and skip the whole suite with it.
export async function saveSecondSession(
	signIn: () => Promise<unknown>,
	file: string,
): Promise<{ ok: true } | { ok: false; why: string }> {
	rmSync(file, { force: true });
	try {
		writeFileSync(file, JSON.stringify(await signIn()));
		if (!sessionSaved(file)) {
			throw new Error("the sign-in succeeded but saved no cookies or local storage");
		}
		return { ok: true };
	} catch (err) {
		rmSync(file, { force: true });
		return { ok: false, why: err instanceof Error ? err.message : "failed" };
	}
}
