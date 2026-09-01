package updates

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func batchWith(id int64) string {
	return fmt.Sprintf(`{"ok":true,"updates":[{"update_id":%d,"text":"u%d"}]}`, id, id)
}

// An update the handler refused to accept must stay redeliverable. Recording it
// as seen before the enqueue succeeds means the 503 asks for a redelivery that
// is then dropped as a duplicate — the update is acknowledged and lost.
func TestWebhookRedeliversAnUpdateItCouldNotAccept(t *testing.T) {
	release := make(chan struct{})

	var mu sync.Mutex
	var handled []int64

	h := NewWebhookHandler(func(_ context.Context, u ym.Update) error {
		<-release
		mu.Lock()
		defer mu.Unlock()
		handled = append(handled, u.UpdateID)

		return nil
	}, WebhookOptions{Workers: 1, Queue: 1})

	// Saturate the handler and note which update it turned away.
	var rejected int64
	for id := int64(1); id <= 50; id++ {
		if rec := post(t, h, "/hook", batchWith(id)); rec.Code == http.StatusServiceUnavailable {
			rejected = id

			break
		}
	}
	if rejected == 0 {
		t.Fatal("expected the queue to saturate")
	}

	close(release) // let the workers drain

	// The API keeps redelivering what it got a 503 for, so retry until the
	// queue has room again rather than racing the workers.
	deadline := time.Now().Add(3 * time.Second)
	accepted := false
	for time.Now().Before(deadline) {
		if rec := post(t, h, "/hook", batchWith(rejected)); rec.Code == http.StatusOK {
			accepted = true

			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !accepted {
		t.Fatal("the redelivery was never accepted")
	}
	if err := h.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	for _, id := range handled {
		if id == rejected {
			return
		}
	}
	t.Fatalf("update %d was refused, then dropped as a duplicate on redelivery; handled=%v", rejected, handled)
}

// ActionRetry promises another attempt at the same update. Treating it like
// ActionContinue advances past the failure, so the update is only seen again
// because the next poll happens to return it — and once the batch offset moves
// past it, it is gone. The poll count is what separates a real retry from a
// re-delivery: a genuine retry re-invokes the handler without polling again.
func TestRunRetriesTheHandlerOnActionRetry(t *testing.T) {
	responses := make([]*http.Response, 0, 8)
	for range 8 {
		responses = append(responses, testutil.NewResponse(http.StatusOK,
			`{"ok":true,"updates":[{"update_id":1,"text":"hi"}]}`))
	}
	doer := &testutil.FakeDoer{Responses: responses}
	svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

	handlerErr := errors.New("transient")
	var calls atomic.Int64

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := svc.Run(ctx, RunOptions{
		Interval:   time.Millisecond,
		MaxBackoff: time.Millisecond,
		OnHandlerError: func(ym.Update, error) ErrorAction {
			if calls.Load() >= 3 {
				return ActionStop
			}

			return ActionRetry
		},
	}, func(context.Context, ym.Update) error {
		calls.Add(1)

		return handlerErr
	})

	if !errors.Is(err, handlerErr) {
		t.Fatalf("expected the handler error once retries were given up, got %v", err)
	}
	if got := calls.Load(); got < 3 {
		t.Fatalf("handler ran %d time(s), want at least 3", got)
	}
	if polls := doer.CallCount(); polls != 1 {
		t.Fatalf("ActionRetry re-polled instead of retrying the update: %d polls for %d handler calls",
			polls, calls.Load())
	}
}

// Shutdown closing the queue while a request is mid-flight must not panic the
// serving goroutine with "send on closed channel".
func TestWebhookShutdownIsSafeDuringConcurrentDeliveries(t *testing.T) {
	h := NewWebhookHandler(func(context.Context, ym.Update) error { return nil },
		WebhookOptions{Workers: 2, Queue: 8})

	var panicked atomic.Bool
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			// ServeHTTP runs in this goroutine, so a send on a closed channel
			// surfaces here.
			defer func() {
				if r := recover(); r != nil {
					panicked.Store(true)
					t.Errorf("ServeHTTP panicked during shutdown: %v", r)
				}
			}()
			post(t, h, "/hook", batchWith(id))
		}(int64(i + 1))
	}

	go func() { _ = h.Shutdown(context.Background()) }()

	wg.Wait()

	if panicked.Load() {
		t.Fatal("Shutdown raced with in-flight deliveries")
	}
}
