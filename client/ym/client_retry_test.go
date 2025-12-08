package ym

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/internal/testutil"
)

type stubNetError struct{}

func (stubNetError) Error() string   { return "network down" }
func (stubNetError) Timeout() bool   { return true }
func (stubNetError) Temporary() bool { return true }

func TestDoRequestRetriesOnRateLimit(t *testing.T) {
	client := NewClientWithHTTP(Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{
				MaxAttempts:    2,
				InitialBackoff: time.Millisecond,
				MaxBackoff:     2 * time.Millisecond,
			},
			RateLimitHandling: ymerrors.RateLimitHandling{
				UseRetryAfter:  true,
				DefaultBackoff: time.Millisecond,
			},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			newResponse(http.StatusTooManyRequests, `{"ok":false}`, map[string]string{"Retry-After": "0"}),
			newResponse(http.StatusOK, `{"ok":true}`, nil),
		},
	})

	resp, err := client.DoRequest(context.Background(), http.MethodGet, "/path", nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestDoRequestRetriesOnNetworkError(t *testing.T) {
	client := NewClientWithHTTP(Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{
				MaxAttempts:    2,
				InitialBackoff: time.Millisecond,
				MaxBackoff:     2 * time.Millisecond,
				RetryNetwork:   true,
			},
			RateLimitHandling: ymerrors.RateLimitHandling{
				DefaultBackoff: time.Millisecond,
			},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			nil,
			newResponse(http.StatusOK, `{"ok":true}`, nil),
		},
		Errors: []error{
			stubNetError{},
			nil,
		},
	})

	resp, err := client.DoRequest(context.Background(), http.MethodGet, "/path", nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestDoRequestContextDeadline(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Millisecond))
	defer cancel()

	client := NewClientWithHTTP(Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{
				MaxAttempts:    2,
				InitialBackoff: time.Millisecond,
				MaxBackoff:     2 * time.Millisecond,
				RetryNetwork:   true,
			},
			RateLimitHandling: ymerrors.RateLimitHandling{
				DefaultBackoff: time.Millisecond,
			},
		},
	}, &testutil.FakeDoer{
		Errors: []error{context.DeadlineExceeded},
	})

	_, err := client.DoRequest(ctx, http.MethodGet, "/path", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func newResponse(status int, body string, headers map[string]string) *http.Response {
	resp := &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{},
	}
	for k, v := range headers {
		resp.Header.Set(k, v)
	}
	return resp
}
