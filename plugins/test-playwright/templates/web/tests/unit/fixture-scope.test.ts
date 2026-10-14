// Managed by Albitor (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { FixtureScope } from "../helpers/fixture-scope.ts";

type Call = { method: string; path: string; headers?: Record<string, string> };

// A fake APIRequestContext: just the three verbs FixtureScope uses.
function fake(opts: { createStatus?: number; deleteStatus?: (path: string) => number; getStatus?: number } = {}) {
	const calls: Call[] = [];
	let next = 0;
	const res = (status: number, body: unknown = {}) => ({
		ok: () => status >= 200 && status < 300,
		status: () => status,
		text: async () => JSON.stringify(body),
		json: async () => body,
	});
	const request = {
		post: async (path: string, o: { headers?: Record<string, string> }) => {
			calls.push({ method: "POST", path, headers: o.headers });
			next += 1;
			return res(opts.createStatus ?? 201, { id: `id${next}` });
		},
		delete: async (path: string, o: { headers?: Record<string, string> }) => {
			calls.push({ method: "DELETE", path, headers: o.headers });
			return res(opts.deleteStatus?.(path) ?? 204);
		},
		get: async (path: string, o: { headers?: Record<string, string> }) => {
			calls.push({ method: "GET", path, headers: o.headers });
			return res(opts.getStatus ?? 404);
		},
	};
	return { request: request as never, calls };
}

describe("FixtureScope", () => {
	it("refuses a collection that is not a relative API path", async () => {
		const { request, calls } = fake();
		const scope = new FixtureScope(request);
		for (const bad of ["/api/../admin", "https://evil.example/api/x", "api/items", "/other/items", "/api/it ems", "/api/"]) {
			await assert.rejects(scope.create(bad, {}), /relative API path/, bad);
		}
		assert.equal(calls.length, 0);
	});

	it("refuses a record whose id is missing or a dot segment", async () => {
		for (const id of ["", ".", ".."]) {
			const { request } = fake();
			(request as { post: unknown }).post = async () => ({
				ok: () => true,
				status: () => 201,
				text: async () => "",
				json: async () => ({ id }),
			});
			await assert.rejects(new FixtureScope(request).create("/api/items", {}), /no usable id/);
		}
	});

	it("deletes only what it created, children before parents", async () => {
		const { request, calls } = fake();
		const scope = new FixtureScope(request, { Authorization: "Bearer t" });
		await scope.create("/api/items", {});
		await scope.create("/api/items", {});
		await scope.cleanup();
		const deletes = calls.filter((c) => c.method === "DELETE").map((c) => c.path);
		assert.deepEqual(deletes, ["/api/items/id2", "/api/items/id1"]);
		assert.ok(calls.every((c) => c.headers?.Authorization === "Bearer t"));
		await assert.rejects(scope.remove("/api/items/other"), /unowned fixture/);
	});

	it("encodes the id it deletes, so it cannot climb out of the collection", async () => {
		const { request, calls } = fake();
		(request as { post: unknown }).post = async () => ({
			ok: () => true,
			status: () => 201,
			text: async () => "",
			json: async () => ({ id: "a/../b" }),
		});
		const scope = new FixtureScope(request);
		await scope.create("/api/items", {});
		await scope.cleanup();
		assert.deepEqual(
			calls.filter((c) => c.method === "DELETE").map((c) => c.path),
			["/api/items/a%2F..%2Fb"],
		);
	});

	it("reports a leaked record instead of swallowing it", async () => {
		const { request } = fake({ deleteStatus: () => 500 });
		const scope = new FixtureScope(request);
		await scope.create("/api/items", {});
		await assert.rejects(scope.cleanup(), /left owned records behind/);
	});

	it("does not take a 404 on delete as proof of deletion", async () => {
		const { request } = fake({ deleteStatus: () => 404, getStatus: 200 });
		const scope = new FixtureScope(request);
		await scope.create("/api/items", {});
		await assert.rejects(scope.cleanup(), /left owned records behind/);
	});

	it("takes a 404 on delete plus a 404 on get as gone", async () => {
		const { request } = fake({ deleteStatus: () => 404, getStatus: 404 });
		const scope = new FixtureScope(request);
		await scope.create("/api/items", {});
		await scope.cleanup();
	});

	it("makes names unique to the scope and within 80 characters", () => {
		const a = new FixtureScope(fake().request);
		const b = new FixtureScope(fake().request);
		assert.notEqual(a.uniqueName("x"), a.uniqueName("x"));
		assert.notEqual(a.uniqueName("x"), b.uniqueName("x"));
		assert.ok(a.uniqueName("p".repeat(200)).length <= 80);
	});
});
