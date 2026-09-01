package messages

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func newTestService(t *testing.T, body string) (*Service, *testutil.FakeDoer) {
	t.Helper()

	doer := &testutil.FakeDoer{Responses: []*http.Response{testutil.NewResponse(http.StatusOK, body)}}
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL:              "http://example.com",
		DisableAutoPayloadID: true,
	}, doer)

	return NewService(client), doer
}

// decodeBody returns the JSON body of the single recorded request.
func decodeBody(t *testing.T, doer *testutil.FakeDoer) map[string]any {
	t.Helper()

	if doer.CallCount() != 1 {
		t.Fatalf("expected exactly 1 request, got %d", doer.CallCount())
	}
	raw, err := io.ReadAll(doer.Requests[0].Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode body %s: %v", raw, err)
	}

	return out
}

func TestNewEndpointsSendExpectedRequest(t *testing.T) {
	chat := ym.ChatTarget("chat-1")

	cases := []struct {
		name     string
		response string
		call     func(context.Context, *Service) error
		wantPath string
		wantBody map[string]any
	}{
		{
			name:     "pin",
			response: `{"ok":true,"message_id":7,"chat":{"type":"group","id":"chat-1"}}`,
			call: func(ctx context.Context, s *Service) error {
				_, err := s.Pin(ctx, chat, 7, nil)

				return err
			},
			wantPath: ym.EndpointMessagesPin,
			wantBody: map[string]any{"chat_id": "chat-1", "message_id": float64(7)},
		},
		{
			name:     "unpin",
			response: `{"ok":true,"chat":{"type":"group","id":"chat-1"}}`,
			call: func(ctx context.Context, s *Service) error {
				_, err := s.Unpin(ctx, chat, 7, nil)

				return err
			},
			wantPath: ym.EndpointMessagesUnpin,
			wantBody: map[string]any{"chat_id": "chat-1", "message_id": float64(7)},
		},
		{
			name:     "sendReaction",
			response: `{"ok":true}`,
			call: func(ctx context.Context, s *Service) error {
				return s.SendReaction(ctx, chat, 7, ym.DefaultReaction("like"), nil)
			},
			wantPath: ym.EndpointMessagesSendReaction,
			wantBody: map[string]any{
				"chat_id":    "chat-1",
				"message_id": float64(7),
				"reaction":   map[string]any{"type": "default_reaction", "name": "like"},
			},
		},
		{
			name:     "removeReaction omits the reaction field",
			response: `{"ok":true}`,
			call: func(ctx context.Context, s *Service) error {
				return s.RemoveReaction(ctx, chat, 7, nil)
			},
			wantPath: ym.EndpointMessagesSendReaction,
			wantBody: map[string]any{"chat_id": "chat-1", "message_id": float64(7)},
		},
		{
			name:     "getReactions",
			response: `{"ok":true,"reactions_type":"public","reactions_list":[]}`,
			call: func(ctx context.Context, s *Service) error {
				_, err := s.GetReactions(ctx, chat, 7, nil)

				return err
			},
			wantPath: ym.EndpointMessagesGetReactions,
			wantBody: map[string]any{"chat_id": "chat-1", "message_id": float64(7)},
		},
		{
			name:     "sendSticker",
			response: `{"ok":true,"message_id":9}`,
			call: func(ctx context.Context, s *Service) error {
				_, err := s.SendSticker(ctx, chat, "set-1", "st-1", nil)

				return err
			},
			wantPath: ym.EndpointMessagesSendSticker,
			wantBody: map[string]any{"chat_id": "chat-1", "sticker_set_id": "set-1", "sticker_id": "st-1"},
		},
		{
			name:     "sendSystemMessage",
			response: `{"ok":true,"message_id":9}`,
			call: func(ctx context.Context, s *Service) error {
				_, err := s.SendSystemMessage(ctx, chat, "deployed", nil)

				return err
			},
			wantPath: ym.EndpointMessagesSendSystemMessage,
			wantBody: map[string]any{"chat_id": "chat-1", "text": "deployed"},
		},
		{
			name:     "sendTyping",
			response: `{"ok":true}`,
			call: func(ctx context.Context, s *Service) error {
				return s.SendTyping(ctx, chat, nil)
			},
			wantPath: ym.EndpointMessagesSendTyping,
			wantBody: map[string]any{"chat_id": "chat-1"},
		},
		{
			name:     "shareFile",
			response: `{"ok":true,"message_id":9}`,
			call: func(ctx context.Context, s *Service) error {
				_, err := s.ShareFile(ctx, chat, ym.SharedFile{FileID: "f1"}, nil)

				return err
			},
			wantPath: ym.EndpointMessagesShareFile,
			wantBody: map[string]any{"chat_id": "chat-1", "document": map[string]any{"file_id": "f1"}},
		},
		{
			name:     "shareImage",
			response: `{"ok":true,"message_id":9}`,
			call: func(ctx context.Context, s *Service) error {
				_, err := s.ShareImage(ctx, chat, ym.SharedImage{FileID: "i1", Width: 10, Height: 20}, nil)

				return err
			},
			wantPath: ym.EndpointMessagesShareImage,
			wantBody: map[string]any{
				"chat_id": "chat-1",
				"image":   map[string]any{"file_id": "i1", "width": float64(10), "height": float64(20)},
			},
		},
		{
			name:     "shareGallery",
			response: `{"ok":true,"message_id":9}`,
			call: func(ctx context.Context, s *Service) error {
				_, err := s.ShareGallery(ctx, chat, []ym.SharedImage{{FileID: "i1", Width: 1, Height: 2}}, nil)

				return err
			},
			wantPath: ym.EndpointMessagesShareGallery,
			wantBody: map[string]any{
				"chat_id": "chat-1",
				"images":  []any{map[string]any{"file_id": "i1", "width": float64(1), "height": float64(2)}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, doer := newTestService(t, tc.response)

			if err := tc.call(context.Background(), svc); err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if got := doer.Requests[0].URL.Path; got != tc.wantPath {
				t.Fatalf("path: got %s, want %s", got, tc.wantPath)
			}
			if got := doer.Requests[0].Method; got != http.MethodPost {
				t.Fatalf("method: got %s, want POST", got)
			}

			body := decodeBody(t, doer)
			for key, want := range tc.wantBody {
				got, ok := body[key]
				if !ok {
					t.Fatalf("body is missing %q; got %v", key, body)
				}
				if !jsonEqual(got, want) {
					t.Fatalf("body[%q]: got %#v, want %#v", key, got, want)
				}
			}
			if _, unexpected := body["reaction"]; unexpected && tc.name == "removeReaction omits the reaction field" {
				t.Fatal("removing a reaction must not send a reaction field")
			}
		})
	}
}

func jsonEqual(got, want any) bool {
	a, errA := json.Marshal(got)
	b, errB := json.Marshal(want)

	return errA == nil && errB == nil && string(a) == string(b)
}

func TestGetReactionsDecodesBothShapes(t *testing.T) {
	t.Run("public chats carry authors", func(t *testing.T) {
		svc, _ := newTestService(t, `{"ok":true,"reactions_type":"public","reactions_list":[
			{"reaction":{"type":"default_reaction","name":"like"},"timestamp":17,"user":{"login":"a@example.org"}}]}`)

		page, err := svc.GetReactions(context.Background(), ym.ChatTarget("c"), 1, nil)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if page.Type != ym.ReactionsPublic {
			t.Fatalf("expected a public page, got %q", page.Type)
		}
		if len(page.List) != 1 || page.List[0].User.Login != "a@example.org" {
			t.Fatalf("unexpected list: %#v", page.List)
		}
		if len(page.Counts) != 0 {
			t.Fatalf("counts must stay empty for a public page, got %#v", page.Counts)
		}
	})

	t.Run("channels carry anonymous tallies", func(t *testing.T) {
		svc, _ := newTestService(t, `{"ok":true,"reactions_type":"private","reactions_count":[
			{"reaction":{"type":"default_reaction","name":"fire"},"count":12}]}`)

		page, err := svc.GetReactions(context.Background(), ym.ChatTarget("c"), 1, nil)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if page.Type != ym.ReactionsPrivate {
			t.Fatalf("expected a private page, got %q", page.Type)
		}
		if len(page.Counts) != 1 || page.Counts[0].Count != 12 {
			t.Fatalf("unexpected counts: %#v", page.Counts)
		}
	})
}

func TestNewEndpointsRejectBadInput(t *testing.T) {
	ctx := context.Background()
	chat := ym.ChatTarget("c")

	cases := []struct {
		name string
		call func(*Service) error
		want error
	}{
		{
			name: "no target",
			call: func(s *Service) error { _, err := s.Pin(ctx, ym.Target{}, 1, nil); return err },
			want: ym.ErrNoTarget,
		},
		{
			name: "two targets",
			call: func(s *Service) error {
				_, err := s.Pin(ctx, ym.Target{ChatID: "c", Login: "l"}, 1, nil)

				return err
			},
			want: ym.ErrAmbiguousTarget,
		},
		{
			name: "pin without a message",
			call: func(s *Service) error { _, err := s.Pin(ctx, chat, 0, nil); return err },
			want: ErrMessageIDRequired,
		},
		{
			name: "reaction without a message",
			call: func(s *Service) error { return s.SendReaction(ctx, chat, 0, nil, nil) },
			want: ErrMessageIDRequired,
		},
		{
			name: "sticker without identifiers",
			call: func(s *Service) error { _, err := s.SendSticker(ctx, chat, "", "", nil); return err },
			want: ErrStickerRequired,
		},
		{
			name: "share without a file id",
			call: func(s *Service) error { _, err := s.ShareFile(ctx, chat, ym.SharedFile{}, nil); return err },
			want: ErrFileIDRequired,
		},
		{
			name: "processing indicator without content",
			call: func(s *Service) error {
				return s.SendTyping(ctx, chat, &SendTypingOptions{Type: ym.TypingProcessing})
			},
			want: ErrProcessingContentRequired,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, doer := newTestService(t, `{"ok":true}`)

			err := tc.call(svc)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if doer.CallCount() != 0 {
				t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
			}
		})
	}
}

func TestSendTypingRejectsOutOfRangeTimeout(t *testing.T) {
	svc, doer := newTestService(t, `{"ok":true}`)

	err := svc.SendTyping(context.Background(), ym.ChatTarget("c"), &SendTypingOptions{Timeout: ym.Ptr(61)})
	if err == nil {
		t.Fatal("expected a timeout range error")
	}
	if doer.CallCount() != 0 {
		t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
	}
}

// Editing a message is modelled as sendText carrying message_id.
func TestEditTextSetsMessageID(t *testing.T) {
	svc, doer := newTestService(t, `{"ok":true,"message_id":5}`)

	_, err := svc.EditText(context.Background(), ym.ChatTarget("c"), 5, "corrected", nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	body := decodeBody(t, doer)
	if got := body["message_id"]; got != float64(5) {
		t.Fatalf("message_id: got %#v, want 5", got)
	}
	if got := body["text"]; got != "corrected" {
		t.Fatalf("text: got %#v", got)
	}
}

// A caller's options must survive EditText rather than being replaced wholesale.
func TestEditTextKeepsCallerOptions(t *testing.T) {
	svc, doer := newTestService(t, `{"ok":true,"message_id":5}`)

	_, err := svc.EditText(context.Background(), ym.ChatTarget("c"), 5, "hi", &SendMessageOptions{
		Important: ym.Ptr(true),
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	body := decodeBody(t, doer)
	if got := body["important"]; got != true {
		t.Fatalf("important: got %#v, want true", got)
	}
	if got := body["message_id"]; got != float64(5) {
		t.Fatalf("message_id: got %#v, want 5", got)
	}
}

func TestSendTextCarriesNewOptions(t *testing.T) {
	svc, doer := newTestService(t, `{"ok":true,"message_id":1}`)

	_, err := svc.SendText(context.Background(), ym.ChatTarget("c"), "hi", &SendMessageOptions{
		ReplyToMessageID: ym.Ptr(ym.MessageID(3)),
		ReplyQuote:       "fragment",
		ActionButtons:    &ym.ActionButtons{Buttons: []ym.ActionButton{{ID: "b1", Title: "Yes"}}},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	body := decodeBody(t, doer)
	for _, key := range []string{"reply_message_id", "reply_quote", "action_buttons"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("body is missing %q; got %v", key, body)
		}
	}
}

func TestForwardsAreSerialised(t *testing.T) {
	svc, doer := newTestService(t, `{"ok":true,"message_id":1}`)

	_, err := svc.SendText(context.Background(), ym.ChatTarget("c"), "", &SendMessageOptions{
		Forwards: []ym.Forward{{ChatID: "src", MessageIDs: []ym.MessageID{1, 2}}},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	body := decodeBody(t, doer)
	want := []any{map[string]any{"chat_id": "src", "message_ids": []any{float64(1), float64(2)}}}
	if !jsonEqual(body["forwards"], want) {
		t.Fatalf("forwards: got %#v", body["forwards"])
	}
}
