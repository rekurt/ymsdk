package ym

import (
	"testing"
)

func TestValidateRecipient(t *testing.T) {
	t.Run("both nil", func(t *testing.T) {
		err := ValidateRecipient(nil, nil)
		if err == nil {
			t.Fatal("expected error when both nil")
		}
	})

	t.Run("both empty", func(t *testing.T) {
		chatID := ChatID("")
		login := UserLogin("")
		err := ValidateRecipient(&chatID, &login)
		if err == nil {
			t.Fatal("expected error when both empty")
		}
	})

	t.Run("both set", func(t *testing.T) {
		chatID := ChatID("c1")
		login := UserLogin("u1")
		err := ValidateRecipient(&chatID, &login)
		if err == nil {
			t.Fatal("expected error when both set")
		}
	})

	t.Run("only chatID", func(t *testing.T) {
		chatID := ChatID("c1")
		err := ValidateRecipient(&chatID, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("only login", func(t *testing.T) {
		login := UserLogin("u1")
		err := ValidateRecipient(nil, &login)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("chatID set login nil", func(t *testing.T) {
		chatID := ChatID("c1")
		err := ValidateRecipient(&chatID, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("chatID nil login set", func(t *testing.T) {
		login := UserLogin("u1")
		err := ValidateRecipient(nil, &login)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
