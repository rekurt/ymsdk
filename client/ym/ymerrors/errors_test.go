package ymerrors

import (
	"errors"
	"testing"
	"time"
)

func TestAPIErrorErrorNotEmpty(t *testing.T) {
	err := &APIError{
		Kind:        KindBadRequest,
		Code:        400,
		HTTPStatus:  400,
		Description: "bad request",
		RequestID:   "req-1",
		Method:      "GET",
		Endpoint:    "/path",
		RetryAfter:  time.Second,
	}
	if err.Error() == "" {
		t.Fatalf("expected non-empty error string")
	}
}

func TestAPIErrorUnwrapRateLimited(t *testing.T) {
	err := &APIError{Kind: KindRateLimited}
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected errors.Is to match ErrRateLimited")
	}
}

func TestAPIErrorUnwrapInvalidToken(t *testing.T) {
	err := &APIError{Kind: KindInvalidToken}
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected errors.Is to match ErrInvalidToken")
	}
}

func TestAPIErrorAs(t *testing.T) {
	err := &APIError{Kind: KindUnauthorized}
	var target *APIError
	if !errors.As(err, &target) {
		t.Fatalf("expected errors.As to populate APIError")
	}
	if target.Kind != KindUnauthorized {
		t.Fatalf("unexpected kind: %v", target.Kind)
	}
}
