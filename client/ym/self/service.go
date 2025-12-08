package self

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

type SelfUpdateRequest struct {
	WebhookURL *string `json:"webhook_url,omitempty"`
}

func (s *Service) Update(ctx context.Context, req *SelfUpdateRequest) (*ym.BotSelf, error) {
	resp, err := s.client.DoRequest(ctx, http.MethodPost, "/bot/v1/self/update/", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed struct {
		OK            bool    `json:"ok"`
		ID            string  `json:"id"`
		DisplayName   string  `json:"display_name"`
		WebhookURL    *string `json:"webhook_url"`
		Organizations []int64 `json:"organizations"`
		Login         string  `json:"login"`
		Description   string  `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("%w: decode self.update response: %w", ymerrors.ErrInvalidResponse, err)
	}
	if !parsed.OK {
		return nil, &ymerrors.APIError{
			Kind:        ymerrors.KindBadRequest,
			HTTPStatus:  resp.StatusCode,
			Description: parsed.Description,
			Method:      http.MethodPost,
			Endpoint:    "/bot/v1/self/update/",
		}
	}

	return &ym.BotSelf{
		ID:            parsed.ID,
		DisplayName:   parsed.DisplayName,
		WebhookURL:    parsed.WebhookURL,
		Organizations: parsed.Organizations,
		Login:         ym.UserLogin(parsed.Login),
	}, nil
}
