package ym

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func TestDoMultipartRequestSuccess(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"ok":true,"message_id":1}`)),
				Header:     http.Header{},
			},
		},
	}
	client := NewClientWithHTTP(Config{
		BaseURL: "http://example.com",
		Token:   "tok",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, doer)

	resp, err := client.DoMultipartRequest(context.Background(), http.MethodPost, "/upload", "multipart/form-data; boundary=abc", []byte("payload"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	req := doer.Requests[0]
	if req.Header.Get("Content-Type") != "multipart/form-data; boundary=abc" {
		t.Fatalf("unexpected content type: %s", req.Header.Get("Content-Type"))
	}
	if req.Header.Get("Authorization") != "OAuth tok" {
		t.Fatalf("unexpected auth: %s", req.Header.Get("Authorization"))
	}
}

func TestDoMultipartRequestRetryOn500(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewBufferString(`{"ok":false}`)),
				Header:     http.Header{},
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"ok":true}`)),
				Header:     http.Header{},
			},
		},
	}
	client := NewClientWithHTTP(Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{
				MaxAttempts:    2,
				InitialBackoff: 1,
			},
		},
	}, doer)

	resp, err := client.DoMultipartRequest(context.Background(), http.MethodPost, "/upload", "multipart/form-data", []byte("data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if doer.CallCount() != 2 {
		t.Fatalf("expected 2 attempts, got %d", doer.CallCount())
	}
}

func TestDoMultipartRequestRateLimitFallback(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(bytes.NewBufferString(`{"ok":false}`)),
				Header:     http.Header{"Retry-After": []string{"0"}},
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"ok":true}`)),
				Header:     http.Header{},
			},
		},
	}
	client := NewClientWithHTTP(Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{
				MaxAttempts:    2,
				InitialBackoff: 1,
			},
			RateLimitHandling: ymerrors.RateLimitHandling{
				UseRetryAfter:  true,
				DefaultBackoff: 1,
			},
		},
	}, doer)

	resp, err := client.DoMultipartRequest(context.Background(), http.MethodPost, "/upload", "multipart/form-data", []byte("data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if doer.CallCount() != 2 {
		t.Fatalf("expected 2 attempts (rate limit + retry), got %d", doer.CallCount())
	}
}
