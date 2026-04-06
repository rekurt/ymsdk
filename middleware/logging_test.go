package middleware

import (
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

func TestWithRequestID(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-123")
	got, ok := ctx.Value(requestIDKey).(string)
	if !ok || got != "req-123" {
		t.Fatalf("expected req-123, got %q", got)
	}
}

func TestLogErrorNilLogger(t *testing.T) {
	// Should not panic with nil logger.
	LogError(nil, context.Background(), nil, "GET", "/test", nil)
}

func TestLogErrorNilError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	// Should not panic with nil error.
	LogError(logger, context.Background(), nil, "GET", "/test", nil)
}

func TestLogErrorWithAPIError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	apiErr := &ymerrors.APIError{
		Kind:        ymerrors.KindRateLimited,
		HTTPStatus:  429,
		Description: "too many requests",
		RequestID:   "abc",
	}
	// Should not panic.
	LogError(logger, context.Background(), apiErr, "POST", "/send", map[string]any{"chat_id": "c1"})
}

func TestLogErrorWithGenericError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := WithRequestID(context.Background(), "req-456")
	// Should log as generic client error.
	LogError(logger, ctx, context.DeadlineExceeded, "GET", "/updates", nil)
}

func TestLogUpdateWithRawDataNilLogger(t *testing.T) {
	// Should not panic.
	LogUpdateWithRawData(nil, context.Background(), ym.Update{}, nil)
}

func TestLogUpdateWithRawDataWithMessage(t *testing.T) {
	logger := zaptest.NewLogger(t)
	update := ym.Update{
		UpdateID:  1,
		MessageID: 42,
		Chat:      &ym.Chat{ID: "c1"},
		From:      &ym.Sender{Login: "u1"},
		Text:      "hello",
	}
	raw, _ := json.Marshal(update)
	// Should log at Info level.
	LogUpdateWithRawData(logger, context.Background(), update, raw)
}

func TestLogUpdateWithRawDataWithoutMessage(t *testing.T) {
	logger := zaptest.NewLogger(t)
	update := ym.Update{
		UpdateID: 2,
	}
	// Should log at Warn level.
	LogUpdateWithRawData(logger, context.Background(), update, []byte(`{}`))
}

func TestLogUnparsedUpdateNilLogger(t *testing.T) {
	LogUnparsedUpdate(nil, context.Background(), nil)
}

func TestLogUnparsedUpdate(t *testing.T) {
	logger := zaptest.NewLogger(t)
	LogUnparsedUpdate(logger, context.Background(), []byte(`{"unknown":"data"}`))
}

func TestLogUnparsedUpdateTruncation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	longJSON := make([]byte, 600)
	for i := range longJSON {
		longJSON[i] = 'a'
	}
	// Should truncate at 500 chars without panic.
	LogUnparsedUpdate(logger, context.Background(), longJSON)
}

func init() {
	// Suppress zap logger output during tests.
	_ = zap.L()
}
