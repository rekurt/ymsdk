package ymerrors

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// ErrorKind classifies API errors into broad categories for programmatic handling.
type ErrorKind int

const (
	// KindUnknown is an unclassified error.
	KindUnknown ErrorKind = iota
	// KindRateLimited indicates the request was throttled (HTTP 429).
	KindRateLimited
	// KindInvalidToken indicates the OAuth token is invalid (HTTP 403).
	KindInvalidToken
	// KindUnauthorized indicates missing or expired credentials (HTTP 401).
	KindUnauthorized
	// KindBadRequest indicates a malformed request (HTTP 400).
	KindBadRequest
	// KindNetwork indicates a transport-level failure (DNS, TCP, 5xx).
	KindNetwork
)

// Sentinel errors for use with errors.Is.
var (
	ErrRateLimited     = errors.New("yandex-messenger: rate limited")
	ErrInvalidToken    = errors.New("yandex-messenger: invalid token")
	ErrUnauthorized    = errors.New("yandex-messenger: unauthorized")
	ErrRequestTimeout  = errors.New("yandex-messenger: request timeout")
	ErrNetworkError    = errors.New("yandex-messenger: network error")
	ErrInvalidResponse = errors.New("yandex-messenger: invalid response")
)

// APIError is a structured error returned by the Yandex Messenger API.
// Use errors.As to extract it and errors.Is to match sentinel errors.
type APIError struct {
	Kind        ErrorKind
	Code        int
	HTTPStatus  int
	Description string
	RequestID   string
	Method      string
	Endpoint    string
	RetryAfter  time.Duration
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString("yandex-messenger/apierror")
	b.WriteString(": kind=")
	b.WriteString(strconv.Itoa(int(e.Kind)))
	if e.HTTPStatus > 0 {
		b.WriteString(" http=")
		b.WriteString(strconv.Itoa(e.HTTPStatus))
	}
	if e.Code != 0 {
		b.WriteString(" code=")
		b.WriteString(strconv.Itoa(e.Code))
	}
	if e.RequestID != "" {
		b.WriteString(" request_id=")
		b.WriteString(e.RequestID)
	}
	if e.Method != "" || e.Endpoint != "" {
		b.WriteString(" op=")
		b.WriteString(strings.TrimSpace(strings.Join([]string{e.Method, e.Endpoint}, " ")))
	}
	if e.RetryAfter > 0 {
		b.WriteString(" retry_after=")
		b.WriteString(e.RetryAfter.String())
	}
	if e.Description != "" {
		b.WriteString(": ")
		b.WriteString(e.Description)
	}

	return b.String()
}

// Unwrap returns the sentinel error matching the error kind,
// enabling errors.Is checks (e.g. errors.Is(err, ErrRateLimited)).
func (e *APIError) Unwrap() error {
	if e == nil {
		return nil
	}
	switch e.Kind {
	case KindRateLimited:
		return ErrRateLimited
	case KindInvalidToken:
		return ErrInvalidToken
	case KindUnauthorized:
		return ErrUnauthorized
	case KindNetwork:
		return ErrNetworkError
	default:
		return nil
	}
}
