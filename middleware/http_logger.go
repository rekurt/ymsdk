package middleware

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HTTPLogger wraps an http.Client to log raw request/response bodies.
type HTTPLogger struct {
	client      *http.Client
	debugLogger *DebugLogger
}

// NewHTTPLogger creates a new HTTP logger wrapper.
func NewHTTPLogger(client *http.Client, debugLogger *DebugLogger) *HTTPLogger {
	if client == nil {
		client = &http.Client{}
	}
	if debugLogger == nil {
		debugLogger = &DebugLogger{}
	}

	return &HTTPLogger{client: client, debugLogger: debugLogger}
}

// Do implements the HttpDoer interface, logging request/response details.
func (hl *HTTPLogger) Do(req *http.Request) (*http.Response, error) {
	// Log request
	var reqBody []byte
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err == nil {
			reqBody = body
			// Restore body for actual request
			req.Body = io.NopCloser(strings.NewReader(string(body)))
		}
	}
	hl.debugLogger.LogRequest(req.Context(), req, reqBody)

	// Make request
	resp, err := hl.client.Do(req)
	if err != nil {
		hl.debugLogger.LogWarning(req.Context(), fmt.Sprintf("HTTP request failed: %v", err))

		return nil, err
	}

	// Log response
	respBody, err := io.ReadAll(resp.Body)
	if err == nil {
		hl.debugLogger.LogResponse(req.Context(), resp, respBody)
		// Restore body for downstream processing
		resp.Body = io.NopCloser(strings.NewReader(string(respBody)))
	}

	return resp, nil
}
