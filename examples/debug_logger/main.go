package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/updates"
	"github.com/rekurt/ymsdk/middleware"
)

// This example demonstrates how to use the enhanced debug logging
// to inspect raw HTTP request/response bodies and handle updates
// that arrive without message data.

func main() {
	token := os.Getenv("YM_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "YM_TOKEN environment variable is required")
		os.Exit(1)
	}

	// Create a structured logger with debug level
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	logger, err := cfg.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync()
	}()

	// Create debug logger for HTTP inspection
	debugLogger := middleware.NewDebugLogger(logger, middleware.LogLevelDebug)

	// Create HTTP client with logging wrapper
	baseHTTPClient := &http.Client{Timeout: 15 * time.Second}
	httpLoggerClient := middleware.NewHTTPLogger(baseHTTPClient, debugLogger)

	// Create YM client with HTTP logging
	ymClient := ym.NewClientWithHTTP(ym.Config{Token: token}, httpLoggerClient)

	// Wrap in SDK client
	cs := client.Wrap(ymClient)

	logger.Info("Starting poll loop with debug logging enabled")

	// Poll for updates
	err = cs.Updates.PollLoop(
		context.Background(),
		updates.GetUpdatesParams{Limit: ptr(10)},
		func(ctx context.Context, update ym.Update) error {
			logger.Info("Processing update",
				zap.Int64("update_id", update.UpdateID),
				zap.Bool("has_message", update.MessageID > 0),
			)

			// Log update with raw data for debugging
			if update.MessageID > 0 && update.Chat != nil && update.From != nil {
				logger.Info("Update has message",
					zap.Int64("message_id", int64(update.MessageID)),
					zap.String("chat_id", string(update.Chat.ID)),
					zap.String("sender", string(update.From.Login)),
					zap.String("text", update.Text),
				)
			} else {
				// Update without message - this is normal
				logger.Warn("Update received without message",
					zap.Int64("update_id", update.UpdateID),
					zap.String("reason", "update may be a non-message event (edit, delete, etc)"),
				)
			}

			return nil
		},
	)

	if err != nil {
		logger.Error("poll loop failed", zap.Error(err))
		os.Exit(1)
	}
}

func ptr[T any](v T) *T {
	return &v
}
