// @journey sign-in
// Seeded by Albitor (auth-hosted), then yours. The sign-in journey the plugin owes the app: a
// signed-out visitor is handed to the API's sign-in (which goes to Cognito's hosted page), a sign-in
// that cannot finish says so and does not loop, the refusal page for an address the app does not
// allow, and a signed-in person's account, sign-out and deletion screens. There is no password
// field anywhere in the app, and this spec never types into Cognito's page: the suite's one
// auth.setup.ts signs in there, and every other project here starts from its saved session.
//
// Nothing here deletes an account or ends the suite's session: the calls that would are answered by
// the spec itself.
import { expect, test as base } from "@playwright/test";
import { expectNoA11yViolations } from "../helpers/a11y";
import { test as journey } from "../helpers/journey";
import { EMPTY_STORAGE_STATE } from "../helpers/storage-state";

// The browser is sent to the API, which sends it to Cognito: stand in for that first request so the
// spec stays on the app, and report what it was asked for.
async function captureLoginRequests(page: import("@playwright/test").Page): Promise<URL[]> {
  const seen: URL[] = [];
  await page.route("**/api/auth/login**", (route) => {
    seen.push(new URL(route.request().url()));
    return route.fulfill({ status: 200, contentType: "text/html", body: "<title>Hosted sign-in</title><p>stand-in</p>" });
  });
  return seen;
}

base.describe("signed out", () => {
  base.use({ storageState: EMPTY_STORAGE_STATE });

  base("a signed-out visitor is handed to the API's sign-in, remembering where they were headed", async ({ page, request }) => {
    // The guard only asks the API who is signed in, so there must be an API that says nobody is.
    const me = await request.get("/api/me").catch(() => undefined);
    base.skip(!me || me.status() !== 401, "no API that refuses an anonymous visitor behind /api/me");

    const logins = await captureLoginRequests(page);
    await page.goto("/reports?tab=2");
    await expect.poll(() => logins.length).toBeGreaterThan(0);
    expect(logins[0]!.pathname).toBe("/api/auth/login");
    expect(logins[0]!.searchParams.get("next")).toBe("/reports?tab=2");
  });

  base("only a path on this site is remembered", async ({ page, request }) => {
    const me = await request.get("/api/me").catch(() => undefined);
    base.skip(!me || me.status() !== 401, "no API that refuses an anonymous visitor behind /api/me");

    const logins = await captureLoginRequests(page);
    await page.goto("/sign-in?next=%2F%2Fevil.example");
    await expect.poll(() => logins.length).toBeGreaterThan(0);
    expect(logins[0]!.searchParams.get("next")).toBe("/");
  });

  base("a sign-in that could not finish says so, and does not send the person round again", async ({ page }) => {
    const logins = await captureLoginRequests(page);
    await page.goto("/sign-in?error=expired");
    await expect(page.getByRole("heading", { level: 1, name: "We could not sign you in" })).toBeVisible();
    await expect(page.getByRole("alert").filter({ hasText: "took too long" })).toBeVisible();
    // No password field, here or on the page the redirect lands on: Cognito owns that.
    await expect(page.locator("input[type=password]")).toHaveCount(0);
    await expectNoA11yViolations(page);

    await page.waitForTimeout(500);
    expect(logins).toHaveLength(0);

    await page.getByRole("button", { name: "Continue to sign in" }).click();
    await expect.poll(() => logins.length).toBe(1);

    // A code this page does not know is a general message, never reflected into the page.
    await page.goto("/sign-in?error=%3Cb%3Eboom%3C%2Fb%3E");
    await expect(page.getByRole("alert")).toHaveText("We could not finish signing you in.");
  });

  base("a second arrival at sign-in so soon does not send the person round again", async ({ page }) => {
    const logins = await captureLoginRequests(page);
    await page.goto("/sign-in");
    await expect.poll(() => logins.length).toBe(1);
    await page.goto("/sign-in");
    await expect(page.getByRole("heading", { level: 1, name: "We could not keep you signed in" })).toBeVisible();
    await page.waitForTimeout(500);
    expect(logins).toHaveLength(1);
    await expectNoA11yViolations(page);
  });

  base("an address the app does not allow gets the refusal page, with a way out", async ({ page }) => {
    await page.goto("/not-allowed");
    await expect(
      page.getByRole("heading", { level: 1, name: "You cannot use this service with that email address" }),
    ).toBeVisible();
    await expect(page.getByRole("link", { name: "Sign out and use a different email address" })).toHaveAttribute("href", "/sign-out");
    await expect(page.locator("input[type=password]")).toHaveCount(0);
    await expectNoA11yViolations(page);
  });
});

journey.describe("signed in", () => {
  journey("the account page offers deletion only behind a confirmation", async ({ page }) => {
    // Whatever happens, the spec itself answers the deletion call, so the suite's account survives.
    await page.route("**/api/auth/delete", (route) =>
      route.fulfill({
        status: 403,
        contentType: "application/json",
        body: JSON.stringify({ code: "scope_missing", error: "this app cannot delete accounts yet: its sign-in is not set up to allow it" }),
      }),
    );

    await page.goto("/account");
    await expect(page.getByRole("heading", { level: 1, name: "Your account" })).toBeVisible();
    await expect(page.getByRole("heading", { level: 2, name: "Delete your account" })).toBeVisible();
    await expect(page.getByTestId("account-email")).toContainText("@");
    await expectNoA11yViolations(page);

    await page.getByRole("button", { name: "Delete account" }).click();
    await expect(page).toHaveURL(/\/account\/delete$/);
    await expect(page.getByRole("heading", { level: 1, name: "Are you sure you want to delete your account?" })).toBeVisible();
    await expectNoA11yViolations(page);

    // Cancel changes nothing.
    await page.getByRole("link", { name: "Cancel" }).click();
    await expect(page).toHaveURL(/\/account$/);

    // A refusal is announced as an alert, with the reason, and the person stays where they are.
    await page.getByRole("button", { name: "Delete account" }).click();
    await page.getByRole("button", { name: "Yes, delete my account" }).click();
    await expect(page.getByRole("alert").filter({ hasText: "cannot delete accounts" })).toBeFocused();
    await expect(page).toHaveURL(/\/account\/delete$/);
  });

  journey("signing out goes through Cognito's sign-out page", async ({ page }) => {
    // Neither call may really end the suite's session: answer the API, and the Cognito page it names.
    await page.route("**/api/auth/sign-out", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ logout_url: "https://logout.example.test/logout?client_id=c" }),
      }),
    );
    const reached = page.waitForRequest("https://logout.example.test/**");
    await page.route("https://logout.example.test/**", (route) =>
      route.fulfill({ status: 200, contentType: "text/html", body: "<title>Signed out</title>" }),
    );

    await page.goto("/sign-out");
    await reached;
  });
});
