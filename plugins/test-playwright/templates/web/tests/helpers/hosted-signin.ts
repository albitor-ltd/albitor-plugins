// Starting point (test-playwright). Pure decisions behind auth.setup.ts's hosted sign-in path.

// HOSTED SIGN-IN LEAVES THE APP. An app signed in through an identity provider's hosted page (the
// Cognito classic hosted UI, reached from the app's /sign-in through /api/auth/login) sends the
// browser to a host that is not the app's, and brings it back through /api/auth/callback. The setup
// recognises it by the host alone: it is not told which provider it is, and a page on the app's own
// host is the in-app form.

/**
 * Whether `current` is an http(s) page on a different host (name and port) from the app at
 * `appBase`. about:blank, chrome-error:// and the like are not another host, whatever they parse as.
 */
export function isForeignHost(appBase: string, current: string): boolean {
	try {
		const now = new URL(current);
		if (now.protocol !== "http:" && now.protocol !== "https:") return false;
		return now.host !== new URL(appBase).host;
	} catch {
		return false;
	}
}

/** Whether a URL is the app's own auth plumbing (/api/auth/login, /api/auth/callback), not a page. */
export function isAuthPlumbing(u: URL): boolean {
	return u.pathname === "/api/auth" || u.pathname.startsWith("/api/auth/");
}

/**
 * Whether any cookie in a saved storage state belongs to the app's host (cookies set for the
 * identity provider's host do not count: they prove nothing about the app's session). A cookie's
 * domain matches the host when equal, or when it is a parent domain written `.example.com`.
 */
export function hasAppHostCookie(cookies: { domain?: string }[], appHostname: string): boolean {
	const host = appHostname.toLowerCase();
	return cookies.some((c) => {
		const d = (c.domain ?? "").toLowerCase().replace(/^\./, "");
		return d !== "" && (host === d || host.endsWith(`.${d}`));
	});
}

/** Names of the cookies in a storage state that belong to any of the given hosts. */
export function appHostCookieNames(cookies: { name: string; domain?: string }[], hostnames: string[]): Set<string> {
	return new Set(cookies.filter((c) => hostnames.some((h) => hasAppHostCookie([c], h))).map((c) => c.name));
}

/**
 * Whether sign-in left a NEW cookie on the app: one for the app's host whose name was not among
 * `before`, the names present just before the credentials were submitted. The app's login route
 * sets transient state cookies before redirecting out, and a failed callback leaves them, so
 * "some cookie for the app" proves nothing; the session cookie is new.
 */
export function hasNewAppHostCookie(
	cookies: { name: string; domain?: string }[],
	hostnames: string[],
	before: ReadonlySet<string>,
	isSession: (name: string) => boolean = () => true,
): boolean {
	return cookies.some((c) => !before.has(c.name) && isSession(c.name) && hostnames.some((h) => hasAppHostCookie([c], h)));
}

/**
 * The default for which new app cookies can be a hosted sign-in's session: auth-hosted names its
 * session cookies `__Host-...` and its transient ones `auth_state`, `auth_verifier`, `auth_next`.
 * A locale, CSRF or consent cookie from an ordinary landing page is neither.
 */
export const DEFAULT_SESSION_COOKIE_PATTERN = "^__Host-(?!auth_(state|verifier|next)$)";

/** The session-name predicate from SESSION_COOKIE_PATTERN (a regular expression), or the default. */
export function sessionCookiePredicate(pattern: string | undefined): (name: string) => boolean {
	let re: RegExp;
	try {
		re = new RegExp(pattern?.trim() || DEFAULT_SESSION_COOKIE_PATTERN);
	} catch {
		throw new Error("SESSION_COOKIE_PATTERN is not a valid regular expression");
	}
	return (name) => re.test(name);
}

/** Whether a URL is still the sign-in route: /sign-in or anything under /sign-in/. */
export function isSignInRoute(u: URL): boolean {
	return u.pathname === "/sign-in" || u.pathname.startsWith("/sign-in/");
}

/** A URL for an error message: origin and path, never the query or fragment (codes, state). */
export function withoutQuery(u: string | URL): string {
	try {
		const url = new URL(u);
		return url.origin + url.pathname;
	} catch {
		return String(u).split(/[?#]/)[0];
	}
}

const unquote = (k: string) => k.trim().replace(/^(["'])(.*)\1$/, "$2");

// Whether an inline flow mapping (`{auth: ..., data: ...}`) has `auth` as a top-level key. An
// unterminated one (continues on the next line) cannot be judged here and counts as declaring.
function flowHasAuth(flow: string): boolean {
	// An escaped quote could hide a key from this scan: unreadable, so strict.
	if (flow.includes("\\")) return true;
	let depth = 0;
	let quote = "";
	let key = "";
	let atKey = true;
	for (const ch of flow) {
		if (quote) {
			if (ch === quote) quote = "";
			else if (depth === 1 && atKey) key += ch;
			continue;
		}
		if (ch === '"' || ch === "'") {
			quote = ch;
			continue;
		}
		if (ch === "{" || ch === "[") {
			depth++;
			if (depth === 1) {
				atKey = true;
				key = "";
			}
		} else if (ch === "}" || ch === "]") {
			depth--;
			if (depth === 0) return false;
		} else if (depth === 1) {
			if (ch === ":" && atKey) {
				if (key.trim() === "auth") return true;
				atKey = false;
			} else if (ch === ",") {
				atKey = true;
				key = "";
			} else if (atKey) {
				key += ch;
			}
		}
	}
	return depth > 0;
}

/**
 * Whether an albitor.yaml declares `services.auth`: a `services:` mapping with an `auth:` key
 * beneath it (quoted keys and the inline `services: {auth: ...}` form included). auth-none
 * contributes none, so its absence is how an app with no sign-in is recognised; any real auth
 * plugin contributes it. A line scan, not a YAML parse: these tests carry no parser. A
 * `services:` value that is inline and not a plain mapping (an alias, a tag) cannot be read, and
 * counts as declaring auth: the failure then is a strict setup, never a silent signed-out run.
 */
export function declaresServicesAuth(yaml: string): boolean {
	let inServices = false;
	let childIndent = -1;
	let baseIndent = -1;
	for (const raw of yaml.split(/\r?\n/)) {
		const line = raw.replace(/\s+#.*$/, "");
		const text = line.trimStart();
		if (text === "" || text.startsWith("#")) continue;
		if (text === "---" || text === "...") continue;
		const indent = line.length - text.length;
		// The document's own left margin: an all-indented file's top level is not column 0.
		if (baseIndent < 0) baseIndent = indent;
		const colon = text.search(/:(\s|$)/);
		const key = unquote(colon < 0 ? text : text.slice(0, colon));
		if (indent === baseIndent) {
			inServices = false;
			if (key === "services" && colon >= 0) {
				const rest = text.slice(colon + 1).trim();
				if (rest === "") {
					inServices = true;
					childIndent = -1;
				} else if (rest.startsWith("{")) {
					if (flowHasAuth(rest)) return true;
				} else {
					return true;
				}
			}
			continue;
		}
		if (!inServices) continue;
		if (childIndent < 0) childIndent = indent;
		if (indent === childIndent && key === "auth" && colon >= 0) return true;
	}
	return false;
}

/**
 * Whether a run with this evidence must find a sign-in and sign in. False is the one sanctioned
 * signed-out run: an app with no sign-in, which has no user pool, so no demo account was provisioned
 * and the app declares no services.auth. A password in hand is proof of a user pool, so a missing
 * sign-in screen is then a defect, never a reason to run signed out.
 */
export function signInRequired(evidence: {
	hasPassword: boolean;
	signInScreenFlag: boolean;
	sourceNamesScreen: boolean;
	declaresAuth: boolean;
}): boolean {
	return evidence.hasPassword || evidence.signInScreenFlag || evidence.sourceNamesScreen || evidence.declaresAuth;
}
