// Managed by Albitor (test-playwright). The harness's own smoke test: the shell loads, the API (if
// the app has one) is up and keeps the edge error contract. State-free: it asserts no records.
import { expect, test } from "./helpers/journey";
import { journeyStep } from "./helpers/journey";

test("the app shell loads", async ({ page }) => {
	await journeyStep(page, "open the app", async () => {
		await page.goto("/");
		// A landmark and one top-level heading: what a person and a screen reader land on.
		await expect(page.getByRole("main")).toBeVisible();
		await expect(page.getByRole("heading", { level: 1 })).toHaveCount(1);
	});
});

test.describe("the API", () => {
	// A web-only app has no /api; the proxy answers 502 and these have nothing to hold to the
	// contract. They run, and fail, for any app whose /api/health answers.
	test.beforeEach(async ({ request }) => {
		const res = await request.get("/api/health").catch(() => undefined);
		test.skip(!res || res.status() === 502, "this app has no API behind /api");
	});

	test("the health endpoint answers JSON status ok", async ({ request }) => {
		const res = await request.get("/api/health");
		expect(res.status()).toBe(200);
		expect(res.headers()["content-type"] ?? "").toContain("application/json");
		expect(await res.json()).toMatchObject({ status: "ok" });
	});

	// An unknown API path must reach the client as the API wrote it: an error, in JSON. A 200
	// text/html here is the SPA shell answering for the API, and every client that checks res.ok then
	// sees success and throws parsing HTML as JSON. An app with sign-in may refuse an
	// unauthenticated request before routing it, which is the API's answer too.
	test("an unknown API path is a JSON error, not the app shell", async ({ request }) => {
		const res = await request.get(`/api/no-such-route-${Date.now()}`);
		expect([401, 403, 404]).toContain(res.status());
		expect(res.headers()["content-type"] ?? "").toContain("application/json");
	});
});
