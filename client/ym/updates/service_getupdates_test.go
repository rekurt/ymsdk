package updates

import (
	"context"
	"net/http"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func TestGetUpdatesSuccess(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":true,"updates":[{"update_id":5,"message_id":10}],"next_offset":6}`),
		},
	})

	svc := NewService(client)
	limit := 10
	upds, nextOffset, err := svc.GetUpdates(context.Background(), GetUpdatesParams{Limit: &limit})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(upds) != 1 {
		t.Fatalf("expected 1 update, got %d", len(upds))
	}
	if nextOffset != 6 {
		t.Fatalf("expected next_offset=6, got %d", nextOffset)
	}
}

func TestGetUpdatesWithOffset(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":true,"updates":[],"next_offset":0}`),
		},
	})

	svc := NewService(client)
	offset := int64(5)
	upds, nextOffset, err := svc.GetUpdates(context.Background(), GetUpdatesParams{Offset: &offset})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(upds) != 0 {
		t.Fatalf("expected 0 updates, got %d", len(upds))
	}
	if nextOffset != 5 {
		t.Fatalf("expected next_offset=5 (from current), got %d", nextOffset)
	}
}

func TestGetUpdatesCalculatesOffsetFromUpdates(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":true,"updates":[{"update_id":10},{"update_id":12}],"next_offset":0}`),
		},
	})

	svc := NewService(client)
	upds, nextOffset, err := svc.GetUpdates(context.Background(), GetUpdatesParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(upds) != 2 {
		t.Fatalf("expected 2 updates, got %d", len(upds))
	}
	if nextOffset != 13 {
		t.Fatalf("expected next_offset=13 (max update_id+1), got %d", nextOffset)
	}
}

func TestPollLoopStopsOnContextCancel(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":true,"updates":[],"next_offset":0}`),
		},
	})

	svc := NewService(client)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.PollLoop(ctx, GetUpdatesParams{}, func(_ context.Context, _ ym.Update) error {
		t.Fatal("handler should not be called on cancelled context")

		return nil
	})
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestPollLoopStopsOnGetUpdatesError(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `not json`),
		},
	})

	svc := NewService(client)
	err := svc.PollLoop(context.Background(), GetUpdatesParams{}, func(_ context.Context, _ ym.Update) error {
		t.Fatal("handler should not be called on error")

		return nil
	})
	if err == nil {
		t.Fatal("expected error from GetUpdates")
	}
}
