package ym

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateText(t *testing.T) {
	t.Run("accepts the maximum", func(t *testing.T) {
		if err := ValidateText(strings.Repeat("a", MaxTextLength)); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("rejects one rune too many", func(t *testing.T) {
		err := ValidateText(strings.Repeat("a", MaxTextLength+1))

		var limitErr *LimitError
		if !errors.As(err, &limitErr) {
			t.Fatalf("expected a LimitError, got %v", err)
		}
		if limitErr.Field != "text" || limitErr.Limit != MaxTextLength {
			t.Fatalf("unexpected error: %#v", limitErr)
		}
	})

	// The API counts characters, so multi-byte text must not be measured in bytes.
	t.Run("counts runes, not bytes", func(t *testing.T) {
		if err := ValidateText(strings.Repeat("я", MaxTextLength)); err != nil {
			t.Fatalf("expected 6000 Cyrillic runes to pass, got %v", err)
		}
	})
}

func TestValidateSuggestButtons(t *testing.T) {
	t.Run("nil is fine", func(t *testing.T) {
		if err := ValidateSuggestButtons(nil); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("counts buttons across rows", func(t *testing.T) {
		rows := make([][]InlineSuggestButton, 0, 21)
		for range 21 {
			row := make([]InlineSuggestButton, 5)
			rows = append(rows, row)
		}
		err := ValidateSuggestButtons(&SuggestButtons{Buttons: rows})

		var limitErr *LimitError
		if !errors.As(err, &limitErr) || limitErr.Value != 105 {
			t.Fatalf("expected 105 buttons to be rejected, got %v", err)
		}
	})

	t.Run("rejects an over-long title", func(t *testing.T) {
		err := ValidateSuggestButtons(&SuggestButtons{Buttons: [][]InlineSuggestButton{{
			{Title: strings.Repeat("a", MaxButtonFieldLength+1)},
		}}})
		if err == nil {
			t.Fatal("expected a limit error")
		}
	})

	t.Run("rejects too many directives", func(t *testing.T) {
		err := ValidateSuggestButtons(&SuggestButtons{Buttons: [][]InlineSuggestButton{{
			{Directives: make([]Directive, MaxDirectivesPerButton+1)},
		}}})
		if err == nil {
			t.Fatal("expected a limit error")
		}
	})
}

func TestValidateActionButtons(t *testing.T) {
	if err := ValidateActionButtons(nil); err != nil {
		t.Fatalf("nil must be accepted, got %v", err)
	}
	err := ValidateActionButtons(&ActionButtons{Buttons: make([]ActionButton, MaxActionButtons+1)})
	if err == nil {
		t.Fatalf("expected more than %d action buttons to be rejected", MaxActionButtons)
	}
}

func TestValidatePageLimit(t *testing.T) {
	for _, bad := range []int{0, -1, MaxPageLimit + 1} {
		if err := ValidatePageLimit(bad); err == nil {
			t.Fatalf("expected %d to be rejected", bad)
		}
	}
	for _, good := range []int{1, 100, MaxPageLimit} {
		if err := ValidatePageLimit(good); err != nil {
			t.Fatalf("expected %d to be accepted, got %v", good, err)
		}
	}
}
