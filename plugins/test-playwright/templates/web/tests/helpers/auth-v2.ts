// Starting point (test-playwright). The auth@2 interface every auth plugin implements, and the
// one a spec or auth.setup.ts calls to be a role: roles(), seed(role), signInAs(role).
//
// A SPEC NEVER SIGNS IN BY HAND AND NEVER NAMES A TEST USER. It asks the app's auth plugin
// (web/tests/auth/provider.ts, shipped by auth-none, auth-cognito or auth-hosted) which roles exist,
// and signs in as one:
//
//   import { auth } from "./auth/provider";
//   for (const role of auth.roles()) { ... await auth.signInAs(role, { context, request, baseURL }); }
//
// ONE ACCOUNT PER ROLE, HOLDING THAT ROLE'S GROUP AND NO OTHER. A role is a group of
// services.auth.groups. The account signInAs signs in as is in that group alone, so a spec that
// signs in as `manager` proves what a manager can see, and the screens of every other role stay
// refused to it. No account is given every group.
//
// WHERE THE ACCOUNT COMES FROM
//   locally   api-go's local issuer (ALBITOR_LOCAL_AUTH=1): seed(role) registers the role's identity
//             in the running API (POST /api/_local/auth/seed), signInAs(role) mints its tokens and
//             attaches them to the context's calls to the app. No sign-up, no email, no AWS.
//   deployed  the account deploy-byoa-aws's seed created, `<role>@albitor-role.invalid`, with the
//             password the deploy wrote to the app's secret store (see roles.ts for how it reaches the
//             run). seed(role) only confirms the credentials reached this run: the deploy seeds, a
//             spec never creates an account. signInAs(role) signs in through the browser, in the
//             plugin's own sign-in (in-app form or hosted page).
import type { APIRequestContext, BrowserContext } from "@playwright/test";
import { browserSignIn, type SignInShape } from "./browser-signin";
import { readCredentials } from "./credentials";
import { attachAuth, authHeader, localIssuerRunning, localTokens, LOCAL_AUTH_PATH } from "./local-auth";
import { readDeclaredRoles, readRoleCredentials, localRoleIdentity, ROLE_PATTERN } from "./roles";
import { WEB_ROOT } from "./storage-state";

/** What a role's account is, as seeded. */
export type SeededRole = {
  role: string;
  /** "local": the in-process issuer. "deployed": the app's own user pool. */
  mode: "local" | "deployed";
  /** The account's email: what the app sees in /api/me. */
  account: string;
  /**
   * The groups the account holds, exactly [role]. Reported by the local issuer only; a deployed
   * account's groups are not known here, so the app's own /api/me is what proves them.
   */
  groups?: string[];
};

export type SeedContext = { request: APIRequestContext };
export type SignInContext = { context: BrowserContext; request: APIRequestContext; baseURL: string };

export interface AuthV2 {
  readonly version: 2;
  /** The roles the app declares (services.auth.groups): none for an app with no sign-in or no groups. */
  roles(): string[];
  /** Make the role's account exist (locally) or confirm the deploy's seed reached this run (deployed). Idempotent. */
  seed(role: string, ctx: SeedContext): Promise<SeededRole>;
  /** Sign `ctx.context` in as the role's account. Seeds first. Throws, never skips, when it cannot. */
  signInAs(role: string, ctx: SignInContext): Promise<SeededRole>;
}

export type AuthV2Options = {
  /** The sign-in the plugin ships. "none": no sign-in, so no roles. */
  shape: SignInShape | "none";
  /** Overridable for tests. */
  declaredRoles?: () => string[];
  env?: Record<string, string | undefined>;
};

export function createAuthV2(options: AuthV2Options): AuthV2 {
  const env = options.env ?? process.env;
  const declared = options.declaredRoles ?? (() => readDeclaredRoles(`${WEB_ROOT}/..`));
  const roles = () => (options.shape === "none" ? [] : declared());

  const known = (role: string): void => {
    if (!ROLE_PATTERN.test(role) || !roles().includes(role)) {
      const have = roles();
      throw new Error(
        `"${role}" is not a role of this app: ` +
          (have.length > 0 ? `it declares ${have.join(", ")}` : "it declares no services.auth.groups, so it has no roles"),
      );
    }
  };

  const seed = async (role: string, { request }: SeedContext): Promise<SeededRole> => {
    known(role);
    if (await localIssuerRunning(request)) {
      const res = await request.post(`${LOCAL_AUTH_PATH}/seed`, { data: { role } });
      if (!res.ok()) {
        throw new Error(`seeding role ${role} in the local issuer failed: ${res.status()} ${await res.text()}`);
      }
      const id = (await res.json()) as { email: string; groups: string[] };
      return { role, mode: "local", account: id.email, groups: id.groups };
    }
    const creds = readRoleCredentials(env, role);
    if (!creds) {
      throw new Error(
        `no account for role ${role} reached this run: the deploy seeds ${role}@albitor-role.invalid, and its password ` +
          `comes only from the journey credentials file (roles.${role}.password), and it has none`,
      );
    }
    return { role, mode: "deployed", account: creds.username };
  };

  return {
    version: 2,
    roles,
    seed,
    async signInAs(role, ctx) {
      const seeded = await seed(role, ctx);
      if (seeded.mode === "local") {
        const tokens = await localTokens(ctx.request, localRoleIdentity(role));
        await attachAuth(ctx.context, authHeader(tokens), new URL(ctx.baseURL).origin);
        return seeded;
      }
      const creds = readRoleCredentials(env, role);
      if (!creds) throw new Error(`no account for role ${role} reached this run`);
      const page = await ctx.context.newPage();
      try {
        // The sign-in lands its session in this context; the page is only the way there.
        await browserSignIn(
          page,
          ctx.baseURL,
          { ...creds, signInRoute: readCredentials(env).signInRoute },
          options.shape === "none" ? "any" : options.shape,
        );
      } finally {
        await page.close().catch(() => {});
      }
      return seeded;
    },
  };
}
