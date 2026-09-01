package updates

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/rekurt/ymsdk/client/ym"
)

const (
	defaultWebhookWorkers = 8
	defaultWebhookQueue   = 256
	defaultDedupeWindow   = 4096
	maxWebhookBody        = 8 << 20 // 8 MiB
)

// WebhookOptions configures [NewWebhookHandler].
type WebhookOptions struct {
	// Secret, when set, is compared against the "secret" query parameter and
	// the request is rejected unless it matches.
	//
	// The API signs nothing and sends no custom headers, so the only thing that
	// authenticates a delivery is the webhook URL being unguessable. Treat the
	// whole URL — path and query — as the credential.
	Secret string

	// Workers bounds how many updates are processed concurrently. Default 8.
	Workers int
	// Queue bounds how many accepted updates may wait for a worker. Default 256.
	// When the queue is full the handler replies 503 so the API retries.
	Queue int
	// DedupeWindow is how many recent update IDs are remembered. Default 4096.
	// Set to a negative value to disable deduplication.
	DedupeWindow int

	// OnError reports decoding failures, rejected requests and handler errors.
	//
	// For request-path errors it runs after the response has been written and
	// flushed, so a slow callback cannot spend the API's one-second budget.
	// It still holds the serving goroutine, so keep it brief or hand the work
	// to something of your own.
	OnError func(error)
}

// WebhookHandler serves update deliveries pushed by the API.
//
// It is built around three facts from the API's delivery contract:
//
//   - The API gives a webhook 100ms to connect and 1s to respond, and treats a
//     timeout as failure. The handler therefore acknowledges immediately and
//     runs the caller's Handler on a worker pool, rather than inline.
//   - Delivery is at-least-once, so the same update can arrive more than once.
//     Recently seen update IDs are remembered and repeats are dropped.
//   - 2xx and 4xx replies are final while 5xx is retried for up to 24 hours.
//     The handler answers 200 once an update is accepted and only returns 5xx
//     when it genuinely could not take it.
//
// Call [WebhookHandler.Shutdown] to drain in-flight updates before exiting.
type WebhookHandler struct {
	handler Handler
	opts    WebhookOptions

	queue chan queuedUpdate
	wg    sync.WaitGroup
	seen  *dedupe

	// mu guards closed together with the send on queue. Shutdown takes the
	// write lock, so it cannot close the channel while a delivery is being
	// enqueued — otherwise a request in flight panics with "send on closed
	// channel" and takes the process down.
	mu     sync.RWMutex
	closed bool
}

type queuedUpdate struct {
	update ym.Update
}

// NewWebhookHandler starts a worker pool and returns a handler ready to mount
// on an HTTP server.
func NewWebhookHandler(handler Handler, opts WebhookOptions) *WebhookHandler {
	if opts.Workers <= 0 {
		opts.Workers = defaultWebhookWorkers
	}
	if opts.Queue <= 0 {
		opts.Queue = defaultWebhookQueue
	}

	window := opts.DedupeWindow
	if window == 0 {
		window = defaultDedupeWindow
	}

	h := &WebhookHandler{
		handler: handler,
		opts:    opts,
		queue:   make(chan queuedUpdate, opts.Queue),
		seen:    newDedupe(window),
	}

	h.wg.Add(opts.Workers)
	for range opts.Workers {
		go h.worker()
	}

	return h
}

func (h *WebhookHandler) worker() {
	defer h.wg.Done()

	for item := range h.queue {
		if err := h.handler(context.Background(), item.update); err != nil {
			h.reportError(err)
		}
	}
}

// ServeHTTP accepts a delivery, acknowledges it, and queues its updates.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}
	if !h.authorised(r) {
		// 4xx is final for the API, so a wrong secret is not retried.
		h.respond(w, http.StatusForbidden, "forbidden",
			errors.New("yandex-messenger/webhook: rejected a delivery with a bad secret"))

		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody))
	if err != nil {
		h.respond(w, http.StatusBadRequest, "cannot read body", err)

		return
	}

	updates, err := ParseWebhookBody(body)
	if err != nil {
		// A body we cannot parse will not parse on a retry either.
		h.respond(w, http.StatusBadRequest, "cannot parse body", err)

		return
	}

	for _, u := range updates {
		if h.seen.admit(u.UpdateID, func() bool { return h.enqueue(u) }) != admissionRefused {
			continue
		}

		// Nothing was recorded, so the redelivery this 503 asks for will be
		// processed rather than mistaken for a duplicate.
		h.respond(w, http.StatusServiceUnavailable, "unavailable",
			errors.New("yandex-messenger/webhook: cannot accept the delivery, asking for redelivery"))

		return
	}

	w.WriteHeader(http.StatusOK)
}

// Shutdown stops accepting updates and waits for in-flight ones to finish, or
// for ctx to be cancelled.
func (h *WebhookHandler) Shutdown(ctx context.Context) error {
	h.mu.Lock()
	if !h.closed {
		h.closed = true
		close(h.queue)
	}
	h.mu.Unlock()

	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// enqueue hands u to a worker, reporting false when the handler is shutting
// down or its queue is saturated. Both cases refuse the delivery so the API
// sends it again rather than the update being silently dropped.
func (h *WebhookHandler) enqueue(u ym.Update) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.closed {
		return false
	}

	select {
	case h.queue <- queuedUpdate{update: u}:
		return true
	default:
		return false
	}
}

func (h *WebhookHandler) authorised(r *http.Request) bool {
	if h.opts.Secret == "" {
		return true
	}
	got := r.URL.Query().Get("secret")

	return subtle.ConstantTimeCompare([]byte(got), []byte(h.opts.Secret)) == 1
}

// respond writes the status, flushes it, and only then reports err.
//
// The API allows one second for a reply and a caller's OnError may do blocking
// I/O, so reporting first would spend that budget on the callback: the API
// would see a timeout rather than the final 4xx or the retryable 503, and
// redeliver something already settled. Flushing puts the answer on the wire
// before the callback can hold the goroutine.
func (h *WebhookHandler) respond(w http.ResponseWriter, status int, message string, err error) {
	http.Error(w, message, status)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	h.reportError(err)
}

func (h *WebhookHandler) reportError(err error) {
	if h.opts.OnError != nil {
		h.opts.OnError(err)
	}
}

// ParseWebhookBody decodes a webhook delivery.
//
// The documented body matches a getUpdates response, but a bare update object
// is accepted too so that a delivery format change does not drop messages.
func ParseWebhookBody(body []byte) ([]ym.Update, error) {
	// Decide the shape before decoding the contents. Falling back to the
	// single-update path on a batch failure used to make the envelope itself
	// parse as an empty update, so the delivery was acknowledged and every
	// update in it disappeared without reaching OnError.
	var envelope struct {
		OK      bool             `json:"ok"`
		Updates *json.RawMessage `json:"updates"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Updates != nil {
		var updates []ym.Update
		if err := json.Unmarshal(*envelope.Updates, &updates); err != nil {
			return nil, fmt.Errorf("yandex-messenger/webhook: decode delivery batch: %w", err)
		}

		return updates, nil
	}

	var single ym.Update
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, fmt.Errorf("yandex-messenger/webhook: decode delivery: %w", err)
	}
	if single.UpdateID == 0 {
		return nil, nil
	}

	return []ym.Update{single}, nil
}

// dedupe remembers the most recent update IDs in a fixed-size ring so that
// at-least-once delivery does not turn into at-least-once processing.
type dedupe struct {
	mu      sync.Mutex
	seen    map[int64]struct{}
	ring    []int64
	pos     int
	enabled bool
}

func newDedupe(window int) *dedupe {
	if window < 0 {
		return &dedupe{enabled: false}
	}

	return &dedupe{
		seen:    make(map[int64]struct{}, window),
		ring:    make([]int64, window),
		enabled: true,
	}
}

// admission is the outcome of offering one update to the handler.
type admission int

const (
	// admissionNew means the update was accepted and recorded.
	admissionNew admission = iota
	// admissionDuplicate means the update had already been recorded.
	admissionDuplicate
	// admissionRefused means the handler could not take the update.
	admissionRefused
)

// admit offers id to accept and records it only once accept has taken it.
//
// The whole decision happens under one lock. Recording first and rolling back on
// failure left a window where a concurrent delivery of the same update saw the
// record, was answered 200 — final, as far as the API is concerned — and then
// the first copy failed to enqueue and undid the record. Nothing had processed
// the update, and the API had its success.
//
// accept must not block: it is called with the lock held.
func (d *dedupe) admit(id int64, accept func() bool) admission {
	if !d.enabled {
		if accept() {
			return admissionNew
		}

		return admissionRefused
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if _, dup := d.seen[id]; dup {
		return admissionDuplicate
	}
	if !accept() {
		return admissionRefused
	}
	d.record(id)

	return admissionNew
}

// record remembers id, evicting the oldest entry. The caller holds mu.
func (d *dedupe) record(id int64) {
	if evicted := d.ring[d.pos]; evicted != 0 {
		delete(d.seen, evicted)
	}
	d.ring[d.pos] = id
	d.pos = (d.pos + 1) % len(d.ring)
	d.seen[id] = struct{}{}
}
