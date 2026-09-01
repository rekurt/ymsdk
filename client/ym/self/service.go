package self

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

// Service provides methods for reading and updating the bot's own settings.
type Service struct {
	client *ym.Client
}

// NewService creates a new self Service.
func NewService(client *ym.Client) *Service {
	return &Service{client: client}
}

// SelfUpdateRequest contains parameters for updating the bot's settings.
type SelfUpdateRequest struct {
	// WebhookURL sets or clears the endpoint updates are pushed to. Pass a
	// pointer to an empty string to clear it.
	WebhookURL *string `json:"webhook_url,omitempty"`
	// Settings toggles the bot's feature flags. Reaction and membership events
	// are not delivered until the matching flag is on.
	Settings *ym.BotSettings `json:"settings,omitempty"`
}

// selfResponse is the shape both self endpoints return.
type selfResponse struct {
	OK            bool            `json:"ok"`
	ID            string          `json:"id"`
	DisplayName   string          `json:"display_name"`
	WebhookURL    *string         `json:"webhook_url"`
	Organizations []int64         `json:"organizations"`
	Login         string          `json:"login"`
	Settings      *ym.BotSettings `json:"settings"`
	Description   string          `json:"description"`
}

func (r selfResponse) toBotSelf() *ym.BotSelf {
	return &ym.BotSelf{
		ID:            r.ID,
		DisplayName:   r.DisplayName,
		WebhookURL:    r.WebhookURL,
		Organizations: r.Organizations,
		Login:         ym.UserLogin(r.Login),
		Settings:      r.Settings,
	}
}

// Get returns information about the bot: its identity, the webhook URL in
// effect and which optional update types are enabled.
func (s *Service) Get(ctx context.Context) (*ym.BotSelf, error) {
	return s.call(ctx, http.MethodGet, ym.EndpointSelfGet, nil)
}

// Update modifies the bot's settings and returns the updated bot information.
func (s *Service) Update(ctx context.Context, req *SelfUpdateRequest) (*ym.BotSelf, error) {
	return s.call(ctx, http.MethodPost, ym.EndpointSelfUpdate, req)
}

func (s *Service) call(ctx context.Context, method, path string, body any) (*ym.BotSelf, error) {
	resp, err := s.client.DoRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed selfResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("%w: decode %s response: %w", ymerrors.ErrInvalidResponse, path, err)
	}
	if !parsed.OK {
		return nil, &ymerrors.APIError{
			Kind:        ymerrors.KindBadRequest,
			HTTPStatus:  resp.StatusCode,
			Description: parsed.Description,
			Method:      method,
			Endpoint:    path,
		}
	}

	return parsed.toBotSelf(), nil
}
