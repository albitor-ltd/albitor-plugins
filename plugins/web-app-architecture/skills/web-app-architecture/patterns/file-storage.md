# file-storage: bytes never go in the data store or through Lambda

## Shape

Files live in the app's object bucket (`services.objects`). The browser uploads and downloads
directly, over presigned URLs, through the app's own origin at `/_objects/<name>/`. The API only
issues the URL and records metadata. The record keeps the object key, content type, byte size,
owner and a status; never the bytes, never a base64 string.

Read the `object-s3` plugin's skill for the code and the same-origin contract (key prefix
`_objects/<name>/`, the `ALBITOR_OBJECTS_PATH_<NAME>` environment variable, presigning against the
bucket host and handing the browser the relative path and query). Compose it where the deploy
target provides buckets (hosted). Plan it as its own early node: others depend on the key shape.

## Limits

Numbers are in `limits.md`; the ones that force this pattern:

- A synchronous Lambda request or response is capped at 6 MB, and the gateway ceiling for a service
  call body is smaller still. An upload through the API fails for any real recording or scan.
- A DynamoDB item is capped at 400 KB including attribute names, and every write is billed per KB.
  A 1 MB file cannot be stored there at all.

## Sketch

```text
browser -> POST /api/uploads {name, type, size}      API checks owner, type, size cap,
                                                     creates record {id, status: pending, key}
        <- {id, uploadPath}                          key = _objects/<name>/ then owner/<id>/<safe-name>
browser -> PUT  uploadPath  (bytes, same origin)     straight to the bucket
browser -> POST /api/uploads/<id>/complete           API checks the object exists, sets ready
browser -> GET  /api/files/<id>  -> {downloadPath}   short-lived presigned path, owner checked
```

Retention means deleting the files with the record (the `object-s3` skill's `DeleteRecordFiles`),
so no record outlives or orphans its object. The bucket's optional `expire_after_days` is only an
age backstop for current objects, not the retention design.

## Anti-patterns

- Base64 in JSON, or multipart through the API: breaks at the 6 MB payload limit and wastes memory.
- Putting content (a transcript, an extracted document) on the record "just this once": records
  stop fitting at 400 KB. Store large derived text as an object too, keep a short preview on the
  record.
- Keys built from user-supplied names without an owner prefix: one user can overwrite or read
  another's file. Prefix by owner and use a generated id.
- A public bucket or a permanent URL. Presign each access, short expiry.
- Trusting the client's declared size or type. Cap them when issuing the URL and re-check on
  complete.

## How a walk verifies it

A journey uploads a small real file through the browser, reloads, and sees it listed with the right
size; downloads it and compares the bytes (or a hash) with the original; deletes it and asserts the
download now fails. The API never receives the bytes, so no walk should assert on an API response
body containing file content.
