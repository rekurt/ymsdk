package ym

import (
	"fmt"

	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

// Documented API limits. Requests that exceed them are rejected server-side,
// so the SDK checks them before spending a round trip.
const (
	// MaxTextLength is the longest message text the API accepts.
	MaxTextLength = 6000
	// MaxSuggestButtons is the largest suggest keyboard.
	MaxSuggestButtons = 100
	// MaxActionButtons is the largest row of action buttons.
	MaxActionButtons = 6
	// MaxDirectivesPerButton is how many directives one button may carry.
	MaxDirectivesPerButton = 3
	// MaxButtonFieldLength caps a button's id and title.
	MaxButtonFieldLength = 255
	// MaxGalleryImages is the largest album.
	MaxGalleryImages = 10
	// MaxPageLimit is the largest page the paginated endpoints return.
	MaxPageLimit = 1000
	// MaxChatNameLength caps a chat or channel name.
	MaxChatNameLength = 200
	// MaxChatDescriptionLength caps a chat or channel description.
	MaxChatDescriptionLength = 500
	// MaxChatAdmins is how many administrators one request may carry.
	MaxChatAdmins = 100
	// MaxChatMembers is how many members or subscribers one request may carry.
	MaxChatMembers = 500
	// MinTypingTimeout and MaxTypingTimeout bound how long a typing indicator
	// stays visible, in seconds.
	MinTypingTimeout = 1
	MaxTypingTimeout = 60
	// MinProcessingTextLength and MaxProcessingTextLength bound the text of a
	// processing indicator.
	MinProcessingTextLength = 1
	MaxProcessingTextLength = 100
	// MinPollAnswers and MaxPollAnswers bound a poll's answer list.
	MinPollAnswers = 2
	MaxPollAnswers = 100
)

// LimitError reports a value outside a documented API limit.
type LimitError struct {
	// Field names the offending parameter, using the API's own name.
	Field string
	// Value is what the caller supplied.
	Value int
	// Limit is the documented maximum.
	Limit int
	// Min is the documented minimum, set only for parameters with a lower
	// bound. Zero means the parameter only has a ceiling.
	Min int
}

func (e *LimitError) Error() string {
	if e.Min > 0 {
		return fmt.Sprintf("yandex-messenger: %s %d is out of range [%d, %d]", e.Field, e.Value, e.Min, e.Limit)
	}

	return fmt.Sprintf("yandex-messenger: %s exceeds the API limit: %d (max %d)", e.Field, e.Value, e.Limit)
}

func newLimitError(field string, value, limit int) error {
	return &LimitError{Field: field, Value: value, Limit: limit}
}

// ValidateLength checks a text field against a documented maximum. Length is
// counted in runes, matching how the API counts characters rather than bytes.
func ValidateLength(field, value string, limit int) error {
	if n := len([]rune(value)); n > limit {
		return newLimitError(field, n, limit)
	}

	return nil
}

// ValidateCount checks a collection against a documented maximum size.
func ValidateCount(field string, count, limit int) error {
	if count > limit {
		return newLimitError(field, count, limit)
	}

	return nil
}

// ValidateText checks a message body against the documented length limit.
func ValidateText(text string) error {
	return ValidateLength("text", text, MaxTextLength)
}

// ValidateSuggestButtons checks a keyboard's size and the shape of its buttons.
func ValidateSuggestButtons(sb *SuggestButtons) error {
	if sb == nil {
		return nil
	}

	total := 0
	for _, row := range sb.Buttons {
		total += len(row)
		for _, b := range row {
			if err := validateButtonFields(b.ID, b.Title, len(b.Directives)); err != nil {
				return err
			}
		}
	}
	if total > MaxSuggestButtons {
		return newLimitError("suggest_buttons", total, MaxSuggestButtons)
	}

	return nil
}

// ValidateActionButtons checks a row of action buttons.
func ValidateActionButtons(ab *ActionButtons) error {
	if ab == nil {
		return nil
	}
	if n := len(ab.Buttons); n > MaxActionButtons {
		return newLimitError("action_buttons", n, MaxActionButtons)
	}
	for i, b := range ab.Buttons {
		if err := validateButtonFields(b.ID, b.Title, len(b.Directives)); err != nil {
			return err
		}
		// Unlike a suggest button, whose fields are all optional, an action
		// button must carry a title, an icon and at least one directive.
		if err := validateActionButtonRequired(i, b); err != nil {
			return err
		}
	}

	return nil
}

// validateActionButtonRequired checks the fields whose absence is unambiguously
// invalid.
//
// The reference also marks directives required, but the documented icons are
// like and dislike, which suggests those buttons carry behaviour of their own —
// and that cannot be confirmed without calling the API. Rejecting a request the
// server would have accepted is worse than letting it answer, so directives are
// left to the API.
func validateActionButtonRequired(index int, b ActionButton) error {
	if b.Title == "" || b.Icon.Type == "" || b.Icon.Value == "" {
		return fmt.Errorf("yandex-messenger: action button %d: %w", index, ymerrors.ErrIncompleteActionButton)
	}

	return nil
}

func validateButtonFields(id, title string, directives int) error {
	if n := len([]rune(id)); n > MaxButtonFieldLength {
		return newLimitError("button id", n, MaxButtonFieldLength)
	}
	if n := len([]rune(title)); n > MaxButtonFieldLength {
		return newLimitError("button title", n, MaxButtonFieldLength)
	}
	if directives > MaxDirectivesPerButton {
		return newLimitError("button directives", directives, MaxDirectivesPerButton)
	}

	return nil
}

// ValidateRange checks a value against a documented range, reporting a
// [LimitError] so that callers can match every documented limit violation the
// same way.
func ValidateRange(field string, value, minimum, maximum int) error {
	if value < minimum || value > maximum {
		return &LimitError{Field: field, Value: value, Limit: maximum, Min: minimum}
	}

	return nil
}

// ValidatePageLimit checks a pagination limit.
func ValidatePageLimit(limit int) error {
	return ValidateRange("limit", limit, 1, MaxPageLimit)
}
