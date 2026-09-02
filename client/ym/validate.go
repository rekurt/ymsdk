package ym

import (
	"errors"

	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

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
		return ymerrors.ErrNoTarget
	case set > 1:
		return ymerrors.ErrAmbiguousTarget
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
	case errors.Is(err, ymerrors.ErrNoTarget):
		return errors.New("either chat_id or login is required")
	case errors.Is(err, ymerrors.ErrAmbiguousTarget):
		return errors.New("only one of chat_id or login must be set")
	default:
		return err
	}
}
