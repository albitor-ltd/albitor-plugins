// @journey sign-in
// Managed by Albitor (auth-cognito). The sign-in journey the plugin owes the app: a signed-out
// visitor is sent to the sign-in screen and can reach the sign-up and reset screens from it, and a
// local test identity (alice, from api-go's local issuer) reaches the app. Signing in with a
// password needs a user pool, so a deployed run's sign-in is the suite's one auth.setup.ts.
import { expect, test as base } from "@playwright/test";
import { expectNoA11yViolations } from "../helpers/a11y";
import { test as journey } from "../helpers/journey";
import { authHeader, localIssuerRunning, localTokens } from "../helpers/local-auth";
import { EMPTY_STORAGE_STATE } from "../helpers/storage-state";

base.describe("signed out", () => {
  base.use({ storageState: EMPTY_STORAGE_STATE });

  base("a signed-out visitor is sent to sign-in and back to where they were headed", async ({ page, request }) => {
    // The guard only asks the API who is signed in, so there must be an API that says nobody is.
    const me = await request.get("/api/me").catch(() => undefined);
    base.skip(!me || me.status() !== 401, "no API that refuses an anonymous visitor behind /api/me");

    await page.goto("/");
    await expect(page).toHaveURL(/\/sign-in\?next=%2F$/);
    await expect(page.getByRole("heading", { level: 1, name: "Sign in" })).toBeVisible();
    // Role and label locators, not test ids: the form is the kit's SignInPage, which differs in
    // markup (and a little in wording) from kit to kit.
    await expect(page.getByLabel(/^Email/)).toBeVisible();
    await expect(page.getByLabel(/^Password/)).toBeVisible();
    await expect(page.getByRole("button", { name: /^Sign in/ })).toBeVisible();
    await expectNoA11yViolations(page);

    // Another page of the app is remembered, and only a path on this origin is.
    await page.goto("/reports?tab=2");
    await expect(page).toHaveURL(/\/sign-in\?next=%2Freports%3Ftab%3D2$/);
    await page.goto("/sign-in?next=%2F%2Fevil.example");
    await expect(page.getByRole("heading", { level: 1, name: "Sign in" })).toBeVisible();
  });

  base("the sign-up, confirm and reset screens are one click from sign-in and from each other", async ({ page, request }) => {
    const me = await request.get("/api/me").catch(() => undefined);
    base.skip(!me || me.status() !== 401, "no API that refuses an anonymous visitor behind /api/me");

    await page.goto("/sign-in");
    await page.getByRole("link", { name: "Create an account" }).click();
    await expect(page).toHaveURL(/\/sign-up$/);
    await expect(page.getByTestId("sign-up-form")).toBeVisible();
    await expectNoA11yViolations(page);

    await page.getByTestId("sign-up-sign-in-link").click();
    await page.getByRole("link", { name: "Forgot your password?" }).click();
    await expect(page).toHaveURL(/\/reset$/);
    await expect(page.getByTestId("reset-request-form")).toBeVisible();
    await expectNoA11yViolations(page);

    await page.goto("/confirm?email=a%40example.test");
    await expect(page.getByTestId("confirm-form")).toBeVisible();
    await expect(page.getByTestId("confirm-email")).toHaveValue("a@example.test");
    await expectNoA11yViolations(page);
  });
});

journey.describe("signed in through the local issuer", () => {
  journey.beforeEach(async ({ request }) => {
    journey.skip(!(await localIssuerRunning(request)), "no local issuer: not a local run with ALBITOR_LOCAL_AUTH=1");
  });

  journey("a local identity reaches the app, and the API knows who it is", async ({ page, request }) => {
    const alice = await localTokens(request, "alice");
    // The journey fixture already signs the context in as alice; ask the API directly as well.
    const me = await request.get("/api/me", { headers: authHeader(alice) });
    expect(me.status()).toBe(200);
    expect(await me.json()).toMatchObject({ sub: alice.sub, email: "alice@example.test" });

    await page.goto("/");
    await expect(page).not.toHaveURL(/\/sign-in/);
    await expect(page.getByRole("heading", { level: 1 })).toHaveCount(1);
  });

  journey("the other identities are other people", async ({ request }) => {
    const bob = await localTokens(request, "bob");
    const admin = await localTokens(request, "admin");
    const asBob = await (await request.get("/api/me", { headers: authHeader(bob) })).json();
    const asAdmin = await (await request.get("/api/me", { headers: authHeader(admin) })).json();
    expect(asBob.sub).not.toBe(asAdmin.sub);
    expect(asAdmin.groups).toEqual(["admin"]);
  });
});
