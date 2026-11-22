// Starting point (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { afterEach, describe, it } from "node:test";
import type { APIRequestContext } from "@playwright/test";
import { localIssuerRunning } from "../helpers/local-auth.ts";

const saved = process.env.ALBITOR_JOURNEY_BASE_URL;
afterEach(() => {
	if (saved === undefined) delete process.env.ALBITOR_JOURNEY_BASE_URL;
	else process.env.ALBITOR_JOURNEY_BASE_URL = saved;
});

const counting = () => {
	const calls: string[] = [];
	const request = {
		get: async (url: string) => {
			calls.push(url);
			return { ok: () => true, headers: () => ({ "content-type": "application/json" }) };
		},
	} as unknown as APIRequestContext;
	return { calls, request };
};

describe("localIssuerRunning", () => {
	it("returns false without a request when the walk targets a deployed app", async () => {
		process.env.ALBITOR_JOURNEY_BASE_URL = "https://app.example.test";
		const { calls, request } = counting();
		assert.equal(await localIssuerRunning(request), false);
		assert.deepEqual(calls, []);
	});
	it("probes the issuer in a local run", async () => {
		delete process.env.ALBITOR_JOURNEY_BASE_URL;
		const { calls, request } = counting();
		assert.equal(await localIssuerRunning(request), true);
		assert.deepEqual(calls, ["/api/_local/auth/identities"]);
	});
});
