package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/polls"
	"github.com/rekurt/ymsdk/client/ym/updates"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

func main() {
	token := os.Getenv("YM_TOKEN")
	chat := os.Getenv("YM_CHAT_ID")
	if token == "" || chat == "" {
		log.Fatal("YM_TOKEN and YM_CHAT_ID required")
	}

	cfg := ym.Config{
		Token: token,
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy:     ymerrors.RetryStrategy{MaxAttempts: 3, RetryNetwork: true, InitialBackoff: 500 * time.Millisecond, MaxBackoff: 3 * time.Second},
			RateLimitHandling: ymerrors.RateLimitHandling{UseRetryAfter: true, DefaultBackoff: time.Second},
		},
	}
	client := ym.NewClient(cfg)
	pollSvc := polls.NewService(client)
	updateSvc := updates.NewService(client)

	ctx := context.Background()
	msg, err := pollSvc.Create(ctx, &polls.CreatePollRequest{
		ChatID:  ptrChat(ym.ChatID(chat)),
		Title:   "Tea or coffee?",
		Answers: []string{"Tea", "Coffee"},
	})
	if err != nil {
		log.Fatalf("create poll failed: %v", err)
	}
	fmt.Printf("Poll sent with message id %d\n", msg.ID)

	// simple polling for results
	offset := int64(0)
	limit := 20
	for i := 0; i < 3; i++ {
		upds, next, err := updateSvc.GetUpdates(ctx, updates.GetUpdatesParams{Limit: &limit, Offset: &offset})
		if err != nil {
			log.Fatalf("get updates: %v", err)
		}
		for _, u := range upds {
			if u.Message != nil {
				fmt.Printf("update %d text=%s\n", u.UpdateID, u.Message.Text)
			}
		}
		offset = next
		time.Sleep(time.Second)
	}
}

func ptrChat(id ym.ChatID) *ym.ChatID {
	return &id
}
