# limits: numbers a design must fit

Defaults at 2026-11-08, each cited. Quotas change and some are adjustable; the platform may set
tighter ones. Check the deploy target's own skill before relying on a number in code.

| Limit | Value | Source | Mitigation |
| --- | --- | --- | --- |
| Lambda synchronous request payload | 6 MB | [Lambda quotas][lambda] | `file-storage.md` |
| Lambda synchronous response payload | 6 MB | [Lambda quotas][lambda] | `screen-data.md` (page the list; embed summaries, not bodies) |
| Lambda asynchronous payload | 1 MB | [Lambda quotas][lambda] | `slow-supplier-work.md` (queue a reference) |
| Lambda function timeout | 30 s on the hosted target (the `gateway-services` skill's step budget); 900 s is the AWS maximum | [Lambda quotas][lambda] | `slow-supplier-work.md` |
| API Gateway HTTP API integration timeout | 30 s, not adjustable | [HTTP API quotas][http] | `slow-supplier-work.md` |
| API Gateway REST API integration timeout | 29 s default | [REST API quotas][rest] | `slow-supplier-work.md` |
| API Gateway payload | 10 MB (so Lambda's 6 MB bites first) | [HTTP][http], [REST][rest] | `file-storage.md` |
| DynamoDB item size | 400 KB, attribute names included | [DynamoDB constraints][ddb] | one item per entry (below) |
| DynamoDB write | 1 write unit per 1 KB, rounded up; transactional writes cost 2x | [Read and write operations][rw] | split hot small fields (below) |
| DynamoDB Query or Scan page | 1 MB per call | [DynamoDB constraints][ddb] | `screen-data.md` (cursor paging) |
| DynamoDB BatchWriteItem | 25 items | [DynamoDB constraints][ddb] | chunk writes by 25 |
| Hosted queue message | 256 KiB; queue a reference, never the payload | `api-go` skill | queue a reference, `slow-supplier-work.md` |
| SQS message size (AWS maximum) | 1,048,576 bytes (1 MiB) | [SQS quotas][sqs] | `slow-supplier-work.md` |
| SQS delivery delay (message timer) | 0 s to 15 min | [SQS quotas][sqs] | `slow-supplier-work.md` |
| Translate TranslateText input | 10,000 bytes UTF-8 per request | [Translate quotas][tr] | chunk by bytes (below) |
| Textract synchronous | 10 MB, PDF and TIFF limited to 1 page | [Textract limits][tx] | `slow-supplier-work.md` |
| Textract asynchronous | 500 MB and 3,000 pages for PDF and TIFF; 10 MB for images | [Textract limits][tx] | `slow-supplier-work.md` |
| Hosted gateway service-call body | 5,500,000 bytes inline, larger by upload handle | `gateway-services` skill | `file-storage.md` |
| Hosted gateway daily credits | a per-app daily cap, owned by the platform; the number is not a design input | `gateway-services` skill | treat a refusal as "unavailable" (below) |
| Hosted HTTP function concurrency | 10 reserved; over it the HTTP API answers 503 `Service Unavailable` | hosted deploy module | `screen-data.md`: one call per screen; retry GET on 429 or 503 |
| Hosted HTTP function size | 1024 MB memory, 30 s timeout | hosted deploy module | `slow-supplier-work.md` |
| Hosted API request rate | 50 requests per second steady, burst 100; over it 429, and the request never reaches the function | hosted deploy module | `screen-data.md`: fewer, larger reads |
| Hosted schedules | at most 3, each `rate()` no faster than every 15 minutes; a failed run is retried up to 3 times within 1 h (maximum event age 1 h) | hosted deploy module | work that needs seconds is a queue, not a schedule; handlers are idempotent |
| Hosted queues | at most 2; `batch_size` 1 to 10; `max_receives` 1 to 5 | hosted deploy module | `slow-supplier-work.md`; idempotent handlers |
| Hosted queue timing | visibility timeout 180 s; retention 1 day; dead-letter queue retention 14 days | hosted deploy module | a batch finishes well inside 30 s per message |
| Hosted queue consumer concurrency | 5 invocations at once per queue | hosted deploy module | `slow-supplier-work.md`; do not assume a burst drains in parallel |
| Hosted DynamoDB | on-demand, so no throughput ceiling to tune; point-in-time recovery and deletion protection on | hosted deploy module | read cost still scales with items read: `screen-data.md` |
| Hosted log retention | 30 days | hosted deploy module | keep evidence a journey needs in the record, not in logs |

## What the numbers mean for a design

- **Request size.** Nothing over about 4 MB moves through the API; see `file-storage.md`. Base64
  inflates a file by about a third, so a file over about 4.5 MB will not fit in 6 MB once encoded;
  the hosted gateway's 5,500,000-byte body ceiling is the tighter figure for service calls.
- **Request time.** An API request is dead at 29 to 30 s. Plan for a few seconds at most; anything
  that calls a model or document service goes to `slow-supplier-work.md`. Check which API type the
  deploy target uses.
- **Record size.** 400 KB counts attribute names too. A list that grows without bound (comments,
  segments, history) is one item per entry, not one array on a parent item. A record carrying a
  full transcript or extracted text is a file (`file-storage.md`).
- **Write cost.** Every write is billed by the item's size rounded up to 1 KB, and updating one
  attribute still costs the full item. A 40 KB record that is edited often costs 40 units per edit;
  split hot small fields (a status, a counter) from cold large ones. Each global secondary index an
  item appears in is written too, billed against the index entry's size (keys plus projected
  attributes): project only what the query needs, and every extra index is a multiple on write
  cost. The index write cost is documented in the DynamoDB guide on global secondary indexes; read
  it before adding more than one.
- **Concurrency and rate.** The hosted function runs at most 10 requests at once and the API takes
  50 a second. A page that fires 14 detail reads at load overflows the first and answers 503 to
  the overflow, and it passes a walk whose fixtures hold three records. Count a screen's calls at
  the largest list the UI shows, not at fixture size: the answer is one call per screen
  (`screen-data.md`), with a client that retries a GET on 429 or 503 and caps requests in flight
  as the second line.
- **Schedules and queues.** A schedule cannot fire faster than every 15 minutes and at most 3
  exist, and a failed run is retried up to 3 times within 1 h (maximum event age 1 h); a queue
  consumer takes at most 5 batches at once and a message is dead-lettered after
  `max_receives` tries, so every scheduled and queue handler must be idempotent, and a queue
  handler must finish inside the 180 s visibility window.
- **Gateway caps.** The per-app daily credit cap switches a feature off for everyone when burnt;
  the app treats a cap refusal as "unavailable", not as an error to retry. Bound what one user
  action can spend. Input must be chunked to the supplier's byte limit (Translate: 10,000 bytes
  UTF-8, so count bytes, not characters); documents longer than one page need the asynchronous
  Textract path and therefore `slow-supplier-work.md`.

## Anti-patterns

- Designing the happy path at demo size and never naming the largest realistic input.
- Counting characters where the limit is bytes (Translate; non-Latin text is 2 to 4 bytes per
  character).
- An unbounded array on one record.
- A page that fetches the detail of every row it lists, or `Promise.all` over a list of ids: ten
  concurrent requests is the whole hosted budget (`screen-data.md`).
- Reading a limit from memory into a constant: link or look it up.

## How a walk verifies it

Add one journey step at a realistic upper size (not a toy): the largest file the plan accepts, a
list at the largest page the UI shows, a long text through the translate path, a list page of the
largest size with every row's detail rendered (count the requests it makes: one, not one per row).
Assert the feature works or refuses with a clear message, not a 413, a 504 or a blank screen.

[lambda]: https://docs.aws.amazon.com/lambda/latest/dg/gettingstarted-limits.html
[http]: https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api-quotas.html
[rest]: https://docs.aws.amazon.com/apigateway/latest/developerguide/api-gateway-execution-service-limits-table.html
[ddb]: https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Constraints.html
[rw]: https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/read-write-operations.html
[sqs]: https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/quotas-messages.html
[tr]: https://docs.aws.amazon.com/translate/latest/dg/what-is-limits.html
[tx]: https://docs.aws.amazon.com/textract/latest/dg/limits-document.html
