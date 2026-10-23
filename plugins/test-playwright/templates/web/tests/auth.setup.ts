// Managed by Albitor (test-playwright).
import { type Dirent, mkdirSync, readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { dirname, join, relative, sep } from "node:path";
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
// app's own front end anywhere under web/ renders the kit's SignInPage (outside the kit itself).
// Once it is true the
// setup no longer looks first: it signs in, a sign-in screen that has gone missing fails the run,
// and so does a deployed run with no password to sign in with. Neither can then skip every
// signed-in spec, green.
//
// While it is false (an app with no sign-in, or a password provisioned before the screen exists)
// the setup looks for the screen first; a password field with no email field beside it fails
// rather than running signed out, so an unlabelled screen cannot hide behind the leniency.
//
// THE SCREEN IS FOUND BY ROLE AND LABEL, not test ids, because the standard sign-in screen is the
// component-kit@2 SignInPage, which is the same in every kit and carries none: a field labelled
// "Email..." (the kit's wording is "Email address"), a field labelled "Password", and a button
// named "Sign in".
const EMAIL_LABEL = /^email/i;
const PASSWORD_LABEL = /^password/i;
const SIGN_IN_BUTTON = /^sign in/i;

// What a front end that renders the sign-in screen names in its source: the kit's component, or
// the old test id of a hand-written screen. The kit's own files (web/src/lib/kit) hold the former
// whatever the app, so they are not evidence of a screen.
const SIGN_IN_MARKERS = ["SignInPage", "sign-in-email"];
const KIT_DIR = join("src", "lib", "kit");

// How long each step of typing the password and signing in may take: well under the test timeout,
// so a stuck step fails into the catch that empties the password field.
const SIGN_IN_STEP_MS = 5_000;

// How long loading the sign-in page may take, and the longest everything before the password goes
// in can take: the goto, the screen probe, the email.
const SIGN_IN_GOTO_MS = 15_000;
const SIGN_IN_PRELUDE_MS = SIGN_IN_GOTO_MS + 10_000 + SIGN_IN_STEP_MS;

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
// suite (which names the screen itself), and build and report output.
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

function namesSignInScreen(path: string): boolean {
  const rel = relative(WEB_ROOT, path);
  if (rel === KIT_DIR || rel.startsWith(KIT_DIR + sep)) {
    return false;
  }
  const text = readFileSync(path, "utf8");
  return SIGN_IN_MARKERS.some((m) => text.includes(m));
}

// sourceNamingSignInField returns the first file of the app's own front end that
// renders the sign-in screen (see SIGN_IN_MARKERS), or "" when none does.
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
        if (statSync(path).size <= SCAN_MAX_BYTES && namesSignInScreen(path)) {
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

// fillSecret sets an input's value and tells the page, the way typing would: through the native
// value setter (so a framework that wraps the property still sees it) and the input and change
// events. Playwright's own fill() would print the secret in the step title of the report.
async function fillSecret(field: Locator, secret: string): Promise<void> {
  await field.evaluate(
    (el, value) => {
      const input = el as HTMLInputElement;
      const set = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set;
      set?.call(input, value);
      input.dispatchEvent(new Event("input", { bubbles: true }));
      input.dispatchEvent(new Event("change", { bubbles: true }));
    },
    secret,
    { timeout: SIGN_IN_STEP_MS },
  );
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

  // THE SIGN-IN SCREEN'S CONTRACT: an email field labelled "Email...", a password field labelled
  // "Password" and a button named "Sign in" (the kit's SignInPage). A screen that labels them
  // differently is the auth plugin's defect to fix, not a reason to edit this file.
  // THE PASSWORD ONLY GOES IN WITH HEADROOM. Everything before the try below is bounded (the probe
  // 10s, goto 15s, the email 5s) and the test timeout is raised to cover them plus three steps and
  // the clean-up, so the test timeout can never fire while the password is in the field: a timeout
  // never reaches the catch that empties it.
  setup.setTimeout(SIGN_IN_PRELUDE_MS + 5 * SIGN_IN_STEP_MS + 10_000);
  await page.goto(signInRoute, { timeout: SIGN_IN_GOTO_MS });
  const emailField = page.getByLabel(EMAIL_LABEL);

  if (!signInScreenExists && !(await signInScreenAppears(emailField))) {
    // Something sign-in shaped IS on the page: the screen exists but is not labelled
    // as the contract says. Running signed out would hide that.
    if ((await page.locator('input[type="password"]').count()) > 0) {
      throw new Error(
        `${signInRoute} shows a password field but no field labelled "Email...": ` +
          'the sign-in screen must label its fields "Email...", "Password" and its button "Sign in"',
      );
    }
    runSignedOut(`no sign-in screen at ${signInRoute} yet`);
    return;
  }

  await emailField.fill(username, { timeout: SIGN_IN_STEP_MS });
  const passwordField = page.getByLabel(PASSWORD_LABEL);
  // THE PASSWORD MUST NOT OUTLIVE THE SIGN-IN IN ANYTHING A FAILURE LEAVES BEHIND. Playwright
  // writes a page snapshot (error-context.md) for a failed test, and CI uploads the HTML report; a
  // password sitting in the field at that moment, or printed in a `fill("...")` step title, would
  // be in both. So it goes in through the page (the step is named "evaluate", its argument is not
  // printed) and, if anything below fails, the field is emptied before the snapshot is taken.
  //
  // EVERY WAIT HERE HAS ITS OWN TIMEOUT, shorter than the test's. A click on a disabled or covered
  // button otherwise waits out the whole test timeout, and a test that times out never reaches the
  // catch: the snapshot is taken with the password still in the field.
  try {
    await fillSecret(passwordField, password);
    await page.getByRole("button", { name: SIGN_IN_BUTTON }).click({ timeout: SIGN_IN_STEP_MS });

    // Prove the sign-in worked before saving it. A saved session that is not a
    // session turns every later spec red on its first assertion, one screen away
    // from the thing that actually broke.
    //
    // NOT expect(...).toBeHidden(): a failed matcher takes its page snapshot at the moment it
    // fails, with the password still in the field. waitFor fails first, the field is emptied,
    // and a plain Error is thrown, so the snapshot Playwright writes for the failed test is taken
    // after.
    await passwordField.waitFor({ state: "hidden", timeout: SIGN_IN_STEP_MS });
    // The password field going away is not enough: a challenge step (a new password) or a landing
    // page with a field of its own would pass. Signed in means the page left the sign-in route.
    await page.waitForURL((u) => u.pathname !== new URL(signInRoute, u).pathname, { timeout: SIGN_IN_STEP_MS });
  } catch (err) {
    await fillSecret(passwordField, "").catch(() => {});
    throw new Error(`the demo account did not sign in at ${signInRoute}: ${err instanceof Error ? err.name : "failed"}`);
  }

  await page.context().storageState({ path: DEMO_STORAGE_STATE });

  // And prove the saved session holds something. An app that keeps its session
  // in sessionStorage or in memory signs in fine and saves an empty state, and
  // every spec guarding on hasDemoSession() would then skip, green.
  expect(
    hasDemoSession(),
    "the sign-in succeeded but saved no cookies or local storage; keep the session in a cookie or localStorage so the suite can reuse it",
  ).toBe(true);
});
