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
- **The client is the second line.** It retries a GET on 429 or 503 with jittered backoff and caps
  requests in flight; neither replaces a screen that asks for the right thing once.

## Before and after

Before: the page loads the list, then one detail per row.

```ts
const rows = await get('/api/opportunities');
const details = await Promise.all(rows.items.map((r) => get(`/api/opportunities/${r.id}`)));
```

With 14 rows that is 15 requests at once against a limit of 10.

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

See `limits.md` for the numbers and `slow-supplier-work.md` for work that cannot finish inside a
request.
