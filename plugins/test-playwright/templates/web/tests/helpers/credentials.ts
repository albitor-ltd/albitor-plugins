// Managed by Albitor (test-playwright). Where the one sign-in gets its account.
import { readFileSync } from "node:fs";
import { safeOpenPath } from "./screen-rules.ts";

export type DemoCredentials = {
	username: string;
	password: string;
	signInRoute: string;
};

const DEFAULT_ROUTE = "/sign-in";

// TWO ROUTES IN, AND THE FILE WINS.
//
// An automated walk of this app's journeys writes a JSON file, mode 0600, named by
// ALBITOR_JOURNEY_CREDENTIALS and deleted when the walk ends, holding
// { "username", "password", "sign_in_route" }. A file has a lifetime and an owner; an environment
// is handed to every browser process the run spawns and printed by any step that dumps it.
// Outside a walk the same three values come from the environment, and the password is never in
// this repository: it is provisioned into the app's own parameter store and reaches the run from
// outside the checkout.
//
// Neither being present is the ordinary case for an app with no sign-in, so reading is forgiving:
// an unreadable or unexpected file falls through to the environment and then to a signed-out
// session. The sign-in route must be a path on the app; anything a browser would read as another
// host is replaced by the default.
export function readCredentials(env: Record<string, string | undefined>): DemoCredentials {
	const fromEnv: DemoCredentials = {
		// The app's own seeded demo user: the account its owner signs in with, and the one the
		// app's screens are photographed through.
		username: env.DEMO_ACCOUNT_USERNAME || "demo@albitor-demo.invalid",
		password: env.ALBITOR_DEMO_ACCOUNT_PASSWORD || "",
		signInRoute: safeOpenPath(env.DEMO_ACCOUNT_SIGN_IN_ROUTE) ?? DEFAULT_ROUTE,
	};

	const credentialsPath = env.ALBITOR_JOURNEY_CREDENTIALS || "";
	if (!credentialsPath) {
		return fromEnv;
	}
	try {
		const parsed = JSON.parse(readFileSync(credentialsPath, "utf8")) as {
			username?: string;
			password?: string;
			sign_in_route?: string;
		};
		if (!parsed.password || typeof parsed.password !== "string") {
			return fromEnv;
		}
		return {
			username: typeof parsed.username === "string" && parsed.username ? parsed.username : fromEnv.username,
			password: parsed.password,
			signInRoute: safeOpenPath(parsed.sign_in_route) ?? fromEnv.signInRoute,
		};
	} catch {
		return fromEnv;
	}
}

/**
 * Whether the page has left the sign-in route. `signIn` is the URL recorded right after the
 * sign-in page loaded (so a redirect to /login is the route), `now` the current one. Query strings
 * do not count (a challenge page on the same route often adds one); under hash routing
 * (`#/sign-in`) the hash is part of the route. A same-route challenge page is not signed in.
 */
export function leftSignInRoute(signIn: URL, now: URL): boolean {
	if (now.origin !== signIn.origin) return true;
	const hashRouted = signIn.hash.startsWith("#/");
	const route = (u: URL) => (hashRouted ? u.pathname + u.hash : u.pathname);
	return route(now) !== route(signIn);
}
