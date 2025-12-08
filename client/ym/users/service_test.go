package users

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func TestGetUserLinkSuccess(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			newResponse(http.StatusOK, `{"ok":true,"id":"u1","chat_link":"cl","call_link":"call"}`),
		},
	})
	svc := NewService(client)

	link, err := svc.GetUserLink(context.Background(), "login1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if link.ChatLink != "cl" || link.CallLink != "call" {
		t.Fatalf("unexpected links: %+v", link)
	}
}

func TestGetUserLinkAPIError(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			newResponse(http.StatusOK, `{"ok":false,"description":"not found"}`),
		},
	})
	svc := NewService(client)

	_, err := svc.GetUserLink(context.Background(), "login1")
	var apiErr *ymerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected api error")
	}
	if apiErr.HTTPStatus != http.StatusOK {
		t.Fatalf("expected status 200, got %d", apiErr.HTTPStatus)
	}
}

func newResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{},
	}
}
