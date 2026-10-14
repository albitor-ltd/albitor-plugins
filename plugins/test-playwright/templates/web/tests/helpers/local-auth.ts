// Managed by Albitor (test-playwright).
import type { APIRequestContext, BrowserContext } from "@playwright/test";

// SIGN IN AS A NAMED TEST IDENTITY, IN A LOCAL RUN ONLY.
//
// A local run of the API with ALBITOR_LOCAL=1 ALBITOR_LOCAL_AUTH=1 starts an
// in-process issuer (api/internal/localauth) in place of the user pool. It
// mints Cognito-shaped tokens for fixed identities — two users and an admin —
// with no sign-up, no e-mail and no call to Cognito. A spec that needs to be
// someone, or two different someones, signs in here.
//
// The issuer only issues. The tokens mean something once the app's sign-in
// verifies them and trusts this issuer in a local run; an app with no sign-in
// ignores them, and every request is still anonymous.
//
// It is never there against a deployed app: the API refuses to start the
// issuer on a deployed function or beside a real user pool. So a spec using it
// checks first and skips when it is absent:
//
//   test.beforeEach(async ({ request }) => {
//     test.skip(!(await localIssuerRunning(request)), "no local issuer: not a local run with ALBITOR_LOCAL_AUTH=1");
//   });
//
// Signing in through the app's own screens is a different spec — the one
// journey that tests sign-in (see tests/auth.setup.ts).

export const LOCAL_AUTH_PATH = "/api/_local/auth";

export type LocalIdentityName = "alice" | "bob" | "admin";

export type LocalTokens = {
  idToken: string;
  accessToken: string;
  sub: string;
  email: string;
  groups: string[];
};

// Whether the API under test is running the local issuer.
export async function localIssuerRunning(request: APIRequestContext): Promise<boolean> {
  try {
    const res = await request.get(`${LOCAL_AUTH_PATH}/identities`);
    return res.ok();
  } catch {
    return false;
  }
}

// Mint tokens for one identity. Throws on anything but a 200, so a failed
// sign-in reads as one rather than as a 401 three steps later.
export async function localTokens(request: APIRequestContext, identity: LocalIdentityName): Promise<LocalTokens> {
  const res = await request.post(`${LOCAL_AUTH_PATH}/token`, { data: { identity } });
  if (!res.ok()) {
    throw new Error(`local sign-in as ${identity} failed: ${res.status()} ${await res.text()}`);
  }
  const body = (await res.json()) as {
    id_token: string;
    access_token: string;
    identity: { sub: string; email: string; groups: string[] };
  };
  return {
    idToken: body.id_token,
    accessToken: body.access_token,
    sub: body.identity.sub,
    email: body.identity.email,
    groups: body.identity.groups,
  };
}

// The header an API call carries to act as these tokens' identity.
export function authHeader(tokens: LocalTokens): Record<string, string> {
  return { Authorization: `Bearer ${tokens.idToken}` };
}

// Sign a whole browser context in: every request its pages make, the app's own
// fetches to /api/ included, carries the identity's token.
export async function signInAs(
  context: BrowserContext,
  request: APIRequestContext,
  identity: LocalIdentityName,
): Promise<LocalTokens> {
  const tokens = await localTokens(request, identity);
  await context.setExtraHTTPHeaders(authHeader(tokens));
  return tokens;
}
