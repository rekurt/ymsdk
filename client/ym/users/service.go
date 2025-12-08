package users

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

type Service struct {
	client *ym.Client
}

func NewService(client *ym.Client) *Service {
	return &Service{client: client}
}

type userLinkResponse struct {
	OK   bool         `json:"ok"`
	Link *ym.UserLink `json:"link,omitempty"`

	ID       string `json:"id,omitempty"`
	ChatLink string `json:"chat_link,omitempty"`
	CallLink string `json:"call_link,omitempty"`
	Error    string `json:"description,omitempty"`
}

func (s *Service) GetUserLink(ctx context.Context, login ym.UserLogin) (*ym.UserLink, error) {
	if login == "" {
		return nil, errors.New("login is required")
	}

	params := url.Values{}
	params.Set("login", string(login))
	path := "/bot/v1/users/getUserLink/?" + params.Encode()

	resp, err := s.client.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed userLinkResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("%w: decode getUserLink response: %w", ymerrors.ErrInvalidResponse, err)
	}
	if !parsed.OK {
		return nil, &ymerrors.APIError{
			Kind:        ymerrors.KindBadRequest,
			HTTPStatus:  resp.StatusCode,
			Description: parsed.Error,
			Method:      http.MethodGet,
			Endpoint:    "/bot/v1/users/getUserLink/",
		}
	}

	if parsed.Link != nil {
		return parsed.Link, nil
	}

	return &ym.UserLink{
		ID:       parsed.ID,
		ChatLink: parsed.ChatLink,
		CallLink: parsed.CallLink,
	}, nil
}
