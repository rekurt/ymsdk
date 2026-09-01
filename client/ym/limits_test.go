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

// Every documented limit violation should be reachable with errors.As, so
// callers can handle them uniformly.
func TestValidatePageLimitReportsALimitError(t *testing.T) {
	for _, limit := range []int{0, -1, MaxPageLimit + 1} {
		err := ValidatePageLimit(limit)

		var limitErr *LimitError
		if !errors.As(err, &limitErr) {
			t.Fatalf("limit %d: expected a *LimitError, got %T (%v)", limit, err, err)
		}
		if limitErr.Value != limit || limitErr.Limit != MaxPageLimit {
			t.Fatalf("limit %d: unexpected error contents %#v", limit, limitErr)
		}
	}
}

// The API marks title, icon and directives required on an action button, while
// a suggest button has none of them required. The shared field check cannot
// enforce that difference, so it belongs to the action-button validator.
func TestValidateActionButtonsRequiresDocumentedFields(t *testing.T) {
	complete := ActionButton{
		Title:      "Like",
		Icon:       ActionButtonIcon{Type: "messenger_icons", Value: "like"},
		Directives: []Directive{{Type: DirectiveServerAction, Name: "like"}},
	}

	cases := []struct {
		name   string
		button ActionButton
	}{
		{"no title", ActionButton{Icon: complete.Icon, Directives: complete.Directives}},
		{"no icon type", ActionButton{
			Title: "Like", Icon: ActionButtonIcon{Value: "like"}, Directives: complete.Directives,
		}},
		{"no icon value", ActionButton{
			Title: "Like", Icon: ActionButtonIcon{Type: "messenger_icons"}, Directives: complete.Directives,
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateActionButtons(&ActionButtons{Buttons: []ActionButton{tc.button}}); err == nil {
				t.Fatal("expected the incomplete button to be rejected")
			}
		})
	}

	// Directives are documented as required, but the like/dislike icons suggest
	// built-in behaviour and that cannot be confirmed without the live API, so
	// a button without them is left for the server to judge.
	t.Run("a button without directives is left to the API", func(t *testing.T) {
		bare := ActionButton{Title: "Like", Icon: complete.Icon}
		if err := ValidateActionButtons(&ActionButtons{Buttons: []ActionButton{bare}}); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("a complete button is accepted", func(t *testing.T) {
		if err := ValidateActionButtons(&ActionButtons{Buttons: []ActionButton{complete}}); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})
}

// Suggest buttons keep their optional fields.
func TestValidateSuggestButtonsAllowsEmptyFields(t *testing.T) {
	err := ValidateSuggestButtons(&SuggestButtons{
		Buttons: [][]InlineSuggestButton{{{}}},
	})
	if err != nil {
		t.Fatalf("suggest button fields are optional; got %v", err)
	}
}
