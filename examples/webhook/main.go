package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/messages"
	"github.com/rekurt/ymsdk/client/ym/updates"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

// Webhook receiver that parses incoming updates and replies with an echo.
//
// The API's delivery contract shapes this example:
//
//   - It allows 100ms to connect and 1s to respond. Replying to a message means
//     calling the API, which takes far longer than that, so updates.WebhookHandler
//     acknowledges the delivery immediately and does the work on a worker pool.
//     Handling deliveries inline would time out and trigger endless retries.
//   - Delivery is at-least-once, so the same update arrives more than once. The
//     handler remembers recent update IDs and drops repeats.
//   - Nothing about a delivery is signed and no custom headers are sent, so the
//     only credential is the webhook URL itself. Keep the path unguessable and
//     optionally add ?secret=… to it.
//
// Env:
//
//	YM_TOKEN          (required) OAuth bot token
//	YM_WEBHOOK_SECRET (required) value expected in the ?secret= query parameter
//	YM_REPLY_CHAT     (optional) override reply chat ID; defaults to incoming chat
//	YM_PORT           (optional) HTTP listen port; defaults to 8080
func main() {
	token := mustEnv("YM_TOKEN")
	// Required, not optional: without it the route below accepts any request,
	// and a forged Update would make the bot send an authenticated reply to a
	// chat of the caller's choosing.
	webhookSecret := mustEnv("YM_WEBHOOK_SECRET")
	port := envOrDefault("YM_PORT", "8080")

	cs := client.New(ym.Config{
		Token:       token,
		UpdatesMode: ymerrors.UpdatesModeWebhook,
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{
				MaxAttempts:    2,
				InitialBackoff: 300 * time.Millisecond,
				MaxBackoff:     2 * time.Second,
			},
			RateLimitHandling: ymerrors.RateLimitHandling{
				UseRetryAfter:  true,
				DefaultBackoff: time.Second,
			},
		},
	})

	hook := updates.NewWebhookHandler(
		func(ctx context.Context, upd ym.Update) error {
			return processUpdate(ctx, cs, upd)
		},
		updates.WebhookOptions{
			Secret:  webhookSecret,
			Workers: 8,
			OnError: func(err error) { log.Printf("webhook: %v", err) },
		},
	)

	mux := http.NewServeMux()
	// The secret is the credential here. Registering the bot on an unguessable
	// path as well costs nothing and keeps the secret out of access logs.
	mux.Handle("/webhook", hook)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Closed once the drain has finished. Shutting the server down makes
	// ListenAndServe return immediately, so main has to wait for this or the
	// process exits while accepted updates are still being processed — losing
	// exactly the work the early acknowledgement promised to do.
	drained := make(chan struct{})

	go func() {
		defer close(drained)

		<-ctx.Done()
		log.Println("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
			log.Printf("http shutdown error: %v", shutdownErr)
		}
		// Drain updates already accepted but not yet processed.
		if drainErr := hook.Shutdown(shutdownCtx); drainErr != nil {
			log.Printf("webhook drain error: %v", drainErr)
		}
	}()

	log.Printf("listening on :%s (endpoints: /webhook, /health)", port)
	if listenErr := srv.ListenAndServe(); listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
		log.Fatalf("server error: %v", listenErr)
	}

	<-drained
	log.Println("shutdown complete")
}

func processUpdate(ctx context.Context, cs *client.YMClient, upd ym.Update) error {
	if upd.Chat == nil || upd.From == nil || upd.MessageID == 0 {
		log.Printf("update %d: no chat/sender/message info, skipping", upd.UpdateID)

		return nil
	}

	target := upd.Chat.ID
	if override := os.Getenv("YM_REPLY_CHAT"); override != "" {
		target = ym.ChatID(override)
	}

	var replyText string
	switch {
	case upd.Sticker != nil:
		replyText = fmt.Sprintf("Nice sticker! %s", upd.Sticker.Emoji)
	case len(upd.Images) > 0:
		// Images arrive as size variants per image; count images, not variants.
		originals := upd.OriginalImages()
		if len(originals) == 1 {
			replyText = fmt.Sprintf("Got your image (%dx%d)", originals[0].Width, originals[0].Height)
		} else {
			replyText = fmt.Sprintf("Got %d images in gallery", len(originals))
		}
	case upd.Document != nil:
		replyText = fmt.Sprintf("Got file: %s (%d bytes)", upd.Document.Name, upd.Document.Size)
	case len(upd.ForwardedMessages) > 0:
		replyText = fmt.Sprintf("Got %d forwarded message(s)", len(upd.ForwardedMessages))
	case upd.Text != "":
		// Incoming text is attacker-controlled: ** and __ in it would otherwise
		// be rendered as formatting the bot appears to have authored.
		replyText = "echo: " + ym.EscapeMarkdown(upd.Text)
	default:
		log.Printf("update %d: unsupported type, skipping", upd.UpdateID)

		return nil
	}

	opts := &messages.SendMessageOptions{}
	// reply_message_id must name a message from the target chat, so a redirected
	// reply has to be sent on its own or the API rejects it.
	if target == upd.Chat.ID {
		replyID := upd.MessageID
		opts.ReplyToMessageID = &replyID
	}

	if _, err := cs.Messages.SendToChat(ctx, target, replyText, opts); err != nil {
		return fmt.Errorf("reply to %s in %s: %w", upd.From.Login, upd.Chat.ID, err)
	}
	log.Printf("replied to %s in %s", upd.From.Login, upd.Chat.ID)

	return nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is required", key)
	}

	return v
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}
