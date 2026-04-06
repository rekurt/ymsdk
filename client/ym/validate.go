package ym

import "errors"

// ValidateRecipient checks that exactly one of chatID or login is provided.
// It returns an error if both are empty or both are set.
func ValidateRecipient(chatID *ChatID, login *UserLogin) error {
	if (chatID == nil || *chatID == "") && (login == nil || *login == "") {
		return errors.New("either chat_id or login is required")
	}
	if chatID != nil && *chatID != "" && login != nil && *login != "" {
		return errors.New("only one of chat_id or login must be set")
	}

	return nil
}
