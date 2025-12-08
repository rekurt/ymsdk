package messages

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"

	"github.com/rekurt/ymsdk/internal/testutil"
)

func TestSendFileValidation(t *testing.T) {
	svc := NewService(nil)
	_, err := svc.SendFile(context.Background(), &SendFileRequest{})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestSendFileSuccess(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":true,"message":{"id":1,"chat":{"id":"c1","type":"private"},"from":{"login":"u1"},"text":"file","created_at":"now"}}`),
		},
	})
	svc := NewService(client)
	msg, err := svc.SendFile(context.Background(), &SendFileRequest{
		ChatID:   ptrChat("c1"),
		Document: bytes.NewBufferString("data"),
		Filename: "f.txt",
	})
	if err != nil || msg == nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteMessageAPIError(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":false,"description":"denied"}`),
		},
	})
	svc := NewService(client)
	err := svc.Delete(context.Background(), &DeleteMessageRequest{
		ChatID:    ptrChat("c1"),
		MessageID: 1,
	})
	var apiErr *ymerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected api error")
	}
}

func TestGetFileJSONError(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{})
	svc := NewService(client)
	resp := testutil.NewResponse(http.StatusOK, `{"ok":false,"description":"not found"}`)
	resp.Header.Set("Content-Type", "application/json")
	client.HTTPDoer().(*testutil.FakeDoer).Responses = []*http.Response{resp}

	_, _, err := svc.GetFile(context.Background(), "file1")
	var apiErr *ymerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected api error")
	}
}

func ptrChat(id ym.ChatID) *ym.ChatID { return &id }
