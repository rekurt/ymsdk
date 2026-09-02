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

// id, chat_link and call_link are all documented as required. Returning empty
// strings with a nil error gives the caller links that go nowhere and hides a
// malformed response — the same gap the other single-object decoders had.
func TestGetUserLinkRejectsAnIncompleteResponse(t *testing.T) {
	cases := []string{
		`{"ok":true}`,
		`{"ok":true,"id":"u1"}`,
		`{"ok":true,"id":"u1","chat_link":"https://example/chat"}`,
	}

	for _, body := range cases {
		doer := &testutil.FakeDoer{Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, body),
		}}
		svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

		link, err := svc.GetUserLink(context.Background(), "user@example.org")
		if !errors.Is(err, ymerrors.ErrInvalidResponse) {
			t.Fatalf("%s: expected ErrInvalidResponse, got %v", body, err)
		}
		if link != nil {
			t.Fatalf("%s: expected no link alongside the error, got %#v", body, link)
		}
	}
}

func TestGetUserLinkAcceptsACompleteResponse(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK,
			`{"ok":true,"id":"u1","chat_link":"https://example/chat","call_link":"messenger://call"}`),
	}}
	svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

	link, err := svc.GetUserLink(context.Background(), "user@example.org")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if link.ID != "u1" || link.ChatLink == "" || link.CallLink == "" {
		t.Fatalf("unexpected link: %#v", link)
	}
}
