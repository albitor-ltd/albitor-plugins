// Starting point (auth-none). This app's auth@2 provider: the one object a spec or auth.setup.ts
// calls to learn the roles and to be one of them. This app has no sign-in, so it has no roles: roles() is always empty and seed and signInAs refuse.
//
//   import { auth } from "./auth/provider";
//   auth.roles();                                     // the declared services.auth.groups
//   await auth.seed(role, { request });              // idempotent
//   await auth.signInAs(role, { context, request, baseURL });
//
// The implementation is test-playwright's (helpers/auth-v2.ts); this file is where the plugin states
// which sign-in it ships, so a screen that moved fails here rather than passing through the other one.
import { createAuthV2 } from "../helpers/auth-v2";

export const auth = createAuthV2({ shape: "none" });
