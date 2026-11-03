---
name: web-app-architecture
description: Use when planning a web_app or writing its foundation, to decide architecture before the work is split into nodes - where uploaded files live, how slow supplier work (transcription, document reading, long model calls) advances, which platform limits the design must fit, and how deployed walks get their data. Index of patterns; read only the ones that apply.
---

# Web app architecture patterns

This is an index. Each pattern below says when it applies, when it does not and how to decide. Read
`patterns/<name>.md` only for the patterns your brief touches; skip the rest. The patterns are
decisions for the plan and the foundation (what is built where, in what shape), not line-by-line
code. They point at the plugin skills that hold the code.

Decide each applicable pattern once, in the plan, and write the decision into the nodes that need it.
A build node that finds the plan silent on one of these should not invent an answer.

## file-storage

Read `patterns/file-storage.md`.

- **Use when** users upload, download or keep files: audio, video, images, documents, exports.
- **Avoid when** the data is small text or numbers that belong on the record (a note, a name).
- **How to decide** As a heuristic: a value that comes from a file, or could approach the 400 KB
  DynamoDB item limit, is a file: bytes go to the object bucket, the record keeps metadata and
  the key.

## slow-supplier-work

Read `patterns/slow-supplier-work.md`.

- **Use when** a step calls a supplier that can take longer than a request (transcription, document
  analysis, a long model call) or that returns a job id to poll.
- **Avoid when** the call returns well inside one request and one user waiting is fine.
- **How to decide** Estimate the worst case, not the typical one. As a heuristic, if it is over
  about a third of the request limit (see `limits`), or the supplier is job-based, use a queue, if
  the deploy target renders queues; otherwise the plan must say how the work advances.

## limits

Read `patterns/limits.md`.

- **Use when** the plan moves payloads, stores lists or blobs on one record, or fans out to
  suppliers; skim the table once for any plan.
- **Avoid when** n/a - it is a reference. Do not copy its numbers into code as constants without
  re-checking the deploy target.
- **How to decide** For every payload and every record, name the largest realistic size and compare
  it with the table. As a heuristic, a design within 2x of a limit changes shape instead.

## seeding-for-deployed-walks

Read `patterns/seeding-for-deployed-walks.md`.

- **Use when** journeys run against a deployed preview and need records (lists, detail pages,
  media) to exist.
- **Avoid when** the journeys only exercise empty states or sign-in.
- **How to decide** If a journey needs data, the plan names the app-API calls that create it and the
  node that owns them; no test-only route is ever planned.
