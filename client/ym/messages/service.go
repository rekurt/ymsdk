package messages

import (
	"context"
	"errors"

	"github.com/rekurt/ymsdk/client/ym"
)

// ErrReplyQuoteNeedsReply is returned when a quoted fragment is supplied
// without the message it quotes.
var ErrReplyQuoteNeedsReply = errors.New("yandex-messenger: reply_quote requires reply_message_id")

// ErrForwardsWithReply is returned when forwarding is combined with a reply.
// The API accepts one or the other, never both in the same request.
var ErrForwardsWithReply = errors.New("yandex-messenger: forwards cannot be combined with reply_message_id")

// Service provides methods for sending and managing messages in Yandex Messenger.
type Service struct {
	client *ym.Client
}

// NewService creates a new messages Service.
func NewService(client *ym.Client) *Service {
	return &Service{client: client}
}

// SendMessageOptions holds optional parameters accepted by the message-sending
// endpoints.
//
// The same type serves sendText, sendSticker and the share* methods. Every
// field is omitted from the request when unset, so passing an option an
// endpoint does not support simply sends nothing rather than an invalid body.
type SendMessageOptions struct {
	// PayloadID is the idempotency key. Left empty, the SDK generates one so
	// that a retried request cannot deliver the message twice. See
	// [ym.NewPayloadID].
	PayloadID string
	// MessageID edits an existing message in place instead of posting a new one.
	// The message must belong to the same chat.
	MessageID *ym.MessageID
	// ReplyToMessageID makes the message a reply. Cannot be combined with Forwards.
	ReplyToMessageID *ym.MessageID
	// ReplyQuote quotes a fragment of the message being replied to. Requires
	// ReplyToMessageID, and must be a substring of that message.
	ReplyQuote string
	// Forwards forwards messages from other chats. Cannot be combined with
	// ReplyToMessageID.
	Forwards []ym.Forward
	// DisableNotification suppresses the recipient's notification.
	DisableNotification *bool
	// Important marks the message as important.
	Important *bool
	// DisableWebPagePreview suppresses link previews.
	DisableWebPagePreview *bool
	// ThreadID posts the message into a thread.
	ThreadID *ym.ThreadID
	// SuggestButtons attaches a keyboard. At most 100 buttons.
	SuggestButtons *ym.SuggestButtons
	// ActionButtons attaches action buttons. At most 6.
	ActionButtons *ym.ActionButtons
	// InlineKeyboard attaches legacy inline buttons.
	//
	// Deprecated: the API marks inline_keyboard as obsolete. Use SuggestButtons.
	//nolint:staticcheck // the API parameter is deprecated but must stay reachable
	InlineKeyboard []ym.Button
}

// messageEnvelope carries the parameters shared by every send-style endpoint.
// It is embedded into the concrete request types so the fields marshal inline.
type messageEnvelope struct {
	ym.Target
	PayloadID             string             `json:"payload_id,omitempty"`
	MessageID             *ym.MessageID      `json:"message_id,omitempty"`
	ReplyMessageID        *ym.MessageID      `json:"reply_message_id,omitempty"`
	ReplyQuote            string             `json:"reply_quote,omitempty"`
	Forwards              []ym.Forward       `json:"forwards,omitempty"`
	DisableNotification   *bool              `json:"disable_notification,omitempty"`
	Important             *bool              `json:"important,omitempty"`
	DisableWebPagePreview *bool              `json:"disable_web_page_preview,omitempty"`
	ThreadID              *ym.ThreadID       `json:"thread_id,omitempty"`
	SuggestButtons        *ym.SuggestButtons `json:"suggest_buttons,omitempty"`
	ActionButtons         *ym.ActionButtons  `json:"action_buttons,omitempty"`
	//nolint:staticcheck // mirrors the deprecated inline_keyboard API parameter
	InlineKeyboard []ym.Button `json:"inline_keyboard,omitempty"`
}

type sendTextRequest struct {
	messageEnvelope
	Text string `json:"text"`
}

// SendToChat sends a text message to a chat identified by chatID.
func (s *Service) SendToChat(
	ctx context.Context, chatID ym.ChatID, text string, opts *SendMessageOptions,
) (*ym.Message, error) {
	return s.SendText(ctx, ym.ChatTarget(chatID), text, opts)
}

// SendToLogin sends a text message to a user identified by login.
func (s *Service) SendToLogin(
	ctx context.Context, login ym.UserLogin, text string, opts *SendMessageOptions,
) (*ym.Message, error) {
	return s.SendText(ctx, ym.LoginTarget(login), text, opts)
}

// SendText sends a text message to target. When opts.MessageID is set the
// existing message is edited instead, which is how the API models edits.
func (s *Service) SendText(
	ctx context.Context, target ym.Target, text string, opts *SendMessageOptions,
) (*ym.Message, error) {
	if err := ym.ValidateTarget(target); err != nil {
		return nil, err
	}

	if err := validateSend(text, opts); err != nil {
		return nil, err
	}

	req := sendTextRequest{
		messageEnvelope: s.envelope(target, opts),
		Text:            text,
	}

	return s.postForMessage(ctx, ym.EndpointMessagesSendText, req)
}

// EditText replaces the text of an existing message. It is a thin wrapper over
// [Service.SendText] with opts.MessageID set, mirroring how the API exposes edits.
func (s *Service) EditText(
	ctx context.Context, target ym.Target, messageID ym.MessageID, text string, opts *SendMessageOptions,
) (*ym.Message, error) {
	if messageID == 0 {
		return nil, ErrMessageIDRequired
	}

	edit := SendMessageOptions{}
	if opts != nil {
		edit = *opts
	}
	edit.MessageID = &messageID

	return s.SendText(ctx, target, text, &edit)
}

// envelope converts caller-facing options into the wire envelope, stamping an
// idempotency key when the caller did not supply one.
//
// Only use it for endpoints the API documents payload_id on: sendText,
// sendSticker and sendSystemMessage. The share* endpoints take
// [Service.shareEnvelope] instead.
func (s *Service) envelope(target ym.Target, opts *SendMessageOptions) messageEnvelope {
	env := newEnvelope(target, opts)
	env.PayloadID = s.stampPayloadID(env.PayloadID)

	return env
}

// shareEnvelope builds the envelope for shareFile, shareImage and shareGallery.
//
// Those endpoints do not document payload_id, so no key is sent — including one
// the caller supplied, which the API has no contract to honour. The consequence
// is that resending an attachment is not idempotent: a retry after the server
// already accepted it posts it twice.
func (s *Service) shareEnvelope(target ym.Target, opts *SendMessageOptions) messageEnvelope {
	env := newEnvelope(target, opts)
	env.PayloadID = ""

	return env
}

func newEnvelope(target ym.Target, opts *SendMessageOptions) messageEnvelope {
	env := messageEnvelope{Target: target}
	if opts != nil {
		env.PayloadID = opts.PayloadID
		env.MessageID = opts.MessageID
		env.ReplyMessageID = opts.ReplyToMessageID
		env.ReplyQuote = opts.ReplyQuote
		env.Forwards = opts.Forwards
		env.DisableNotification = opts.DisableNotification
		env.Important = opts.Important
		env.DisableWebPagePreview = opts.DisableWebPagePreview
		env.ThreadID = opts.ThreadID
		env.SuggestButtons = opts.SuggestButtons
		env.ActionButtons = opts.ActionButtons
		env.InlineKeyboard = opts.InlineKeyboard
	}

	return env
}

// stampPayloadID returns current, or a fresh idempotency key when the caller
// left it empty and automatic keys are enabled.
//
// A retried request replays an identical body, so the key travels with every
// attempt and the API collapses the duplicates into a single message.
func (s *Service) stampPayloadID(current string) string {
	if current == "" && s.client.AutoPayloadID() {
		return ym.NewPayloadID()
	}

	return current
}

// validateSend checks the documented limits that apply to every send-style
// endpoint, so an over-long message fails locally instead of at the API.
func validateSend(text string, opts *SendMessageOptions) error {
	if err := ym.ValidateText(text); err != nil {
		return err
	}
	if opts == nil {
		return nil
	}
	if opts.ReplyQuote != "" && opts.ReplyToMessageID == nil {
		return ErrReplyQuoteNeedsReply
	}
	if len(opts.Forwards) > 0 && opts.ReplyToMessageID != nil {
		return ErrForwardsWithReply
	}
	if err := ym.ValidateSuggestButtons(opts.SuggestButtons); err != nil {
		return err
	}

	return ym.ValidateActionButtons(opts.ActionButtons)
}
