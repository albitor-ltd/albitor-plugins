// Starting point (test-playwright). The journey suite's one Playwright config.
//
// Run it from web/tests (`npm test`) or from the repository root
// (`npx playwright test -c web/tests/playwright.config.ts <spec files>`); paths below are derived
// from this file, never from the working directory.
import { existsSync } from "node:fs";
import { resolve } from "node:path";
import { defineConfig, devices } from "@playwright/test";
import { DEMO_STORAGE_STATE, TESTS_DIR, WEB_ROOT } from "./helpers/storage-state.ts";

const isCI = !!process.env.CI;

// Where the results and the HTML report go. Only the harness's own tests move it.
const outRoot = process.env.TESTS_OUT_DIR || TESTS_DIR;

// ALBITOR_JOURNEY_BASE_URL: an automated walk of this app's journeys drives an app that is ALREADY
// deployed and published, so nothing is served from the runner. The suite boots no server and
// browses that origin.
const journeyBaseURL = (process.env.ALBITOR_JOURNEY_BASE_URL || "").trim();

// ALBITOR_LOCAL_BASE_URL: the app is ALREADY booted on this machine under the local run contract,
// so the suite browses it and boots nothing.
const localBaseURL = (process.env.ALBITOR_LOCAL_BASE_URL || "").trim();

// PLAYWRIGHT_API_ORIGIN: serve THIS bundle against an API that is already deployed (a deploy's
// pre-publish verify). Only the web side boots; a local API would answer /api/* from a fresh
// in-process store and prove nothing about the release.
const apiOrigin = (process.env.PLAYWRIGHT_API_ORIGIN || "").trim();

const webPort = (process.env.LOCAL_WEB_PORT || "3000").trim();
const apiPort = (process.env.LOCAL_API_PORT || "8080").trim();
const bootsNothing = journeyBaseURL !== "" || localBaseURL !== "";

// THE LOCAL RUN CONTRACT. Every other run boots the whole stack the way a person would:
//
//   api/  `ALBITOR_LOCAL=1 ALBITOR_LOCAL_AUTH=1 go run .`  in-process store, local issuer (api-runtime)
//   web/  `npm run local`                                  builds, serves build/, forwards /api (web-framework)
//
// The API half is skipped when the app has no api/ (a web-only profile) or when the API is
// deployed. Each server's own timeout covers a cold build.
const servers = [];
if (!bootsNothing && !apiOrigin && existsSync(resolve(WEB_ROOT, "..", "api", "main.go"))) {
	servers.push({
		command: "go run .",
		cwd: resolve(WEB_ROOT, "..", "api"),
		url: `http://127.0.0.1:${apiPort}/api/health`,
		reuseExistingServer: !isCI,
		timeout: 180_000,
		env: { ALBITOR_LOCAL: "1", ALBITOR_LOCAL_AUTH: "1", PORT: apiPort },
	});
}
if (!bootsNothing) {
	servers.push({
		command: "npm run local",
		cwd: WEB_ROOT,
		url: `http://localhost:${webPort}/healthz`,
		reuseExistingServer: !isCI,
		timeout: 180_000,
		env: { PORT: webPort, APP_API_ORIGIN: apiOrigin || `http://127.0.0.1:${apiPort}` },
	});
}

export default defineConfig({
	testDir: TESTS_DIR,
	testMatch: /.*\.spec\.ts|auth\.setup\.ts/,
	forbidOnly: isCI,
	retries: isCI ? 1 : 0,
	timeout: 30_000,
	expect: { timeout: 5_000 },
	reporter: isCI ? [["list"], ["html", { open: "never", outputFolder: resolve(outRoot, "playwright-report") }]] : "list",
	outputDir: resolve(outRoot, ".results"),
	use: {
		baseURL: journeyBaseURL || localBaseURL || `http://localhost:${webPort}`,
		trace: "on-first-retry",
	},
	// ONE SIGN-IN FOR THE WHOLE SUITE, not one per spec. The `setup` project runs first, signs in as
	// the app's seeded demo account and saves the session; every browser context in `chromium` then
	// starts from that saved session.
	//
	// The alternative - each spec signing itself in, or worse registering itself - is what exhausts
	// the AWS account's shared Cognito email allowance: 50 confirmation emails a day for every app in
	// the account, not per app. The one spec that must start signed out (the one testing sign-in or
	// registration) opts out with `test.use({ storageState: EMPTY_STORAGE_STATE })`.
	//
	// `testIgnore` on chromium is not optional: without it the setup file matches chromium's glob as
	// well and the sign-in runs twice.
	projects: [
		{
			name: "setup",
			testMatch: /auth\.setup\.ts/,
			// THIS PROJECT TYPES THE DEMO PASSWORD INTO A FORM. A trace, a screenshot or a video of that
			// step holds it, and CI uploads the report. Nothing is recorded here, on a retry either.
			use: { trace: "off", screenshot: "off", video: "off" },
		},
		{
			name: "chromium",
			use: { ...devices["Desktop Chrome"], storageState: DEMO_STORAGE_STATE },
			testIgnore: /auth\.setup\.ts/,
			dependencies: ["setup"],
		},
	],
	webServer: servers.length > 0 ? servers : undefined,
});
