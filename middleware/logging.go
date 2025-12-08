package middleware

import (
	"context"
	"errors"

	"go.uber.org/zap"

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
