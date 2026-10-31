// Starting point (test-playwright).
import type { APIRequestContext, BrowserContext } from "@playwright/test";
import { authFor } from "./screen-rules";

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

export type LocalIdentityName = "alice" | "bob" | "admin" | `role-${string}`;

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
    // A single-page app answers every unknown path 200 with its HTML shell; only the issuer
    // answers this one with JSON.
    return res.ok() && (res.headers()["content-type"] ?? "").includes("json");
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

// The session cookies a signed-in browser carries, for a local run. The auth plugins read the
// session from an HttpOnly cookie when no bearer is sent: auth-cognito names it `session`,
// auth-hosted `__Host-session`. Setting both is harmless (the one a plugin does not read is
// ignored) and keeps this helper from knowing which plugin is composed. Chromium treats
// http://localhost as secure, so the Secure attribute the plugins require is accepted there.
export function localSessionCookies(tokens: LocalTokens, baseURL: string) {
  return ["session", "__Host-session"].map((name) => ({
    name,
    value: tokens.idToken,
    // domain and path, not url: Chromium refuses a `__Host-` cookie given by url over http.
    domain: new URL(baseURL).hostname,
    path: "/",
    httpOnly: true,
    secure: true,
    sameSite: "Strict" as const,
  }));
}

// attachAuth adds `auth` to the app's own fetch and XHR calls to `origin`, by a route that checks
// the origin (see authFor). Three things it is not:
//
//  - Not for pages, scripts, module preloads, images or EventSource: they go as they are, and a
//    request whose Sec-Fetch-Dest is not "empty" (a preload, a script) is never round-tripped.
//  - Not for streams. A call is answered by route.fetch, which buffers the whole response before the
//    browser sees any of it, so a fetch() that streams (Accept: text/event-stream) is left alone
//    and goes WITHOUT the bearer: give such a call its own Authorization header, which is also
//    always left alone.
//  - Not for the hop after a redirect. The browser follows a 3xx we hand it itself, and that hop
//    does not come back through this route, so it carries no bearer: an API that redirects to
//    another API path sees an anonymous request and fails closed. That is the price of never
//    handing the bearer to a redirect's target, and the reason route.continue({ headers }) is not
//    used (Chromium keeps an overridden header across a redirect, so an app-origin URL that
//    redirects to another host would hand that host the bearer).
//
// A request still in flight when the context closes is dropped quietly: route.fetch and
// route.fulfill throw "disposed" errors then, and an error out of a route handler fails the test.
export async function attachAuth(
  context: BrowserContext,
  auth: Record<string, string>,
  origin: string,
): Promise<void> {
  await context.route("**/*", async (route) => {
    const req = route.request();
    const type = req.resourceType();
    const sent = req.headers();
    const streams = (sent.accept ?? "").includes("text/event-stream");
    const headers =
      (type === "fetch" || type === "xhr") && !streams && (sent["sec-fetch-dest"] ?? "empty") === "empty"
        ? authFor(req.url(), origin, sent, auth)
        : undefined;
    try {
      if (!headers) {
        await route.continue();
        return;
      }
      const response = await route.fetch({ headers, maxRedirects: 0 });
      await route.fulfill({ response });
    } catch {
      // The page or context went away mid-request (teardown), or the request failed outright. Fail
      // the request if it can still be failed, and never throw out of the handler.
      await route.abort().catch(() => {});
    }
  });
}

// Sign a whole browser context in: every request its pages make to the APP'S OWN ORIGIN, the
// app's fetches to /api/ included, carries the identity's token. A request to any other host (a
// CDN, an identity provider) goes without it: context.setExtraHTTPHeaders would send the bearer
// everywhere, so the header is added by a route that checks the origin instead.
export async function signInAs(
  context: BrowserContext,
  request: APIRequestContext,
  identity: LocalIdentityName,
  origin: string,
): Promise<LocalTokens> {
  const tokens = await localTokens(request, identity);
  await attachAuth(context, authHeader(tokens), origin);
  return tokens;
}
