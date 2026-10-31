// Starting point (test-playwright).
import { randomUUID } from "node:crypto";
import type { TestInfo } from "@playwright/test";
import { FixtureScope } from "./fixture-scope";
import { test as base } from "./journey";

// Seeding helpers. A spec that needs data creates its own through the app's own API and asserts on
// what the app renders back.
//
// This file exists so no product route ever has to carry test data. Hard-coding a list of records
// inside a component to give a spec something to assert on pins that data into the shipped app -
// the first screen a customer opens then shows records belonging to nobody - and once specs depend
// on it, removing it turns the suite red, so nothing removes it. Seed here instead:
//
//   test("lists my claim", async ({ page, fixtureScope }) => {
//     const claim = await fixtureScope.create<{ id: string }>("/api/claims", {
//       title: fixtureScope.uniqueName("claim"),
//     });
//     ...
//   });
//
// `fixtureScope` deletes exactly what the test created when the test ends, and fails the test if
// something was left behind. Never seed by writing to a database.

// A random namespace isolates separate runs; a counter separates calls within the worker.
let seedCounter = 0;
const runNamespace = randomUUID();

// A title unique to this test, this worker and this call, so parallel workers and reruns never
// collide on the same string.
export function uniqueTitle(testInfo: TestInfo, prefix = "seeded"): string {
	seedCounter += 1;
	const suffix = `-${runNamespace}-w${testInfo.workerIndex}-${seedCounter}`;
	const label = `${prefix}-${testInfo.testId}`;
	return `${label.slice(0, Math.max(1, 80 - suffix.length))}${suffix}`;
}

type SeedFixtures = {
	fixtureScope: FixtureScope;
};

// `test` - the journey helper's, whose page is watched for blocked scripts and uncaught errors -
// extended with a `fixtureScope` that carries the journey identity's bearer on its direct API
// calls. The journey helper signs the local browser in as alice; against a deployed app both use
// the one demo session loaded by Playwright, so neither creates identities. A sign-in or
// signed-out spec imports @playwright/test directly and controls its own identity instead.
export const test = base.extend<SeedFixtures>({
	fixtureScope: async ({ appAuthHeaders, request }, use) => {
		const scope = new FixtureScope(request, appAuthHeaders);
		try {
			await use(scope);
		} finally {
			await scope.cleanup();
		}
	},
});

export { expect } from "@playwright/test";
