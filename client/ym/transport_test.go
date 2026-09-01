package ym

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func retryingClient(t *testing.T, doer HTTPDoer, backoff time.Duration) *Client {
	t.Helper()

	return NewClientWithHTTP(Config{
		BaseURL: "http://example.com",
		Token:   "test-token",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{
				MaxAttempts:    3,
				InitialBackoff: backoff,
				MaxBackoff:     backoff,
				RetryHTTP:      []int{http.StatusInternalServerError},
				DisableJitter:  true,
			},
		},
	}, doer)
}

func TestSleepContext(t *testing.T) {
	t.Run("returns after the delay", func(t *testing.T) {
		if err := SleepContext(context.Background(), time.Millisecond); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("returns immediately when the context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		start := time.Now()
		err := SleepContext(ctx, 5*time.Second)
		elapsed := time.Since(start)

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
		if elapsed > time.Second {
			t.Fatalf("expected an immediate return, waited %v", elapsed)
		}
	})

	t.Run("non-positive delay does not block", func(t *testing.T) {
		if err := SleepContext(context.Background(), 0); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})
}

// A cancelled context must abort a pending retry back-off instead of letting a
// shutting-down bot block for the full MaxBackoff.
func TestDoRequestAbortsBackoffOnCancel(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusInternalServerError, `{"ok":false}`),
			testutil.NewResponse(http.StatusOK, `{"ok":true}`),
		},
	}
	client := retryingClient(t, doer, 10*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := client.DoRequest(ctx, http.MethodPost, "/path", map[string]string{"a": "b"})
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("cancellation was ignored during back-off: waited %v", elapsed)
	}
	if doer.CallCount() != 1 {
		t.Fatalf("expected the retry to be abandoned, got %d calls", doer.CallCount())
	}
}

// Every retry attempt must replay byte-identical bytes, which is what lets the
// API's payload_id key collapse a retried send into a single message.
func TestDoRequestReplaysIdenticalBody(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusInternalServerError, `{"ok":false}`),
			testutil.NewResponse(http.StatusInternalServerError, `{"ok":false}`),
			testutil.NewResponse(http.StatusOK, `{"ok":true}`),
		},
	}
	client := retryingClient(t, doer, time.Millisecond)

	body := map[string]string{"payload_id": "fixed-key", "text": "hi"}
	resp, err := client.DoRequest(context.Background(), http.MethodPost, "/path", body)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	defer resp.Body.Close()

	if doer.CallCount() != 3 {
		t.Fatalf("expected 3 attempts, got %d", doer.CallCount())
	}

	var first string
	for i, req := range doer.Requests {
		raw, readErr := io.ReadAll(req.Body)
		if readErr != nil {
			t.Fatalf("attempt %d: read body: %v", i, readErr)
		}
		if i == 0 {
			first = string(raw)

			continue
		}
		if string(raw) != first {
			t.Fatalf("attempt %d sent a different body:\n got %s\nwant %s", i, raw, first)
		}
	}
}

func TestRequestHeaders(t *testing.T) {
	t.Run("body-less request sends no Content-Type", func(t *testing.T) {
		doer := &testutil.FakeDoer{Responses: []*http.Response{testutil.NewResponse(http.StatusOK, `{"ok":true}`)}}
		client := retryingClient(t, doer, time.Millisecond)

		resp, err := client.DoRequest(context.Background(), http.MethodGet, "/path", nil)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		defer resp.Body.Close()

		if ct := doer.Requests[0].Header.Get("Content-Type"); ct != "" {
			t.Fatalf("expected no Content-Type on a GET, got %q", ct)
		}
	})

	t.Run("JSON body sets Content-Type", func(t *testing.T) {
		doer := &testutil.FakeDoer{Responses: []*http.Response{testutil.NewResponse(http.StatusOK, `{"ok":true}`)}}
		client := retryingClient(t, doer, time.Millisecond)

		resp, err := client.DoRequest(context.Background(), http.MethodPost, "/path", map[string]string{"a": "b"})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		defer resp.Body.Close()

		if ct := doer.Requests[0].Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("expected application/json, got %q", ct)
		}
	})

	t.Run("default User-Agent identifies the SDK", func(t *testing.T) {
		doer := &testutil.FakeDoer{Responses: []*http.Response{testutil.NewResponse(http.StatusOK, `{"ok":true}`)}}
		client := retryingClient(t, doer, time.Millisecond)

		resp, err := client.DoRequest(context.Background(), http.MethodGet, "/path", nil)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		defer resp.Body.Close()

		if ua := doer.Requests[0].Header.Get("User-Agent"); ua != defaultUserAgent {
			t.Fatalf("expected %q, got %q", defaultUserAgent, ua)
		}
	})

	t.Run("custom User-Agent is honoured", func(t *testing.T) {
		doer := &testutil.FakeDoer{Responses: []*http.Response{testutil.NewResponse(http.StatusOK, `{"ok":true}`)}}
		client := NewClientWithHTTP(Config{BaseURL: "http://example.com", UserAgent: "my-bot/1.0"}, doer)

		resp, err := client.DoRequest(context.Background(), http.MethodGet, "/path", nil)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		defer resp.Body.Close()

		if ua := doer.Requests[0].Header.Get("User-Agent"); ua != "my-bot/1.0" {
			t.Fatalf("expected my-bot/1.0, got %q", ua)
		}
	})
}

func TestBuildURL(t *testing.T) {
	client := NewClientWithHTTP(Config{BaseURL: "http://example.com/"}, &testutil.FakeDoer{})

	t.Run("trims the base URL slash", func(t *testing.T) {
		if got := client.buildURL("/bot/v1/self/get", nil); got != "http://example.com/bot/v1/self/get" {
			t.Fatalf("unexpected URL: %s", got)
		}
	})

	t.Run("appends the query", func(t *testing.T) {
		got := client.buildURL("/p", map[string][]string{"a": {"1"}})
		if got != "http://example.com/p?a=1" {
			t.Fatalf("unexpected URL: %s", got)
		}
	})
}

func TestApplyJitter(t *testing.T) {
	t.Run("stays within half the delay and the delay", func(t *testing.T) {
		const d = time.Second
		for range 200 {
			got := applyJitter(d, false)
			if got < d/2 || got > d {
				t.Fatalf("jittered %v outside [%v, %v]", got, d/2, d)
			}
		}
	})

	t.Run("disabled returns the delay unchanged", func(t *testing.T) {
		if got := applyJitter(time.Second, true); got != time.Second {
			t.Fatalf("expected 1s, got %v", got)
		}
	})

	t.Run("non-positive delay is returned unchanged", func(t *testing.T) {
		if got := applyJitter(0, false); got != 0 {
			t.Fatalf("expected 0, got %v", got)
		}
	})
}

func TestNewPayloadID(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for range 100 {
		id := NewPayloadID()
		if len(id) != 32 {
			t.Fatalf("expected a 32-character hex key, got %q", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate payload id: %s", id)
		}
		seen[id] = struct{}{}
	}
}

func TestAutoPayloadID(t *testing.T) {
	t.Run("enabled by the zero value", func(t *testing.T) {
		client := NewClientWithHTTP(Config{}, &testutil.FakeDoer{})
		if !client.AutoPayloadID() {
			t.Fatal("expected idempotency to be on by default")
		}
	})

	t.Run("opt-out honoured", func(t *testing.T) {
		client := NewClientWithHTTP(Config{DisableAutoPayloadID: true}, &testutil.FakeDoer{})
		if client.AutoPayloadID() {
			t.Fatal("expected idempotency to be disabled")
		}
	})
}
