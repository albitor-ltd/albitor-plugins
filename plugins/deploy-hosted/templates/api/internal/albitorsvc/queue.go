package albitorsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// The queue half of the client: a send to the app's own queue is plain IAM on the
// function role, signed here for service "sqs" with the role's environment credentials; it does
// not go through the gateway, which is for metered AI services.

const (
	// QueueURLEnvPrefix and QueueURLEnvSuffix wrap the upper-cased queue name:
	// ALBITOR_QUEUE_TRANSCRIBE_URL. The platform sets them; the app never builds a URL.
	QueueURLEnvPrefix = "ALBITOR_QUEUE_"
	QueueURLEnvSuffix = "_URL"

	// MaxMessageBytes is SQS's ceiling for a message body. A larger envelope is refused before
	// the call: put a reference (an object key, a record id) on the queue, never the data.
	MaxMessageBytes = 256 << 10
	// MaxDelay is SQS's longest delivery delay.
	MaxDelay = 900 * time.Second
	// MaxAttempt is the most re-enqueues one chain of messages may make.
	MaxAttempt = 50
	// MaxHorizon is the longest a chain of messages may run from its first send.
	MaxHorizon = time.Hour
	// MaxBatch is SQS's most entries in one SendMessageBatch.
	MaxBatch = 10

	maxQueueResponse = 1 << 20
)

// PartialRetry says that a 2xx answer carrying {"batchItemFailures": [...]} retries only the
// listed messages. This is an assumption to confirm on the first
// deployed queue (the Lambda Web Adapter may not return the response body to Lambda). If it
// proves false, set this to false: WriteResult then answers 5xx for any partial failure, so the
// whole batch is retried, and no handler changes. Handlers must be idempotent either way.
const PartialRetry = true

var (
	// ErrQueueNotConfigured means no ALBITOR_QUEUE_<NAME>_URL is set for the queue: the manifest
	// does not declare it. Fix albitor.yaml; the send cannot work.
	ErrQueueNotConfigured = errors.New("albitorsvc: no URL for this queue: declare it under queues in albitor.yaml")
	// ErrAttemptsExceeded means a handler re-enqueued more than MaxAttempt times down one chain.
	ErrAttemptsExceeded = errors.New("albitorsvc: this message chain has been re-enqueued too many times")
	// ErrDeadlinePassed means the chain's deadline (set at its first send) has passed.
	ErrDeadlinePassed = errors.New("albitorsvc: this message chain's deadline has passed")
	// ErrMessageTooLarge means the envelope is over MaxMessageBytes.
	ErrMessageTooLarge = errors.New("albitorsvc: the message is over 256 KiB; queue a reference, not the data")
	// ErrNotQueueEvent is ParseEvent's answer to a body with no Records key, such as a schedule's.
	ErrNotQueueEvent = errors.New("albitorsvc: not a queue event")

	queueNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)
)

// QueueEnv is the environment variable holding a queue's URL.
func QueueEnv(queue string) string {
	return QueueURLEnvPrefix + strings.ToUpper(strings.ReplaceAll(queue, "-", "_")) + QueueURLEnvSuffix
}

// Envelope is the body of every message: the platform client writes it, ParseEvent hands it
// back as a Message.
type Envelope struct {
	Queue    string          `json:"queue"`
	Payload  json.RawMessage `json:"payload"`
	Attempt  int             `json:"attempt"`
	Deadline time.Time       `json:"deadline"`
}

// Message is one received queue message.
type Message struct {
	// ID is the SQS messageId: the identifier Failures and WriteResult report.
	ID string
	// Queue is the declared queue name, from the envelope.
	Queue string
	// Payload is the app's JSON, as passed to Send.
	Payload json.RawMessage
	// Attempt is 0 for a message sent from a request or a schedule and one more for each
	// re-enqueue down a chain.
	Attempt int
	// Deadline is when the chain stops being allowed to re-enqueue.
	Deadline time.Time
	// ReceiveCount is SQS's delivery count for this message (1 on the first delivery).
	ReceiveCount int
}

// Decode unmarshals the payload into v.
func (m Message) Decode(v any) error { return json.Unmarshal(m.Payload, v) }

// SendOptions adjusts one send. The zero value sends at once and, for a first send, allows the
// chain MaxHorizon.
type SendOptions struct {
	// Delay holds delivery back, 0 to MaxDelay. A consumer polling an async gateway job
	// re-enqueues itself with a Delay instead of looping.
	Delay time.Duration
	// Horizon bounds a first send's whole chain, at most MaxHorizon; zero means MaxHorizon.
	// Ignored when Parent is set: the chain keeps its deadline.
	Horizon time.Duration
	// Parent is the message the handler is handling, when it re-enqueues. The new message
	// carries Attempt+1 and the same Deadline, and Send refuses past either bound.
	Parent *Message
}

// Queues sends to the app's queues and parses queue events. Under ALBITOR_LOCAL=1 it is the
// in-process queue (see local.go); otherwise it signs SQS calls with the function's role.
type Queues struct {
	local *localQueue

	getenv   func(string) string
	region   string
	creds    Credentials
	http     *http.Client
	now      func() time.Time
	endpoint string // test seam: overrides the endpoint derived from the queue URL
}

// OpenQueues builds a Queues from the environment, read through getenv (os.Getenv in main).
// With ALBITOR_LOCAL=1 it returns the in-process queue, which refuses to start when
// AWS_LAMBDA_FUNCTION_NAME is set; call Register on it once the http.Handler exists.
func OpenQueues(getenv func(string) string) (*Queues, error) {
	env := func(k string) string { return strings.TrimSpace(getenv(k)) }
	if env("ALBITOR_LOCAL") == "1" {
		if env("AWS_LAMBDA_FUNCTION_NAME") != "" {
			return nil, errors.New("albitorsvc: ALBITOR_LOCAL=1 on a deployed function: refusing the in-process queue")
		}
		return &Queues{local: newLocalQueue()}, nil
	}
	region := env("AWS_REGION")
	creds := Credentials{
		AccessKeyID:     env("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: env("AWS_SECRET_ACCESS_KEY"),
		SessionToken:    env("AWS_SESSION_TOKEN"),
	}
	if region == "" || creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
		return nil, errors.New("albitorsvc: no Region or role credentials in the environment")
	}
	hc := &http.Client{Timeout: 20 * time.Second}
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Queues{getenv: env, region: region, creds: creds, http: hc, now: time.Now}, nil
}

// Register gives the in-process queue the app's http.Handler, which it calls with POST /events.
// It does nothing on a deployed Queues.
func (q *Queues) Register(h http.Handler) {
	if q.local != nil {
		q.local.setHandler(h)
	}
}

func (q *Queues) clock() time.Time {
	if q.local != nil || q.now == nil {
		return time.Now()
	}
	return q.now()
}

// Send puts payload (any value that marshals to JSON) on the named queue.
func (q *Queues) Send(ctx context.Context, queue string, payload any, opts SendOptions) error {
	env, err := q.envelope(queue, payload, opts)
	if err != nil {
		return err
	}
	return q.deliver(ctx, queue, []Envelope{env}, opts.Delay)
}

// SendBatch sends up to MaxBatch payloads to one queue in one call. All share opts. A partial
// failure is an error: the batch is not retried for you.
func (q *Queues) SendBatch(ctx context.Context, queue string, payloads []any, opts SendOptions) error {
	if len(payloads) == 0 {
		return nil
	}
	if len(payloads) > MaxBatch {
		return fmt.Errorf("albitorsvc: a batch holds at most %d messages", MaxBatch)
	}
	envs := make([]Envelope, 0, len(payloads))
	total := 0
	for _, p := range payloads {
		env, err := q.envelope(queue, p, opts)
		if err != nil {
			return err
		}
		b, _ := json.Marshal(env)
		if total += len(b); total > MaxMessageBytes {
			return ErrMessageTooLarge
		}
		envs = append(envs, env)
	}
	return q.deliver(ctx, queue, envs, opts.Delay)
}

func (q *Queues) envelope(queue string, payload any, opts SendOptions) (Envelope, error) {
	if !queueNamePattern.MatchString(queue) {
		return Envelope{}, fmt.Errorf("albitorsvc: %q is not a queue name", queue)
	}
	if opts.Delay < 0 || opts.Delay > MaxDelay {
		return Envelope{}, errors.New("albitorsvc: the delay must be 0 to 900 seconds")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, fmt.Errorf("albitorsvc: encode payload: %w", err)
	}
	now := q.clock()
	env := Envelope{Queue: queue, Payload: raw}
	if p := opts.Parent; p != nil {
		env.Attempt = p.Attempt + 1
		env.Deadline = p.Deadline
		if env.Attempt > MaxAttempt {
			return Envelope{}, ErrAttemptsExceeded
		}
		if !now.Before(env.Deadline) {
			return Envelope{}, ErrDeadlinePassed
		}
	} else {
		h := opts.Horizon
		if h < 0 || h > MaxHorizon {
			return Envelope{}, errors.New("albitorsvc: the horizon must be at most one hour")
		}
		if h == 0 {
			h = MaxHorizon
		}
		env.Deadline = now.Add(h)
	}
	env.Deadline = env.Deadline.UTC().Truncate(time.Second)
	if b, _ := json.Marshal(env); len(b) > MaxMessageBytes {
		return Envelope{}, ErrMessageTooLarge
	}
	return env, nil
}

func (q *Queues) deliver(ctx context.Context, queue string, envs []Envelope, delay time.Duration) error {
	if q.local != nil {
		for _, e := range envs {
			q.local.enqueue(e, delay)
		}
		return nil
	}
	queueURL := q.getenv(QueueEnv(queue))
	if queueURL == "" {
		return ErrQueueNotConfigured
	}
	u, err := url.Parse(queueURL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && !isLoopback(u)) {
		return errors.New("albitorsvc: the queue URL must be https")
	}
	endpoint := q.endpoint
	if endpoint == "" {
		endpoint = u.Scheme + "://" + u.Host + "/"
	}
	secs := int(delay / time.Second)
	if len(envs) == 1 {
		body, _ := json.Marshal(envs[0])
		req := map[string]any{"QueueUrl": queueURL, "MessageBody": string(body)}
		if secs > 0 {
			req["DelaySeconds"] = secs
		}
		return q.call(ctx, endpoint, "SendMessage", req, nil)
	}
	entries := make([]map[string]any, 0, len(envs))
	for i, e := range envs {
		body, _ := json.Marshal(e)
		entry := map[string]any{"Id": fmt.Sprint(i), "MessageBody": string(body)}
		if secs > 0 {
			entry["DelaySeconds"] = secs
		}
		entries = append(entries, entry)
	}
	var resp struct {
		Failed []struct{ Id, Code string } `json:"Failed"`
	}
	if err := q.call(ctx, endpoint, "SendMessageBatch", map[string]any{"QueueUrl": queueURL, "Entries": entries}, &resp); err != nil {
		return err
	}
	if len(resp.Failed) > 0 {
		return fmt.Errorf("albitorsvc: %d of %d messages were not accepted by the queue", len(resp.Failed), len(envs))
	}
	return nil
}

// call makes one SQS JSON-protocol request. Nothing is retried: a send may have landed, and the
// queue is at least once anyway, so the caller decides.
func (q *Queues) call(ctx context.Context, endpoint, action string, req, resp any) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	hr.Header.Set("Content-Type", "application/x-amz-json-1.0")
	hr.Header.Set("X-Amz-Target", "AmazonSQS."+action)
	signRequest(hr, body, q.creds, q.region, "sqs", q.now())
	res, err := q.http.Do(hr)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("albitorsvc: the queue request failed to send")
	}
	defer func() { _ = res.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, maxQueueResponse))
	if res.StatusCode/100 != 2 {
		// SQS names the failure in __type; its message can echo the queue URL, so log only the type.
		var e struct {
			Type string `json:"__type"`
		}
		_ = json.Unmarshal(raw, &e)
		if i := strings.LastIndex(e.Type, "#"); i >= 0 {
			e.Type = e.Type[i+1:]
		}
		if e.Type == "" {
			e.Type = "unknown"
		}
		return fmt.Errorf("albitorsvc: sqs %s: %d %s", action, res.StatusCode, e.Type)
	}
	if resp != nil {
		if err := json.Unmarshal(raw, resp); err != nil {
			return fmt.Errorf("albitorsvc: decode sqs response: %w", err)
		}
	}
	return nil
}

// Event is a parsed queue event.
type Event struct {
	Messages []Message
	// Malformed are the ids of records whose body is not an envelope. They can never succeed:
	// log them and leave them out of the failures, so SQS deletes them.
	Malformed []string
}

// ParseEvent parses the body of a POST /events that came from an SQS event source mapping. It
// returns ErrNotQueueEvent when the body has no Records key (a schedule's body, say), and an
// Event with no Messages for the deploy probe's {"Records": []}, which the handler answers 2xx.
func ParseEvent(body []byte) (Event, error) {
	var raw struct {
		Records *[]struct {
			MessageID  string            `json:"messageId"`
			Body       string            `json:"body"`
			Attributes map[string]string `json:"attributes"`
		} `json:"Records"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || raw.Records == nil {
		return Event{}, ErrNotQueueEvent
	}
	ev := Event{Messages: make([]Message, 0, len(*raw.Records)), Malformed: []string{}}
	for _, r := range *raw.Records {
		var env Envelope
		if json.Unmarshal([]byte(r.Body), &env) != nil || !queueNamePattern.MatchString(env.Queue) {
			ev.Malformed = append(ev.Malformed, r.MessageID)
			continue
		}
		count := 0
		_, _ = fmt.Sscan(r.Attributes["ApproximateReceiveCount"], &count)
		ev.Messages = append(ev.Messages, Message{
			ID: r.MessageID, Queue: env.Queue, Payload: env.Payload,
			Attempt: env.Attempt, Deadline: env.Deadline, ReceiveCount: count,
		})
	}
	return ev, nil
}

// Failures builds the response body that retries only the listed messages.
func Failures(ids ...string) []byte {
	type item struct {
		ItemIdentifier string `json:"itemIdentifier"`
	}
	out := struct {
		BatchItemFailures []item `json:"batchItemFailures"`
	}{BatchItemFailures: make([]item, 0, len(ids))}
	for _, id := range ids {
		out.BatchItemFailures = append(out.BatchItemFailures, item{id})
	}
	b, _ := json.Marshal(out)
	return b
}

// WriteResult answers a queue event: 200 when every message succeeded (the batch is deleted),
// and when some failed either 200 with Failures (only those are retried) or, if PartialRetry is
// false, 500 (the whole batch is retried). Pass the ids of the messages that failed.
func WriteResult(w http.ResponseWriter, failed []string) {
	w.Header().Set("Content-Type", "application/json")
	if len(failed) > 0 && !PartialRetry {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(Failures(failed...))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(Failures(failed...))
}
