package ym

import (
	"encoding/json"
	"strings"
	"testing"
)

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

// The Bot API takes a flat button array when layout is "false" and a nested one
// when it is "true". The SDK stores rows either way, so the shape is decided at
// marshal time.
func TestSuggestButtonsMarshalShapeFollowsLayout(t *testing.T) {
	rows := [][]InlineSuggestButton{
		{{Title: "a"}, {Title: "b"}},
		{{Title: "c"}},
	}
	tests := []struct {
		name   string
		layout *string
		want   string
	}{
		{
			name:   "layout false flattens every row into one array",
			layout: Ptr(SuggestLayoutFlat),
			want:   `"buttons":[{"title":"a"},{"title":"b"},{"title":"c"}]`,
		},
		{
			name:   "layout true keeps the rows nested",
			layout: Ptr(SuggestLayoutRows),
			want:   `"buttons":[[{"title":"a"},{"title":"b"}],[{"title":"c"}]]`,
		},
		{
			name:   "layout unset keeps the previous nested shape",
			layout: nil,
			want:   `"buttons":[[{"title":"a"},{"title":"b"}],[{"title":"c"}]]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := json.Marshal(SuggestButtons{Layout: tt.layout, Buttons: rows})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if !strings.Contains(string(encoded), tt.want) {
				t.Fatalf("want %s in payload, got %s", tt.want, encoded)
			}
		})
	}
}

// A type with a custom marshaller that cannot read its own output is a trap,
// so both wire shapes must unmarshal back into rows.
func TestSuggestButtonsUnmarshalsBothShapes(t *testing.T) {
	tests := []struct {
		name     string
		payload  string
		wantRows int
		wantLen0 int
	}{
		{name: "flat", payload: `{"layout":"false","buttons":[{"title":"a"},{"title":"b"}]}`, wantRows: 1, wantLen0: 2},
		{name: "nested", payload: `{"layout":"true","buttons":[[{"title":"a"}],[{"title":"b"}]]}`, wantRows: 2, wantLen0: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb SuggestButtons
			if err := json.Unmarshal([]byte(tt.payload), &sb); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(sb.Buttons) != tt.wantRows || len(sb.Buttons[0]) != tt.wantLen0 {
				t.Fatalf("want %d rows first of %d, got %+v", tt.wantRows, tt.wantLen0, sb.Buttons)
			}
		})
	}
}
