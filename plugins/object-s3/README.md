# Object storage plugin (`object-s3`)

The `object-store@1` plugin. A schema-2 plugin: `.claude-plugin/plugin.json` declares what it
provides, requires and contributes, and the platform composer maps the `file-storage` capability to
it.

It is a skill with examples and seeds nothing. It contributes the `uploads` bucket to
`services.objects` in `albitor.yaml` and the `object-s3` skill; the deploy target creates the bucket
and hands the function its name as `ALBITOR_BUCKET_UPLOADS`. No Terraform.

## What the skill carries

| Path (under `skills/object-s3/`) | What |
| --- | --- |
| `SKILL.md` | The rule (bytes never through the API or the data store), authorising before signing, limits, deleting with the record, retention, deployed limits |
| `examples/api/internal/objects/` | `Store`, the S3 adapter with hand-written SigV4 query signing, the in-memory fake, `Policy`, key helpers |
| `examples/api/internal/app/files.go`, `files_test.go` | Upload, complete, download and delete routes; `DeleteRecordFiles` |
| `examples/web/src/lib/files/` | The browser side: upload by signed URL, download link, delete |

The Go examples are standard library only, so `api/go.mod` gains no dependency. `scripts/verify-object-s3.sh`
(part of `just ci`) copies them over api-go's tree with auth-hosted's identity package, then builds,
vets and tests them with the race detector; the signer is held to the AWS documentation's published
presigned-URL signature.

## Requires

`api-runtime@1` and `deploy-target@1`. The web example assumes `web-framework@1` is `web-svelte`.

## Same origin, hosted lane only

The hosted app module serves each bucket on the app's own origin at `/_objects/<name>/*`, so the
browser needs no CORS rule or CSP change; the API presigns against the bucket's regional host and
returns the relative path. Object keys start `_objects/<name>/`; the app reads
`ALBITOR_BUCKET_<NAME>` and `ALBITOR_OBJECTS_PATH_<NAME>`. `deploy-byoa-aws` does not render
`services.objects`, so the composer refuses this plugin on a BYOA target at Start. A deployed upload
is verified only by a walk once that route ships; see the skill's "Deployed limits".
