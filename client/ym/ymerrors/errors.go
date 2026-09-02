package ymerrors

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// ErrorKind classifies API errors into broad categories for programmatic handling.
type ErrorKind int

const (
	// KindUnknown is an unclassified error.
	KindUnknown ErrorKind = iota
	// KindRateLimited indicates the request was throttled (HTTP 429).
	KindRateLimited
	// KindInvalidToken indicates the OAuth token is invalid (HTTP 403).
	KindInvalidToken
	// KindUnauthorized indicates missing or expired credentials (HTTP 401).
	KindUnauthorized
	// KindBadRequest indicates a malformed request (HTTP 400).
	KindBadRequest
	// KindNetwork indicates a transport-level failure (DNS, TCP, 5xx).
	KindNetwork
	// KindNotFound indicates the requested resource was not found (HTTP 404).
	KindNotFound
	// KindConflict indicates a conflict with the current state (HTTP 409).
	KindConflict
	// KindPayloadTooLarge indicates the request body exceeds the size limit (HTTP 413).
	KindPayloadTooLarge
)

// Sentinel errors for use with errors.Is.
var (
	ErrRateLimited     = errors.New("yandex-messenger: rate limited")
	ErrInvalidToken    = errors.New("yandex-messenger: invalid token")
	ErrUnauthorized    = errors.New("yandex-messenger: unauthorized")
	ErrBadRequest      = errors.New("yandex-messenger: bad request")
	ErrNotFound        = errors.New("yandex-messenger: not found")
	ErrConflict        = errors.New("yandex-messenger: conflict")
	ErrPayloadTooLarge = errors.New("yandex-messenger: payload too large")
	ErrRequestTimeout  = errors.New("yandex-messenger: request timeout")
	ErrNetworkError    = errors.New("yandex-messenger: network error")
	ErrInvalidResponse = errors.New("yandex-messenger: invalid response")
)

// APIError is a structured error returned by the Yandex Messenger API.
// Use errors.As to extract it and errors.Is to match sentinel errors.
type APIError struct {
	Kind        ErrorKind
	Code        int
	HTTPStatus  int
	Description string
	RequestID   string
	Method      string
	Endpoint    string
	RetryAfter  time.Duration
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString("yandex-messenger/apierror")
	b.WriteString(": kind=")
	b.WriteString(strconv.Itoa(int(e.Kind)))
	if e.HTTPStatus > 0 {
		b.WriteString(" http=")
		b.WriteString(strconv.Itoa(e.HTTPStatus))
	}
	if e.Code != 0 {
		b.WriteString(" code=")
		b.WriteString(strconv.Itoa(e.Code))
	}
	if e.RequestID != "" {
		b.WriteString(" request_id=")
		b.WriteString(e.RequestID)
	}
	if e.Method != "" || e.Endpoint != "" {
		b.WriteString(" op=")
		b.WriteString(strings.TrimSpace(strings.Join([]string{e.Method, e.Endpoint}, " ")))
	}
	if e.RetryAfter > 0 {
		b.WriteString(" retry_after=")
		b.WriteString(e.RetryAfter.String())
	}
	if e.Description != "" {
		b.WriteString(": ")
		b.WriteString(e.Description)
	}

	return b.String()
}

// Unwrap returns the sentinel error matching the error kind,
// enabling errors.Is checks (e.g. errors.Is(err, ErrRateLimited)).
func (e *APIError) Unwrap() error {
	if e == nil {
		return nil
	}
	switch e.Kind {
	case KindRateLimited:
		return ErrRateLimited
	case KindInvalidToken:
		return ErrInvalidToken
	case KindUnauthorized:
		return ErrUnauthorized
	case KindBadRequest:
		return ErrBadRequest
	case KindNetwork:
		return ErrNetworkError
	case KindNotFound:
		return ErrNotFound
	case KindConflict:
		return ErrConflict
	case KindPayloadTooLarge:
		return ErrPayloadTooLarge
	default:
		return nil
	}
}

// Request validation sentinels. These are returned before a request is sent,
// when the caller's arguments cannot produce a valid call. They live here, and
// not beside the service that returns them, so matching one with errors.Is
// needs a single import rather than one per API domain.

// ErrNoTarget is returned when an operation names no recipient.
var ErrNoTarget = errors.New("yandex-messenger: exactly one of chat_id, login or user_id is required")

// ErrAmbiguousTarget is returned when an operation names more than one recipient.
var ErrAmbiguousTarget = errors.New("yandex-messenger: only one of chat_id, login or user_id may be set")

// ErrIncompleteActionButton is returned when an action button omits a field
// the API requires.
var ErrIncompleteActionButton = errors.New(
	"yandex-messenger: an action button requires a title and an icon",
)

// ErrReplyQuoteNeedsReply is returned when a quoted fragment is supplied
// without the message it quotes.
var ErrReplyQuoteNeedsReply = errors.New("yandex-messenger: reply_quote requires reply_message_id")

// ErrForwardsWithReply is returned when forwarding is combined with a reply.
// The API accepts one or the other, never both in the same request.
var ErrForwardsWithReply = errors.New("yandex-messenger: forwards cannot be combined with reply_message_id")

// ErrTextRequired is returned when an endpoint that carries nothing but text
// is called without any.
var ErrTextRequired = errors.New("yandex-messenger: text is required")

// ErrMessageIDRequired is returned when an operation that acts on a specific
// message is called without one.
var ErrMessageIDRequired = errors.New("yandex-messenger: message_id is required")

// ErrStickerRequired is returned when a sticker is sent without identifying it.
var ErrStickerRequired = errors.New("yandex-messenger: sticker_set_id and sticker_id are required")

// ErrIncompleteReaction is returned when a reaction is supplied without both
// of the fields the API requires. A nil reaction is not an error: it removes
// whatever the bot had set.
var ErrIncompleteReaction = errors.New("yandex-messenger: reaction type and name are required")

// ErrFileIDRequired is returned when a share operation omits the file identifier.
var ErrFileIDRequired = errors.New("yandex-messenger: file_id is required")

// ErrImageDimensionsRequired is returned when a shared image carries no size.
// The API requires width and height on every shared image, so sending zeroes is
// a request that can only fail.
var ErrImageDimensionsRequired = errors.New("yandex-messenger: shared image width and height are required")

// ErrProcessingContentRequired is returned when a processing indicator is
// requested without the content that describes how to render it.
var ErrProcessingContentRequired = errors.New(
	"yandex-messenger: processing_content is required when the typing type is processing",
)

// ErrProcessingDisplayRequired is returned when processing content omits the
// display mode the API uses to decide how to render the indicator.
var ErrProcessingDisplayRequired = errors.New(
	`yandex-messenger: processing_content.display must be "default" or "text"`,
)

// ErrTypingTypeInvalid is returned when the indicator type is neither of the
// two the API documents. The zero value is allowed: it is omitted from the
// payload, leaving the server to apply its own default.
var ErrTypingTypeInvalid = errors.New(
	`yandex-messenger: type must be "text" or "processing"`,
)

// ErrChatIDRequired is returned when a chat-scoped query omits the chat.
var ErrChatIDRequired = errors.New("yandex-messenger: chat_id is required")

// ErrUnknownMemberRole is returned when the role filter is neither empty nor
// one of the three roles the API documents.
var ErrUnknownMemberRole = errors.New(
	`yandex-messenger: role must be "admin", "member" or "subscriber"`,
)
