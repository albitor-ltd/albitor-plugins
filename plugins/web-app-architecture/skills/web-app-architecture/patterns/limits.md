# limits: numbers a design must fit

Defaults at 2026-11-03, each cited. Quotas change and some are adjustable; the platform may set
tighter ones. Check the deploy target's own skill before relying on a number in code.

| Limit | Value | Source |
| --- | --- | --- |
| Lambda synchronous request payload | 6 MB | [Lambda quotas][lambda] |
| Lambda synchronous response payload | 6 MB | [Lambda quotas][lambda] |
| Lambda asynchronous payload | 1 MB | [Lambda quotas][lambda] |
| Lambda function timeout | 30 s on the hosted target (the `gateway-services` skill's step budget); 900 s is the AWS maximum | [Lambda quotas][lambda] |
| API Gateway HTTP API integration timeout | 30 s, not adjustable | [HTTP API quotas][http] |
| API Gateway REST API integration timeout | 29 s default | [REST API quotas][rest] |
| API Gateway payload | 10 MB (so Lambda's 6 MB bites first) | [HTTP][http], [REST][rest] |
| DynamoDB item size | 400 KB, attribute names included | [DynamoDB constraints][ddb] |
| DynamoDB write | 1 write unit per 1 KB, rounded up; transactional writes cost 2x | [Read and write operations][rw] |
| DynamoDB Query or Scan page | 1 MB per call | [DynamoDB constraints][ddb] |
| DynamoDB BatchWriteItem | 25 items | [DynamoDB constraints][ddb] |
| Hosted queue message | 256 KiB; queue a reference, never the payload | `api-go` skill |
| SQS message size (AWS maximum) | 1,048,576 bytes (1 MiB) | [SQS quotas][sqs] |
| SQS delivery delay (message timer) | 0 s to 15 min | [SQS quotas][sqs] |
| Translate TranslateText input | 10,000 bytes UTF-8 per request | [Translate quotas][tr] |
| Textract synchronous | 10 MB, PDF and TIFF limited to 1 page | [Textract limits][tx] |
| Textract asynchronous | 500 MB and 3,000 pages for PDF and TIFF; 10 MB for images | [Textract limits][tx] |
| Hosted gateway service-call body | 5,500,000 bytes inline, larger by upload handle | `gateway-services` skill |
| Hosted gateway daily credits | a per-app daily cap, owned by the platform; the number is not a design input | `gateway-services` skill |

## What the numbers mean for a design

- **Request size.** Nothing over about 4 MB moves through the API; see `file-storage.md`. The
  Base64 inflates a file by about a third, so a file over about 4.5 MB will not fit in 6 MB once encoded; the hosted gateway's 5,500,000-byte body ceiling is the tighter figure for service calls.
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
- Reading a limit from memory into a constant: link or look it up.

## How a walk verifies it

Add one journey step at a realistic upper size (not a toy): the largest file the plan accepts, a
list at the largest page the UI shows, a long text through the translate path. Assert the feature
works or refuses with a clear message, not a 413, a 504 or a blank screen.

[lambda]: https://docs.aws.amazon.com/lambda/latest/dg/gettingstarted-limits.html
[http]: https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api-quotas.html
[rest]: https://docs.aws.amazon.com/apigateway/latest/developerguide/api-gateway-execution-service-limits-table.html
[ddb]: https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Constraints.html
[rw]: https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/read-write-operations.html
[sqs]: https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/quotas-messages.html
[tr]: https://docs.aws.amazon.com/translate/latest/dg/what-is-limits.html
[tx]: https://docs.aws.amazon.com/textract/latest/dg/limits-document.html
