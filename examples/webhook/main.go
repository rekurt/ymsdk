package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/messages"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

// Minimal webhook receiver: starts HTTP server, parses updates from YM and replies via SendToChat.
// Env:
// YM_TOKEN (required), YM_REPLY_CHAT (optional default from incoming update),
// YM_WEBHOOK_SECRET (required, value expected in X-Webhook-Secret header),
// YM_PORT (default 8080).
func main() {
	token := os.Getenv("YM_TOKEN")
	if token == "" {
		log.Fatal("YM_TOKEN is required")
	}
	webhookSecret := os.Getenv("YM_WEBHOOK_SECRET")
	if webhookSecret == "" {
		log.Fatal("YM_WEBHOOK_SECRET is required")
	}
	port := os.Getenv("YM_PORT")
	if port == "" {
		port = "8080"
	}

	s := client.New(ym.Config{
		Token:       token,
		UpdatesMode: ymerrors.UpdatesModeWebhook,
	})

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Webhook-Secret")), []byte(webhookSecret)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)

			return
		}

		var upd ym.Update
		if err := json.Unmarshal(body, &upd); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)

			return
		}

		log.Printf("got update %d", upd.UpdateID)
		if upd.MessageID > 0 && upd.Chat != nil && upd.From != nil {
			replyChat := os.Getenv("YM_REPLY_CHAT")
			target := upd.Chat.ID
			if replyChat != "" {
				target = ym.ChatID(replyChat)
			}

			_, err := s.Messages.SendToChat(r.Context(), target, "echo: "+upd.Text, &messages.SendMessageOptions{
				ReplyToMessageID: fmt.Sprintf("%d", upd.MessageID),
			})
			if err != nil {
				log.Printf("send reply failed: %v", err)
			}
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"ok":true}`))
		if err != nil {
			log.Printf("write failed: %v", err)

			return
		}
	})

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
