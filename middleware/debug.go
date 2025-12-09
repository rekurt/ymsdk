package middleware

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// LogLevel represents logging verbosity level.
type LogLevel int

const (
	LogLevelSilent LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// DebugLogger provides detailed HTTP request/response logging with raw body inspection.
type DebugLogger struct {
	logger   *zap.Logger
	logLevel LogLevel
}

// NewDebugLogger creates a new debug logger with specified level.
func NewDebugLogger(logger *zap.Logger, level LogLevel) *DebugLogger {
	if logger == nil {
		return &DebugLogger{logLevel: level}
	}

	return &DebugLogger{logger: logger, logLevel: level}
}

// LogRequest logs HTTP request details including raw body.
func (dl *DebugLogger) LogRequest(ctx context.Context, req *http.Request, body []byte) {
	if dl.logger == nil || dl.logLevel < LogLevelDebug {
		return
	}

	headers := make(map[string]string)
	for k, vv := range req.Header {
		// Don't log sensitive headers
		if strings.ToLower(k) != "authorization" {
			headers[k] = strings.Join(vv, ",")
		}
	}

	bodyStr := string(body)
	if len(bodyStr) > 1000 {
		bodyStr = bodyStr[:1000] + "...(truncated)"
	}

	dl.logger.Debug("HTTP Request",
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.Any("headers", headers),
		zap.String("body", bodyStr),
	)
}

// LogResponse logs HTTP response details including raw body.
func (dl *DebugLogger) LogResponse(ctx context.Context, resp *http.Response, body []byte) {
	if dl.logger == nil || dl.logLevel < LogLevelDebug {
		return
	}

	headers := make(map[string]string)
	for k, vv := range resp.Header {
		headers[k] = strings.Join(vv, ",")
	}

	bodyStr := string(body)
	if len(bodyStr) > 1000 {
		bodyStr = bodyStr[:1000] + "...(truncated)"
	}

	dl.logger.Debug("HTTP Response",
		zap.Int("status_code", resp.StatusCode),
		zap.String("status", resp.Status),
		zap.Any("headers", headers),
		zap.String("body", bodyStr),
	)
}

// LogParsedUpdate logs parsed update data at info level.
func (dl *DebugLogger) LogParsedUpdate(ctx context.Context, updateID int64, data map[string]interface{}) {
	if dl.logger == nil || dl.logLevel < LogLevelInfo {
		return
	}

	dl.logger.Info("Parsed Update",
		zap.Int64("update_id", updateID),
		zap.Any("data", data),
	)
}

// LogWarning logs a warning-level message.
func (dl *DebugLogger) LogWarning(ctx context.Context, msg string, fields ...zap.Field) {
	if dl.logger == nil || dl.logLevel < LogLevelWarn {
		return
	}
	dl.logger.Warn(msg, fields...)
}

// LogDebug logs a debug-level message.
func (dl *DebugLogger) LogDebug(ctx context.Context, msg string, fields ...zap.Field) {
	if dl.logger == nil || dl.logLevel < LogLevelDebug {
		return
	}
	dl.logger.Debug(msg, fields...)
}

// RespBodyReader reads response body and returns both the bytes and a new reader.
// This is useful for logging the body while still being able to pass it downstream.
func RespBodyReader(resp *http.Response) ([]byte, io.ReadCloser, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read response body: %w", err)
	}

	// Close original body
	resp.Body.Close()

	// Return body bytes and a new reader
	return body, io.NopCloser(strings.NewReader(string(body))), nil
}

// RequestBodyReader reads request body and returns both the bytes and a new reader.
func RequestBodyReader(req *http.Request) ([]byte, io.ReadCloser, error) {
	if req.Body == nil {
		return nil, nil, nil
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read request body: %w", err)
	}

	// Close original body
	req.Body.Close()

	// Return body bytes and a new reader
	return body, io.NopCloser(strings.NewReader(string(body))), nil
}
