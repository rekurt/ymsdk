package messages

import (
	"context"
	"errors"

	"github.com/rekurt/ymsdk/client/ym"
)

// ErrTextRequired is returned when an endpoint that carries nothing but text
// is called without any.
var ErrTextRequired = errors.New("yandex-messenger: text is required")

// SendSystemMessageOptions holds optional parameters for system messages.
// The endpoint accepts a narrower set than the ordinary send methods.
type SendSystemMessageOptions struct {
	// PayloadID is the idempotency key. Left empty, the SDK generates one.
	PayloadID string
	// ThreadID posts the message into a thread.
	ThreadID *ym.ThreadID
	// DisableNotification suppresses the recipient's notification.
	DisableNotification *bool
}

type sendSystemMessageRequest struct {
	ym.Target
	Text                string       `json:"text"`
	ThreadID            *ym.ThreadID `json:"thread_id,omitempty"`
	DisableNotification *bool        `json:"disable_notification,omitempty"`
	PayloadID           string       `json:"payload_id,omitempty"`
}

// SendSystemMessage posts a system message: it appears in the chat as a neutral
// notice with no sender, rather than as a message from the bot.
func (s *Service) SendSystemMessage(
	ctx context.Context, target ym.Target, text string, opts *SendSystemMessageOptions,
) (*ym.Message, error) {
	if err := ym.ValidateTarget(target); err != nil {
		return nil, err
	}

	// Unlike an ordinary send, this endpoint carries no attachment or forward
	// that could give an empty body meaning, and the API marks the text required.
	if text == "" {
		return nil, ErrTextRequired
	}
	if err := ym.ValidateText(text); err != nil {
		return nil, err
	}

	req := sendSystemMessageRequest{Target: target, Text: text}
	if opts != nil {
		req.ThreadID = opts.ThreadID
		req.DisableNotification = opts.DisableNotification
		req.PayloadID = opts.PayloadID
	}
	req.PayloadID = s.stampPayloadID(req.PayloadID)

	return s.postForMessage(ctx, ym.EndpointMessagesSendSystemMessage, req)
}
