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

func TestAPIErrorUnwrapBadRequest(t *testing.T) {
	err := &APIError{Kind: KindBadRequest}
	if !errors.Is(err, ErrBadRequest) {
		t.Fatalf("expected errors.Is to match ErrBadRequest")
	}
}

func TestAPIErrorUnwrapNetwork(t *testing.T) {
	err := &APIError{Kind: KindNetwork}
	if !errors.Is(err, ErrNetworkError) {
		t.Fatalf("expected errors.Is to match ErrNetworkError")
	}
}

func TestAPIErrorUnwrapNotFound(t *testing.T) {
	err := &APIError{Kind: KindNotFound}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected errors.Is to match ErrNotFound")
	}
}

func TestAPIErrorUnwrapConflict(t *testing.T) {
	err := &APIError{Kind: KindConflict}
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected errors.Is to match ErrConflict")
	}
}

func TestAPIErrorUnwrapPayloadTooLarge(t *testing.T) {
	err := &APIError{Kind: KindPayloadTooLarge}
	if !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("expected errors.Is to match ErrPayloadTooLarge")
	}
}

func TestAPIErrorUnwrapUnknown(t *testing.T) {
	err := &APIError{Kind: KindUnknown}
	if err.Unwrap() != nil {
		t.Fatalf("expected KindUnknown to unwrap to nil")
	}
}
