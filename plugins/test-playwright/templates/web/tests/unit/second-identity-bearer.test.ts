// Starting point (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { createServer, type Server } from "node:http";
import type { AddressInfo } from "node:net";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { describe, it } from "node:test";
import { bearerFromStorageState, NO_SECOND_IDENTITY, resolveSecondIdentity } from "../helpers/second-identity.ts";

const dir = mkdtempSync(join(tmpdir(), "second-identity-"));
const state = (name: string, body: unknown) => {
	const p = join(dir, name);
	writeFileSync(p, JSON.stringify(body));
	return p;
};
const jwt = (sub: string) => `h.${sub}.s`;

describe("bearerFromStorageState", () => {
	it("reads the ID token from the session cookie, whichever plugin named it", () => {
		for (const name of ["session", "__Host-session"]) {
			const f = state(`${name}.json`, { cookies: [{ name: "other", value: "x" }, { name, value: jwt("b") }], origins: [] });
			assert.equal(bearerFromStorageState(f), jwt("b"));
		}
	});
	it("falls back to an ID token in local storage", () => {
		const f = state("ls.json", { cookies: [], origins: [{ localStorage: [{ name: "app.idToken", value: jwt("ls") }] }] });
		assert.equal(bearerFromStorageState(f), jwt("ls"));
	});
	it("returns nothing for a missing file, a signed-out state, or a value that is not a token", () => {
		assert.equal(bearerFromStorageState(join(dir, "missing.json")), undefined);
		assert.equal(bearerFromStorageState(state("empty.json", { cookies: [], origins: [] })), undefined);
		assert.equal(bearerFromStorageState(state("opaque.json", { cookies: [{ name: "session", value: "opaque" }] })), undefined);
	});
});

describe("resolveSecondIdentity", () => {
	const saved = state("verifier-b.json", { cookies: [{ name: "session", value: jwt("verifier-b") }], origins: [] });
	it("is bob's header when the local issuer answers, whatever session is saved", async () => {
		const got = await resolveSecondIdentity(async () => ({ Authorization: "Bearer bob" }), saved);
		assert.deepEqual(got, { Authorization: "Bearer bob" });
	});
	it("is verifier-b's bearer on a deployed run (no local issuer)", async () => {
		assert.deepEqual(await resolveSecondIdentity(async () => null, saved), { Authorization: `Bearer ${jwt("verifier-b")}` });
	});
	it("is null with neither, and the skip reason says it is for want of a second identity", async () => {
		assert.equal(await resolveSecondIdentity(async () => null, join(dir, "none.json")), null);
		assert.match(NO_SECOND_IDENTITY, /^no second signed-in identity/);
	});
});

// The shape of a cross-account spec, run against a tiny API that scopes records by bearer: the
// primary identity creates a record, the second identity must be refused it. The same spec body
// runs on both paths; only where the second bearer comes from differs.
describe("a cross-account spec on both paths", () => {
	function api() {
		const owners = new Map<string, string>();
		const server: Server = createServer((req, res) => {
			const who = (req.headers.authorization ?? "").replace("Bearer ", "");
			if (req.method === "POST") owners.set("r1", who);
			const ok = req.method === "POST" || owners.get("r1") === who;
			res.writeHead(req.method === "POST" ? 201 : ok ? 200 : 404).end();
		});
		return new Promise<{ url: string; close: () => void }>((resolve) =>
			server.listen(0, "127.0.0.1", () =>
				resolve({ url: `http://127.0.0.1:${(server.address() as AddressInfo).port}`, close: () => server.close() }),
			),
		);
	}
	async function crossAccountSpec(url: string, primary: Record<string, string>, second: Record<string, string> | null) {
		assert.ok(second, NO_SECOND_IDENTITY);
		assert.equal((await fetch(`${url}/r`, { method: "POST", headers: primary })).status, 201);
		assert.equal((await fetch(`${url}/r`, { headers: primary })).status, 200);
		assert.equal((await fetch(`${url}/r`, { headers: second })).status, 404, "refuses another account's record");
	}
	it("local: alice creates, bob is refused", async () => {
		const a = await api();
		try {
			const second = await resolveSecondIdentity(async () => ({ Authorization: "Bearer bob" }), join(dir, "none.json"));
			await crossAccountSpec(a.url, { Authorization: "Bearer alice" }, second);
		} finally {
			a.close();
		}
	});
	it("deployed: the primary creates, verifier-b (from its saved session) is refused", async () => {
		const a = await api();
		try {
			const f = state("deployed-b.json", { cookies: [{ name: "__Host-session", value: jwt("verifier-b") }], origins: [] });
			const second = await resolveSecondIdentity(async () => null, f);
			await crossAccountSpec(a.url, { Authorization: `Bearer ${jwt("verifier-a")}` }, second);
		} finally {
			a.close();
		}
	});
});
