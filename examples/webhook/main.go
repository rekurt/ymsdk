package main

import (
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
// YM_PORT (default 8080).
func main() {
	token := os.Getenv("YM_TOKEN")
	if token == "" {
		log.Fatal("YM_TOKEN is required")
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
		body, _ := io.ReadAll(r.Body)

		var upd ym.Update
		if err := json.Unmarshal(body, &upd); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)

			return
		}

		log.Printf("got update %d", upd.UpdateID)
		if upd.Message != nil {
			replyChat := os.Getenv("YM_REPLY_CHAT")
			target := upd.Message.Chat.ID
			if replyChat != "" {
				target = ym.ChatID(replyChat)
			}
			_, err := s.Messages.SendToChat(r.Context(), target, "echo: "+upd.Message.Text, &messages.SendMessageOptions{
				ReplyToMessageID: fmt.Sprintf("%d", upd.Message.ID),
			})
			if err != nil {
				log.Printf("send reply failed: %v", err)
			}
		}
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"ok":true}`))
		if err != nil {
			log.Printf("write failed: %v", err)

			return
		}
	})

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
