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

	// Run keeps polling through transient API failures instead of exiting on
	// the first one, and returns promptly when ctx is cancelled.
	err := cs.Updates.Run(ctx, updates.RunOptions{
		Limit: ym.Ptr(20),
		OnPollError: func(err error) updates.ErrorAction {
			logger.Warn("poll failed, backing off", zap.Error(err))

			return updates.ActionRetry
		},
		OnHandlerError: func(u ym.Update, err error) updates.ErrorAction {
			logger.Error("handler failed, skipping update",
				zap.Int64("update_id", u.UpdateID), zap.Error(err))

			return updates.ActionContinue
		},
	}, func(ctx context.Context, u ym.Update) error {
		logUpdate(logger, u)

		return nil
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		middleware.LogError(logger, ctx, err, "GET", ym.EndpointMessagesGetUpdates, nil)
		log.Fatalf("poll loop failed: %v", err)
	}

	log.Println("shutdown complete")
}

func logUpdate(logger *zap.Logger, u ym.Update) {
	// Reaction and membership events carry no sender, so they have to be
	// handled before the guard that requires message-shaped fields — otherwise
	// they are only ever reported as missing sender information.
	switch {
	case u.Reaction != nil:
		logger.Info("reaction event",
			zap.String("action", string(u.Reaction.Action)),
			zap.String("reaction", u.Reaction.Reaction.Name),
			zap.Int64("message_id", int64(u.Reaction.MessageID)),
		)

		return

	case u.ChatMembersUpdate != nil:
		logger.Info("membership changed",
			zap.Int("added", len(u.ChatMembersUpdate.NewChatMembers)),
			zap.Int("removed", len(u.ChatMembersUpdate.RemovedChatMembers)),
		)

		return
	}

	if u.Chat == nil || u.From == nil {
		logger.Warn("update without chat/sender info",
			zap.Int64("update_id", u.UpdateID),
		)

		return
	}

	chatID := string(u.Chat.ID)
	sender := string(u.From.Login)

	switch {
	case len(u.ForwardedMessages) > 0:
		log.Printf("[%s] %s forwarded %d message(s)", chatID, sender, len(u.ForwardedMessages))

	case u.Sticker != nil:
		log.Printf("[%s] %s sent sticker: %s (set=%s id=%s)", chatID, sender, u.Sticker.Emoji, u.Sticker.SetID, u.Sticker.ID)

	case len(u.Images) > 0:
		// Each image arrives as a list of size variants; OriginalImages picks
		// the full-size one of each.
		for i, img := range u.OriginalImages() {
			log.Printf("[%s] %s sent image %d: %dx%d (id=%s)", chatID, sender, i+1, img.Width, img.Height, img.FileID)
		}

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
