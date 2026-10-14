// Managed by Albitor (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { describe, it } from "node:test";
import { readCredentials } from "../helpers/credentials.ts";
import { parseVerification } from "../helpers/mailbox-contract.ts";
import { DEMO_STORAGE_STATE, sessionSaved } from "../helpers/storage-state.ts";

const dir = mkdtempSync(join(tmpdir(), "albitor-tests-"));
const write = (name: string, body: string) => {
	const p = join(dir, name);
	writeFileSync(p, body);
	return p;
};

describe("storage state", () => {
	it("lives under tests/.auth whatever the working directory", () => {
		assert.ok(DEMO_STORAGE_STATE.endsWith("/tests/.auth/demo.json"));
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
