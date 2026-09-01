package messages

import (
	"context"
	"errors"

	"github.com/rekurt/ymsdk/client/ym"
)

// ErrMessageIDRequired is returned when an operation that acts on a specific
// message is called without one.
var ErrMessageIDRequired = errors.New("yandex-messenger: message_id is required")

// PinOptions holds optional parameters for pinning and unpinning.
type PinOptions struct {
	// ThreadID scopes the operation to a thread.
	ThreadID *ym.ThreadID
}

type pinRequest struct {
	ym.Target
	MessageID ym.MessageID `json:"message_id"`
	ThreadID  *ym.ThreadID `json:"thread_id,omitempty"`
}

// PinResult describes the message that was pinned.
type PinResult struct {
	MessageID ym.MessageID
	// Chat is the chat the message was pinned in. It may be nil.
	Chat *ym.Chat
}

// Pin pins a message in the target chat.
//
// The bot must be a member or admin of the chat, and the chat must not have
// pinning disabled in its settings.
func (s *Service) Pin(
	ctx context.Context, target ym.Target, messageID ym.MessageID, opts *PinOptions,
) (*PinResult, error) {
	req, err := buildPinRequest(target, messageID, opts)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		OK          bool         `json:"ok"`
		MessageID   ym.MessageID `json:"message_id"`
		Chat        *ym.Chat     `json:"chat"`
		Description string       `json:"description"`
	}
	if err := s.post(ctx, ym.EndpointMessagesPin, req, &parsed); err != nil {
		return nil, err
	}
	if !parsed.OK {
		return nil, okFalseError(ym.EndpointMessagesPin, parsed.Description)
	}

	return &PinResult{MessageID: parsed.MessageID, Chat: parsed.Chat}, nil
}

// Unpin removes a pinned message and returns the chat it was pinned in, which
// may be nil.
func (s *Service) Unpin(
	ctx context.Context, target ym.Target, messageID ym.MessageID, opts *PinOptions,
) (*ym.Chat, error) {
	req, err := buildPinRequest(target, messageID, opts)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		OK          bool     `json:"ok"`
		Chat        *ym.Chat `json:"chat"`
		Description string   `json:"description"`
	}
	if err := s.post(ctx, ym.EndpointMessagesUnpin, req, &parsed); err != nil {
		return nil, err
	}
	if !parsed.OK {
		return nil, okFalseError(ym.EndpointMessagesUnpin, parsed.Description)
	}

	return parsed.Chat, nil
}

func buildPinRequest(target ym.Target, messageID ym.MessageID, opts *PinOptions) (pinRequest, error) {
	if err := ym.ValidateTarget(target); err != nil {
		return pinRequest{}, err
	}
	if messageID == 0 {
		return pinRequest{}, ErrMessageIDRequired
	}

	req := pinRequest{Target: target, MessageID: messageID}
	if opts != nil {
		req.ThreadID = opts.ThreadID
	}

	return req, nil
}
