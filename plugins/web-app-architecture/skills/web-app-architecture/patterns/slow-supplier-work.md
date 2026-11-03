# slow-supplier-work: a queue with delayed re-enqueue

## Shape

Only work that fits inside one request advances on the request. Anything longer is a job:

1. The route starts it (calls the supplier's `Start...` with an idempotency key, or enqueues the
   work), writes a record with `status: processing`, and returns at once with the record id.
2. A queue consumer advances it. For a supplier job (Transcribe, Textract) the consumer asks for
   the result; still running means it re-enqueues itself with a delay (back off, do not poll
   tightly); done means it stores the result and enqueues the next step; failed means it marks the
   record failed.
3. The UI polls a status route that only reads the record. Status polling never calls the supplier.

Messages carry a reference (record id, job id), never the audio, the document or the result.
Every step is idempotent: a queue message can arrive twice. A polling chain must be finite: the
platform bounds attempts and total age, and the consumer treats exceeding either as a failure to
record, not something to retry around.

Do not poll from a schedule: schedules fire too rarely for a user who is waiting.

For the code, read the `deploy-hosted` plugin's `gateway-services` skill (section "Async jobs: poll
by delayed re-enqueue", and its queue example) and the `api-go` skill's "Queues" section for the
`POST /events` consumer. Declare the queue under `queues` in `albitor.yaml`. Respect the queue and
function caps those skills state (message size, maximum delay, per-step time); do not restate or
raise them in the plan. Platform-level SQS limits are in the cited docs under `limits.md`.

## Limits

A step must fit the function timeout and, for steps on a request, the API timeout (see `limits.md`).
A supplier step that charges per call (each poll is a gateway call) costs credits: back off.

## Sketch

```text
POST /api/recordings/<id>/transcribe   -> Start job, status=processing, enqueue poll(+5s)   (202)
queue poll(job)                        -> running: re-enqueue(+10s, parent=this)
                                          done:    store result, enqueue summarise(id)
                                          failed / attempts exceeded: status=failed
GET  /api/recordings/<id>              -> reads status only
```

## Anti-patterns

- Holding the request open until the supplier finishes: it dies at the API timeout.
- The status route calling the supplier "to refresh": every viewer spends credits and hits rate
  limits.
- A consumer that catches the attempts or deadline error and sends again with no parent: an
  unbounded chain.
- Retrying a refused call (credits exhausted, forbidden): mark the job unavailable and stop.
- Putting the result into the queue message.

## How a walk verifies it

The journey starts the job, asserts the UI shows a processing state at once, then waits (with a
generous, explicit timeout) for the done state and asserts the result. It also asserts a failed job
shows a failed state, using a supplier fake or an unavailable-supplier case. A walk that passes
only because the work finished before the first assertion proves nothing: assert the processing
state first.
