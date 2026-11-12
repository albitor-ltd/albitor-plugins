// Starting point (test-playwright). The walk's source-text check under test: it fails on a page
// that shows developer source text and passes on a clean one, in a real browser.
import { expect, test } from "@playwright/test";
import { pageSourceTextProblem } from "./helpers/source-text";

test("fails a page that shows developer source text", async ({ page }) => {
	await page.setContent(`<header>// Starting point: the claims header</header><main><h1>Claims</h1></main>`);
	expect(await pageSourceTextProblem(page)).toMatch(/source text/);
});

test("passes a clean page", async ({ page }) => {
	await page.setContent(`<main><h1>Claims</h1><p>See https://example.com/a // b</p></main>`);
	expect(await pageSourceTextProblem(page)).toBe("");
});

test("skips code, editable regions and the data-allow-source-text opt-out", async ({ page }) => {
	await page.setContent(`<main>
		<pre>// a code sample</pre>
		<textarea>// typed</textarea>
		<div contenteditable>// editable</div>
		<div data-allow-source-text>/* a user's snippet */</div>
		<h1>Claims</h1></main>`);
	expect(await pageSourceTextProblem(page)).toBe("");
});

test("ignores hidden text but fails on shown text beside an opt-out", async ({ page }) => {
	await page.setContent(`<main><p style="display:none">// hidden</p>
		<div data-allow-source-text>// snippet</div><p><!-- x -->ok</p><p>/* leaked */</p></main>`);
	expect(await pageSourceTextProblem(page)).toMatch(/leaked/);
});
