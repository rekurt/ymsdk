package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/updates"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/middleware"
)

func main() {
	token := os.Getenv("YM_TOKEN")
	if token == "" {
		log.Fatal("YM_TOKEN is required")
	}

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
	updateSvc := updates.NewService(client)
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	ctx := middleware.WithRequestID(context.Background(), "poller")
	offset := ""

	for {
		upds, nextOffset, err := updateSvc.Get(ctx, 20, offset)
		if err != nil {
			if handleAPIError(err) {
				middleware.LogError(logger, ctx, err, "GET", "/bot/v1/messages/getUpdates", map[string]any{"offset": offset})

				continue
			}
			log.Fatalf("get updates failed: %v", err)
		}

		for _, u := range upds {
			if u.Message != nil {
				fmt.Printf("[%s] %s: %s\n", u.Message.Chat.ID, u.Message.From.Login, u.Message.Text)
			}
		}

		offset = nextOffset
		time.Sleep(time.Second)
	}
}

func handleAPIError(err error) bool {
	var apiErr *ymerrors.APIError
	if errors.As(err, &apiErr) {
		fmt.Printf("api error kind=%d http=%d desc=%s\n", apiErr.Kind, apiErr.HTTPStatus, apiErr.Description)
		if errors.Is(err, ymerrors.ErrRateLimited) && apiErr.RetryAfter > 0 {
			time.Sleep(apiErr.RetryAfter)

			return true
		}
		if errors.Is(err, ymerrors.ErrInvalidToken) || errors.Is(err, ymerrors.ErrUnauthorized) {
			return false
		}

		return true
	}

	return false
}
