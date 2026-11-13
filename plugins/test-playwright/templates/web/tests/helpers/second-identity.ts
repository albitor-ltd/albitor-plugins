// Starting point (test-playwright). The second verifier identity's sign-in, which never throws.
import { readFileSync, rmSync, writeFileSync } from "node:fs";
import { sessionSaved, VERIFIER_B_STORAGE_STATE } from "./storage-state.ts";

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

// WHY A SKIPPED SPEC SAYS THIS. An automated walk reads a skip for want of a second person as
// "unverified", so the wording is part of the contract: keep the leading "no second signed-in
// identity".
export const NO_SECOND_IDENTITY =
	"no second signed-in identity: this run has neither the local issuer (bob) nor a verifier-b session";

// The cookies an auth plugin keeps the session's ID token in: auth-cognito `session`, auth-hosted
// `__Host-session`. The API accepts that same token as a bearer header (auth-hosted: "the token
// arrives as a bearer header or a session cookie"), which is how verifier-b's session becomes a
// bearer on a direct `request` call.
const SESSION_COOKIES = ["session", "__Host-session"];

function looksLikeJwt(value: unknown): value is string {
	return typeof value === "string" && /^[\w-]+\.[\w-]+\.[\w-]+$/.test(value);
}

// bearerFromStorageState reads the second identity's ID token out of its saved browser state: a
// session cookie first, then a local-storage value whose key names an ID token. It returns
// undefined when the file is missing, unreadable, or holds no token shaped like a JWT.
export function bearerFromStorageState(file: string = VERIFIER_B_STORAGE_STATE): string | undefined {
	let state: {
		cookies?: { name?: string; value?: string }[];
		origins?: { localStorage?: { name?: string; value?: string }[] }[];
	};
	try {
		state = JSON.parse(readFileSync(file, "utf8"));
	} catch {
		return undefined;
	}
	for (const cookie of state.cookies ?? []) {
		if (SESSION_COOKIES.includes(cookie.name ?? "") && looksLikeJwt(cookie.value)) {
			return cookie.value;
		}
	}
	for (const origin of state.origins ?? []) {
		for (const item of origin.localStorage ?? []) {
			if (/id[_-]?token/i.test(item.name ?? "") && looksLikeJwt(item.value)) {
				return item.value;
			}
		}
	}
	return undefined;
}

// resolveSecondIdentity is the pure half of secondIdentityHeaders (helpers/local-auth.ts): the
// second person's Authorization header, or null when this run has none. `local` mints bob's header
// from the local issuer (null when there is no issuer); a deployed run falls back to verifier-b's
// saved session at `file`.
export async function resolveSecondIdentity(
	local: () => Promise<Record<string, string> | null>,
	file: string = VERIFIER_B_STORAGE_STATE,
): Promise<Record<string, string> | null> {
	const bob = await local();
	if (bob) return bob;
	const token = bearerFromStorageState(file);
	return token ? { Authorization: `Bearer ${token}` } : null;
}
