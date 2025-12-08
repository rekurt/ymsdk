package chats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

const (
	maxAdminsChat      = 100
	maxMembersChat     = 500
	maxSubscribersChat = 500
)

type Service struct {
	client *ym.Client
}

func NewService(client *ym.Client) *Service {
	return &Service{client: client}
}

type ChatCreateRequest struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	AvatarURL   *string      `json:"avatar_url,omitempty"`
	Channel     bool         `json:"channel"`
	Admins      []ym.UserRef `json:"admins,omitempty"`
	Members     []ym.UserRef `json:"members,omitempty"`
	Subscribers []ym.UserRef `json:"subscribers,omitempty"`
}

type chatCreateResponse struct {
	OK      bool      `json:"ok"`
	ChatID  ym.ChatID `json:"chat_id"`
	Chat    *ym.Chat  `json:"chat,omitempty"`
	Message string    `json:"description,omitempty"`
}

func (s *Service) Create(ctx context.Context, req *ChatCreateRequest) (*ym.Chat, error) {
	if err := validateCreate(req); err != nil {
		return nil, err
	}

	resp, err := s.client.DoRequest(ctx, http.MethodPost, "/bot/v1/chats/create/", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed chatCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("%w: decode create chat response: %w", ymerrors.ErrInvalidResponse, err)
	}
	if !parsed.OK {
		return nil, &ymerrors.APIError{
			Kind:        ymerrors.KindBadRequest,
			HTTPStatus:  resp.StatusCode,
			Description: parsed.Message,
			Method:      http.MethodPost,
			Endpoint:    "/bot/v1/chats/create/",
		}
	}

	chat := parsed.Chat
	if chat == nil {
		chat = &ym.Chat{
			ID:          parsed.ChatID,
			Title:       req.Name,
			Description: req.Description,
			IsChannel:   req.Channel,
		}
	}

	return chat, nil
}

type ChatUpdateMembersRequest struct {
	ChatID      ym.ChatID    `json:"chat_id"`
	Members     []ym.UserRef `json:"members,omitempty"`
	Admins      []ym.UserRef `json:"admins,omitempty"`
	Subscribers []ym.UserRef `json:"subscribers,omitempty"`
	Remove      []ym.UserRef `json:"remove,omitempty"`
}

type chatUpdateResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

func (s *Service) UpdateMembers(ctx context.Context, req *ChatUpdateMembersRequest) error {
	if err := validateUpdateMembers(req); err != nil {
		return err
	}

	resp, err := s.client.DoRequest(ctx, http.MethodPost, "/bot/v1/chats/updateMembers/", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var parsed chatUpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("%w: decode updateMembers response: %w", ymerrors.ErrInvalidResponse, err)
	}
	if !parsed.OK {
		return &ymerrors.APIError{
			Kind:        ymerrors.KindBadRequest,
			HTTPStatus:  resp.StatusCode,
			Description: parsed.Description,
			Method:      http.MethodPost,
			Endpoint:    "/bot/v1/chats/updateMembers/",
		}
	}

	return nil
}

func validateCreate(req *ChatCreateRequest) error {
	if req == nil {
		return errors.New("chat create request is nil")
	}
	if req.Name == "" {
		return errors.New("chat name is required")
	}
	if req.Channel {
		if len(req.Members) > 0 {
			return errors.New("members must be empty when creating a channel")
		}
		if len(req.Subscribers) > maxSubscribersChat {
			return fmt.Errorf("subscribers limit exceeded: %d", len(req.Subscribers))
		}
	} else {
		if len(req.Subscribers) > 0 {
			return errors.New("subscribers must be empty when creating a chat")
		}
		if len(req.Members) > maxMembersChat {
			return fmt.Errorf("members limit exceeded: %d", len(req.Members))
		}
	}
	if len(req.Admins) > maxAdminsChat {
		return fmt.Errorf("admins limit exceeded: %d", len(req.Admins))
	}

	return nil
}

func validateUpdateMembers(req *ChatUpdateMembersRequest) error {
	if req == nil {
		return errors.New("update members request is nil")
	}
	if req.ChatID == "" {
		return errors.New("chat_id is required")
	}
	total := len(req.Members) + len(req.Admins) + len(req.Subscribers) + len(req.Remove)
	if total == 0 {
		return errors.New("at least one of members/admins/subscribers/remove is required")
	}
	if len(req.Members) > maxMembersChat || len(req.Subscribers) > maxSubscribersChat || len(req.Admins) > maxAdminsChat {
		return errors.New("members/admins/subscribers limit exceeded")
	}
	seen := map[ym.UserLogin]struct{}{}
	for _, lst := range [][]ym.UserRef{req.Members, req.Admins, req.Subscribers, req.Remove} {
		for _, u := range lst {
			if _, ok := seen[u.Login]; ok {
				return fmt.Errorf("duplicate user login: %s", u.Login)
			}
			seen[u.Login] = struct{}{}
		}
	}

	return nil
}
