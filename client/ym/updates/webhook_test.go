package updates

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rekurt/ymsdk/client/ym"
)

const batchPayload = `{"ok":true,"updates":[{"update_id":1,"text":"hi"}]}`

// collector records the updates a handler receives and signals when enough have
// arrived, so tests never rely on sleeping.
type collector struct {
	mu   sync.Mutex
	got  []ym.Update
	want int
	done chan struct{}
}

func newCollector(want int) *collector {
	return &collector{want: want, done: make(chan struct{})}
}

func (c *collector) handle(_ context.Context, u ym.Update) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.got = append(c.got, u)
	if len(c.got) == c.want {
		close(c.done)
	}

	return nil
}

func (c *collector) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.got)
}

func post(t *testing.T, h http.Handler, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	return rec
}

func TestWebhookAcknowledgesAndDispatches(t *testing.T) {
	c := newCollector(1)
	h := NewWebhookHandler(c.handle, WebhookOptions{})
	t.Cleanup(func() { _ = h.Shutdown(context.Background()) })

	rec := post(t, h, "/hook", batchPayload)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	select {
	case <-c.done:
	case <-time.After(2 * time.Second):
		t.Fatal("the update was never dispatched")
	}
	if c.got[0].Text != "hi" {
		t.Fatalf("unexpected update: %#v", c.got[0])
	}
}

// Delivery is at-least-once, so the same update_id can arrive repeatedly. It
// must reach the handler once.
func TestWebhookDeduplicatesRedeliveries(t *testing.T) {
	c := newCollector(1)
	h := NewWebhookHandler(c.handle, WebhookOptions{})
	t.Cleanup(func() { _ = h.Shutdown(context.Background()) })

	for range 5 {
		if rec := post(t, h, "/hook", batchPayload); rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	}

	select {
	case <-c.done:
	case <-time.After(2 * time.Second):
		t.Fatal("the update was never dispatched")
	}

	// Give any duplicate a chance to slip through before asserting.
	if err := h.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if got := c.count(); got != 1 {
		t.Fatalf("expected the repeat deliveries to be dropped, handler ran %d times", got)
	}
}

func TestWebhookSecret(t *testing.T) {
	c := newCollector(1)
	h := NewWebhookHandler(c.handle, WebhookOptions{Secret: "s3cret"})
	t.Cleanup(func() { _ = h.Shutdown(context.Background()) })

	t.Run("rejects a wrong secret with a final status", func(t *testing.T) {
		rec := post(t, h, "/hook?secret=wrong", batchPayload)
		// 4xx is final for the API: a bad secret must not be retried for 24h.
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("accepts the right secret", func(t *testing.T) {
		if rec := post(t, h, "/hook?secret=s3cret", batchPayload); rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

func TestWebhookRejectsBadRequests(t *testing.T) {
	h := NewWebhookHandler(func(context.Context, ym.Update) error { return nil }, WebhookOptions{})
	t.Cleanup(func() { _ = h.Shutdown(context.Background()) })

	t.Run("non-POST", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/hook", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})

	t.Run("unparsable body is final, not retried", func(t *testing.T) {
		rec := post(t, h, "/hook", `{{{`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

// A full queue must ask for redelivery rather than silently dropping updates.
func TestWebhookAsksForRedeliveryWhenSaturated(t *testing.T) {
	block := make(chan struct{})
	var started atomic.Int64

	h := NewWebhookHandler(func(context.Context, ym.Update) error {
		started.Add(1)
		<-block

		return nil
	}, WebhookOptions{Workers: 1, Queue: 1})
	t.Cleanup(func() {
		close(block)
		_ = h.Shutdown(context.Background())
	})

	var got503 bool
	for i := range 20 {
		body := `{"ok":true,"updates":[{"update_id":` + string(rune('1'+i%9)) + `00}]}`
		if rec := post(t, h, "/hook", body); rec.Code == http.StatusServiceUnavailable {
			got503 = true

			break
		}
	}
	if !got503 {
		t.Fatal("a saturated handler must return 503 so the API redelivers")
	}
}

func TestParseWebhookBody(t *testing.T) {
	t.Run("batch shape", func(t *testing.T) {
		got, err := ParseWebhookBody([]byte(batchPayload))
		if err != nil || len(got) != 1 || got[0].UpdateID != 1 {
			t.Fatalf("got %#v, err %v", got, err)
		}
	})

	t.Run("bare update object", func(t *testing.T) {
		got, err := ParseWebhookBody([]byte(`{"update_id":7,"text":"solo"}`))
		if err != nil || len(got) != 1 || got[0].UpdateID != 7 {
			t.Fatalf("got %#v, err %v", got, err)
		}
	})

	t.Run("empty batch", func(t *testing.T) {
		got, err := ParseWebhookBody([]byte(`{"ok":true,"updates":[]}`))
		if err != nil || len(got) != 0 {
			t.Fatalf("got %#v, err %v", got, err)
		}
	})

	t.Run("garbage", func(t *testing.T) {
		if _, err := ParseWebhookBody([]byte(`not json`)); err == nil {
			t.Fatal("expected a decode error")
		}
	})
}

func TestDedupeEviction(t *testing.T) {
	// The window is never narrower than one delivery, so eviction is exercised
	// at that size rather than at an arbitrary small one.
	d := newDedupe(1)
	take := func() bool { return true }

	for id := int64(1); id <= ym.MaxPageLimit; id++ {
		if d.admit(id, take) != admissionNew {
			t.Fatalf("id %d should have been accepted", id)
		}
	}
	if d.admit(1, take) != admissionDuplicate {
		t.Fatal("a remembered id must be reported as a duplicate")
	}

	// One more id past the window pushes the oldest out.
	if d.admit(ym.MaxPageLimit+1, take) != admissionNew {
		t.Fatal("a fresh id should have been accepted")
	}
	if d.admit(1, take) != admissionNew {
		t.Fatal("expected the oldest id to have been evicted")
	}
}

func TestDedupeCanBeDisabled(t *testing.T) {
	d := newDedupe(-1)
	take := func() bool { return true }

	if d.admit(1, take) != admissionNew || d.admit(1, take) != admissionNew {
		t.Fatal("a disabled window must accept every delivery")
	}
	if d.admit(1, func() bool { return false }) != admissionRefused {
		t.Fatal("a refused delivery must be reported even with the window disabled")
	}
}

// A batch whose updates will not decode must be reported. Falling through to
// the single-update path made the envelope itself parse as an empty update, so
// ServeHTTP answered 200, OnError never fired, and every update in the batch
// was dropped without a trace — the exact silent loss this handler exists to
// prevent.
func TestParseWebhookBodyReportsBatchDecodeFailures(t *testing.T) {
	const malformed = `{"ok":true,"updates":[{"update_id":1,"text":12345}]}`

	got, err := ParseWebhookBody([]byte(malformed))
	if err == nil {
		t.Fatalf("expected a decode error, got %d updates", len(got))
	}
	if got != nil {
		t.Fatalf("expected no updates alongside the error, got %#v", got)
	}
}

func TestWebhookSurfacesBatchDecodeFailures(t *testing.T) {
	var reported atomic.Int64
	h := NewWebhookHandler(func(context.Context, ym.Update) error { return nil },
		WebhookOptions{OnError: func(error) { reported.Add(1) }})
	t.Cleanup(func() { _ = h.Shutdown(context.Background()) })

	rec := post(t, h, "/hook", `{"ok":true,"updates":[{"update_id":1,"text":12345}]}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if reported.Load() == 0 {
		t.Fatal("a batch that would not decode was acknowledged without reporting")
	}
}

// The invariant: no delivery may be told "duplicate" unless a copy is actually
// queued. Recording the id before the enqueue breaks it — a concurrent delivery
// of the same update sees the record and is answered 200, which is final for the
// API, while the first copy then fails to enqueue and rolls the record back.
// Nothing processed the update and the API has its success.
//
// With every acceptance refused, no copy is ever queued, so no caller may hear
// "duplicate".
func TestAdmitNeverCallsADeliveryDuplicateBeforeOneIsQueued(t *testing.T) {
	// Repeated because a single round can miss the window by luck; the flaw is
	// hit in most rounds, but a test that passes half the time would report the
	// defect as absent.
	for round := range 50 {
		d := newDedupe(64)

		const callers = 64

		var wg sync.WaitGroup
		results := make([]admission, callers)

		for i := range callers {
			wg.Add(1)
			go func(slot int) {
				defer wg.Done()
				results[slot] = d.admit(7, func() bool { return false })
			}(i)
		}
		wg.Wait()

		for i, got := range results {
			if got == admissionDuplicate {
				t.Fatalf("round %d, caller %d was told 'duplicate' although nothing was ever queued", round, i)
			}
			if got != admissionRefused {
				t.Fatalf("round %d, caller %d: expected refused, got %v", round, i, got)
			}
		}
	}
}

// Once a copy is queued, later deliveries of the same update are duplicates.
func TestAdmitReportsDuplicatesOnceQueued(t *testing.T) {
	d := newDedupe(64)

	if got := d.admit(7, func() bool { return true }); got != admissionNew {
		t.Fatalf("expected the first delivery to be accepted, got %v", got)
	}
	if got := d.admit(7, func() bool { return true }); got != admissionDuplicate {
		t.Fatalf("expected a repeat to be a duplicate, got %v", got)
	}
}

// A refused delivery leaves no trace, so the redelivery it asks for is processed.
func TestAdmitLeavesNoTraceWhenRefused(t *testing.T) {
	d := newDedupe(64)

	if got := d.admit(7, func() bool { return false }); got != admissionRefused {
		t.Fatalf("expected refused, got %v", got)
	}
	if got := d.admit(7, func() bool { return true }); got != admissionNew {
		t.Fatalf("the refused delivery was remembered; got %v", got)
	}
}

// The API allows one second for a reply. A caller's OnError may write to a
// remote log, so running it before the status is on the wire lets a slow
// callback consume that budget: the API then sees a timeout instead of the
// final 400/403 or the retryable 503, and redelivers what should have been
// settled.
func TestWebhookAnswersBeforeReportingErrors(t *testing.T) {
	var rec *httptest.ResponseRecorder
	seenByCallback := make(chan int, 4)

	h := NewWebhookHandler(func(context.Context, ym.Update) error { return nil },
		WebhookOptions{
			Secret:  "s3cret",
			OnError: func(error) { seenByCallback <- rec.Code },
		})
	t.Cleanup(func() { _ = h.Shutdown(context.Background()) })

	cases := []struct {
		name   string
		target string
		body   string
		want   int
	}{
		{"bad secret", "/hook?secret=wrong", batchPayload, http.StatusForbidden},
		{"unparsable body", "/hook?secret=s3cret", `{{{`, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.target, strings.NewReader(tc.body))
			rec = httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, rec.Code)
			}

			select {
			case got := <-seenByCallback:
				if got != tc.want {
					t.Fatalf("OnError ran before the response was written: it saw %d, want %d", got, tc.want)
				}
			default:
				t.Fatal("OnError was not called")
			}
		})
	}
}

// Deduplication keys on UpdateID alone, so several entries without one would
// collapse into a single delivery and the rest would vanish behind a 200. The
// bare-update path already discards a zero id; a batch carrying one is
// malformed and has to say so.
func TestParseWebhookBodyRejectsZeroUpdateIDsInABatch(t *testing.T) {
	cases := []string{
		`{"ok":true,"updates":[{"text":"no id"}]}`,
		`{"ok":true,"updates":[{"update_id":1,"text":"ok"},{"update_id":0,"text":"missing"}]}`,
	}

	for _, body := range cases {
		got, err := ParseWebhookBody([]byte(body))
		if err == nil {
			t.Fatalf("expected an error for %s, got %d updates", body, len(got))
		}
		if got != nil {
			t.Fatalf("expected no updates alongside the error, got %#v", got)
		}
	}
}

// A window smaller than a delivery evicts ids from the very batch being
// admitted. If that batch then hits a full queue and answers 503, the API
// redelivers all of it and the evicted prefix — already processed — is admitted
// a second time. The window has to be able to hold a whole delivery.
func TestDedupeWindowCoversAFullDelivery(t *testing.T) {
	d := newDedupe(10)
	take := func() bool { return true }

	// The API caps a batch at the documented page limit, so that many ids must
	// survive together.
	for id := int64(1); id <= ym.MaxPageLimit; id++ {
		if got := d.admit(id, take); got != admissionNew {
			t.Fatalf("id %d: expected it to be accepted, got %v", id, got)
		}
	}

	if got := d.admit(1, take); got != admissionDuplicate {
		t.Fatalf("the first id of the batch was evicted before the batch ended, got %v", got)
	}
}

func TestDedupeCanStillBeDisabled(t *testing.T) {
	d := newDedupe(-1)
	take := func() bool { return true }

	if d.admit(1, take) != admissionNew || d.admit(1, take) != admissionNew {
		t.Fatal("a disabled window must accept every delivery")
	}
}

// A body past the cap used to be truncated silently, and the unparsable
// remainder answered 400 — final for the API, so the whole delivery was lost.
// An oversized body has to be refused in a way that asks for redelivery.
func TestWebhookAsksForRedeliveryWhenTheBodyIsTooLarge(t *testing.T) {
	var reported atomic.Int64
	h := NewWebhookHandler(func(context.Context, ym.Update) error { return nil },
		WebhookOptions{
			MaxBodyBytes: 64,
			OnError:      func(error) { reported.Add(1) },
		})
	t.Cleanup(func() { _ = h.Shutdown(context.Background()) })

	oversized := `{"ok":true,"updates":[{"update_id":1,"text":"` + strings.Repeat("a", 200) + `"}]}`

	rec := post(t, h, "/hook", oversized)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 so the API redelivers, got %d", rec.Code)
	}
	if reported.Load() == 0 {
		t.Fatal("an oversized body was refused without reporting")
	}
}

// A delivery within the cap is processed as usual.
func TestWebhookAcceptsABodyWithinTheCap(t *testing.T) {
	c := newCollector(1)
	h := NewWebhookHandler(c.handle, WebhookOptions{MaxBodyBytes: 1 << 20})
	t.Cleanup(func() { _ = h.Shutdown(context.Background()) })

	if rec := post(t, h, "/hook", batchPayload); rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	select {
	case <-c.done:
	case <-time.After(2 * time.Second):
		t.Fatal("the update was never dispatched")
	}
}
