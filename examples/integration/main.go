package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/chats"
	"github.com/rekurt/ymsdk/client/ym/messages"
	"github.com/rekurt/ymsdk/client/ym/polls"
	"github.com/rekurt/ymsdk/client/ym/self"
	"github.com/rekurt/ymsdk/client/ym/updates"
	"github.com/rekurt/ymsdk/client/ym/users"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

// Integration exercise for all SDK methods.
// Configure via env vars before running:
// YM_TOKEN (required), YM_CHAT_ID, YM_LOGIN, YM_FILE_PATH, YM_IMAGE_PATH,
// YM_GALLERY_PATHS (comma-separated), YM_WEBHOOK_URL, YM_CREATE_CHAT_NAME,
// YM_MEMBER_LOGIN, YM_FILE_ID (for getFile).
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
	ctx := context.Background()

	msgSvc := messages.NewService(client)
	chatSvc := chats.NewService(client)
	userSvc := users.NewService(client)
	pollSvc := polls.NewService(client)
	selfSvc := self.NewService(client)
	updateSvc := updates.NewService(client)

	chatID := ym.ChatID(os.Getenv("YM_CHAT_ID"))
	login := ym.UserLogin(os.Getenv("YM_LOGIN"))

	if login != "" {
		if link, err := userSvc.GetUserLink(ctx, login); err != nil {
			log.Printf("getUserLink failed: %v", err)
		} else {
			log.Printf("getUserLink: %+v", link)
		}
	}

	var lastMsgID ym.MessageID
	if chatID != "" {
		if msg, err := msgSvc.SendToChat(ctx, chatID, "integration: hello chat", nil); err != nil {
			log.Printf("send text to chat failed: %v", err)
		} else {
			lastMsgID = msg.ID
			log.Printf("sent text to chat %s message_id=%d", chatID, msg.ID)
		}
	}
	if login != "" {
		if msg, err := msgSvc.SendToLogin(ctx, login, "integration: hello login", nil); err != nil {
			log.Printf("send text to login failed: %v", err)
		} else {
			lastMsgID = msg.ID
			log.Printf("sent text to login %s message_id=%d", login, msg.ID)
		}
	}

	if fp := os.Getenv("YM_FILE_PATH"); fp != "" && chatID != "" {
		if msg, err := sendFile(ctx, msgSvc, chatID, fp); err != nil {
			log.Printf("sendFile failed: %v", err)
		} else {
			lastMsgID = msg.ID
			log.Printf("sendFile ok message_id=%d", msg.ID)
		}
	}

	if ip := os.Getenv("YM_IMAGE_PATH"); ip != "" && chatID != "" {
		if msg, err := sendImage(ctx, msgSvc, chatID, ip); err != nil {
			log.Printf("sendImage failed: %v", err)
		} else {
			lastMsgID = msg.ID
			log.Printf("sendImage ok message_id=%d", msg.ID)
		}
	}

	if gp := os.Getenv("YM_GALLERY_PATHS"); gp != "" && chatID != "" {
		if msg, err := sendGallery(ctx, msgSvc, chatID, gp); err != nil {
			log.Printf("sendGallery failed: %v", err)
		} else {
			lastMsgID = msg.ID
			log.Printf("sendGallery ok message_id=%d", msg.ID)
		}
	}

	if lastMsgID != 0 && chatID != "" {
		if err := msgSvc.Delete(ctx, &messages.DeleteMessageRequest{ChatID: &chatID, MessageID: lastMsgID}); err != nil {
			log.Printf("delete message failed: %v", err)
		} else {
			log.Printf("delete message %d ok", lastMsgID)
		}
	}

	if fid := os.Getenv("YM_FILE_ID"); fid != "" {
		rc, meta, err := msgSvc.GetFile(ctx, fid)
		if err != nil {
			log.Printf("getFile failed: %v", err)
		} else {
			defer rc.Close()
			io.Copy(io.Discard, rc)
			log.Printf("getFile ok id=%s content_type=%s length=%d", meta.FileID, meta.ContentType, meta.ContentLength)
		}
	}

	if chatID != "" {
		msg, err := pollSvc.Create(ctx, &polls.CreatePollRequest{
			ChatID:  &chatID,
			Title:   "integration poll",
			Answers: []string{"Yes", "No"},
		})
		if err != nil {
			log.Printf("create poll failed: %v", err)
		} else {
			log.Printf("poll created message_id=%d", msg.ID)
			if res, err := pollSvc.GetResults(ctx, polls.PollResultsParams{ChatID: &chatID, MessageID: msg.ID}); err != nil {
				log.Printf("getResults failed: %v", err)
			} else {
				log.Printf("getResults ok voted=%d answers=%v", res.VotedCount, res.Answers)
			}
			if voters, err := pollSvc.GetVotersPage(ctx, polls.PollVotersParams{ChatID: &chatID, MessageID: msg.ID, AnswerID: 1, Limit: intPtr(10)}); err != nil {
				log.Printf("getVoters failed: %v", err)
			} else {
				log.Printf("getVoters ok count=%d cursor=%d", voters.VotedCount, voters.Cursor)
			}
		}
	}

	if name := os.Getenv("YM_CREATE_CHAT_NAME"); name != "" {
		channel := strings.ToLower(os.Getenv("YM_CREATE_CHAT_CHANNEL")) == "1"
		req := &chats.ChatCreateRequest{
			Name:        name,
			Description: "integration chat",
			Channel:     channel,
		}
		if chat, err := chatSvc.Create(ctx, req); err != nil {
			log.Printf("create chat failed: %v", err)
		} else {
			log.Printf("create chat ok id=%s", chat.ID)
			if ml := os.Getenv("YM_MEMBER_LOGIN"); ml != "" && !channel {
				err := chatSvc.UpdateMembers(ctx, &chats.ChatUpdateMembersRequest{
					ChatID:  chat.ID,
					Members: []ym.UserRef{{Login: ym.UserLogin(ml)}},
				})
				if err != nil {
					log.Printf("updateMembers failed: %v", err)
				} else {
					log.Printf("updateMembers ok")
				}
			}
		}
	}

	if wh := os.Getenv("YM_WEBHOOK_URL"); wh != "" {
		if selfObj, err := selfSvc.Update(ctx, &self.SelfUpdateRequest{WebhookURL: &wh}); err != nil {
			log.Printf("self.update webhook failed: %v", err)
		} else {
			log.Printf("self.update webhook ok: %+v", selfObj)
		}
	}

	limit := 10
	upds, next, err := updateSvc.GetUpdates(ctx, updates.GetUpdatesParams{Limit: &limit})
	if err != nil {
		log.Printf("getUpdates failed: %v", err)
	} else {
		log.Printf("getUpdates ok updates=%d next_offset=%d", len(upds), next)
		for _, u := range upds {
			if u.MessageID > 0 && u.Chat != nil {
				log.Printf("update %d chat=%s text=%s", u.UpdateID, u.Chat.ID, u.Text)
			}
		}
	}
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

func intPtr(v int) *int { return &v }
