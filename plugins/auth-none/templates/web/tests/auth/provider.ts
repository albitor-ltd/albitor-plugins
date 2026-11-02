// The auth provider specs use: the one object a spec or auth.setup.ts calls to learn the roles
// and to be one of them.
//
//   import { auth } from "./auth/provider";
//   auth.roles();                                     // the declared services.auth.groups
//   await auth.seed(role, { request });              // idempotent
//   await auth.signInAs(role, { context, request, baseURL });
//
// The implementation lives in helpers/auth-v2.ts; this file states which kind of sign-in the app has.
import { createAuthV2 } from "../helpers/auth-v2";

export const auth = createAuthV2({ shape: "none" });
