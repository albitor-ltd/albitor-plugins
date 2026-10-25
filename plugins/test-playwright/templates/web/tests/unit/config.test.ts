// Managed by Albitor (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import config from "../playwright.config.ts";

const pkg = JSON.parse(readFileSync(new URL("../package.json", import.meta.url), "utf8")) as {
	devDependencies: Record<string, string>;
};
const lock = JSON.parse(readFileSync(new URL("../package-lock.json", import.meta.url), "utf8")) as {
	packages: Record<string, { version: string }>;
};

describe("a project that signs in leaves no recording of it", () => {
	// auth.setup.ts types the demo password into a form. A trace, a screenshot or a video of that
	// step holds the password, and CI uploads the report. The setup project turns all three off.
	it("turns trace, screenshot and video off for the setup project", () => {
		const setup = config.projects?.find((p) => p.name === "setup");
		assert.ok(setup, "no setup project");
		assert.equal(setup.use?.trace, "off");
		assert.equal(setup.use?.screenshot, "off");
		assert.equal(setup.use?.video, "off");
	});

	it("does not let the suite-wide defaults record a video or screenshot", () => {
		const use = config.use ?? {};
		for (const key of ["screenshot", "video"] as const) {
			const v = use[key];
			assert.ok(v === undefined || v === "off", `use.${key} is ${JSON.stringify(v)}`);
		}
	});

	it("gives every project that may sign in the same protection", () => {
		for (const p of config.projects ?? []) {
			const signsIn = p.testMatch !== undefined && String(p.testMatch).includes("auth");
			if (signsIn) {
				assert.equal(p.use?.trace, "off", `${p.name} signs in with trace on`);
			}
		}
	});
});

describe("dependencies", () => {
	it("pins both packages exactly", () => {
		for (const [name, range] of Object.entries(pkg.devDependencies)) {
			assert.match(range, /^\d+\.\d+\.\d+$/, `${name} is ${range}`);
		}
	});

	it("locks the pinned Playwright, with playwright and playwright-core on the same version", () => {
		const want = pkg.devDependencies["@playwright/test"];
		for (const name of ["@playwright/test", "playwright", "playwright-core"]) {
			assert.equal(lock.packages[`node_modules/${name}`]?.version, want, name);
		}
	});

	it("keeps the Playwright pin at least fourteen days old", () => {
		// 1.64.1 was published 2026-10-14 and 1.64.0 on 2026-10-07: the pin must be 1.64.0 or older.
		assert.match(pkg.devDependencies["@playwright/test"], /^1\.(?:[0-5]\d|6[0-4])\.\d+$/);
		assert.notEqual(pkg.devDependencies["@playwright/test"], "1.64.1");
	});
});
