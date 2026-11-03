# seeding-for-deployed-walks: real API, once per preview

## Shape

A journey that runs against a deployed preview needs records to exist. Create them through the
app's real API, as a signed-in user, the same way the UI would. Seed once per preview and reuse the
records across that preview's walks; do not re-create them per test.

Never add a route, flag or environment switch that exists only so a test can plant data or skip
auth. A deployed test route is an unauthenticated write path in production code. The test-playwright
skill (`fixtureScope` and the seed helpers) holds the mechanics; this pattern is about what the plan
must provide so seeding is possible.

## What the plan must provide

- A real create endpoint for every record type a journey reads, owned by a named node that lands
  before the journeys that need it. If a seed call 404s, the dependency is not built; that is a
  missing node, not a reason to add a test route.
- A real way to attach media: a short recording or document uploaded through the file-storage path
  (`file-storage.md`), small and real (a few seconds of speech, a one-page PDF), not a stub that the
  supplier would reject.
- A seed that is idempotent per preview: look up by a stable name first, create if absent.
- Supplier-backed content (a transcript, a translation) is seeded by running the real flow once
  and reusing the result, so most walks do not spend credits or wait on a queue.

## Limits

One seed run should stay inside the daily gateway cap with room for the journeys themselves
(`limits.md`). A seed that triggers many supplier jobs per preview burns the cap for everyone.

## Sketch

```text
globalSetup / first walk: sign in -> GET by stable name -> missing? POST via API (and upload media)
every walk:               reuse the ids; create only throwaway records, delete them at the end
```

## Anti-patterns

- `/api/test/seed`, `/__reset`, a `?test=1` bypass.
- Writing to the data store from the test runner.
- Seeding in every test: slow, and burns supplier credits.
- Fixtures too fake for the real supplier path, so the walk passes only on the stand-in.
- Seeds with hard-coded ids that collide across previews.

## How a walk verifies it

Walks assert on seeded content as a user sees it. A cold preview (no seed yet) and a warm one must
both pass; running the seed twice must not duplicate records.
