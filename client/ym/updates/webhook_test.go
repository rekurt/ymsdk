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
	d := newDedupe(2)

	if !d.markSeen(1) || !d.markSeen(2) {
		t.Fatal("fresh ids must be reported as new")
	}
	if d.markSeen(1) {
		t.Fatal("a remembered id must be reported as a duplicate")
	}
	// The window holds two ids; adding a third evicts the oldest.
	if !d.markSeen(3) {
		t.Fatal("expected id 3 to be new")
	}
	if !d.markSeen(1) {
		t.Fatal("expected id 1 to have been evicted from the window")
	}
}

func TestDedupeCanBeDisabled(t *testing.T) {
	d := newDedupe(-1)
	if !d.markSeen(1) || !d.markSeen(1) {
		t.Fatal("a disabled window must report every id as new")
	}
}
