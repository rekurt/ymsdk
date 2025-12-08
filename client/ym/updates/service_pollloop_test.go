package updates

import (
	"context"
	"net/http"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func TestPollLoopStopsOnHandlerError(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":true,"updates":[{"update_id":1,"message":{"id":1,"chat":{"id":"c1","type":"private"},"from":{"login":"u1"},"text":"hi","created_at":"now"}}],"next_offset":2}`),
		},
	})
	svc := NewService(client)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handlerCalled := false
	err := svc.PollLoop(ctx, GetUpdatesParams{Limit: intPtr(10)}, func(ctx context.Context, u ym.Update) error {
		handlerCalled = true
		return context.Canceled
	})
	if err == nil {
		t.Fatalf("expected error from handler")
	}
	if !handlerCalled {
		t.Fatalf("handler not called")
	}
}

func intPtr(i int) *int { return &i }
