package albitorsvc

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// The in-process queue for a local run. It runs the same POST /events route
// and the same handler as the deployed consumer, and nothing else: no visibility timeout, no
// DLQ, nothing kept across restarts. Do not grow it toward SQS.

const (
	// localReceives is how many times a message is delivered before it is logged and dropped,
	// where the cloud would dead-letter it.
	localReceives = 3
)

// localRetryDelay is the wait before a redelivery; tests set it to zero.
var localRetryDelay = time.Second

type localQueue struct {
	mu      sync.Mutex
	handler http.Handler
	ready   chan struct{}
}

func newLocalQueue() *localQueue { return &localQueue{ready: make(chan struct{})} }

func (l *localQueue) setHandler(h http.Handler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.handler == nil {
		l.handler = h
		close(l.ready)
	}
}

// enqueue delivers env on its own goroutine, one message per call, after delay. A message sent
// before Register waits for the handler: the goroutine blocks until Register is called.
func (l *localQueue) enqueue(env Envelope, delay time.Duration) {
	go func() {
		if delay > 0 {
			time.Sleep(delay)
		}
		<-l.ready
		id := localID()
		for receive := 1; receive <= localReceives; receive++ {
			if l.deliverOnce(env, id, receive) {
				return
			}
			if receive < localReceives {
				time.Sleep(localRetryDelay)
			}
		}
		log.Printf("albitorsvc: local queue %q dropped message %s after %d failed deliveries", env.Queue, id, localReceives)
	}()
}

type recorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (r *recorder) Header() http.Header         { return r.header }
func (r *recorder) Write(b []byte) (int, error) { return r.body.Write(b) }
func (r *recorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
}

// deliverOnce posts one SQS-shaped event and reports whether the message was accepted: a 2xx
// without this message in batchItemFailures.
func (l *localQueue) deliverOnce(env Envelope, id string, receive int) (ok bool) {
	body, _ := json.Marshal(env)
	event, _ := json.Marshal(map[string]any{"Records": []map[string]any{{
		"messageId":   id,
		"body":        string(body),
		"eventSource": "aws:sqs",
		"attributes":  map[string]string{"ApproximateReceiveCount": itoa(receive)},
	}}})
	req, err := http.NewRequest(http.MethodPost, "http://localhost/events", bytes.NewReader(event))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	rec := &recorder{header: http.Header{}}
	defer func() {
		if p := recover(); p != nil { // a deployed panic is a 5xx; so is this
			log.Printf("albitorsvc: local queue handler panicked: %v", p)
			ok = false
		}
	}()
	l.handler.ServeHTTP(rec, req)
	if rec.status == 0 {
		rec.status = http.StatusOK
	}
	if rec.status/100 == 5 { // the adapter treats only 5xx as an error; a 4xx would delete the message
		return false
	}
	var res struct {
		BatchItemFailures []struct {
			ItemIdentifier string `json:"itemIdentifier"`
		} `json:"batchItemFailures"`
	}
	if json.Unmarshal(rec.body.Bytes(), &res) == nil {
		for _, f := range res.BatchItemFailures {
			if f.ItemIdentifier == id {
				return false
			}
		}
	}
	return true
}

func localID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "local-" + hex.EncodeToString(b)
}

func itoa(n int) string { return strconv.Itoa(n) }
