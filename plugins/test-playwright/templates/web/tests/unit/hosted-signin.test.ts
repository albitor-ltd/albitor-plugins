// Managed by Albitor (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
	appHostCookieNames,
	declaresServicesAuth,
	hasAppHostCookie,
	hasNewAppHostCookie,
	sessionCookiePredicate,
	isAuthPlumbing,
	isForeignHost,
	isSignInRoute,
	signInRequired,
	withoutQuery,
} from "../helpers/hosted-signin.ts";

describe("isForeignHost", () => {
	const app = "https://app.example.com";
	it("is false on the app's own host, whatever the path", () => {
		assert.equal(isForeignHost(app, "https://app.example.com/sign-in"), false);
		assert.equal(isForeignHost(app, "https://app.example.com/api/auth/callback?code=x"), false);
	});
	it("is true on the Cognito hosted UI host", () => {
		assert.equal(isForeignHost(app, "https://my-app.auth.eu-west-2.amazoncognito.com/login?client_id=x"), true);
	});
	it("compares the port too", () => {
		assert.equal(isForeignHost("http://localhost:4173", "http://localhost:4174/"), true);
		assert.equal(isForeignHost("http://localhost:4173", "http://localhost:4173/sign-in"), false);
	});
	it("is false for pages that are not http(s): about:blank, chrome-error://, data:", () => {
		assert.equal(isForeignHost(app, "about:blank"), false);
		assert.equal(isForeignHost(app, "chrome-error://chromewebdata/"), false);
		assert.equal(isForeignHost(app, "data:text/html,x"), false);
	});
	it("is false for a URL it cannot parse", () => {
		assert.equal(isForeignHost(app, "not a url"), false);
	});
});

describe("declaresServicesAuth", () => {
	it("finds auth under services", () => {
		assert.equal(declaresServicesAuth("name: x\nservices:\n  auth:\n    provider: cognito\n"), true);
	});
	it("is false when services has no auth (auth-none)", () => {
		assert.equal(declaresServicesAuth("services:\n  data:\n    provider: dynamodb\n"), false);
		assert.equal(declaresServicesAuth("name: x\n"), false);
	});
	it("ignores auth keys elsewhere and comments", () => {
		assert.equal(declaresServicesAuth("auth:\n  x: 1\nservices:\n  data: {}\n"), false);
		assert.equal(declaresServicesAuth("services:\n  # auth:\n  data: {}\n"), false);
		assert.equal(declaresServicesAuth("services:\n  data:\n    auth: none\n"), false);
	});
	it("stops at the end of the services block", () => {
		assert.equal(declaresServicesAuth("services:\n  data: {}\nother:\n  auth: {}\n"), false);
	});
});

describe("declaresServicesAuth, other spellings", () => {
	it("reads quoted keys", () => {
		assert.equal(declaresServicesAuth('"services":\n  "auth":\n    provider: x\n'), true);
		assert.equal(declaresServicesAuth("'services':\n  'data': {}\n"), false);
	});
	it("reads the inline flow form", () => {
		assert.equal(declaresServicesAuth("services: {auth: {provider: cognito}}\n"), true);
		assert.equal(declaresServicesAuth('services: {"data": {}, "auth": {}}\n'), true);
		assert.equal(declaresServicesAuth("services: {data: {auth: x}}\n"), false);
		assert.equal(declaresServicesAuth("services: {}\n"), false);
	});
	it("counts a services value it cannot read as declaring auth", () => {
		assert.equal(declaresServicesAuth("services: *shared\n"), true);
		assert.equal(declaresServicesAuth("services: {data: {},\n  other: {}}\n"), true);
	});
});

describe("isAuthPlumbing", () => {
	it("is the callback and login routes, not pages", () => {
		assert.equal(isAuthPlumbing(new URL("https://a.test/api/auth/callback?code=x")), true);
		assert.equal(isAuthPlumbing(new URL("https://a.test/api/auth/login")), true);
		assert.equal(isAuthPlumbing(new URL("https://a.test/api/authors")), false);
		assert.equal(isAuthPlumbing(new URL("https://a.test/")), false);
	});
});

describe("hasAppHostCookie", () => {
	it("counts only cookies for the app's host", () => {
		assert.equal(hasAppHostCookie([{ domain: "my.auth.eu-west-2.amazoncognito.com" }], "app.example.com"), false);
		assert.equal(hasAppHostCookie([{ domain: "app.example.com" }], "app.example.com"), true);
		assert.equal(hasAppHostCookie([{ domain: ".example.com" }], "app.example.com"), true);
		assert.equal(hasAppHostCookie([{ domain: "other.com" }, { domain: "" }], "app.example.com"), false);
		assert.equal(hasAppHostCookie([], "app.example.com"), false);
	});
});

describe("declaresServicesAuth, hard documents", () => {
	it("fails strict on a backslash in an inline services value", () => {
		assert.equal(declaresServicesAuth('services: {"da\\"auth": 1, data: {}}\n'), true);
	});
	it("reads an all-indented document at its own margin", () => {
		assert.equal(declaresServicesAuth("  name: x\n  services:\n    auth: {}\n"), true);
		assert.equal(declaresServicesAuth("  name: x\n  services:\n    data: {}\n"), false);
		assert.equal(declaresServicesAuth("  services: {auth: {}}\n"), true);
		assert.equal(declaresServicesAuth("---\nservices:\n  auth: {}\n"), true);
	});
});

describe("new cookies", () => {
	const host = ["app.test"];
	const state = { name: "__Host-auth_state", domain: "app.test" };
	const session = { name: "session", domain: "app.test" };
	it("names only the app's cookies", () => {
		assert.deepEqual([...appHostCookieNames([state, { name: "x", domain: "cognito.test" }], host)], ["__Host-auth_state"]);
	});
	it("does not count a transient cookie that was there before the submit", () => {
		const before = appHostCookieNames([state], host);
		assert.equal(hasNewAppHostCookie([state], host, before), false);
		assert.equal(hasNewAppHostCookie([state, session], host, before), true);
	});
	it("does not count a new cookie on a foreign host", () => {
		assert.equal(hasNewAppHostCookie([{ name: "XSRF-TOKEN", domain: "cognito.test" }], host, new Set()), false);
	});
	it("counts a landed host named alongside the base host", () => {
		assert.equal(hasNewAppHostCookie([{ name: "s", domain: "www.test" }], ["test", "www.test"], new Set()), true);
	});
});

describe("sessionCookiePredicate", () => {
	const ok = sessionCookiePredicate(undefined);
	it("by default takes __Host- cookies that are not the transient ones", () => {
		assert.equal(ok("__Host-session"), true);
		assert.equal(ok("__Host-auth_state"), false);
		assert.equal(ok("__Host-auth_verifier"), false);
		assert.equal(ok("__Host-auth_next"), false);
		assert.equal(ok("locale"), false);
		assert.equal(ok("session"), false);
	});
	it("takes a pattern from the environment, and refuses a bad one", () => {
		assert.equal(sessionCookiePredicate("^sid$")("sid"), true);
		assert.equal(sessionCookiePredicate("^sid$")("locale"), false);
		assert.throws(() => sessionCookiePredicate("("), /valid regular expression/);
	});
	it("filters new cookies", () => {
		const cookies = [{ name: "locale", domain: "app.test" }];
		assert.equal(hasNewAppHostCookie(cookies, ["app.test"], new Set(), ok), false);
		assert.equal(hasNewAppHostCookie([{ name: "__Host-s", domain: "app.test" }], ["app.test"], new Set(), ok), true);
	});
});

describe("isSignInRoute and withoutQuery", () => {
	it("is /sign-in and everything under it", () => {
		for (const p of ["/sign-in", "/sign-in/", "/sign-in/good"]) assert.equal(isSignInRoute(new URL(`https://a.test${p}`)), true, p);
		for (const p of ["/", "/sign-inx", "/app/sign-in"]) assert.equal(isSignInRoute(new URL(`https://a.test${p}`)), false, p);
	});
	it("drops query and fragment", () => {
		assert.equal(withoutQuery("https://a.test/api/auth/callback?code=abc&state=z#f"), "https://a.test/api/auth/callback");
		assert.equal(withoutQuery("/sign-in?next=/x"), "/sign-in");
	});
});

describe("signInRequired", () => {
	const none = { hasPassword: false, signInScreenFlag: false, sourceNamesScreen: false, declaresAuth: false };
	it("allows a signed-out run only for an app with no password, flag, screen or services.auth", () => {
		assert.equal(signInRequired(none), false);
	});
	it("requires sign-in on any one signal, a password above all", () => {
		for (const k of Object.keys(none)) {
			assert.equal(signInRequired({ ...none, [k]: true }), true, k);
		}
	});
});
