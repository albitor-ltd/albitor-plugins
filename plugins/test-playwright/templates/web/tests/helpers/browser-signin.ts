// Starting point (test-playwright). Signing in through a browser, in the app's own form or on
// the identity provider's hosted page. Moved out of auth.setup.ts so one implementation serves the
// suite's demo sign-in and auth@2's signInAs(role) on a deployed app.
import type { Locator, Page } from "@playwright/test";
import { leftSignInRoute } from "./credentials";
import {
  appHostCookieNames,
  hasNewAppHostCookie,
  isAuthPlumbing,
  isForeignHost,
  isSignInRoute,
  sessionCookiePredicate,
  withoutQuery,
} from "./hosted-signin";

// The account to sign in as, and the app's sign-in route.
export type BrowserCredentials = { username: string; password: string; signInRoute: string };

// The browser state a sign-in leaves behind, in the shape Playwright saves and loads.
export type BrowserState = Awaited<ReturnType<ReturnType<Page["context"]>["storageState"]>>;

// Which sign-in an auth plugin ships. "any" detects it (the suite's demo sign-in does); an auth
// plugin's auth@2 provider names its own, so a plugin whose screen moved fails instead of passing
// through the other shape.
export type SignInShape = "in-app" | "hosted" | "any";

// Raised when the sign-in came back to the app but its callback set no session. The caller decides
// what to leave on disk.
export class NoSessionError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "NoSessionError";
  }
}

// THE SCREEN IS FOUND BY ROLE AND LABEL, not test ids, because the standard sign-in screen is the
// component-kit@2 SignInPage, which is the same in every kit and carries none: a field labelled
// "Email..." (the kit's wording is "Email address"), a field labelled "Password", and a button
// named "Sign in".
const EMAIL_LABEL = /^email/i;
const PASSWORD_LABEL = /^password/i;
const SIGN_IN_BUTTON = /^sign in/i;

// How long each step of typing the password and signing in may take: well under the test timeout,
// so a stuck step fails into the catch that empties the password field.
export const SIGN_IN_STEP_MS = 5_000;

// How long loading the sign-in page may take, and the longest everything before the password goes
// in can take: the goto, the screen probe, the email.
export const SIGN_IN_GOTO_MS = 15_000;

// How long the setup waits, after /sign-in loads, to see where it lands: the in-app form, or a
// redirect to another host (a hosted sign-in redirects after a client-side hop or two).
export const SIGN_IN_LANDING_MS = 8_000;

// How long the hosted UI's form may take to appear, and the round trip back through the app's
// callback after the submit.
export const HOSTED_FORM_MS = 15_000;
export const HOSTED_RETURN_MS = 30_000;

// The Cognito classic hosted UI (managed_login_version 1) renders two copies of its form, desktop
// and mobile, one of them hidden: hence :visible.
const HOSTED_USERNAME = "input[name=username]:visible";
const HOSTED_PASSWORD = "input[name=password]:visible";
const HOSTED_SUBMIT = "input[name=signInSubmitButton]:visible";
// What the classic UI shows when it refuses the credentials.
const HOSTED_ERROR = "#loginErrorMessage:visible, .errorMessage-customizable:visible";

// whereSignInLands waits for /sign-in to settle: "in-app" once the email field is visible, whatever
// the host (an apex to www, localhost to 127.0.0.1 or CloudFront to custom-domain redirect of the
// app's own form is still the app's form); "hosted" only on another host AND with the Cognito
// username field visible; "unknown" when neither happens within the window. Polled rather than raced
// so nothing is left dangling.
async function whereSignInLands(
  page: Page,
  base: string,
  emailField: Locator,
): Promise<"hosted" | "in-app" | "unknown"> {
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
  routeLabel: string,
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
    throw new Error(`the demo account did not sign in at ${routeLabel}: ${err instanceof Error ? err.name : "failed"}`);
  }
}

// signInHosted signs in on the identity provider's hosted page (the Cognito classic hosted UI) and
// waits for the callback to bring the browser back to the app, off the sign-in route. The password
// goes in the way the in-app one does, and is emptied if anything fails.
async function signInHosted(page: Page, base: string, username: string, password: string, appSignInURL: URL): Promise<void> {
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
    if (!isForeignHost(base, now.href) && !isAuthPlumbing(now) && !isSignInRoute(now) && leftSignInRoute(appSignInURL, now)) {
      return;
    }
    if (await page.locator(HOSTED_ERROR).first().isVisible().catch(() => false)) {
      return fail(`Cognito refused the credentials on the hosted page at ${hostedHost}: the demo account's password does not match`);
    }
    if (Date.now() >= deadline) {
      const stillHosted = isForeignHost(base, now.href);
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

// browserSignIn signs in as `creds` in `page` and returns the browser state to save. It goes to the
// sign-in route, finds where it lands (the app's own form, or the identity provider's hosted page),
// signs in there, and proves a session came of it. It writes nothing: the caller decides where the
// state goes. A password never outlives the sign-in in anything a failure leaves behind.
export async function browserSignIn(
  page: Page,
  base: string,
  creds: BrowserCredentials,
  shape: SignInShape = "any",
): Promise<BrowserState> {
  const { username, password, signInRoute } = creds;
  await page.goto(signInRoute, { timeout: SIGN_IN_GOTO_MS });
  // Where the sign-in page actually is on the app (it may have redirected): signed in means leaving this.
  const emailField = page.getByLabel(EMAIL_LABEL);
  const landing = await whereSignInLands(page, base, emailField);
  const routeLabel = withoutQuery(signInRoute);
  if (landing !== "unknown" && shape !== "any" && landing !== shape) {
    throw new Error(
      `${routeLabel} shows the ${landing === "hosted" ? "hosted" : "app's own"} sign-in, but this app's auth plugin ` +
        `ships the ${shape === "hosted" ? "hosted" : "in-app"} one`,
    );
  }
  const signInURL = landing === "hosted" ? new URL(signInRoute, base) : new URL(page.url());
  // A landing that touched another host (hosted sign-in, or an in-app form reached through a
  // redirect to a sibling host) must end with a cookie that is NEW to the app: its login route sets
  // transient state cookies before redirecting out, so any cookie at all proves nothing. The hosts
  // that count are the app's, and for an in-app form on a sibling host, the one it landed on.
  const touchedForeign = landing === "hosted" || isForeignHost(base, page.url());
  const appHosts = [new URL(base).hostname];
  if (landing === "in-app") {
    appHosts.push(new URL(page.url()).hostname);
  }
  const cookiesBefore = appHostCookieNames(await page.context().cookies(), appHosts);

  if (landing === "hosted") {
    await signInHosted(page, base, username, password, signInURL);
  } else if (landing === "unknown") {
    // A password is provisioned, so the app has a user pool and a way in; running signed out would
    // hide a missing or unlabelled screen behind a green suite.
    const hasPasswordField = (await page.locator('input[type="password"]').count()) > 0;
    const landedElsewhere = isForeignHost(base, page.url());
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
    await signInInApp(page, emailField, signInURL, username, password, routeLabel);
  }

  const state = await page.context().storageState();
  if (touchedForeign) {
    // The cookies Cognito set on its own host prove nothing, nor do the transient ones the app set
    // before redirecting out: the session is a NEW cookie from the app's callback, on the app's host.
    // Without it, save nothing that looks like a session.
    if (!hasNewAppHostCookie(
        state.cookies,
        appHosts,
        cookiesBefore,
        // The hosted path also needs the cookie to look like a session (default: a `__Host-` cookie that is not
        // a transient state one), so a locale or consent cookie from the landing page cannot pass for it.
        landing === "hosted" ? sessionCookiePredicate(process.env.SESSION_COOKIE_PATTERN) : undefined,
      )) {
      throw new NoSessionError(
        "the sign-in returned to the app but the callback set no session: no new cookie for the app's host " +
          "(cookies set on the identity provider's host, and ones that were there before the submit, do not count)",
      );
    }
  }
  return state;
}
