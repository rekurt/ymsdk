package ym

import "testing"

func TestUpdateToMessageReturnsNilWhenRequiredFieldsMissing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		update *Update
	}{
		{name: "nil update", update: nil},
		{name: "nil chat", update: &Update{From: &Sender{Login: "bot"}}},
		{name: "nil from", update: &Update{Chat: &Chat{ID: "chat-1"}}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.update.ToMessage(); got != nil {
				t.Fatalf("expected nil message, got %#v", got)
			}
		})
	}
}

func TestUpdateToMessageConvertsWhenRequiredFieldsPresent(t *testing.T) {
	t.Parallel()

	update := &Update{
		MessageID: 42,
		Chat:      &Chat{ID: "chat-1"},
		From:      &Sender{Login: "bot"},
		Text:      "hello",
		Timestamp: 100,
	}

	message := update.ToMessage()
	if message == nil {
		t.Fatal("expected message, got nil")
	}
	if message.ID != update.MessageID || message.Chat.ID != update.Chat.ID || message.From.Login != update.From.Login {
		t.Fatalf("unexpected conversion result: %#v", message)
	}
}
