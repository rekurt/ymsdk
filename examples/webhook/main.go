package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/messages"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

// Webhook receiver that parses incoming updates and replies with an echo.
//
// Features shown:
//   - Webhook secret validation with constant-time comparison
//   - Request body size limiting (1 MB)
//   - Handling different update types (text, image, file, sticker)
//   - Reply-to-message for threaded responses
//   - Graceful HTTP server shutdown on SIGINT
//
// Env:
//
//	YM_TOKEN          (required) OAuth bot token
//	YM_WEBHOOK_SECRET (required) shared secret for X-Webhook-Secret header
//	YM_REPLY_CHAT     (optional) override reply chat ID; defaults to incoming chat
//	YM_PORT           (optional) HTTP listen port; defaults to 8080
func main() {
	token := mustEnv("YM_TOKEN")
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

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", webhookHandler(cs, webhookSecret))
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

	go func() {
		<-ctx.Done()
		log.Println("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
			log.Printf("shutdown error: %v", shutdownErr)
		}
	}()

	log.Printf("listening on :%s (endpoints: /webhook, /health)", port)
	if listenErr := srv.ListenAndServe(); listenErr != nil && listenErr != http.ErrServerClosed {
		log.Fatalf("server error: %v", listenErr)
	}

	log.Println("shutdown complete")
}

func webhookHandler(cs *client.YMClient, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

			return
		}

		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Webhook-Secret")), []byte(secret)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			http.Error(w, "bad request", http.StatusBadRequest)

			return
		}

		var upd ym.Update
		if unmarshalErr := json.Unmarshal(body, &upd); unmarshalErr != nil {
			http.Error(w, "bad request", http.StatusBadRequest)

			return
		}

		log.Printf("webhook update %d", upd.UpdateID)
		processUpdate(r.Context(), cs, upd)

		w.WriteHeader(http.StatusOK)
		if _, writeErr := w.Write([]byte(`{"ok":true}`)); writeErr != nil {
			log.Printf("write response failed: %v", writeErr)
		}
	}
}

func processUpdate(ctx context.Context, cs *client.YMClient, upd ym.Update) {
	if upd.Chat == nil || upd.From == nil || upd.MessageID == 0 {
		log.Printf("  skipping: no chat/sender/message info")

		return
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
		// Images arrive as size variants per image; count the images, not the variants.
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
		replyText = "echo: " + upd.Text
	default:
		log.Printf("  skipping: unsupported update type")

		return
	}

	replyID := upd.MessageID
	opts := &messages.SendMessageOptions{
		ReplyToMessageID: &replyID,
	}

	if _, sendErr := cs.Messages.SendToChat(ctx, target, replyText, opts); sendErr != nil {
		log.Printf("  send reply failed: %v", sendErr)
	} else {
		log.Printf("  replied to %s in %s", upd.From.Login, upd.Chat.ID)
	}
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
