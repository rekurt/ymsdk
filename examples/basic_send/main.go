package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/messages"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/middleware"
)

func main() {
	token := os.Getenv("YM_TOKEN")
	if token == "" {
		log.Fatal("YM_TOKEN is required")
	}

	chatID := flag.String("chat", "", "chat id to send message")
	login := flag.String("login", "", "user login to send message")
	text := flag.String("text", "Hello from ymsdk", "text to send")
	flag.Parse()

	cfg := ym.Config{
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
	}

	client := ym.NewClient(cfg)
	msgSvc := messages.NewService(client)
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	ctx := middleware.WithRequestID(context.Background(), "sample-req")

	if *chatID != "" {
		if msg, err := msgSvc.SendToChat(ctx, ym.ChatID(*chatID), *text, nil); err != nil {
			middleware.LogError(logger, ctx, err, "POST", "/bot/v1/messages/sendText", map[string]any{"chat_id": *chatID})
			handleError(err)
		} else {
			fmt.Printf("sent to chat %s message %d\n", *chatID, msg.ID)
		}
	}

	if *login != "" {
		if msg, err := msgSvc.SendToLogin(ctx, ym.UserLogin(*login), *text, nil); err != nil {
			middleware.LogError(logger, ctx, err, "POST", "/bot/v1/messages/sendText", map[string]any{"login": *login})
			handleError(err)
		} else {
			fmt.Printf("sent to user %s message %d\n", *login, msg.ID)
		}
	}
}

func handleError(err error) {
	var apiErr *ymerrors.APIError
	if errors.As(err, &apiErr) {
		fmt.Printf("api error: kind=%d http=%d desc=%s", apiErr.Kind, apiErr.HTTPStatus, apiErr.Description)
		if errors.Is(err, ymerrors.ErrRateLimited) && apiErr.RetryAfter > 0 {
			fmt.Printf(" retry after %s", apiErr.RetryAfter)
		}
		fmt.Println()

		return
	}
	fmt.Printf("unexpected error: %v\n", err)
}
