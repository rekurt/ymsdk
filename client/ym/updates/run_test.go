package updates

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func runService(responses []*http.Response, errs []error) *Service {
	doer := &testutil.FakeDoer{Responses: responses, Errors: errs}

	return NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))
}

func updatesResponse(body string) *http.Response {
	return testutil.NewResponse(http.StatusOK, body)
}

// A single 500 used to end the loop and take the bot down with it.
func TestRunSurvivesTransientPollFailure(t *testing.T) {
	svc := runService([]*http.Response{
		testutil.NewResponse(http.StatusInternalServerError, `{"ok":false}`),
		updatesResponse(`{"ok":true,"updates":[{"update_id":1,"text":"hi"}]}`),
	}, nil)

	var seen atomic.Int64
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := svc.Run(ctx, RunOptions{
		Interval:   time.Millisecond,
		MaxBackoff: time.Millisecond,
	}, func(_ context.Context, u ym.Update) error {
		if seen.Add(1) == 1 && u.UpdateID != 1 {
			t.Errorf("unexpected update %d", u.UpdateID)
		}
		cancel()

		return nil
	})

	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected the loop to end on cancellation, got %v", err)
	}
	if seen.Load() == 0 {
		t.Fatal("the loop gave up after the 500 instead of retrying")
	}
}

func TestRunPollErrorPolicy(t *testing.T) {
	t.Run("ActionStop surfaces the error", func(t *testing.T) {
		svc := runService([]*http.Response{
			testutil.NewResponse(http.StatusInternalServerError, `{"ok":false}`),
		}, nil)

		err := svc.Run(context.Background(), RunOptions{
			Interval:    time.Millisecond,
			OnPollError: func(error) ErrorAction { return ActionStop },
		}, func(context.Context, ym.Update) error { return nil })

		if err == nil {
			t.Fatal("expected the poll error to be returned")
		}
	})
}

func TestRunHandlerErrorPolicy(t *testing.T) {
	const payload = `{"ok":true,"updates":[{"update_id":1},{"update_id":2}]}`
	handlerErr := errors.New("boom")

	t.Run("stops by default", func(t *testing.T) {
		svc := runService([]*http.Response{updatesResponse(payload)}, nil)

		var calls atomic.Int64
		err := svc.Run(context.Background(), RunOptions{Interval: time.Millisecond},
			func(context.Context, ym.Update) error {
				calls.Add(1)

				return handlerErr
			})

		if !errors.Is(err, handlerErr) {
			t.Fatalf("expected the handler error, got %v", err)
		}
		if calls.Load() != 1 {
			t.Fatalf("expected the loop to stop after the first failure, got %d calls", calls.Load())
		}
	})

	t.Run("ActionContinue processes the rest of the batch", func(t *testing.T) {
		svc := runService([]*http.Response{updatesResponse(payload)}, nil)

		var calls atomic.Int64
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := svc.Run(ctx, RunOptions{
			Interval: time.Millisecond,
			OnHandlerError: func(ym.Update, error) ErrorAction {
				if calls.Load() >= 2 {
					cancel()
				}

				return ActionContinue
			},
		}, func(context.Context, ym.Update) error {
			calls.Add(1)

			return handlerErr
		})

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation, got %v", err)
		}
		if calls.Load() < 2 {
			t.Fatalf("expected both updates to be attempted, got %d", calls.Load())
		}
	})
}

// A panicking handler should be containable rather than taking the process down.
func TestRunRecoversPanicWhenAsked(t *testing.T) {
	svc := runService([]*http.Response{
		updatesResponse(`{"ok":true,"updates":[{"update_id":1}]}`),
	}, nil)

	var recovered atomic.Bool
	err := svc.Run(context.Background(), RunOptions{
		Interval: time.Millisecond,
		OnPanic:  func(ym.Update, any) { recovered.Store(true) },
	}, func(context.Context, ym.Update) error {
		panic("handler exploded")
	})

	if err == nil {
		t.Fatal("expected the panic to surface as an error")
	}
	if !recovered.Load() {
		t.Fatal("OnPanic was not called")
	}
}

// Cancelling must not wait out the poll interval.
func TestRunReturnsPromptlyOnCancel(t *testing.T) {
	svc := runService([]*http.Response{
		updatesResponse(`{"ok":true,"updates":[]}`),
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := svc.Run(ctx, RunOptions{Interval: 30 * time.Second},
		func(context.Context, ym.Update) error { return nil })
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("cancellation waited out the interval: %v", elapsed)
	}
}
