package messages

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

// Service provides methods for sending and managing messages in Yandex Messenger.
type Service struct {
	client *ym.Client
}

// NewService creates a new messages Service.
func NewService(client *ym.Client) *Service {
	return &Service{client: client}
}

// SendMessageOptions holds optional parameters for text message sending.
type SendMessageOptions struct {
	PayloadID             string
	ReplyToMessageID      *ym.MessageID
	DisableNotification   *bool
	Important             *bool
	DisableWebPagePreview *bool
	ThreadID              *ym.ThreadID
	SuggestButtons        *ym.SuggestButtons
}

type sendMessageRequest struct {
	ChatID                ym.ChatID          `json:"chat_id,omitempty"`
	Login                 ym.UserLogin       `json:"login,omitempty"`
	Text                  string             `json:"text"`
	PayloadID             string             `json:"payload_id,omitempty"`
	ReplyMessageID        *ym.MessageID      `json:"reply_message_id,omitempty"`
	DisableNotification   *bool              `json:"disable_notification,omitempty"`
	Important             *bool              `json:"important,omitempty"`
	DisableWebPagePreview *bool              `json:"disable_web_page_preview,omitempty"`
	ThreadID              *ym.ThreadID       `json:"thread_id,omitempty"`
	SuggestButtons        *ym.SuggestButtons `json:"suggest_buttons,omitempty"`
}

type sendMessageResponse struct {
	OK        bool         `json:"ok"`
	MessageID ym.MessageID `json:"message_id"`
}

// SendToChat sends a text message to a chat identified by chatID.
func (s *Service) SendToChat(
	ctx context.Context, chatID ym.ChatID, text string, opts *SendMessageOptions,
) (*ym.Message, error) {
	req := buildRequest(text, opts)
	req.ChatID = chatID

	return s.send(ctx, req)
}

// SendToLogin sends a text message to a user identified by login.
func (s *Service) SendToLogin(
	ctx context.Context, login ym.UserLogin, text string, opts *SendMessageOptions,
) (*ym.Message, error) {
	req := buildRequest(text, opts)
	req.Login = login

	return s.send(ctx, req)
}

func (s *Service) send(ctx context.Context, reqBody sendMessageRequest) (*ym.Message, error) {
	resp, err := s.client.DoRequest(ctx, http.MethodPost, "/bot/v1/messages/sendText/", reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed sendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("%w: decode sendText response: %w", ymerrors.ErrInvalidResponse, err)
	}
	if !parsed.OK {
		return nil, fmt.Errorf("%w: ok=false", ymerrors.ErrInvalidResponse)
	}

	return &ym.Message{ID: parsed.MessageID}, nil
}

func buildRequest(text string, opts *SendMessageOptions) sendMessageRequest {
	if opts == nil {
		return sendMessageRequest{Text: text}
	}

	return sendMessageRequest{
		Text:                  text,
		PayloadID:             opts.PayloadID,
		ReplyMessageID:        opts.ReplyToMessageID,
		DisableNotification:   opts.DisableNotification,
		Important:             opts.Important,
		DisableWebPagePreview: opts.DisableWebPagePreview,
		ThreadID:              opts.ThreadID,
		SuggestButtons:        opts.SuggestButtons,
	}
}
