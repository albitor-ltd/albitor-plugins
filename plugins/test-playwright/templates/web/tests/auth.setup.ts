// Managed by Albitor (test-playwright).
import { type Dirent, mkdirSync, readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { expect, type Locator, test as setup } from "@playwright/test";
import { readCredentials } from "./helpers/credentials";
import {
  DEMO_STORAGE_STATE,
  EMPTY_STORAGE_STATE,
  hasDemoSession,
  WEB_ROOT,
} from "./helpers/storage-state";

// ONE SIGN-IN FOR THE WHOLE SUITE.
//
// This runs once, before every other project, signs in as the app's seeded demo
// account and saves the session to disk. playwright.config.ts then loads that
// session into every browser context, so no other spec signs in and no other
// spec signs up.
//
// WHY NO OTHER SPEC MAY REGISTER. Every sign-up through a Cognito user pool sends
// one confirmation email, and the default Cognito sender is capped at 50 a day
// PER AWS ACCOUNT — shared by every app deployed into that account, not budgeted
// per app. Three apps whose suites each registered a fresh account per run
// exhausted the whole account's allowance in an afternoon; from then on every
// sign-up in every app in that account failed, and suites went red for want of a
// session rather than for want of a feature. A seeded account signed into once
// spends none of it.
//
// THE ONE EXCEPTION is the spec that tests signing in or registering, which must
// start signed out. It opts out at the top of its own file:
//
//   test.use({ storageState: EMPTY_STORAGE_STATE });
//
// and it is the only file in this suite that may touch registration.
//
// NO CREDENTIALS, NO SIGN-IN, STILL GREEN. An app with no authentication — and
// this template as delivered — sets no password, so the block below writes a
// signed-out session and the suite runs as an anonymous visitor. That is a
// deliberate no-op rather than a failure: a template that cannot run its own
// tests teaches nothing. The same holds for an app that has a password but no
// sign-in screen yet; see signInScreenExists below.
//
// A SPEC THAT NEEDS A SESSION SAYS SO, and then skips rather than fails when
// the suite is running signed out:
//
//   import { hasDemoSession } from "./helpers/storage-state";
//   test.beforeEach(() => {
//     test.skip(!hasDemoSession(), "no signed-in session: this app has no sign-in yet");
//   });
//
// It has to be a beforeEach (or the first line of the test), not a top-level
// test.skip: the session file is written by this setup, which runs after every
// spec file has been loaded.

// Where the account comes from, and which of the two routes wins, is in helpers/credentials.ts.
const { username, password, signInRoute } = readCredentials(process.env);

// DOES THIS APP HAVE A SIGN-IN SCREEN YET? Decided here, not by an edit.
//
// This file is managed: no builder can flip a flag in it. The answer is true when the platform
// says so (ALBITOR_SIGN_IN_SCREEN=1, set by an auth plugin that ships a sign-in screen) or when the
// app's own front end anywhere under web/ names the sign-in field's test id. Once it is true the
// setup no longer looks first: it signs in, a sign-in screen that has gone missing fails the run,
// and so does a deployed run with no password to sign in with. Neither can then skip every
// signed-in spec, green.
//
// While it is false (an app with no sign-in, or a password provisioned before the screen exists)
// the setup looks for the screen first; found under other test ids, it fails rather than running
// signed out, so a renamed screen cannot hide behind the leniency.
// The test id the setup types the username into, and looks for.
const SIGN_IN_EMAIL_ID = "sign-in-email";

// How long the setup looks for the sign-in screen before deciding there is none.
// Generous on purpose: it is paid once per run, and only by an app with no
// screen, while a cold edge that answers slowly must not be mistaken for one.
const SIGN_IN_PROBE_MS = 10_000;

// Whether this run browses a DEPLOYED app: the deploy's pre-ship verify, or an
// automated journey walk. An ordinary local or CI run boots its own stack,
// which has no user pool to sign in to, so a missing password there is the
// ordinary case. Against a deployed app with a sign-in screen it is a failed
// lookup — and quietly running signed out would hide it.
const browsesDeployedApp = !!(
  (process.env.PLAYWRIGHT_API_ORIGIN || "").trim() ||
  (process.env.ALBITOR_JOURNEY_BASE_URL || "").trim()
);

// signInScreenAppears reports whether the sign-in field shows up within the
// probe window. Only "it never appeared" is a no; anything else — the browser
// crashed, the page closed — is a real failure and is thrown as one.
async function signInScreenAppears(field: Locator): Promise<boolean> {
  try {
    await field.waitFor({ state: "visible", timeout: SIGN_IN_PROBE_MS });
    return true;
  } catch (err) {
    if (err instanceof Error && err.name === "TimeoutError") {
      return false;
    }
    throw err;
  }
}

// Directories under web/ the source scan never enters: dependencies, this test
// suite (which names the test id itself), and build and report output.
const SCAN_SKIP_DIRS = new Set([
  "node_modules",
  "dist",
  "build",
  "coverage",
  ".svelte-kit",
  ".next",
  ".nuxt",
  ".output",
  "playwright-report",
  "test-results",
]);

// A file larger than this is a bundle or an asset, not a screen's source.
const SCAN_MAX_BYTES = 2 * 1024 * 1024;

// sourceNamingSignInField returns the first file of the app's own front end that
// names the sign-in field's test id, or "" when none does.
//
// IT SCANS ALL OF web/, not one conventional directory. This template's front
// end is web/public/index.html and app.js, with no web/src at all; a framework
// swap puts it in web/src, web/app or web/pages. A scan of one of those would
// pass silently over the others. Symbolic links are not followed, and an
// unreadable entry is skipped rather than failing the scan.
function sourceNamingSignInField(): string {
  const pending = [WEB_ROOT];
  while (pending.length > 0) {
    const dir = pending.pop() as string;
    let entries: Dirent[];
    try {
      entries = readdirSync(dir, { withFileTypes: true });
    } catch {
      continue;
    }
    for (const entry of entries) {
      const path = join(dir, entry.name);
      if (entry.isDirectory()) {
        const topLevelTests = dir === WEB_ROOT && entry.name === "tests";
        if (!topLevelTests && !SCAN_SKIP_DIRS.has(entry.name)) {
          pending.push(path);
        }
        continue;
      }
      // A symlink, broken or not, is neither a file nor a directory here.
      if (!entry.isFile()) {
        continue;
      }
      try {
        if (statSync(path).size <= SCAN_MAX_BYTES && readFileSync(path, "utf8").includes(SIGN_IN_EMAIL_ID)) {
          return join("web", relative(WEB_ROOT, path));
        }
      } catch {
        // Unreadable: skip it.
      }
    }
  }
  return "";
}

// runSignedOut saves the signed-out session and says so twice: as an annotation
// on this test, which the report shows, and as a log line an automated walk
// quotes when a journey skips for want of a session.
function runSignedOut(why: string): void {
  setup.info().annotations.push({ type: "signed-out", description: why });
  console.log(`auth.setup: ${why}, so the suite runs signed out and specs that need a session skip themselves`);
  writeFileSync(DEMO_STORAGE_STATE, JSON.stringify(EMPTY_STORAGE_STATE));
}

setup("sign in once as the demo account", async ({ page }) => {
  mkdirSync(dirname(DEMO_STORAGE_STATE), { recursive: true });

  // Strict once the platform says the app has a sign-in screen, or the app's own front end
  // renders the sign-in field. Decided per run, so nothing has to be edited when it appears.
  const signInScreenExists = process.env.ALBITOR_SIGN_IN_SCREEN === "1" || sourceNamingSignInField() !== "";

  if (!password) {
    if (signInScreenExists && browsesDeployedApp) {
      throw new Error(
        "no demo account password reached this run, and the app has a " +
          "sign-in screen: the password lookup failed, so the signed-in specs would all skip",
      );
    }
    runSignedOut("no demo account password");
    return;
  }

  // THE SIGN-IN SCREEN'S CONTRACT: the auth plugin's screen carries the test ids
  // sign-in-form, sign-in-email, sign-in-password and sign-in-button. A screen that names
  // them differently is the auth plugin's defect to fix, not a reason to edit this file.
  await page.goto(signInRoute);
  const emailField = page.getByTestId(SIGN_IN_EMAIL_ID);

  if (!signInScreenExists && !(await signInScreenAppears(emailField))) {
    // Something sign-in shaped IS on the page: the screen exists under other
    // test ids. Running signed out would hide that.
    if ((await page.locator('input[type="password"], [data-testid^="sign-in"]').count()) > 0) {
      throw new Error(
        `${signInRoute} shows a password field or a "sign-in" test id but no "${SIGN_IN_EMAIL_ID}": ` +
          "the sign-in screen must carry the test ids sign-in-form, sign-in-email, sign-in-password and sign-in-button",
      );
    }
    runSignedOut(`no sign-in screen at ${signInRoute} yet`);
    return;
  }

  await emailField.fill(username);
  await page.getByTestId("sign-in-password").fill(password);
  await page.getByTestId("sign-in-button").click();

  // Prove the sign-in worked before saving it. A saved session that is not a
  // session turns every later spec red on its first assertion, one screen away
  // from the thing that actually broke.
  await expect(page.getByTestId("sign-in-form")).toBeHidden();

  await page.context().storageState({ path: DEMO_STORAGE_STATE });

  // And prove the saved session holds something. An app that keeps its session
  // in sessionStorage or in memory signs in fine and saves an empty state, and
  // every spec guarding on hasDemoSession() would then skip, green.
  expect(
    hasDemoSession(),
    "the sign-in succeeded but saved no cookies or local storage; keep the session in a cookie or localStorage so the suite can reuse it",
  ).toBe(true);
});
