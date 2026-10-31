---
name: gateway-services
description: Use when an app composed with the deploy-hosted plugin needs to translate, transcribe, read documents, speak, detect language or call a model, or any AWS AI service, from its Go API. Declare it under services.callable in albitor.yaml and call it through api/internal/albitorsvc. Never add an AWS SDK client for these services, an API key or an IAM permission for them.
metadata:
  capability: deploy-target
---

# Gateway services (deploy-hosted)

A hosted app has no AWS AI-service permission and holds no supplier key. It calls the Albitor
service gateway, which makes the call from Albitor's account, meters it and bills it in credits.
The client is `api/internal/albitorsvc/` (package `webapp/internal/albitorsvc`, standard library
only). The plugin owns that directory; the build owns `api/internal/app/`.

## Starting points

Everything under `api/internal/albitorsvc/`, tests included, is seeded once and is yours from then
on. Keep the client's signing as it is: do not point an AWS SDK client at the gateway or add a
custom signer, because a stock SDK signs for the target service's name (`translate`, `transcribe`),
which the gateway cannot verify, and that is why this client exists.

## Steps to add a feature

1. **Declare it** in `albitor.yaml`; nothing else grants it, and a call outside the declared set is
   `403 not_declared`:

   ```yaml
   services:
     callable:
       - service: translate
       - service: textract
         features: [tables, forms]   # optional; narrows what the app may ask for
       - service: bedrock
         models: [claude-haiku]      # catalogue aliases, never model ids or ARNs
   ```

   Residency, rate limits, Regions, quotas and model ids are platform-owned and refused in the
   manifest. The platform sets `ALBITOR_SERVICES_URL` and the role's permission to call the gateway
   at deploy.
2. **Open the client once** in `main` or the app's setup:
   `svc, err := albitorsvc.Open(os.Getenv)`. `albitorsvc.ErrNotConfigured` means no URL is set (a
   local run, or nothing declared): keep the feature off and carry on, do not fail the start.
3. **Define request and response structs** with AWS's documented JSON field names
   (`TranslateText`: `Text`, `SourceLanguageCode`, `TargetLanguageCode`) and call
   `svc.Call(ctx, "translate", "TranslateText", req, &resp)`. The gateway rebuilds the request from
   an allow-list: a field it does not list is `400 bad_request`, so send only the fields the
   operation needs.
4. **Pass a fresh `albitorsvc.WithIdempotencyKey(...)`** on operations that start work or bill by
   the call (Bedrock, job starts), or the gateway answers `400 idempotency_key_required`. The key
   must match `^[A-Za-z0-9_-]{8,64}$` (a random hex or base64url string; no spaces, slashes or dots),
   and `Call` refuses any other before sending.

## Inputs and outputs are handles, never locations

A request never carries an S3 URI, bucket, key, ARN or URL, and the gateway rejects one. A service
that reads a file gets it through a slot:

```go
slot, err := svc.CreateUploadSlot(ctx, "transcribe", "audio/mpeg", maxBytes)
err = svc.Upload(ctx, slot, data)          // multipart POST; the slot is the credential
// then name the input by slot.Handle in the operation's request
```

A slot expires in 5 minutes: upload at once, and never log it, store it or send it to a browser. A
result the gateway returns as a URL expires in 5 minutes too; fetch it server-side and return the
content, not the URL, unless the user needs the file.

## Errors: say "unavailable", never retry around it

`svc.Call` returns `*albitorsvc.Error` (status and code only; AWS's message never arrives):

| Answer | Test | The app does |
| --- | --- | --- |
| `402 credits_exhausted`, `402 cap_reached` | `errors.Is(err, albitorsvc.ErrCreditsExhausted)` or `ErrCapReached` | Treat the feature as unavailable: tell the user it is unavailable right now and carry on. Do not retry, and do not queue the work to retry later |
| `403` (`not_declared`, `service_disabled`, `app_disabled`, residency) | `errors.Is(err, albitorsvc.ErrForbidden)` | The same: unavailable. A `not_declared` is a build bug, so log it once at startup of the feature |
| `429` | `errors.Is(err, albitorsvc.ErrRateLimited)` | The client has already retried with jittered, capped backoff and given up, or returned at once because the wait asked for is over its 5 second cap: answer "busy, try again shortly" |
| anything else | `*albitorsvc.Error` | A fault: log the code and `RequestID` against the correlation id, answer an opaque 500 through `apikit.WriteFailure` |

`albitorsvc.Unavailable(err)` is true for the first two rows. Never show a user the code, a credit
figure or the word "credits"; the owner sees spend in Albitor, the end user sees the feature
unavailable. Do not wrap `Call` in a retry loop: the client's retries already spend the app's own
rate budget.

## Denial of wallet: a gateway call is the owner's money

An endpoint that calls the gateway and is reachable without signing in is an open, credit-billed AI
proxy for the internet. By default:

- Put every gateway-backed route **behind the app's sign-in** (the `auth` plugin's guard), and take
  the end user's identity from the verified session, never from the body.
- Where the app has no sign-in, or a route must be public, **rate-limit per end user** (or per
  address) in the app itself, with a low daily allowance, before the call reaches `albitorsvc`.
- Bound what one call can cost: cap input length and file size in the app below the gateway's own
  caps, and never let a request choose the model, the feature set or how many calls fan out.
- Never put a gateway-backed call on an unauthenticated schedule or webhook without a counter.

The platform also holds the app to a daily cap and a rate limit; those protect Albitor, not your end
users, and an app that burns its cap turns the feature off for everyone.

## Tests

The app's own tests do not call the gateway. Put the calls behind a small interface the handler
depends on (`Translate(ctx, text, lang) (string, error)`), implement it with `albitorsvc` in
`main`, and fake it in tests, including the unavailable path: a fake that returns `albitorsvc.ErrCreditsExhausted` or
`albitorsvc.ErrForbidden` (both are ready-made `*albitorsvc.Error` values) must produce the
handler's "unavailable" answer.
