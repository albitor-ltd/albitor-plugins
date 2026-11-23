# screen-data: one call per screen

The hosted API runs at most 10 requests at once and takes 50 a second (`limits.md`). A screen that
fans out one request per row spends that budget on a single page load; the overflow is answered
503 or 429, and it looks fine in a walk whose fixtures hold three records. The rule: a screen gets
what it renders from one Lambda call, and work moves into the handler, where one invocation can do
it with DynamoDB `Query` and `BatchGetItem`.

## The rules

- **Lists return what the list renders.** Embed the summaries and counts a row shows (owner name,
  open-task count, last-activity time) in each list item. If a row needs a field, the list has it;
  the client never fetches a detail to fill a row.
- **A detail page is one call.** The record with the child summaries it displays, in one response.
- **A cross-entity screen gets a screen-shaped endpoint.** A dashboard, a meeting-prep page or an
  overview that shows several entities is one endpoint (`GET /api/dashboard`) whose handler reads
  them all and answers one document. Name it for the screen, not the table.
- **When a batch read is unavoidable, take `ids=`.** `GET /api/opportunities?ids=a,b,c` with a
  hard cap (50 is enough; `BatchGetItem` takes 100 keys) answers every id in one invocation.
  Refuse more with a 400 rather than truncating.
- **Never** fetch the detail of each row, `Promise.all` over a list of ids, or call a list and then
  loop over its items. Each of those is N concurrent invocations.
- **Page, do not widen.** A big screen pages the list (`cursor`, a limit of at most 100). Embedding
  summaries is not licence to return whole records: the response is capped at 6 MB and each
  `Query` page at 1 MB.
- **File and image URLs travel in the screen's response.** A screen that shows files (thumbnails,
  attachments, avatars) gets each file's URL in the list or detail it already fetched, as a
  same-origin `/_objects/...` URL the server signed in that one call. Never point `<img src>`,
  `<video>` or `<audio>` at `/api/...`, even a route that redirects: the browser fires one
  invocation per image, all at once, and the client's in-flight cap never sees them. Images carry
  `loading="lazy"`, `decoding="async"`, an `alt` and a width and height. On an image error or an
  expired URL the screen refetches its data once. Never store or log the URLs. The object-s3
  skill's "Showing files in a list or detail" has the handler and the web client.
- **The client is the second line.** It retries a GET on 429 or 503 with jittered backoff and caps
  requests in flight; neither replaces a screen that asks for the right thing once.

## Before and after

Before: the page loads the list, then one detail per row.

```ts
const rows = await get('/api/opportunities');
const details = await Promise.all(rows.items.map((r) => get(`/api/opportunities/${r.id}`)));
```

With 14 rows the list call has finished before the detail calls start, so 14 requests run
concurrently against a limit of 10.

After: the list already carries what the row renders, and the page makes one request.

```ts
const rows = await get('/api/opportunities'); // items: id, name, owner, stage, openTaskCount
```

In the handler, the owners come from one `BatchGetItem` over the distinct owner ids and the counts
from one `Query` per partition (or a maintained counter on the record), all inside the one request.
The cost moves from 15 invocations to one, with a few more read units.

## Deciding

If a screen shows data from more than one list, or a row needs a field the list lacks, add the
field to the list or add a screen endpoint. A new endpoint is cheaper than a page that makes N
calls. The plan names, per screen, how many requests it makes at the largest list it shows.

A walk of the app counts image and media requests to `/api/` against the same per-screen budget as
fetches, so the file pattern above is flagged if it slips in.

See `limits.md` for the numbers and `slow-supplier-work.md` for work that cannot finish inside a
request.
