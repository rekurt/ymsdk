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
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
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

// A limit above the documented maximum is a 400 the API will never accept. With
// the default ActionRetry policy that turns into an endless hot loop, so it has
// to be rejected before the first poll.
func TestRunRejectsOutOfRangeLimit(t *testing.T) {
	for _, limit := range []int{0, -1, ym.MaxPageLimit + 1} {
		responses := make([]*http.Response, 0, 50)
		for range 50 {
			responses = append(responses,
				testutil.NewResponse(http.StatusBadRequest, `{"ok":false,"description":"bad limit"}`))
		}
		doer := &testutil.FakeDoer{Responses: responses}
		svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := svc.Run(ctx, RunOptions{Limit: ym.Ptr(limit), Interval: time.Millisecond},
			func(context.Context, ym.Update) error { return nil })
		cancel()

		var limitErr *ym.LimitError
		if !errors.As(err, &limitErr) {
			t.Fatalf("limit %d: expected a *ym.LimitError, got %T (%v)", limit, err, err)
		}
		if doer.CallCount() != 0 {
			t.Fatalf("limit %d: an impossible request must not be sent, got %d polls", limit, doer.CallCount())
		}
	}
}

func TestRunAcceptsValidLimit(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"updates":[]}`),
	}}
	svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = svc.Run(ctx, RunOptions{Limit: ym.Ptr(ym.MaxPageLimit), Interval: 50 * time.Millisecond},
		func(context.Context, ym.Update) error { return nil })

	if doer.CallCount() == 0 {
		t.Fatal("a valid limit must still poll")
	}
}

// Retrying by default rescued the bot from a transient 500, but it also made
// permanent failures invisible: a revoked token would loop at MaxBackoff
// forever and never reach the caller or a supervisor.
func TestRunStopsOnPermanentPollErrors(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"revoked token", http.StatusUnauthorized, `{"ok":false,"description":"unauthorized"}`, ymerrors.ErrUnauthorized},
		{"invalid token", http.StatusForbidden, `{"ok":false,"description":"forbidden"}`, ymerrors.ErrInvalidToken},
		{"malformed request", http.StatusBadRequest, `{"ok":false,"description":"bad request"}`, ymerrors.ErrBadRequest},
		{"missing resource", http.StatusNotFound, `{"ok":false,"description":"not found"}`, ymerrors.ErrNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			responses := make([]*http.Response, 0, 20)
			for range 20 {
				responses = append(responses, testutil.NewResponse(tc.status, tc.body))
			}
			doer := &testutil.FakeDoer{Responses: responses}
			svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := svc.Run(ctx, RunOptions{Interval: time.Millisecond, MaxBackoff: time.Millisecond},
				func(context.Context, ym.Update) error { return nil })

			if !errors.Is(err, tc.want) {
				t.Fatalf("expected the failure to be reported, got %v", err)
			}
			if polls := doer.CallCount(); polls != 1 {
				t.Fatalf("a permanent failure was retried %d times", polls)
			}
		})
	}
}

// Transient failures must still be retried — that is what the default is for.
func TestRunKeepsRetryingTransientPollErrors(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusTooManyRequests} {
		responses := []*http.Response{
			testutil.NewResponse(status, `{"ok":false}`),
			testutil.NewResponse(http.StatusOK, `{"ok":true,"updates":[{"update_id":1}]}`),
		}
		doer := &testutil.FakeDoer{Responses: responses}
		svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

		ctx, cancel := context.WithCancel(context.Background())
		var seen atomic.Bool

		err := svc.Run(ctx, RunOptions{Interval: time.Millisecond, MaxBackoff: time.Millisecond},
			func(context.Context, ym.Update) error {
				seen.Store(true)
				cancel()

				return nil
			})
		cancel()

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("status %d: expected the loop to survive and be cancelled, got %v", status, err)
		}
		if !seen.Load() {
			t.Fatalf("status %d: the loop gave up instead of retrying", status)
		}
	}
}

// MaxBackoff should bound every wait, including the first one. The initial
// back-off used to be a hardcoded second, so a caller tuning MaxBackoff down
// for a low-latency bot still paid a full second on the first retry.
func TestRunHonoursMaxBackoffOnTheFirstRetry(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusInternalServerError, `{"ok":false}`),
		testutil.NewResponse(http.StatusOK, `{"ok":true,"updates":[{"update_id":1}]}`),
	}}
	svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	err := svc.Run(ctx, RunOptions{Interval: time.Millisecond, MaxBackoff: 5 * time.Millisecond},
		func(context.Context, ym.Update) error {
			cancel()

			return nil
		})
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if elapsed > 300*time.Millisecond {
		t.Fatalf("the first retry ignored MaxBackoff: waited %v", elapsed)
	}
}

type plainTransportError struct{}

func (plainTransportError) Error() string { return "broken transport" }

// The documented default retries network trouble, rate limits and 5xx. Anything
// else — a proxy that keeps returning malformed JSON, or a custom transport
// failing for its own reasons — would otherwise be retried forever behind a
// promise that says the opposite.
func TestDefaultPollPolicyRetriesOnlyTransientFailures(t *testing.T) {
	t.Run("malformed body stops", func(t *testing.T) {
		responses := make([]*http.Response, 0, 20)
		for range 20 {
			responses = append(responses, testutil.NewResponse(http.StatusOK, `not json at all`))
		}
		doer := &testutil.FakeDoer{Responses: responses}
		svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := svc.Run(ctx, RunOptions{Interval: time.Millisecond, MaxBackoff: time.Millisecond},
			func(context.Context, ym.Update) error { return nil })

		if !errors.Is(err, ymerrors.ErrInvalidResponse) {
			t.Fatalf("expected the decode failure to surface, got %v", err)
		}
		if polls := doer.CallCount(); polls != 1 {
			t.Fatalf("a persistent decode failure was retried %d times", polls)
		}
	})

	t.Run("an unclassified transport failure stops", func(t *testing.T) {
		errs := make([]error, 0, 20)
		for range 20 {
			errs = append(errs, plainTransportError{})
		}
		doer := &testutil.FakeDoer{Errors: errs}
		svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := svc.Run(ctx, RunOptions{Interval: time.Millisecond, MaxBackoff: time.Millisecond},
			func(context.Context, ym.Update) error { return nil })

		if err == nil {
			t.Fatal("expected the transport failure to surface")
		}
		if polls := doer.CallCount(); polls != 1 {
			t.Fatalf("an unclassified failure was retried %d times", polls)
		}
	})

	// Real network failures do not carry ErrNetworkError — the client wraps the
	// raw error — so they have to be recognised by their net.Error behaviour.
	t.Run("a network failure still retries", func(t *testing.T) {
		doer := &testutil.FakeDoer{
			Errors:    []error{stubNetworkError{}, nil},
			Responses: []*http.Response{nil, testutil.NewResponse(http.StatusOK, `{"ok":true,"updates":[{"update_id":1}]}`)},
		}
		svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

		ctx, cancel := context.WithCancel(context.Background())
		var seen atomic.Bool

		err := svc.Run(ctx, RunOptions{Interval: time.Millisecond, MaxBackoff: time.Millisecond},
			func(context.Context, ym.Update) error {
				seen.Store(true)
				cancel()

				return nil
			})
		cancel()

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected the loop to survive the network error, got %v", err)
		}
		if !seen.Load() {
			t.Fatal("the loop gave up on a retryable network failure")
		}
	})
}

type stubNetworkError struct{}

func (stubNetworkError) Error() string   { return "dial tcp: connection refused" }
func (stubNetworkError) Timeout() bool   { return false }
func (stubNetworkError) Temporary() bool { return true }

// Get is the older raw entry point and bypassed the validation added to
// GetUpdates and Run, so the documented local check did not hold for callers
// still using it.
func TestGetValidatesLimit(t *testing.T) {
	for _, limit := range []int{-1, ym.MaxPageLimit + 1} {
		doer := &testutil.FakeDoer{Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":true,"updates":[]}`),
		}}
		svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

		_, _, err := svc.Get(context.Background(), limit, "")

		var limitErr *ym.LimitError
		if !errors.As(err, &limitErr) {
			t.Fatalf("limit %d: expected a *ym.LimitError, got %T (%v)", limit, err, err)
		}
		if doer.CallCount() != 0 {
			t.Fatalf("limit %d: invalid input must not reach the network", limit)
		}
	}
}

// Zero keeps meaning "unset" in this signature, since it takes a plain int.
func TestGetTreatsZeroLimitAsUnset(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"updates":[]}`),
	}}
	svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

	if _, _, err := svc.Get(context.Background(), 0, ""); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if q := doer.Requests[0].URL.Query(); q.Has("limit") {
		t.Fatalf("an unset limit must not be sent, got %q", q.Get("limit"))
	}
}
