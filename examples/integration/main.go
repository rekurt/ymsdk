package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/chats"
	"github.com/rekurt/ymsdk/client/ym/messages"
	"github.com/rekurt/ymsdk/client/ym/polls"
	"github.com/rekurt/ymsdk/client/ym/self"
	"github.com/rekurt/ymsdk/client/ym/updates"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

// Integration exercise for all SDK methods via the client.New aggregator.
//
// Features shown:
//   - All major SDK operations: messages, files, images, galleries, polls, chats, users, self, updates
//   - client.New aggregator for single-point initialization
//   - Structured error handling with API error details
//   - Graceful shutdown via SIGINT
//
// Configure via env vars before running:
//
//	YM_TOKEN              (required) OAuth bot token
//	YM_CHAT_ID            chat for messages/polls/files
//	YM_LOGIN              user login for DMs and user link
//	YM_FILE_PATH          local file for sendFile
//	YM_IMAGE_PATH         local image for sendImage
//	YM_GALLERY_PATHS      comma-separated paths for sendGallery
//	YM_FILE_ID            file_id for getFile download
//	YM_WEBHOOK_URL        webhook URL for self.update
//	YM_CREATE_CHAT_NAME   create chat (YM_CREATE_CHAT_CHANNEL=1 for channel)
//	YM_MEMBER_LOGIN       user to add to created chat
func main() {
	token := os.Getenv("YM_TOKEN")
	if token == "" {
		log.Fatal("YM_TOKEN is required")
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

	chatID := ym.ChatID(os.Getenv("YM_CHAT_ID"))
	login := ym.UserLogin(os.Getenv("YM_LOGIN"))

	// --- User Link ---
	if login != "" {
		section("getUserLink")
		link, err := cs.Users.GetUserLink(ctx, login)
		if err != nil {
			logErr("getUserLink", err)
		} else {
			log.Printf("  chat_link=%s call_link=%s", link.ChatLink, link.CallLink)
		}
	}

	// --- Send Text ---
	var lastMsgID ym.MessageID
	if chatID != "" {
		section("sendText to chat")
		msg, err := cs.Messages.SendToChat(ctx, chatID, "integration: hello from ymsdk", &messages.SendMessageOptions{
			MarkImportant: true,
		})
		if err != nil {
			logErr("sendText(chat)", err)
		} else {
			lastMsgID = msg.ID
			log.Printf("  ✓ message_id=%d chat=%s", msg.ID, chatID)
		}
	}
	if login != "" {
		section("sendText to login")
		msg, err := cs.Messages.SendToLogin(ctx, login, "integration: hello via DM", nil)
		if err != nil {
			logErr("sendText(login)", err)
		} else {
			lastMsgID = msg.ID
			log.Printf("  ✓ message_id=%d login=%s", msg.ID, login)
		}
	}

	// --- Send File ---
	if fp := os.Getenv("YM_FILE_PATH"); fp != "" && chatID != "" {
		section("sendFile")
		msg, err := sendFile(ctx, cs.Messages, chatID, fp)
		if err != nil {
			logErr("sendFile", err)
		} else {
			lastMsgID = msg.ID
			log.Printf("  ✓ message_id=%d file=%s", msg.ID, filepath.Base(fp))
		}
	}

	// --- Send Image ---
	if ip := os.Getenv("YM_IMAGE_PATH"); ip != "" && chatID != "" {
		section("sendImage")
		msg, err := sendImage(ctx, cs.Messages, chatID, ip)
		if err != nil {
			logErr("sendImage", err)
		} else {
			lastMsgID = msg.ID
			log.Printf("  ✓ message_id=%d image=%s", msg.ID, filepath.Base(ip))
		}
	}

	// --- Send Gallery ---
	if gp := os.Getenv("YM_GALLERY_PATHS"); gp != "" && chatID != "" {
		section("sendGallery")
		msg, err := sendGallery(ctx, cs.Messages, chatID, gp)
		if err != nil {
			logErr("sendGallery", err)
		} else {
			lastMsgID = msg.ID
			log.Printf("  ✓ message_id=%d images=%d", msg.ID, len(strings.Split(gp, ",")))
		}
	}

	// --- Delete Message ---
	if lastMsgID != 0 && chatID != "" {
		section("deleteMessage")
		if err := cs.Messages.Delete(ctx, &messages.DeleteMessageRequest{ChatID: &chatID, MessageID: lastMsgID}); err != nil {
			logErr("delete", err)
		} else {
			log.Printf("  ✓ deleted message_id=%d", lastMsgID)
		}
	}

	// --- Get File ---
	if fid := os.Getenv("YM_FILE_ID"); fid != "" {
		section("getFile")
		rc, meta, err := cs.Messages.GetFile(ctx, fid)
		if err != nil {
			logErr("getFile", err)
		} else {
			n, _ := io.Copy(io.Discard, rc)
			_ = rc.Close()
			log.Printf("  ✓ file_id=%s content_type=%s declared=%d read=%d",
				meta.FileID, meta.ContentType, meta.ContentLength, n)
		}
	}

	// --- Polls ---
	if chatID != "" {
		section("createPoll")
		msg, err := cs.Polls.Create(ctx, &polls.CreatePollRequest{
			ChatID:      &chatID,
			Title:       "Integration poll: pick one",
			Answers:     []string{"Option A", "Option B", "Option C"},
			IsAnonymous: ym.Ptr(false),
		})
		if err != nil {
			logErr("createPoll", err)
		} else {
			log.Printf("  ✓ poll message_id=%d", msg.ID)

			section("getResults")
			if res, resErr := cs.Polls.GetResults(ctx, polls.PollResultsParams{ChatID: &chatID, MessageID: msg.ID}); resErr != nil {
				logErr("getResults", resErr)
			} else {
				log.Printf("  ✓ voted=%d answers=%v", res.VotedCount, res.Answers)
			}

			section("getVoters (answer #1)")
			if voters, votersErr := cs.Polls.GetVotersPage(ctx, polls.PollVotersParams{
				ChatID:    &chatID,
				MessageID: msg.ID,
				AnswerID:  1,
				Limit:     ym.Ptr(10),
			}); votersErr != nil {
				logErr("getVoters", votersErr)
			} else {
				log.Printf("  ✓ answer_1_voted=%d cursor=%d", voters.VotedCount, voters.Cursor)
			}
		}
	}

	// --- Create Chat/Channel ---
	if name := os.Getenv("YM_CREATE_CHAT_NAME"); name != "" {
		isChannel := strings.EqualFold(os.Getenv("YM_CREATE_CHAT_CHANNEL"), "1")
		kind := "chat"
		if isChannel {
			kind = "channel"
		}
		section("createChat (" + kind + ")")

		chat, err := cs.Chats.Create(ctx, &chats.ChatCreateRequest{
			Name:        name,
			Description: "integration " + kind,
			Channel:     isChannel,
		})
		if err != nil {
			logErr("createChat", err)
		} else {
			log.Printf("  ✓ %s id=%s", kind, chat.ID)

			if ml := os.Getenv("YM_MEMBER_LOGIN"); ml != "" && !isChannel {
				section("updateMembers")
				if memberErr := cs.Chats.UpdateMembers(ctx, &chats.ChatUpdateMembersRequest{
					ChatID:  chat.ID,
					Members: []ym.UserRef{{Login: ym.UserLogin(ml)}},
				}); memberErr != nil {
					logErr("updateMembers", memberErr)
				} else {
					log.Printf("  ✓ added %s to %s", ml, chat.ID)
				}
			}
		}
	}

	// --- Webhook Setup ---
	if wh := os.Getenv("YM_WEBHOOK_URL"); wh != "" {
		section("self.update (webhook)")
		bot, err := cs.Self.Update(ctx, &self.SelfUpdateRequest{WebhookURL: &wh})
		if err != nil {
			logErr("self.update", err)
		} else {
			log.Printf("  ✓ bot=%s display=%s webhook=%v", bot.ID, bot.DisplayName, bot.WebhookURL)
		}
	}

	// --- Get Updates ---
	section("getUpdates")
	upds, next, err := cs.Updates.GetUpdates(ctx, updates.GetUpdatesParams{Limit: ym.Ptr(10)})
	if err != nil {
		logErr("getUpdates", err)
	} else {
		log.Printf("  ✓ updates=%d next_offset=%d", len(upds), next)
		for _, u := range upds {
			if u.Chat != nil {
				log.Printf("    update %d: chat=%s text=%q", u.UpdateID, u.Chat.ID, truncate(u.Text, 80))
			}
		}
	}

	log.Println("\n=== integration complete ===")
}

func section(name string) {
	log.Printf("\n--- %s ---", name)
}

func logErr(op string, err error) {
	var apiErr *ymerrors.APIError
	if errors.As(err, &apiErr) {
		log.Printf("  ✗ %s: API error kind=%d http=%d desc=%q request_id=%s",
			op, apiErr.Kind, apiErr.HTTPStatus, apiErr.Description, apiErr.RequestID)
		if apiErr.RetryAfter > 0 {
			log.Printf("    retry_after=%s", apiErr.RetryAfter)
		}

		return
	}

	log.Printf("  ✗ %s: %v", op, err)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen] + "..."
}

func sendFile(ctx context.Context, svc *messages.Service, chatID ym.ChatID, path string) (*ym.Message, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return svc.SendFile(ctx, &messages.SendFileRequest{
		ChatID:   &chatID,
		Document: bytes.NewReader(data),
		Filename: filepath.Base(path),
	})
}

func sendImage(ctx context.Context, svc *messages.Service, chatID ym.ChatID, path string) (*ym.Message, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return svc.SendImage(ctx, &messages.SendImageRequest{
		ChatID:   &chatID,
		Image:    bytes.NewReader(data),
		Filename: filepath.Base(path),
	})
}

func sendGallery(ctx context.Context, svc *messages.Service, chatID ym.ChatID, paths string) (*ym.Message, error) {
	var parts []messages.FilePart
	for _, p := range strings.Split(paths, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		parts = append(parts, messages.FilePart{
			Reader:   bytes.NewReader(data),
			Filename: filepath.Base(p),
		})
	}

	return svc.SendGallery(ctx, &messages.SendGalleryRequest{
		ChatID: &chatID,
		Images: parts,
	})
}
