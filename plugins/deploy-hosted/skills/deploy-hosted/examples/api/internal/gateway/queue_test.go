package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)

func cloudQueues(t *testing.T, srv *httptest.Server) *Queues {
	t.Helper()
	env := map[string]string{QueueEnv("transcribe"): srv.URL + "/123/app--transcribe"}
	return &Queues{
		getenv: func(k string) string { return env[k] }, region: "eu-west-2",
		creds: Credentials{AccessKeyID: "AKID", SecretAccessKey: "s", SessionToken: "tok"},
		http:  srv.Client(), now: func() time.Time { return fixedNow },
	}
}

func TestQueueEnvUppercasesAndMapsDashes(t *testing.T) {
	if got := QueueEnv("my-queue"); got != "ALBITOR_QUEUE_MY_QUEUE_URL" {
		t.Errorf("QueueEnv = %q", got)
	}
}

func TestSendSignsAnSQSCallWithTheEnvelope(t *testing.T) {
	var target, auth, path string
	var req map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target, auth, path = r.Header.Get("X-Amz-Target"), r.Header.Get("Authorization"), r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &req)
		_, _ = w.Write([]byte(`{"MessageId":"m"}`))
	}))
	defer srv.Close()
	err := cloudQueues(t, srv).Send(context.Background(), "transcribe", map[string]string{"key": "a"},
		SendOptions{Delay: 30 * time.Second, Horizon: 10 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if target != "AmazonSQS.SendMessage" || path != "/" || !strings.Contains(auth, "/eu-west-2/sqs/aws4_request") {
		t.Errorf("target %q path %q auth %q", target, path, auth)
	}
	if req["DelaySeconds"] != float64(30) || !strings.HasSuffix(req["QueueUrl"].(string), "/123/app--transcribe") {
		t.Errorf("request = %v", req)
	}
	var env Envelope
	if err := json.Unmarshal([]byte(req["MessageBody"].(string)), &env); err != nil {
		t.Fatal(err)
	}
	if env.Queue != "transcribe" || env.Attempt != 0 || !env.Deadline.Equal(fixedNow.Add(10*time.Minute)) || string(env.Payload) != `{"key":"a"}` {
		t.Errorf("envelope = %+v", env)
	}
}

func TestSendFromAHandlerIncrementsAttemptAndKeepsTheDeadline(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &req)
		body = req["MessageBody"]
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	parent := Message{Queue: "transcribe", Attempt: 4, Deadline: fixedNow.Add(5 * time.Minute)}
	if err := cloudQueues(t, srv).Send(context.Background(), "transcribe", 1, SendOptions{Parent: &parent, Horizon: time.Minute}); err != nil {
		t.Fatal(err)
	}
	var env Envelope
	_ = json.Unmarshal([]byte(body), &env)
	if env.Attempt != 5 || !env.Deadline.Equal(parent.Deadline) {
		t.Errorf("envelope = %+v", env)
	}
}

func TestSendRefusesPastTheBounds(t *testing.T) {
	q := cloudQueues(t, httptest.NewServer(http.NotFoundHandler()))
	ctx := context.Background()
	cases := map[string]struct {
		opts SendOptions
		want error
	}{
		"attempts": {SendOptions{Parent: &Message{Attempt: MaxAttempt, Deadline: fixedNow.Add(time.Hour)}}, ErrAttemptsExceeded},
		"deadline": {SendOptions{Parent: &Message{Attempt: 1, Deadline: fixedNow}}, ErrDeadlinePassed},
	}
	for name, c := range cases {
		if err := q.Send(ctx, "transcribe", 1, c.opts); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", name, err, c.want)
		}
	}
	// The last permitted attempt still goes through the checks (and then reaches the 404 server).
	last := SendOptions{Parent: &Message{Attempt: MaxAttempt - 1, Deadline: fixedNow.Add(time.Hour)}}
	if err := q.Send(ctx, "transcribe", 1, last); errors.Is(err, ErrAttemptsExceeded) {
		t.Errorf("attempt %d was refused", MaxAttempt)
	}
	for _, bad := range []SendOptions{{Delay: MaxDelay + time.Second}, {Delay: -1}, {Horizon: 2 * time.Hour}} {
		if err := q.Send(ctx, "transcribe", 1, bad); err == nil {
			t.Errorf("%+v was accepted", bad)
		}
	}
	if err := q.Send(ctx, "transcribe", strings.Repeat("x", MaxMessageBytes), SendOptions{}); !errors.Is(err, ErrMessageTooLarge) {
		t.Errorf("oversize err = %v", err)
	}
	if err := q.Send(ctx, "other", 1, SendOptions{}); !errors.Is(err, ErrQueueNotConfigured) {
		t.Errorf("undeclared queue err = %v", err)
	}
}

func TestSendBatchReportsAPartialFailure(t *testing.T) {
	var entries int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Entries []any }
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &req)
		entries = len(req.Entries)
		_, _ = w.Write([]byte(`{"Successful":[],"Failed":[{"Id":"1","Code":"InternalError"}]}`))
	}))
	defer srv.Close()
	err := cloudQueues(t, srv).SendBatch(context.Background(), "transcribe", []any{1, 2}, SendOptions{})
	if err == nil || entries != 2 {
		t.Errorf("err = %v, entries = %d", err, entries)
	}
	if err := cloudQueues(t, srv).SendBatch(context.Background(), "transcribe", make([]any, MaxBatch+1), SendOptions{}); err == nil {
		t.Error("an 11 message batch was accepted")
	}
}

func TestSendErrorNamesTheSQSTypeNotTheMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"__type":"com.amazon.coral.service#AccessDeniedException","message":"secret-queue-url"}`))
	}))
	defer srv.Close()
	err := cloudQueues(t, srv).Send(context.Background(), "transcribe", 1, SendOptions{})
	if err == nil || !strings.Contains(err.Error(), "AccessDeniedException") || strings.Contains(err.Error(), "secret") {
		t.Errorf("err = %v", err)
	}
}

func TestParseEvent(t *testing.T) {
	body := `{"Records":[
	 {"messageId":"a","body":"{\"queue\":\"transcribe\",\"payload\":{\"k\":1},\"attempt\":2,\"deadline\":\"2026-11-02T12:30:00Z\"}","attributes":{"ApproximateReceiveCount":"2"}},
	 {"messageId":"b","body":"not json"}]}`
	ev, err := ParseEvent([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(ev.Messages) != 1 || len(ev.Malformed) != 1 || ev.Malformed[0] != "b" {
		t.Fatalf("event = %+v", ev)
	}
	m := ev.Messages[0]
	var p struct{ K int }
	if err := m.Decode(&p); err != nil {
		t.Fatal(err)
	}
	if m.ID != "a" || m.Queue != "transcribe" || m.Attempt != 2 || m.ReceiveCount != 2 || p.K != 1 || m.Deadline.Minute() != 30 {
		t.Errorf("message = %+v", m)
	}

	probe, err := ParseEvent([]byte(`{"Records": []}`))
	if err != nil || len(probe.Messages) != 0 {
		t.Errorf("empty batch: %+v, %v", probe, err)
	}
	if _, err := ParseEvent([]byte(`{"schedule":"nightly"}`)); !errors.Is(err, ErrNotQueueEvent) {
		t.Errorf("schedule body err = %v", err)
	}
}

func TestFailuresAndWriteResult(t *testing.T) {
	if got := string(Failures()); got != `{"batchItemFailures":[]}` {
		t.Errorf("Failures() = %s", got)
	}
	w := httptest.NewRecorder()
	WriteResult(w, []string{"a", "b"})
	if w.Code != 200 || w.Body.String() != `{"batchItemFailures":[{"itemIdentifier":"a"},{"itemIdentifier":"b"}]}` {
		t.Errorf("WriteResult = %d %s", w.Code, w.Body)
	}
}

func TestLocalQueueRunsTheSameRouteAndRetries(t *testing.T) {
	localRetryDelay = 0
	q, err := OpenQueues(func(k string) string {
		if k == "ALBITOR_LOCAL" {
			return "1"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var seen []Message
	done := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("POST /events", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		ev, err := ParseEvent(b)
		if err != nil {
			t.Error(err)
		}
		mu.Lock()
		seen = append(seen, ev.Messages...)
		n := len(seen)
		mu.Unlock()
		if n < 3 { // fail twice via batchItemFailures, succeed on the third delivery
			WriteResult(w, []string{ev.Messages[0].ID})
			return
		}
		close(done)
		WriteResult(w, nil)
	})
	q.Register(mux)
	// Sent before and after Register would both work; send after here.
	if err := q.Send(context.Background(), "transcribe", map[string]int{"n": 1}, SendOptions{}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("message never delivered a third time")
	}
	if seen[0].ReceiveCount != 1 || seen[2].ReceiveCount != 3 || seen[2].Queue != "transcribe" {
		t.Errorf("deliveries = %+v", seen)
	}
}

func TestLocalQueueRefusesOnALambda(t *testing.T) {
	_, err := OpenQueues(func(k string) string {
		return map[string]string{"ALBITOR_LOCAL": "1", "AWS_LAMBDA_FUNCTION_NAME": "f"}[k]
	})
	if err == nil {
		t.Error("the local queue started on a deployed function")
	}
}

func TestLocalQueueTreatsA4xxAsDeliveredLikeTheAdapter(t *testing.T) {
	l := newLocalQueue()
	l.setHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) }))
	if !l.deliverOnce(Envelope{Queue: "transcribe"}, "id", 1) {
		t.Error("a 4xx was treated as a failure; deployed it deletes the message")
	}
	l.handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) })
	if l.deliverOnce(Envelope{Queue: "transcribe"}, "id", 1) {
		t.Error("a 5xx was treated as delivered")
	}
}
