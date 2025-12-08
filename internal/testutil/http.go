package testutil

import (
	"bytes"
	"io"
	"net/http"
)

// NewResponse constructs an HTTP response with the given status and body string.
func NewResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{},
	}
}
