package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestHTTPLoggerDo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	debugLogger := NewDebugLogger(logger, LogLevelDebug)
	hl := NewHTTPLogger(server.Client(), debugLogger)

	req, err := http.NewRequest(http.MethodPost, server.URL+"/test", strings.NewReader(`{"text":"hello"}`))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := hl.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("body mismatch: %s", body)
	}
}

func TestHTTPLoggerNilDefaults(t *testing.T) {
	hl := NewHTTPLogger(nil, nil)
	if hl.client == nil || hl.debugLogger == nil {
		t.Fatal("expected non-nil defaults")
	}
}
