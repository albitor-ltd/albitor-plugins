// Managed by Albitor (test-playwright).
import { type Dirent, readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { join, relative, sep } from "node:path";
import { expect, test as setup } from "@playwright/test";
import {
  type BrowserState,
  browserSignIn,
  HOSTED_FORM_MS,
  HOSTED_RETURN_MS,
  NoSessionError,
  SIGN_IN_GOTO_MS,
  SIGN_IN_LANDING_MS,
  SIGN_IN_STEP_MS,
} from "./helpers/browser-signin";
import { readCredentials } from "./helpers/credentials";
import { declaresServicesAuth, signInRequired } from "./helpers/hosted-signin";
import {
  DEMO_STORAGE_STATE,
  EMPTY_STORAGE_STATE,
  ensureAuthDir,
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
// NO SIGN-IN IS THE ONLY SIGNED-OUT RUN. An app with no authentication (auth-none) and this
// template as delivered have no user pool, so no demo password is provisioned and the app declares
// no services.auth in albitor.yaml: the block below writes a signed-out session and the suite runs
// as an anonymous visitor. That is a deliberate no-op rather than a failure: a template that cannot
// run its own tests teaches nothing. Any sign of sign-in (a password, services.auth, the platform
// flag, a screen in the source) makes the setup strict; see signInRequired.
//
// TWO SIGN-IN SHAPES. The in-app form (auth-cognito's SignInPage) is signed in as described below.
// A hosted sign-in (auth-hosted) has /sign-in redirect to the identity provider's own host, here the
// Cognito classic hosted UI, which is signed in there and returns to the app through its callback.
// The setup tells them apart by whether the page is on the app's host.
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
// Declaring services.auth in albitor.yaml, or holding a demo password at all (only an app with a
// user pool is given one), also counts: a hosted sign-in has no in-app screen to find in the source.
// Once it is true the setup signs in, a sign-in screen that has gone missing fails the run, and so
// does a deployed run with no password to sign in with. Neither can then skip every signed-in spec,
// green. Only when it is false (an app with no sign-in) does the setup run signed out.
//

// What a front end that renders the sign-in screen names in its source: the kit's component, or
// the old test id of a hand-written screen. The kit's own files (web/src/lib/kit) hold the former
// whatever the app, so they are not evidence of a screen.
const SIGN_IN_MARKERS = ["SignInPage", "sign-in-email"];
const KIT_DIR = join("src", "lib", "kit");


// Whether this run browses a DEPLOYED app: the deploy's pre-ship verify, or an
// automated journey walk. An ordinary local or CI run boots its own stack,
// which has no user pool to sign in to, so a missing password there is the
// ordinary case. Against a deployed app with a sign-in screen it is a failed
// lookup — and quietly running signed out would hide it.
const browsesDeployedApp = !!(
  (process.env.PLAYWRIGHT_API_ORIGIN || "").trim() ||
  (process.env.ALBITOR_JOURNEY_BASE_URL || "").trim()
);

// The app's base URL, as the config resolved it for this run.
function appBaseURL(): string {
  return String(setup.info().project.use.baseURL ?? "");
}

// albitorYamlDeclaresAuth reads the app's albitor.yaml (the repository root, above web/).
function albitorYamlDeclaresAuth(): boolean {
  try {
    return declaresServicesAuth(readFileSync(join(WEB_ROOT, "..", "albitor.yaml"), "utf8"));
  } catch {
    return false;
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
setup("sign in once as the demo account", async ({ page }) => {
  ensureAuthDir();

  // Strict once there is any evidence the app signs in: see signInRequired. A password is the
  // strongest, since only an app with a user pool is given one. Decided per run, so nothing has to
  // be edited when the screen appears.
  const signInScreenFlag = process.env.ALBITOR_SIGN_IN_SCREEN === "1";
  const signInScreenExists = signInRequired({
    hasPassword: !!password,
    signInScreenFlag,
    sourceNamesScreen: sourceNamingSignInField() !== "",
    declaresAuth: albitorYamlDeclaresAuth(),
  });

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

  // THE IN-APP SIGN-IN SCREEN'S CONTRACT: an email field labelled "Email...", a password field labelled
  // "Password" and a button named "Sign in" (the kit's SignInPage). A screen that labels them
  // differently is the auth plugin's defect to fix, not a reason to edit this file.
  // THE PASSWORD ONLY GOES IN WITH HEADROOM. Everything before the try in browser-signin.ts is bounded (the
  // goto 15s, the landing wait 8s, the email 5s; for a hosted page the form 3 x 15s) and the test timeout
  // is raised to cover them plus the steps and the clean-up, so the test timeout can never fire while
  // the password is in the field: a timeout never reaches the catch that empties it.
  setup.setTimeout(SIGN_IN_GOTO_MS + SIGN_IN_LANDING_MS + 3 * HOSTED_FORM_MS + HOSTED_RETURN_MS + 6 * SIGN_IN_STEP_MS + 10_000);
  let state: BrowserState;
  try {
    state = await browserSignIn(page, appBaseURL(), { username, password, signInRoute });
  } catch (err) {
    if (err instanceof NoSessionError) {
      writeFileSync(DEMO_STORAGE_STATE, JSON.stringify(EMPTY_STORAGE_STATE));
    }
    throw err;
  }
  writeFileSync(DEMO_STORAGE_STATE, JSON.stringify(state));

  // And prove the saved session holds something. An app that keeps its session
  // in sessionStorage or in memory signs in fine and saves an empty state, and
  // every spec guarding on hasDemoSession() would then skip, green.
  expect(
    hasDemoSession(),
    "the sign-in succeeded but saved no cookies or local storage; keep the session in a cookie or localStorage so the suite can reuse it",
  ).toBe(true);
});
