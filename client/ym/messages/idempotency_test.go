package messages

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func retryOnceClient(doer ym.HTTPDoer, disableAuto bool) *ym.Client {
	return ym.NewClientWithHTTP(ym.Config{
		BaseURL:              "http://example.com",
		DisableAutoPayloadID: disableAuto,
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{
				MaxAttempts:    2,
				InitialBackoff: time.Millisecond,
				MaxBackoff:     time.Millisecond,
				RetryHTTP:      []int{http.StatusInternalServerError},
				DisableJitter:  true,
			},
		},
	}, doer)
}

func payloadIDOf(t *testing.T, req *http.Request) string {
	t.Helper()

	raw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	var parsed struct {
		PayloadID string `json:"payload_id"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("decode request body %s: %v", raw, err)
	}

	return parsed.PayloadID
}

// Retrying a POST is only safe because both attempts carry the same payload_id:
// the API treats them as duplicates and delivers a single message.
func TestSendToChatReusesPayloadIDAcrossRetries(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusInternalServerError, `{"ok":false}`),
			testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":42}`),
		},
	}
	svc := NewService(retryOnceClient(doer, false))

	msg, err := svc.SendToChat(context.Background(), "chat-1", "hello", nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if msg.ID != 42 {
		t.Fatalf("expected message id 42, got %d", msg.ID)
	}
	if doer.CallCount() != 2 {
		t.Fatalf("expected 2 attempts, got %d", doer.CallCount())
	}

	first := payloadIDOf(t, doer.Requests[0])
	second := payloadIDOf(t, doer.Requests[1])

	if first == "" {
		t.Fatal("expected an auto-generated payload_id, got an empty one")
	}
	if first != second {
		t.Fatalf("payload_id changed between attempts: %q then %q", first, second)
	}
}

func TestSendToChatPayloadIDOptions(t *testing.T) {
	t.Run("caller-supplied key is preserved", func(t *testing.T) {
		doer := &testutil.FakeDoer{
			Responses: []*http.Response{testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":1}`)},
		}
		svc := NewService(retryOnceClient(doer, false))

		_, err := svc.SendToChat(context.Background(), "chat-1", "hi", &SendMessageOptions{PayloadID: "mine"})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if got := payloadIDOf(t, doer.Requests[0]); got != "mine" {
			t.Fatalf("expected the caller's key, got %q", got)
		}
	})

	t.Run("opt-out sends no key", func(t *testing.T) {
		doer := &testutil.FakeDoer{
			Responses: []*http.Response{testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":1}`)},
		}
		svc := NewService(retryOnceClient(doer, true))

		_, err := svc.SendToChat(context.Background(), "chat-1", "hi", nil)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if got := payloadIDOf(t, doer.Requests[0]); got != "" {
			t.Fatalf("expected no key, got %q", got)
		}
	})

	t.Run("SendToLogin also stamps a key", func(t *testing.T) {
		doer := &testutil.FakeDoer{
			Responses: []*http.Response{testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":1}`)},
		}
		svc := NewService(retryOnceClient(doer, false))

		_, err := svc.SendToLogin(context.Background(), "user@example.org", "hi", nil)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if got := payloadIDOf(t, doer.Requests[0]); got == "" {
			t.Fatal("expected an auto-generated payload_id")
		}
	})
}
