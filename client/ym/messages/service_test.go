package messages

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func TestSendToChatSuccess(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":true,"message":{"id":1,"chat":{"id":"c1","type":"private"},"from":{"login":"u1"},"text":"hi","created_at":"now"}}`),
		},
	})

	service := NewService(client)
	msg, err := service.SendToChat(context.Background(), ym.ChatID("c1"), "hi", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil || msg.ID != 1 {
		t.Fatalf("expected message with id=1")
	}
}

func TestSendToChatInvalidResponse(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":false}`),
		},
	})
	service := NewService(client)

	_, err := service.SendToChat(context.Background(), ym.ChatID("c1"), "hi", nil)
	if !errors.Is(err, ymerrors.ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got %v", err)
	}
}

func TestSendToLoginBadJSON(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":`),
		},
	})
	service := NewService(client)

	_, err := service.SendToLogin(context.Background(), ym.UserLogin("login1"), "hi", nil)
	if !errors.Is(err, ymerrors.ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got %v", err)
	}
}

// helper in attachments_test.go
