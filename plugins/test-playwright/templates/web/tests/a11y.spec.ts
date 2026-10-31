// Starting point (test-playwright). The floor accessibility audit: the shell's own screens
// meet WCAG 2.2 A and AA as far as axe can tell. A journey audits the screens it walks with
// `journeyStep(page, name, body, { a11y: true })`; this file covers the screens no journey owns.
import { expect, test } from "./helpers/journey";
import { expectNoA11yViolations } from "./helpers/a11y";

const SHELL_ROUTES = ["/", "/offline"];

for (const route of SHELL_ROUTES) {
	test(`${route} has no WCAG 2.2 A/AA violations`, async ({ page }) => {
		await page.goto(route);
		await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
		await expectNoA11yViolations(page);
	});
}

test("an unknown route is a named error page with no violations", async ({ page }) => {
	await page.goto(`/no-such-page-${Date.now()}`);
	await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
	await expectNoA11yViolations(page);
});

test("the audit fails on a real violation", async ({ page }) => {
	// The audit itself is under test: a button with no accessible name must not pass.
	await page.goto("/");
	await expect(page.getByRole("main")).toBeVisible();
	await page.evaluate(() => {
		document.querySelector("main")?.append(document.createElement("button"));
	});
	await expect(expectNoA11yViolations(page)).rejects.toThrow(/button-name/);
});

test("a waiver with a reason lets exactly that rule through", async ({ page }) => {
	await page.goto("/");
	await expect(page.getByRole("main")).toBeVisible();
	await page.evaluate(() => {
		document.querySelector("main")?.append(document.createElement("button"));
	});
	await expectNoA11yViolations(page, { waive: [{ rule: "button-name", reason: "the test adds a nameless button on purpose" }] });
	await expect(expectNoA11yViolations(page, { waive: [{ rule: "button-name", reason: "" }] })).rejects.toThrow(/needs a reason/);
});
