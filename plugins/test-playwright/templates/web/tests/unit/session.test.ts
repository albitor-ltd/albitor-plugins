// Managed by Albitor (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { chmodSync, mkdtempSync, statSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { describe, it } from "node:test";
import { leftSignInRoute, readCredentials } from "../helpers/credentials.ts";
import { parseVerification } from "../helpers/mailbox-contract.ts";
import { DEMO_STORAGE_STATE, sessionSaved, TESTS_DIR } from "../helpers/storage-state.ts";

const dir = mkdtempSync(join(tmpdir(), "albitor-tests-"));
const write = (name: string, body: string) => {
	const p = join(dir, name);
	writeFileSync(p, body);
	return p;
};

describe("storage state", () => {
	it("lives under the OS temp dir, outside the checkout, whatever the working directory", () => {
		assert.ok(DEMO_STORAGE_STATE.startsWith(join(tmpdir(), "albitor-tests-auth-")));
		assert.ok(!DEMO_STORAGE_STATE.startsWith(TESTS_DIR));
		assert.ok(DEMO_STORAGE_STATE.endsWith("/demo.json"));
	});
	it("is named per checkout, so two apps never share a session", async () => {
		const other = await import("../helpers/storage-state.ts?other-checkout");
		assert.equal(other.DEMO_STORAGE_STATE, DEMO_STORAGE_STATE);
	});
	it("can be moved with ALBITOR_TESTS_AUTH_DIR", async () => {
		const target = join(dir, "moved");
		process.env.ALBITOR_TESTS_AUTH_DIR = target;
		try {
			const moved = await import("../helpers/storage-state.ts?moved");
			assert.equal(moved.DEMO_STORAGE_STATE, join(target, "demo.json"));
		} finally {
			delete process.env.ALBITOR_TESTS_AUTH_DIR;
		}
	});
	it("creates the directory private to the user (0700), tightening one that already exists", async () => {
		const target = join(dir, "private", "nested");
		process.env.ALBITOR_TESTS_AUTH_DIR = target;
		try {
			const moved = await import("../helpers/storage-state.ts?private");
			assert.equal(moved.ensureAuthDir(), target);
			assert.equal(statSync(target).mode & 0o777, 0o700);
			chmodSync(target, 0o755);
			moved.ensureAuthDir();
			assert.equal(statSync(target).mode & 0o777, 0o700);
		} finally {
			delete process.env.ALBITOR_TESTS_AUTH_DIR;
		}
	});
	it("refuses a session path that is not a directory", async () => {
		const file = write("not-a-dir", "x");
		process.env.ALBITOR_TESTS_AUTH_DIR = file;
		try {
			const moved = await import("../helpers/storage-state.ts?file");
			assert.throws(() => moved.ensureAuthDir());
		} finally {
			delete process.env.ALBITOR_TESTS_AUTH_DIR;
		}
	});
	it("is a session only when it holds cookies or origins", () => {
		assert.equal(sessionSaved(join(dir, "missing.json")), false);
		assert.equal(sessionSaved(write("bad.json", "{")), false);
		assert.equal(sessionSaved(write("empty.json", '{"cookies":[],"origins":[]}')), false);
		assert.equal(sessionSaved(write("cookie.json", '{"cookies":[{"name":"s"}],"origins":[]}')), true);
		assert.equal(sessionSaved(write("origin.json", '{"cookies":[],"origins":[{"origin":"http://x"}]}')), true);
	});
});

describe("readCredentials", () => {
	it("reads the environment when no file is named", () => {
		const c = readCredentials({ ALBITOR_DEMO_ACCOUNT_PASSWORD: "pw", DEMO_ACCOUNT_USERNAME: "u@example.com" });
		assert.deepEqual(c, { username: "u@example.com", password: "pw", signInRoute: "/sign-in" });
	});
	it("lets the walk's file win over the environment", () => {
		const file = write("creds.json", JSON.stringify({ username: "w", password: "wp", sign_in_route: "/login" }));
		const c = readCredentials({ ALBITOR_JOURNEY_CREDENTIALS: file, ALBITOR_DEMO_ACCOUNT_PASSWORD: "envpw" });
		assert.deepEqual(c, { username: "w", password: "wp", signInRoute: "/login" });
	});
	it("falls back to the environment for an unreadable or passwordless file", () => {
		const env = { ALBITOR_DEMO_ACCOUNT_PASSWORD: "envpw" };
		assert.equal(readCredentials({ ...env, ALBITOR_JOURNEY_CREDENTIALS: join(dir, "nope.json") }).password, "envpw");
		const file = write("nopw.json", '{"username":"w"}');
		assert.equal(readCredentials({ ...env, ALBITOR_JOURNEY_CREDENTIALS: file }).password, "envpw");
	});
	it("refuses a sign-in route that is not a path on the app", () => {
		const file = write("evil.json", JSON.stringify({ password: "p", sign_in_route: "https://evil.example/" }));
		assert.equal(readCredentials({ ALBITOR_JOURNEY_CREDENTIALS: file }).signInRoute, "/sign-in");
		assert.equal(readCredentials({ ALBITOR_DEMO_ACCOUNT_PASSWORD: "p", DEMO_ACCOUNT_SIGN_IN_ROUTE: "//evil.example" }).signInRoute, "/sign-in");
	});
});

describe("parseVerification", () => {
	it("prefers a code, then a followed link, then a safe path", () => {
		assert.deepEqual(parseVerification('{"code":"123456","open":"/x"}', 1), { code: "123456" });
		assert.deepEqual(parseVerification('{"followed":true}', 1), { followed: true });
		assert.deepEqual(parseVerification('{"open":"/confirm?t=1"}', 1), { open: "/confirm?t=1" });
	});
	it("throws on an answer with nothing usable, or an open path on another host", () => {
		assert.throws(() => parseVerification("{}", 2), /email 2/);
		assert.throws(() => parseVerification('{"open":"//evil.example"}', 3), /email 3/);
		assert.throws(() => parseVerification("not json", 4), /email 4/);
	});
});

describe("leftSignInRoute", () => {
	const at = (p: string) => new URL(p, "https://app.example.test");
	it("is false for a challenge page on the same route, query or not", () => {
		assert.equal(leftSignInRoute(at("/sign-in"), at("/sign-in")), false);
		assert.equal(leftSignInRoute(at("/sign-in"), at("/sign-in?challenge=new-password")), false);
	});
	it("is true once the path changes, including from a redirected sign-in page", () => {
		assert.equal(leftSignInRoute(at("/sign-in"), at("/")), true);
		assert.equal(leftSignInRoute(at("/login"), at("/dashboard")), true);
	});
	it("uses the hash under hash routing", () => {
		assert.equal(leftSignInRoute(at("/#/sign-in"), at("/#/sign-in")), false);
		assert.equal(leftSignInRoute(at("/#/sign-in"), at("/#/new-password")), true);
		assert.equal(leftSignInRoute(at("/#/sign-in"), at("/#/")), true);
	});
});
