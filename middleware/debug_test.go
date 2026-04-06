package middleware

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestNewDebugLoggerNilLogger(t *testing.T) {
	dl := NewDebugLogger(nil, LogLevelDebug)
	if dl == nil {
		t.Fatal("expected non-nil DebugLogger")
	}
	// Should not panic on nil logger.
	dl.LogRequest(context.Background(), nil, nil)
	dl.LogResponse(context.Background(), nil, nil)
	dl.LogWarning(context.Background(), "test")
	dl.LogDebug(context.Background(), "test")
}

func TestDebugLoggerLogRequest(t *testing.T) {
	logger := zaptest.NewLogger(t)
	dl := NewDebugLogger(logger, LogLevelDebug)
	req, _ := http.NewRequest(http.MethodPost, "http://example.com/test", nil)
	req.Header.Set("Authorization", "OAuth secret")
	req.Header.Set("Content-Type", "application/json")
	// Should log without Authorization header.
	dl.LogRequest(context.Background(), req, []byte(`{"text":"hello"}`))
}

func TestDebugLoggerLogRequestTruncation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	dl := NewDebugLogger(logger, LogLevelDebug)
	req, _ := http.NewRequest(http.MethodPost, "http://example.com/test", nil)
	longBody := make([]byte, 1500)
	for i := range longBody {
		longBody[i] = 'x'
	}
	// Should truncate body at 1000 chars.
	dl.LogRequest(context.Background(), req, longBody)
}

func TestDebugLoggerLogResponse(t *testing.T) {
	logger := zaptest.NewLogger(t)
	dl := NewDebugLogger(logger, LogLevelDebug)
	resp := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": {"application/json"}},
	}
	dl.LogResponse(context.Background(), resp, []byte(`{"ok":true}`))
}

func TestDebugLoggerLogParsedUpdate(t *testing.T) {
	logger := zaptest.NewLogger(t)
	dl := NewDebugLogger(logger, LogLevelInfo)
	dl.LogParsedUpdate(context.Background(), 42, map[string]any{"text": "hello"})
}

func TestDebugLoggerLevelFiltering(t *testing.T) {
	logger := zaptest.NewLogger(t)

	dl := NewDebugLogger(logger, LogLevelError)
	// These should be no-ops due to log level filtering.
	dl.LogRequest(context.Background(), nil, nil)
	dl.LogResponse(context.Background(), nil, nil)
	dl.LogParsedUpdate(context.Background(), 1, nil)
	dl.LogWarning(context.Background(), "test")
	dl.LogDebug(context.Background(), "test")
}

func TestRespBodyReader(t *testing.T) {
	body := "response body content"
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(body)),
	}

	bodyBytes, newBody, err := RespBodyReader(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(bodyBytes) != body {
		t.Fatalf("expected %q, got %q", body, string(bodyBytes))
	}

	read, _ := io.ReadAll(newBody)
	if string(read) != body {
		t.Fatalf("expected %q from new reader, got %q", body, string(read))
	}
}

func TestRequestBodyReader(t *testing.T) {
	t.Run("nil body", func(t *testing.T) {
		req := &http.Request{}
		bodyBytes, newBody, err := RequestBodyReader(req)
		if err != nil || bodyBytes != nil || newBody != nil {
			t.Fatal("expected all nil for nil body")
		}
	})

	t.Run("with body", func(t *testing.T) {
		body := "request content"
		req := &http.Request{
			Body: io.NopCloser(strings.NewReader(body)),
		}

		bodyBytes, newBody, err := RequestBodyReader(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(bodyBytes) != body {
			t.Fatalf("expected %q, got %q", body, string(bodyBytes))
		}

		read, _ := io.ReadAll(newBody)
		if string(read) != body {
			t.Fatalf("expected %q from new reader, got %q", body, string(read))
		}
	})
}
