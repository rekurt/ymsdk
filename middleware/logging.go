package middleware

import (
	"context"
	"encoding/json"
	"errors"

	"go.uber.org/zap"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

type ctxKey string

const requestIDKey ctxKey = "ym_request_id"

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func LogError(logger *zap.Logger, ctx context.Context, err error, method, endpoint string, params map[string]any) {
	if logger == nil || err == nil {
		return
	}

	requestID, _ := ctx.Value(requestIDKey).(string)

	var apiErr *ymerrors.APIError
	if errors.As(err, &apiErr) {
		logger.Error("yandex-messenger api error",
			zap.Int("kind", int(apiErr.Kind)),
			zap.Int("http_status", apiErr.HTTPStatus),
			zap.Int("code", apiErr.Code),
			zap.String("description", apiErr.Description),
			zap.Duration("retry_after", apiErr.RetryAfter),
			zap.String("request_id", apiErr.RequestID),
			zap.String("method", method),
			zap.String("endpoint", endpoint),
			zap.Any("params", params),
			zap.Error(err),
		)

		return
	}

	logger.Error("yandex-messenger client error",
		zap.String("request_id", requestID),
		zap.String("method", method),
		zap.String("endpoint", endpoint),
		zap.Any("params", params),
		zap.Error(err),
	)
}

// LogUpdateWithRawData logs a received update along with its raw JSON representation.
// This is useful for debugging parsing issues.
func LogUpdateWithRawData(logger *zap.Logger, ctx context.Context, update ym.Update, rawJSON []byte) {
	if logger == nil {
		return
	}

	var rawData map[string]interface{}
	_ = json.Unmarshal(rawJSON, &rawData)

	fields := []zap.Field{
		zap.Int64("update_id", update.UpdateID),
		zap.Any("raw_data", rawData),
	}

	if update.MessageID > 0 {
		fields = append(fields,
			zap.Int64("message_id", int64(update.MessageID)),
		)
		if update.Chat != nil {
			fields = append(fields, zap.String("chat_id", string(update.Chat.ID)))
		}
		if update.From != nil {
			fields = append(fields, zap.String("sender", string(update.From.Login)))
		}
		logger.Info("Update received with message", fields...)
	} else {
		logger.Warn("Update received without message", fields...)
	}
}

// LogUnparsedUpdate logs when an update structure doesn't match expected format.
func LogUnparsedUpdate(logger *zap.Logger, ctx context.Context, rawJSON []byte) {
	if logger == nil {
		return
	}

	rawStr := string(rawJSON)
	if len(rawStr) > 500 {
		rawStr = rawStr[:500] + "...(truncated)"
	}

	logger.Warn("Unparsed update data",
		zap.String("raw_json", rawStr),
	)
}
