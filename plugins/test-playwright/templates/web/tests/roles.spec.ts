// Managed by Albitor (test-playwright). Every declared role signs in, and holds only its own group.
//
// This is the foundation's per-role check, and it calls only the auth@2 interface of the app's auth
// plugin (./auth/provider): roles() names the roles, seed(role) makes the account, signInAs(role)
// signs a fresh signed-out browser context in as it. The app then says who it thinks that is
// (GET /api/me) and the answer must be that role's group and no other: an account holding two
// groups could pass every role-gated screen and prove nothing about any one of them.
//
// One test per role, named `role <role> signs in and holds only its own group`, so a report says
// which role failed. An app with no sign-in, or no services.auth.groups, has no roles and one skipped
// test says so.
import { expect, test } from "@playwright/test";
import type { AuthV2 } from "./helpers/auth-v2";
import { EMPTY_STORAGE_STATE } from "./helpers/storage-state";

// The provider is the app's auth plugin's (web/tests/auth/provider.ts). A suite composed without an
// auth plugin has none, and so no roles: loading it must not fail the whole file.
const auth: AuthV2 | null = await import("./auth/provider").then(
  (m) => m.auth as AuthV2,
  () => null,
);

// Signed out: the role's own session is the only one these tests may use.
test.use({ storageState: EMPTY_STORAGE_STATE });

const roles = auth?.roles() ?? [];

if (roles.length === 0) {
  test("every declared role signs in (the app declares no roles)", () => {
    test.skip(true, auth ? "this app declares no services.auth.groups, so it has no roles to sign in as" : "this app has no auth plugin, so it has no roles");
  });
}

for (const role of roles) {
  test(`role ${role} signs in and holds only its own group`, async ({ browser, request, baseURL }) => {
    expect(baseURL, "the config resolves a baseURL for every run").toBeTruthy();
    const context = await browser.newContext({ storageState: EMPTY_STORAGE_STATE, baseURL });
    try {
      const seeded = await (auth as AuthV2).signInAs(role, { context, request, baseURL: baseURL as string });
      expect(seeded.groups, `the ${role} account holds only the ${role} group`).toEqual([role]);

      const page = await context.newPage();
      await page.goto("/");
      const me = await page.evaluate(async () => {
        const res = await fetch("/api/me", { credentials: "same-origin", headers: { Accept: "application/json" } });
        return { status: res.status, body: await res.json().catch(() => null) };
      });
      expect(me.status, `GET /api/me as ${role}`).toBe(200);
      const groups = (me.body as { groups?: string[] } | null)?.groups ?? [];
      expect(groups, `the app sees ${role} as holding exactly its own group`).toEqual([role]);
    } finally {
      await context.close();
    }
  });
}
