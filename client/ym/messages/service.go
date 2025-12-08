package messages

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

type Service struct {
	client *ym.Client
}

func NewService(client *ym.Client) *Service {
	return &Service{client: client}
}

type SendMessageOptions struct {
	MarkImportant    bool
	ReplyToMessageID string
}

type sendMessageRequest struct {
	ChatID           ym.ChatID    `json:"chat_id,omitempty"`
	Login            ym.UserLogin `json:"login,omitempty"`
	Text             string       `json:"text"`
	MarkImportant    bool         `json:"mark_important,omitempty"`
	ReplyToMessageID string       `json:"reply_to_message_id,omitempty"`
}

type sendMessageResponse struct {
	OK      bool        `json:"ok"`
	Message *ym.Message `json:"message"`
}

func (s *Service) SendToChat(
	ctx context.Context, chatID ym.ChatID, text string, opts *SendMessageOptions,
) (*ym.Message, error) {
	req := buildRequest(text, opts)
	req.ChatID = chatID

	return s.send(ctx, req)
}

func (s *Service) SendToLogin(
	ctx context.Context, login ym.UserLogin, text string, opts *SendMessageOptions,
) (*ym.Message, error) {
	req := buildRequest(text, opts)
	req.Login = login

	return s.send(ctx, req)
}

func (s *Service) send(ctx context.Context, reqBody sendMessageRequest) (*ym.Message, error) {
	resp, err := s.client.DoRequest(ctx, http.MethodPost, "/bot/v1/messages/sendText", reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed sendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("%w: decode sendText response: %w", ymerrors.ErrInvalidResponse, err)
	}
	if !parsed.OK || parsed.Message == nil {
		return nil, fmt.Errorf(
			"%w: ok=%v message_present=%v", ymerrors.ErrInvalidResponse, parsed.OK, parsed.Message != nil,
		)
	}

	return parsed.Message, nil
}

func buildRequest(text string, opts *SendMessageOptions) sendMessageRequest {
	if opts == nil {
		return sendMessageRequest{Text: text}
	}

	return sendMessageRequest{
		Text:             text,
		MarkImportant:    opts.MarkImportant,
		ReplyToMessageID: opts.ReplyToMessageID,
	}
}
