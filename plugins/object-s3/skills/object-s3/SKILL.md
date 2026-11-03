---
name: object-s3
description: Use when an app composed with the object-s3 plugin (object-store@1) stores files - a person uploads a document or image, downloads one, attaches files to a record, deletes a record that owns files, or the app needs a size or type limit or a retention rule for files. File bytes go to the bucket directly by presigned URL, never through the API function or the data store. The Go handlers and the web client are worked examples to copy, then alter.
metadata:
  capability: object-store
---

# Object storage (object-s3)

Files live in a private S3 bucket the app declares in `albitor.yaml` as `services.objects`. This
plugin declares one, `uploads`. The hosted deploy creates it and hands the function its AWS name as
`ALBITOR_BUCKET_UPLOADS` (the logical name upper-cased, `-` as `_`) and the path the app's own
origin serves it at as `ALBITOR_OBJECTS_PATH_UPLOADS` (`/_objects/uploads/`), with `AWS_REGION` and
the execution role's credentials. The role may `GetObject`, `PutObject`, `DeleteObject` and
`ListBucket` on that bucket and nothing else. There is no Terraform and no bucket name in the
repository. Nothing is seeded: the code is in `examples/` for the foundation node and later
nodes to copy.

**Hosted lane only.** `services.objects` is rendered by the hosted deploy target and not by
`deploy-byoa-aws`, whose `renders` list omits it. Composing this plugin with a BYOA target is refused
at Start (`manifest_unrendered`), not discovered at deploy; no other guard is needed.

## Same origin: how the browser reaches the bucket

Each bucket is served by the app's own CloudFront at `/_objects/<name>/*`. The distribution forwards
to the bucket's regional S3 host unchanged, with caching off, every method, every query string and
the `Content-Type`, `Content-Length` and `x-amz-*` headers, and does not re-sign. So:

- Object keys **must** start `_objects/<name>/`: the path is the key with a leading slash.
- The API presigns against the **bucket's regional virtual-hosted host** (SigV4 signs `host`, and
  CloudFront presents that host to S3, so the signature stays valid) and hands the browser only the
  **relative path and query**, `/_objects/uploads/<owner>/<record>/<file>?X-Amz-...`.
- The browser's request is same-origin: no CORS rule and no CSP change. Use the path as given.
  Never hand out the S3 host, and never build the URL in the browser.
- The examples keep the prefix out of the app: `Store` takes logical keys (`<owner>/<record>/<file>`)
  and `S3` adds `_objects/<name>/` from `ALBITOR_OBJECTS_PATH_<NAME>`.

## The rule: bytes never go through the API or the data store

- The API function takes request bodies up to 6 MB, and a base64 body costs a third more. A real
  file does not fit.
- A DynamoDB item is at most 400 KB, and every byte in it is a billed write unit. Never store file
  content, base64 or a data URL in a record. A record holds the file's **details** (id, name,
  type, size) and the object's key is derived from ids, not stored as a URL.
- So the API only **signs**. It authorises the person, then returns a short-lived URL allowing one
  method on one key; the browser sends or fetches the bytes against the bucket itself.

```
browser                       API (Lambda)                    bucket
  | POST .../files {name,type,size} |                            |
  |-------------------------------->| check owner, size, type    |
  |<-- {file_id, upload{url,headers}}  sign PUT (5 min)          |
  | PUT /_objects/uploads/... (same origin, via CloudFront) ----->|
  | POST .../files/{id}/complete -->| HEAD the object, re-check  |
  |<-- {id,name,content_type,size}  | (keep these on the record) |
```

## Examples (copy, then alter)

Worked examples sit in `examples/` beside this skill, laid out by repository path: copy a file to
the same path in the app. They are the app's code once copied; they never reference this skill. They
use only the standard library, so `api/go.mod` gains no dependency.

| Example | What it is |
| --- | --- |
| `examples/api/internal/objects/` | The storage package. `Store` (presign put and get, head, list, delete), `S3` (SigV4 query signing by hand), `Memory` (the local run and test fake, which refuses a PUT that differs from what was signed, as S3 does), `Policy` (size and type), `Key`/`Prefix`/`NewID`, `DeleteUnder`, `Open` (reads `ALBITOR_BUCKET_*`, never falls back to memory when deployed) |
| `examples/api/internal/app/files.go`, `files_test.go` | The four routes under `/api/records/{record}/files`: start an upload, complete it, get a download URL (JSON, or a 302 with `?redirect=1`), delete one. And `DeleteRecordFiles`. Rename `records` to the app's own noun |
| `examples/web/src/lib/files/upload.ts`, `upload.test.ts` | `uploadFile`, `downloadUrl`, `downloadLink`, `deleteFile`, `checkFile`. They return `{ ok: false, code, message }` rather than throw on a refusal |

To wire them in: copy both Go directories, then in `internal/app/routes.go` build the store once and
register the routes. In a local run also serve the fake bucket.

```go
func Routes(r *http.ServeMux) {
	store, err := objects.Open(os.Getenv, objects.Bucket)
	if err != nil {
		// Do not stop the app: its other routes work. The file routes answer 503
		// and the log says which variable is missing.
		slog.Error("object storage is not configured", "err", err)
		store = objects.Unavailable(err)
	}
	files := &Files{
		Store:      store,
		Policy:     objects.Policy{MaxSize: 10 << 20, Types: map[string]bool{"application/pdf": true, "image/png": true, "image/jpeg": true}},
		OwnsRecord: ownsRecord, // reads the record from the data store; see below
	}
	files.Routes(r)
	if mem, ok := store.(*objects.Memory); ok {
		r.Handle(objects.LocalPath, mem.Handler())
	}
}
```

Add the four routes to `api/openapi.yaml` in the same change, and keep `/api/local-objects/` out of
it. The web client calls them through `web/src/lib/shell/api.ts`.

## Authorise before signing

A signed URL is a bearer token: whoever holds it can use it until it expires. So every decision is
made before it is minted, on the server.

- **Owner-scoped keys.** The key is `<owner>/<record>/<file>`. The owner is the verified identity
  (`identity.Owner(r)`), never a body field or a query parameter. The record id comes from the URL
  and is turned into a safe key segment by `objects.Segment` (an id that is already `[A-Za-z0-9._-]`
  and short stays as it is; anything else, such as a composite sort key `REC#2026/17`, an email or
  a unicode id, becomes `h` plus a hash). Use `Segment` everywhere a record id becomes a key, pass
  the raw id to `OwnsRecord`, and `objects.Key` still refuses a bad segment (no slash, no `..`, no
  leading dot), so a crafted id cannot step out of the owner's prefix. A key that fails is a 404
  and is logged with the correlation id. The file id is `objects.NewID()`, random, so a key cannot be
  guessed.
- **Own the record.** `OwnsRecord` loads the record from the data store and compares its owner to
  the identity before any URL is signed. A nil `OwnsRecord` fails closed (500): nothing is signed. A record that is not there and one that is not theirs both
  answer 404, so a stranger learns nothing. If records are shared, this is where the rule for who
  else may attach or read is written; the key's owner segment stays the uploader's.
- **The person's file name is metadata, never part of the key.** It is cleaned (`cleanName`: base
  name only, no control characters, 200 bytes) and signed into the PUT as `x-amz-meta-filename` and
  as `Content-Disposition: attachment; filename*=...`, which S3 stores with the object. A download is
  then a plain GET: the app's CloudFront may not forward `response-content-*` query parameters, so
  never presign a GET with them (the signature would fail with a 403). The browser must send the
  `Content-Type`, `Content-Disposition` and `x-amz-meta-filename` headers the API returned, exactly.
- **Short-lived.** URLs live 5 minutes by default and never more than 15 (`MaxExpiry`); the function's
  credentials are temporary and a URL stops working when they rotate, whatever its expiry says.
  Sign a download on demand, per click; never put a signed URL in a record or a cache.
- Log the key and the correlation id, never the signed URL: it carries a credential in its query.

## Size and content-type limits

Checked twice, and signed once.

1. `Policy.Check` refuses an empty file, one over `MaxSize`, or a type not on the allowlist, with a
   422, **before** anything is signed. Pick the allowlist per product need; do not accept
   `text/html`, `image/svg+xml` or other types a browser would run.
2. The size and content type are signed into the URL (`content-length` and `content-type` are
   signed headers), so S3 itself answers 403 to a body of another length or type. `Memory` does the
   same, so a local run behaves the same.
3. `complete` does a HEAD and checks the policy against what actually landed; a mismatch deletes
   the object.

`complete` is not bound to the `start` that issued the URL, and does not need to be for safety: the
key is under the caller's own owner and record, the file id is random, and the signed length and
type mean anything there conforms (and is re-checked). It does not make a file single-use or cap a
record's file count: make the record write idempotent on the file id, and cap files per record in
that write if the product needs it.

The type is the browser's claim, not a scan of the bytes: it does not make a file safe to open. Keep
downloads as `attachment` (the examples do), never serve a file inline from the app's own origin,
and if the product needs more (virus scanning, image validation) that is a separate step on the
object after upload.

The bucket's own ceiling for one PUT is 5 GiB, far above any sensible limit here; the bound that
matters is the app's `MaxSize`. A file too large for a single PUT needs multipart upload, which the
examples do not cover.

## Delete with the record

A record that owns files must delete them when it is deleted; the bucket does not know about records.

- In the delete handler call `DeleteRecordFiles(r, store, record)` **before** deleting the record. If
  it fails, the record is still there and the person retries; deleting the record first would leave
  files nothing points at. A delete of a key that is not there succeeds, so a retry is safe.
- Deleting one file: `DELETE /api/records/{record}/files/{file}`, and remove it from the record's list
  in the same handler.
- A person deleting their account: `objects.Prefix(owner, "")` gives every key they own.
- An upload that is started and never completed leaves an orphan under the record's prefix. List the
  prefix (`Store.List`) against the files the record names and delete the difference, in the record's
  own handlers or a scheduled job.

## Retention

The bucket is **private, encrypted (SSE-S3), TLS-only and versioned**, and is not destroyed if
dropped from the manifest (`force_destroy` is off). The platform's lifecycle:

- Noncurrent versions and incomplete multipart uploads expire by default, so a delete leaves a delete
  marker and the old version is removed after that window, not at once. "Deleted" means gone from the
  app, then from storage later.
- `services.objects` gains an optional `expire_after_days` for **current** objects. Declare it for a
  bucket whose files should lapse by age (add it to the manifest fragment; this plugin's own fragment
  does not set it, because the right period is a product decision). Do not invent other keys: the
  manifest is closed.

`expire_after_days` is an age backstop, not the product's rule. Keep the app-side cleanup: records
that end (deleted, or past their own `expires_at`) must delete their files with `DeleteRecordFiles`,
from the delete handler or a schedule (15 minutes or slower; see api-go's `POST /events`) in small
idempotent batches, so a record and its files go together and no file outlives the record that
explains it. Say in the product copy only what both rules guarantee.

## Deployed limits to design around

- **Unverified until a walk.** The same-origin route (`/_objects/<name>/*`) is the platform's
  contract and ships with the hosted app module. Until it is live on the target, a deployed browser
  upload is unproven: treat it as verified only when a walk against the deployed app has uploaded and
  then downloaded a file. A local run proves the handlers, not the route.
- **Local run.** `ALBITOR_LOCAL=1` keeps files in process memory and the API serves the "signed" URLs
  at `/api/local-objects/`; a restart empties it. Seed any files a walk needs through the app's own
  upload route.
- **Two buckets at most**, each `^[a-z][a-z0-9-]{1,31}$`, and a bucket name S3 refuses is rejected.
  The derived AWS name has no dots, so virtual-hosted TLS works.
- **No listing for the browser.** Show the files a record names from the data store, not from the
  bucket.
- **The signer is written out by hand on purpose.** api-go's module has no dependencies and a plugin
  may not add one, so `S3` signs with the standard library, for single-object requests only. Do not
  extend it (multipart upload, checksums, KMS): that is the point to ask for a dependency-bearing
  plugin version instead. A file too large for one PUT is out of scope here.
