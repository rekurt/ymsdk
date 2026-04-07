package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"time"

	"go.uber.org/zap"

	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/updates"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/middleware"
)

// Demonstrates long-polling for updates with the ymsdk PollLoop.
//
// Features shown:
//   - client.New aggregator with retry/rate-limit configuration
//   - PollLoop for continuous update retrieval
//   - Handling different update types (text, image, file, sticker, forward, gallery)
//   - Graceful shutdown via SIGINT
//   - Structured error logging with middleware
//
// Env: YM_TOKEN (required)
func main() {
	token := os.Getenv("YM_TOKEN")
	if token == "" {
		log.Fatal("YM_TOKEN is required")
	}

	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cs := client.New(ym.Config{
		Token: token,
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{
				MaxAttempts:    5,
				InitialBackoff: 500 * time.Millisecond,
				MaxBackoff:     10 * time.Second,
				RetryNetwork:   true,
			},
			RateLimitHandling: ymerrors.RateLimitHandling{
				UseRetryAfter:  true,
				DefaultBackoff: 2 * time.Second,
			},
		},
	})

	log.Println("polling for updates... (Ctrl+C to stop)")

	err := cs.Updates.PollLoop(ctx, updates.GetUpdatesParams{Limit: ym.Ptr(20)},
		func(ctx context.Context, u ym.Update) error {
			logUpdate(logger, u)

			return nil
		},
	)
	if err != nil && !errors.Is(err, context.Canceled) {
		middleware.LogError(logger, ctx, err, "GET", "/bot/v1/messages/getUpdates", nil)
		log.Fatalf("poll loop failed: %v", err)
	}

	log.Println("shutdown complete")
}

func logUpdate(logger *zap.Logger, u ym.Update) {
	if u.Chat == nil || u.From == nil {
		logger.Warn("update without chat/sender info",
			zap.Int64("update_id", u.UpdateID),
		)

		return
	}

	chatID := string(u.Chat.ID)
	sender := string(u.From.Login)

	switch {
	case u.Forward != nil:
		fwdFrom := "unknown"
		if u.Forward.From != nil {
			fwdFrom = string(u.Forward.From.Login)
		}
		log.Printf("[%s] %s forwarded from %s: %s", chatID, sender, fwdFrom, u.Text)

	case u.Sticker != nil:
		log.Printf("[%s] %s sent sticker: %s (id=%s)", chatID, sender, u.Sticker.Emoji, u.Sticker.ID)

	case len(u.Images) > 0:
		log.Printf("[%s] %s sent gallery with %d images", chatID, sender, len(u.Images))

	case u.Image != nil:
		log.Printf("[%s] %s sent image: %dx%d (id=%s)", chatID, sender, u.Image.Width, u.Image.Height, u.Image.FileID)

	case u.Document != nil:
		log.Printf("[%s] %s sent file: %s (%s, %d bytes)", chatID, sender, u.Document.Name, u.Document.MimeType, u.Document.Size)

	case u.Text != "":
		log.Printf("[%s] %s: %s", chatID, sender, u.Text)
		if u.ThreadID != nil {
			log.Printf("  └─ in thread %d", *u.ThreadID)
		}

	default:
		log.Printf("[%s] %s: (empty update, possibly edit/delete event)", chatID, sender)
	}
}
