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
3. **Look up the operation in [references/request-shapes.md](references/request-shapes.md)**,
   define request and response structs with exactly the fields listed there, and call
   `svc.Call(ctx, "translate", "TranslateText", req, &resp)`. This is not the AWS SDK's request: the
   gateway rebuilds each request from a per-operation allow-list, **refuses any field it does not
   list (`400 bad_request`) and never ignores one**, and sets the rest itself (the Transcribe job
   name, media and output locations, role, settings). Never copy an AWS SDK input struct. For
   Transcribe that means `UploadHandle` plus a language, not `TranscriptionJobName`, `MediaFormat`
   or `Media`.
4. **Pass a fresh `albitorsvc.WithIdempotencyKey(...)`** on exactly the operations marked in the
   reference: `StartTranscriptionJob`, Textract's two `Start...` operations and Polly
   `SynthesizeSpeech`. The gateway answers `400 idempotency_key_required` without one. Bedrock,
   Translate, Comprehend, Location and the Get operations take no key. The key must match
   `^[A-Za-z0-9_-]{8,64}$` (a random hex or base64url string; no spaces, slashes or dots), and `Call`
   refuses any other before sending.

## Inputs and outputs are handles, never locations

A request never carries an S3 URI, bucket, key, ARN or URL, and the gateway rejects one. A service
that reads a file gets it through a slot:

```go
slot, err := svc.CreateUploadSlot(ctx, "transcribe", "audio/mpeg", maxBytes)
err = svc.Upload(ctx, slot, data)          // multipart POST; the slot is the credential
// then name the input by slot.Handle in the operation's request
```

The same applies to Bedrock media: an image, document or video block's `source` is exactly one of
`{"bytes": ...}` (inline, within the 5,500,000-byte body ceiling) or `{"uploadHandle": slot.Handle}`
from `CreateUploadSlot(ctx, "bedrock", contentType, maxBytes)`; a video is by handle only. See the
Bedrock section of the reference for exact blocks.

A slot expires in 5 minutes: upload at once, and never log it, store it or send it to a browser. A
result the gateway returns as a URL expires in 5 minutes too; fetch it server-side and return the
content, not the URL, unless the user needs the file.

## Async jobs: poll by delayed re-enqueue, not on a schedule

Transcribe and Textract's `Start...` operations return at once and the result arrives later; the
app polls (`GetTranscriptionJob`, the Textract `Get...`). Do not poll from a `schedules` entry: a
schedule fires at most every 15 minutes, so a user waiting on the result waits up to that long for
a job that took seconds. Use a queue and a consumer that re-enqueues itself with a delay:

1. The route that starts the job calls `Start...` (with its idempotency key), then
   `q.Send(ctx, "transcribe", jobRef{JobID: id}, albitorsvc.SendOptions{Delay: 5 * time.Second})`.
   Queue the job id and the record it belongs to, never the audio or the result.
2. The consumer (`POST /events`, see the api-go skill) calls the `Get...` operation. Still running:
   `q.Send(ctx, queue, ref, albitorsvc.SendOptions{Parent: &m, Delay: 10 * time.Second})`, which
   bumps `attempt` and keeps the chain's deadline. Done: store the result on the record and, for
   the next step of a pipeline, `Send` that step's message. Failed: mark the record failed.
3. `Send` returns `ErrAttemptsExceeded` (50 re-enqueues) or `ErrDeadlinePassed` (the chain's
   horizon, at most one hour) when the job never finishes: mark the record failed and stop. A
   polling loop is finite by construction; do not catch the error and send again with no parent.
4. Keep every step under the 30 second function timeout and idempotent: a message can arrive twice,
   so a `Get...` on a finished job must store the same result, not start a second job. Each poll is
   a gateway call and bills like one, so back the delay off (5, 10, 20 seconds) rather than polling
   tightly; the delay is at most 900 seconds.
5. Answer a `402`/`403` from the gateway as "unavailable": mark the job failed, do not re-enqueue.

Queues need a deploy target that renders `queues`; until then a user-awaited step runs inline.

## Errors: say "unavailable", never retry around it

`svc.Call` returns `*albitorsvc.Error` (status and code; AWS's message never arrives). On a `400
bad_request` it also carries `Field` and `Detail` when the gateway names the refused field (needs
the gateway change in albitor PR #4271; empty on an older gateway, and some checks never name one), and the
text shows them: `albitorsvc: 400 bad_request (field MediaFormat: unknown field)`. Log that text;
a `400` is a build bug in the request shape, never retry it:

| Answer | Test | The app does |
| --- | --- | --- |
| `402 credits_exhausted`, `402 cap_reached` | `errors.Is(err, albitorsvc.ErrCreditsExhausted)` or `ErrCapReached` | Treat the feature as unavailable: tell the user it is unavailable right now and carry on. Do not retry, and do not queue the work to retry later |
| `403` (`not_declared`, `service_disabled`, `app_disabled`, residency) | `errors.Is(err, albitorsvc.ErrForbidden)` | The same: unavailable. A `not_declared` is a build bug, so log it once at startup of the feature |
| `429` | `errors.Is(err, albitorsvc.ErrRateLimited)` | The client has already retried with jittered, capped backoff and given up, or returned at once because the wait asked for is over its 5 second cap: answer "busy, try again shortly" |
| anything else | `*albitorsvc.Error` | A fault: log the error text (code, and `Field`/`Detail` on a 400) and `RequestID` against the correlation id, answer an opaque 500 through `apikit.WriteFailure` |

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
- Never use a schedule to wait on a gateway job a user is waiting for; see the async section above.

The platform also holds the app to a daily cap and a rate limit; those protect Albitor, not your end
users, and an app that burns its cap turns the feature off for everyone.

## Tests

The app's own tests do not call the gateway. Put the calls behind a small interface the handler
depends on (`Translate(ctx, text, lang) (string, error)`), implement it with `albitorsvc` in
`main`, and fake it in tests, including the unavailable path: a fake that returns `albitorsvc.ErrCreditsExhausted` or
`albitorsvc.ErrForbidden` (both are ready-made `*albitorsvc.Error` values) must produce the
handler's "unavailable" answer.
