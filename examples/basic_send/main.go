package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/messages"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/middleware"
)

// Demonstrates sending text messages via the ymsdk aggregator.
//
// Features shown:
//   - client.New aggregator for convenient multi-service access
//   - Sending to chat and/or user login
//   - Reply-to-message and mark-important options
//   - Structured error handling with ymerrors.APIError
//   - Graceful shutdown via signal context
//
// Env: YM_TOKEN (required)
// Flags: -chat, -login, -text, -reply-to, -important
func main() {
	token := os.Getenv("YM_TOKEN")
	if token == "" {
		log.Fatal("YM_TOKEN is required")
	}

	chatID := flag.String("chat", "", "chat ID to send message to")
	login := flag.String("login", "", "user login to send message to")
	text := flag.String("text", "Hello from ymsdk 👋", "message text")
	replyTo := flag.String("reply-to", "", "message ID to reply to (optional)")
	important := flag.Bool("important", false, "mark message as important")
	flag.Parse()

	if *chatID == "" && *login == "" {
		log.Fatal("at least one of -chat or -login is required")
	}

	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cs := client.New(ym.Config{
		Token: token,
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{
				MaxAttempts:    3,
				InitialBackoff: 500 * time.Millisecond,
				MaxBackoff:     5 * time.Second,
				RetryNetwork:   true,
			},
			RateLimitHandling: ymerrors.RateLimitHandling{
				UseRetryAfter:  true,
				DefaultBackoff: time.Second,
			},
		},
	})

	var opts *messages.SendMessageOptions
	if *replyTo != "" || *important {
		opts = &messages.SendMessageOptions{}
		if *important {
			opts.Important = ym.Ptr(true)
		}
		if *replyTo != "" {
			v, err := strconv.ParseInt(*replyTo, 10, 64)
			if err != nil {
				log.Fatalf("invalid reply-to message ID: %v", err)
			}
			mid := ym.MessageID(v)
			opts.ReplyToMessageID = &mid
		}
	}

	reqCtx := middleware.WithRequestID(ctx, "basic-send")

	if *chatID != "" {
		msg, err := cs.Messages.SendToChat(reqCtx, ym.ChatID(*chatID), *text, opts)
		if err != nil {
			middleware.LogError(logger, reqCtx, err, "POST", "/bot/v1/messages/sendText", map[string]any{"chat_id": *chatID})
			handleError(err)
		} else {
			log.Printf("✓ sent to chat %s — message_id=%d", *chatID, msg.ID)
		}
	}

	if *login != "" {
		msg, err := cs.Messages.SendToLogin(reqCtx, ym.UserLogin(*login), *text, opts)
		if err != nil {
			middleware.LogError(logger, reqCtx, err, "POST", "/bot/v1/messages/sendText", map[string]any{"login": *login})
			handleError(err)
		} else {
			log.Printf("✓ sent to user %s — message_id=%d", *login, msg.ID)
		}
	}
}

func handleError(err error) {
	var apiErr *ymerrors.APIError
	if errors.As(err, &apiErr) {
		log.Printf("✗ API error: kind=%d http=%d desc=%q", apiErr.Kind, apiErr.HTTPStatus, apiErr.Description)
		if apiErr.RequestID != "" {
			log.Printf("  request_id=%s", apiErr.RequestID)
		}
		if errors.Is(err, ymerrors.ErrRateLimited) && apiErr.RetryAfter > 0 {
			log.Printf("  rate limited — retry after %s", apiErr.RetryAfter)
		}
		if errors.Is(err, ymerrors.ErrInvalidToken) {
			log.Printf("  hint: check that YM_TOKEN is valid and not expired")
		}

		return
	}

	if errors.Is(err, context.Canceled) {
		log.Println("request cancelled (interrupted)")

		return
	}

	log.Printf("✗ unexpected error: %v", err)
}
