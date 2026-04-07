package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/polls"
	"github.com/rekurt/ymsdk/client/ym/updates"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

// Demonstrates poll creation, result fetching, and voter listing.
//
// Features shown:
//   - Creating a poll with multiple answers and options
//   - Fetching aggregated poll results via GetResults
//   - Paginating individual voters via GetVotersPage
//   - Collecting all voters via GetAllVoters helper
//   - Polling for new updates after poll creation
//   - Graceful shutdown via SIGINT
//
// Env: YM_TOKEN (required), YM_CHAT_ID (required)
func main() {
	token := os.Getenv("YM_TOKEN")
	chat := os.Getenv("YM_CHAT_ID")
	if token == "" || chat == "" {
		log.Fatal("YM_TOKEN and YM_CHAT_ID are required")
	}

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

	chatID := ym.ChatID(chat)

	// --- Create poll ---
	log.Println("creating poll...")
	msg, err := cs.Polls.Create(ctx, &polls.CreatePollRequest{
		ChatID:      &chatID,
		Title:       "What's your favourite language?",
		Answers:     []string{"Go", "Rust", "Python", "TypeScript"},
		MaxChoices:  ym.Ptr(2),
		IsAnonymous: ym.Ptr(false),
	})
	if err != nil {
		log.Fatalf("create poll failed: %v", err)
	}
	log.Printf("✓ poll created — message_id=%d", msg.ID)

	// --- Fetch results ---
	log.Println("fetching poll results...")
	results, err := cs.Polls.GetResults(ctx, polls.PollResultsParams{
		ChatID:    &chatID,
		MessageID: msg.ID,
	})
	if err != nil {
		log.Printf("✗ getResults failed: %v", err)
	} else {
		log.Printf("✓ poll results: total_voted=%d", results.VotedCount)
		answers := []string{"Go", "Rust", "Python", "TypeScript"}
		for id, count := range results.Answers {
			name := "unknown"
			if id > 0 && id <= len(answers) {
				name = answers[id-1]
			}
			log.Printf("  %s: %d votes", name, count)
		}
	}

	// --- Fetch voters for first answer ---
	log.Println("fetching voters for answer #1...")
	voters, err := cs.Polls.GetAllVoters(ctx, polls.PollVotersParams{
		ChatID:    &chatID,
		MessageID: msg.ID,
		AnswerID:  1,
		Limit:     ym.Ptr(50),
	})
	if err != nil {
		log.Printf("✗ getAllVoters failed: %v", err)
	} else {
		log.Printf("✓ answer #1 voters: %d total", len(voters))
		for _, v := range voters {
			log.Printf("  - %s (ts=%d)", v.User.Login, v.Timestamp)
		}
	}

	// --- Poll for updates ---
	log.Println("polling for updates (3 rounds)...")
	var offset int64
	for range 3 {
		select {
		case <-ctx.Done():
			log.Println("interrupted")

			return
		default:
		}

		limit := 20
		upds, next, updateErr := cs.Updates.GetUpdates(ctx, updates.GetUpdatesParams{
			Limit:  &limit,
			Offset: &offset,
		})
		if updateErr != nil {
			log.Printf("✗ getUpdates: %v", updateErr)

			break
		}

		for _, u := range upds {
			if u.MessageID > 0 && u.Chat != nil {
				log.Printf("  update %d: chat=%s text=%q", u.UpdateID, u.Chat.ID, u.Text)
			}
		}

		offset = next
		time.Sleep(2 * time.Second)
	}

	log.Println("done")
}
