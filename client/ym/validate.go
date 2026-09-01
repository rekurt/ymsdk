package ym

import "errors"

// ErrNoTarget is returned when an operation names no recipient.
var ErrNoTarget = errors.New("yandex-messenger: exactly one of chat_id, login or user_id is required")

// ErrAmbiguousTarget is returned when an operation names more than one recipient.
var ErrAmbiguousTarget = errors.New("yandex-messenger: only one of chat_id, login or user_id may be set")

// ValidateTarget checks that t names exactly one recipient.
//
// The API rejects requests carrying more than one of chat_id, login and
// user_id, so this is enforced before the request goes out.
func ValidateTarget(t Target) error {
	set := 0
	if t.ChatID != "" {
		set++
	}
	if t.Login != "" {
		set++
	}
	if t.UserID != "" {
		set++
	}

	switch {
	case set == 0:
		return ErrNoTarget
	case set > 1:
		return ErrAmbiguousTarget
	default:
		return nil
	}
}

// ValidateRecipient checks that exactly one of chatID or login is provided.
//
// Deprecated: use [ValidateTarget], which also covers the user_id form that
// several endpoints accept. This wrapper is kept so existing callers compile.
func ValidateRecipient(chatID *ChatID, login *UserLogin) error {
	var t Target
	if chatID != nil {
		t.ChatID = *chatID
	}
	if login != nil {
		t.Login = *login
	}

	switch err := ValidateTarget(t); {
	case errors.Is(err, ErrNoTarget):
		return errors.New("either chat_id or login is required")
	case errors.Is(err, ErrAmbiguousTarget):
		return errors.New("only one of chat_id or login must be set")
	default:
		return err
	}
}
