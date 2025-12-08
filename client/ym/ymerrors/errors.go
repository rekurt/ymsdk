package ymerrors

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

type ErrorKind int

const (
	KindUnknown ErrorKind = iota
	KindRateLimited
	KindInvalidToken
	KindUnauthorized
	KindBadRequest
	KindNetwork
)

var (
	ErrRateLimited     = errors.New("yandex-messenger: rate limited")
	ErrInvalidToken    = errors.New("yandex-messenger: invalid token")
	ErrUnauthorized    = errors.New("yandex-messenger: unauthorized")
	ErrRequestTimeout  = errors.New("yandex-messenger: request timeout")
	ErrNetworkError    = errors.New("yandex-messenger: network error")
	ErrInvalidResponse = errors.New("yandex-messenger: invalid response")
)

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
