package messages

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

// postForMessage POSTs body and decodes an {ok, message_id} response, which is
// what every send-style endpoint returns.
func (s *Service) postForMessage(ctx context.Context, path string, body any) (*ym.Message, error) {
	var parsed struct {
		OK          bool         `json:"ok"`
		MessageID   ym.MessageID `json:"message_id"`
		Description string       `json:"description"`
	}
	if err := s.post(ctx, path, body, &parsed); err != nil {
		return nil, err
	}
	if !parsed.OK {
		return nil, okFalseError(path, parsed.Description)
	}

	return &ym.Message{ID: parsed.MessageID}, nil
}

// postForOK POSTs body and decodes an {ok} acknowledgement, which is what the
// endpoints that return no payload use.
func (s *Service) postForOK(ctx context.Context, path string, body any) error {
	var parsed struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := s.post(ctx, path, body, &parsed); err != nil {
		return err
	}
	if !parsed.OK {
		return okFalseError(path, parsed.Description)
	}

	return nil
}

// post sends body and decodes the response into out.
func (s *Service) post(ctx context.Context, path string, body, out any) error {
	resp, err := s.client.DoRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%w: decode %s response: %w", ymerrors.ErrInvalidResponse, path, err)
	}

	return nil
}

// okFalseError builds the error for a 200 response whose body reports failure.
//
// The result sits in the chain twice on purpose: callers that matched
// [ymerrors.ErrInvalidResponse] keep working, while callers using errors.As
// reach the *APIError and the server's description. Go's multi-%w makes both
// reachable from one value.
func okFalseError(path, description string) error {
	if description == "" {
		description = "ok=false"
	}

	apiErr := &ymerrors.APIError{
		Kind:        ymerrors.KindBadRequest,
		HTTPStatus:  http.StatusOK,
		Description: description,
		Method:      http.MethodPost,
		Endpoint:    path,
	}

	return fmt.Errorf("%w: %w", ymerrors.ErrInvalidResponse, apiErr)
}
