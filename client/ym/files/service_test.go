package files

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func TestSendToChatSuccess(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{
			newResponse(http.StatusOK, `{"ok":true,"message":{"id":1,"chat":{"id":"c1","type":"private"},"from":{"login":"u1"},"text":"file","created_at":"now"}}`),
		},
	}
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, doer)

	svc := NewService(client)
	msg, err := svc.SendToChat(context.Background(), "c1", "f.txt", "text/plain", []byte("hello"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil || msg.ID != 1 {
		t.Fatalf("expected message m1")
	}
	if len(doer.Requests) != 1 {
		t.Fatalf("expected one request")
	}
	req := doer.Requests[0]
	if req.Method != http.MethodPost {
		t.Fatalf("expected POST, got %s", req.Method)
	}
	if req.URL.Path != "/bot/v1/messages/sendFile" {
		t.Fatalf("unexpected path: %s", req.URL.Path)
	}
	if ct := req.Header.Get("Content-Type"); !strings.HasPrefix(ct, "multipart/form-data") {
		t.Fatalf("expected multipart content type, got %s", ct)
	}
}

func TestSendToLoginInvalidResponse(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			newResponse(http.StatusOK, `{"ok":false}`),
		},
	})

	svc := NewService(client)
	_, err := svc.SendToLogin(context.Background(), "login1", "f.txt", "text/plain", []byte("hello"), nil)
	if !errors.Is(err, ymerrors.ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got %v", err)
	}
}

func TestSendToChatBadJSON(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			newResponse(http.StatusOK, `{"ok":`),
		},
	})

	svc := NewService(client)
	_, err := svc.SendToChat(context.Background(), "c1", "f.txt", "text/plain", []byte("hello"), nil)
	if !errors.Is(err, ymerrors.ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got %v", err)
	}
}

func newResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{},
	}
}
