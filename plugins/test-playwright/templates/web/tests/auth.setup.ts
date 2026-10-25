// Managed by Albitor (test-playwright).
import { type Dirent, mkdirSync, readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { dirname, join, relative, sep } from "node:path";
import { expect, type Locator, type Page, test as setup } from "@playwright/test";
import { leftSignInRoute, readCredentials } from "./helpers/credentials";
import {
  appHostCookieNames,
  declaresServicesAuth,
  hasNewAppHostCookie,
  isAuthPlumbing,
  isForeignHost,
  isSignInRoute,
  sessionCookiePredicate,
  signInRequired,
  withoutQuery,
} from "./helpers/hosted-signin";
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

// How long the setup waits, after /sign-in loads, to see where it lands: the in-app form, or a
// redirect to another host (a hosted sign-in redirects after a client-side hop or two).
const SIGN_IN_LANDING_MS = 8_000;

// How long the hosted UI's form may take to appear, and the round trip back through the app's
// callback after the submit.
const HOSTED_FORM_MS = 15_000;
const HOSTED_RETURN_MS = 30_000;

// Whether this run browses a DEPLOYED app: the deploy's pre-ship verify, or an
// automated journey walk. An ordinary local or CI run boots its own stack,
// which has no user pool to sign in to, so a missing password there is the
// ordinary case. Against a deployed app with a sign-in screen it is a failed
// lookup — and quietly running signed out would hide it.
const browsesDeployedApp = !!(
  (process.env.PLAYWRIGHT_API_ORIGIN || "").trim() ||
  (process.env.ALBITOR_JOURNEY_BASE_URL || "").trim()
);

// The Cognito classic hosted UI (managed_login_version 1) renders two copies of its form, desktop
// and mobile, one of them hidden: hence :visible.
const HOSTED_USERNAME = "input[name=username]:visible";
const HOSTED_PASSWORD = "input[name=password]:visible";
const HOSTED_SUBMIT = "input[name=signInSubmitButton]:visible";
// What the classic UI shows when it refuses the credentials.
const HOSTED_ERROR = "#loginErrorMessage:visible, .errorMessage-customizable:visible";

// The app's base URL, as the config resolved it for this run.
function appBaseURL(): string {
  return String(setup.info().project.use.baseURL ?? "");
}

// whereSignInLands waits for /sign-in to settle: "in-app" once the email field is visible, whatever
// the host (an apex to www, localhost to 127.0.0.1 or CloudFront to custom-domain redirect of the
// app's own form is still the app's form); "hosted" only on another host AND with the Cognito
// username field visible; "unknown" when neither happens within the window. Polled rather than raced
// so nothing is left dangling.
async function whereSignInLands(page: Page, emailField: Locator): Promise<"hosted" | "in-app" | "unknown"> {
  const base = appBaseURL();
  const deadline = Date.now() + SIGN_IN_LANDING_MS;
  for (;;) {
    const foreign = isForeignHost(base, page.url());
    const visible = (sel: string) => page.locator(sel).first().isVisible().catch(() => false);
    // The Cognito submit button on another host settles it: its username field may be labelled "Email".
    if (foreign && (await visible(HOSTED_SUBMIT)) && (await visible(HOSTED_USERNAME))) {
      return "hosted";
    }
    if (await emailField.first().isVisible().catch(() => false)) {
      return "in-app";
    }
    if (Date.now() >= deadline) {
      return "unknown";
    }
    await page.waitForTimeout(250);
  }
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

// signInInApp fills the app's own sign-in form (auth-cognito's SignInPage) and waits to leave the route.
async function signInInApp(
  page: Page,
  emailField: Locator,
  signInURL: URL,
  username: string,
  password: string,
): Promise<void> {
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
    // page with a field of its own would pass. Signed in means the page left the sign-in route
    // (pathname, plus the hash under hash routing). "commit": a slow full-navigation landing page
    // must not fail a good sign-in; the URL changing is all that is asked here.
    await page.waitForURL((u) => leftSignInRoute(signInURL, u), { timeout: SIGN_IN_STEP_MS, waitUntil: "commit" });
  } catch (err) {
    await fillSecret(passwordField, "").catch(() => {});
    throw new Error(`the demo account did not sign in at ${withoutQuery(signInRoute)}: ${err instanceof Error ? err.name : "failed"}`);
  }
}

// signInHosted signs in on the identity provider's hosted page (the Cognito classic hosted UI) and
// waits for the callback to bring the browser back to the app, off the sign-in route. The password
// goes in the way the in-app one does, and is emptied if anything fails.
async function signInHosted(page: Page, username: string, password: string, appSignInURL: URL): Promise<void> {
  const hostedHost = new URL(page.url()).host;
  const usernameField = page.locator(HOSTED_USERNAME).first();
  const passwordField = page.locator(HOSTED_PASSWORD).first();
  const submit = page.locator(HOSTED_SUBMIT).first();

  for (const [what, field] of [
    ["username field", usernameField],
    ["password field", passwordField],
    ["submit button", submit],
  ] as const) {
    try {
      await field.waitFor({ state: "visible", timeout: HOSTED_FORM_MS });
    } catch {
      throw new Error(
        `the hosted sign-in page on ${hostedHost} shows no visible ${what} ` +
          `(${what === "username field" ? HOSTED_USERNAME : what === "password field" ? HOSTED_PASSWORD : HOSTED_SUBMIT}): ` +
          "the Cognito hosted UI changed?",
      );
    }
  }

  await usernameField.fill(username, { timeout: SIGN_IN_STEP_MS });
  const fail = async (message: string): Promise<never> => {
    await fillSecret(passwordField, "").catch(() => {});
    throw new Error(message);
  };
  try {
    await fillSecret(passwordField, password);
    await submit.click({ timeout: SIGN_IN_STEP_MS });
  } catch (err) {
    return fail(`the demo account did not sign in on the hosted page at ${hostedHost}: ${err instanceof Error ? err.name : "failed"}`);
  }

  // BACK IN THE APP means: on the app's host, on a page (not /api/auth/login or /api/auth/callback,
  // which the callback's own 200-with-meta-refresh or redirect chain passes through), and off the
  // sign-in route. Watching for the identity provider's refusal at the same time fails fast.
  const deadline = Date.now() + HOSTED_RETURN_MS;
  for (;;) {
    const now = new URL(page.url());
    if (!isForeignHost(appBaseURL(), now.href) && !isAuthPlumbing(now) && !isSignInRoute(now) && leftSignInRoute(appSignInURL, now)) {
      return;
    }
    if (await page.locator(HOSTED_ERROR).first().isVisible().catch(() => false)) {
      return fail(`Cognito refused the credentials on the hosted page at ${hostedHost}: the demo account's password does not match`);
    }
    if (Date.now() >= deadline) {
      const stillHosted = isForeignHost(appBaseURL(), now.href);
      return fail(
        stillHosted
          ? `the demo account did not sign in on the hosted page at ${hostedHost}: it never returned to the app ` +
              "(a new-password challenge, or the Cognito hosted UI changed?)"
          : `the demo account signed in at ${hostedHost} but the app never left ${now.pathname}: the callback did not finish`,
      );
    }
    await page.waitForTimeout(250);
  }
}

setup("sign in once as the demo account", async ({ page }) => {
  mkdirSync(dirname(DEMO_STORAGE_STATE), { recursive: true });

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
  // THE PASSWORD ONLY GOES IN WITH HEADROOM. Everything before the try below is bounded (the goto
  // 15s, the landing wait 8s, the email 5s; for a hosted page the form 3 x 15s) and the test timeout
  // is raised to cover them plus the steps and the clean-up, so the test timeout can never fire while
  // the password is in the field: a timeout never reaches the catch that empties it.
  setup.setTimeout(SIGN_IN_GOTO_MS + SIGN_IN_LANDING_MS + 3 * HOSTED_FORM_MS + HOSTED_RETURN_MS + 6 * SIGN_IN_STEP_MS + 10_000);
  await page.goto(signInRoute, { timeout: SIGN_IN_GOTO_MS });
  // Where the sign-in page actually is on the app (it may have redirected): signed in means leaving this.
  const emailField = page.getByLabel(EMAIL_LABEL);
  const landing = await whereSignInLands(page, emailField);
  const signInURL = landing === "hosted" ? new URL(signInRoute, appBaseURL()) : new URL(page.url());
  // A landing that touched another host (hosted sign-in, or an in-app form reached through a
  // redirect to a sibling host) must end with a cookie that is NEW to the app: its login route sets
  // transient state cookies before redirecting out, so any cookie at all proves nothing. The hosts
  // that count are the app's, and for an in-app form on a sibling host, the one it landed on.
  const touchedForeign = landing === "hosted" || isForeignHost(appBaseURL(), page.url());
  const appHosts = [new URL(appBaseURL()).hostname];
  if (landing === "in-app") {
    appHosts.push(new URL(page.url()).hostname);
  }
  const cookiesBefore = appHostCookieNames(await page.context().cookies(), appHosts);
  const routeLabel = withoutQuery(signInRoute);

  if (landing === "hosted") {
    await signInHosted(page, username, password, signInURL);
  } else if (landing === "unknown") {
    // A password is provisioned, so the app has a user pool and a way in; running signed out would
    // hide a missing or unlabelled screen behind a green suite.
    const hasPasswordField = (await page.locator('input[type="password"]').count()) > 0;
    const landedElsewhere = isForeignHost(appBaseURL(), page.url());
    throw new Error(
      landedElsewhere
        ? `${routeLabel} landed on ${new URL(page.url()).host}, another host, with neither the app's sign-in form ` +
            "nor the Cognito hosted sign-in fields: the Cognito hosted UI changed?"
        : hasPasswordField
        ? `${routeLabel} shows a password field but no field labelled "Email...": ` +
            'the sign-in screen must label its fields "Email...", "Password" and its button "Sign in"'
        : `a demo account password reached this run but ${routeLabel} shows no sign-in screen and ` +
            "does not redirect to a hosted sign-in page: the app has a user pool, so it must have a way in",
    );
  } else {
    await signInInApp(page, emailField, signInURL, username, password);
  }

  if (touchedForeign) {
    // The cookies Cognito set on its own host prove nothing, nor do the transient ones the app set
    // before redirecting out: the session is a NEW cookie from the app's callback, on the app's host.
    // Without it, save nothing that looks like a session.
    const state = await page.context().storageState();
    if (!hasNewAppHostCookie(
        state.cookies,
        appHosts,
        cookiesBefore,
        // The hosted path also needs the cookie to look like a session (default: a `__Host-` cookie that is not
        // a transient state one), so a locale or consent cookie from the landing page cannot pass for it.
        landing === "hosted" ? sessionCookiePredicate(process.env.ALBITOR_SESSION_COOKIE_PATTERN) : undefined,
      )) {
      writeFileSync(DEMO_STORAGE_STATE, JSON.stringify(EMPTY_STORAGE_STATE));
      throw new Error(
        "the sign-in returned to the app but the callback set no session: no new cookie for the app's host " +
          "(cookies set on the identity provider's host, and ones that were there before the submit, do not count)",
      );
    }
    writeFileSync(DEMO_STORAGE_STATE, JSON.stringify(state));
  } else {
    await page.context().storageState({ path: DEMO_STORAGE_STATE });
  }

  // And prove the saved session holds something. An app that keeps its session
  // in sessionStorage or in memory signs in fine and saves an empty state, and
  // every spec guarding on hasDemoSession() would then skip, green.
  expect(
    hasDemoSession(),
    "the sign-in succeeded but saved no cookies or local storage; keep the session in a cookie or localStorage so the suite can reuse it",
  ).toBe(true);
});
