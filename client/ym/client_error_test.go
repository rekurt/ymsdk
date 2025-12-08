package ym

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

func TestNewAPIErrorRateLimited(t *testing.T) {
	client := &Client{}
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Body:       io.NopCloser(bytes.NewBufferString(`{"ok":false,"description":"wait","code":429}`)),
		Header:     http.Header{"Retry-After": []string{"2"}, "X-Request-Id": []string{"req-1"}},
	}

	apiErr, err := client.newAPIError("GET", "/path", resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr.Kind != ymerrors.KindRateLimited {
		t.Fatalf("expected KindRateLimited, got %v", apiErr.Kind)
	}
	if apiErr.HTTPStatus != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", apiErr.HTTPStatus)
	}
	if apiErr.RetryAfter != 2*time.Second {
		t.Fatalf("expected RetryAfter 2s, got %v", apiErr.RetryAfter)
	}
	if apiErr.RequestID != "req-1" {
		t.Fatalf("expected request id from header")
	}
}

func TestNewAPIErrorUnauthorized(t *testing.T) {
	client := &Client{}
	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Body:       io.NopCloser(bytes.NewBufferString(`{"ok":false,"description":"unauthorized"}`)),
		Header:     http.Header{},
	}

	apiErr, err := client.newAPIError("POST", "/path", resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr.Kind != ymerrors.KindUnauthorized {
		t.Fatalf("expected KindUnauthorized, got %v", apiErr.Kind)
	}
}

func TestNewAPIErrorInvalidToken(t *testing.T) {
	client := &Client{}
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(bytes.NewBufferString(`{"ok":false,"description":"invalid token"}`)),
		Header:     http.Header{},
	}

	apiErr, err := client.newAPIError("POST", "/path", resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr.Kind != ymerrors.KindInvalidToken {
		t.Fatalf("expected KindInvalidToken, got %v", apiErr.Kind)
	}
}

func TestNewAPIErrorNoBody(t *testing.T) {
	client := &Client{}
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(bytes.NewBuffer(nil)),
		Header:     http.Header{},
	}

	apiErr, err := client.newAPIError("POST", "/path", resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr.Description != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("expected default status text, got %q", apiErr.Description)
	}
}
