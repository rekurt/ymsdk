package messages

import (
	"context"
	"net/http"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/internal/testutil"
)

// serviceWithIdempotency builds a service with automatic payload_id left on,
// which is the default a real caller gets.
func serviceWithIdempotency(t *testing.T) (*Service, *testutil.FakeDoer) {
	t.Helper()

	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":1}`),
	}}
	client := ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer)

	return NewService(client), doer
}

// The API documents payload_id for sendText, sendSticker, sendSystemMessage and
// createPoll only. Stamping it on a share endpoint sends an undocumented field
// and, worse, makes a retry look idempotent when it is not.
func TestShareEndpointsSendNoPayloadID(t *testing.T) {
	cases := []struct {
		name string
		call func(*Service) error
	}{
		{
			name: "ShareFile",
			call: func(s *Service) error {
				_, err := s.ShareFile(context.Background(), ym.ChatTarget("c"),
					ym.SharedFile{FileID: "f1"}, nil)

				return err
			},
		},
		{
			name: "ShareImage",
			call: func(s *Service) error {
				_, err := s.ShareImage(context.Background(), ym.ChatTarget("c"),
					ym.SharedImage{FileID: "f1", Width: 1, Height: 2}, nil)

				return err
			},
		},
		{
			name: "ShareGallery",
			call: func(s *Service) error {
				_, err := s.ShareGallery(context.Background(), ym.ChatTarget("c"),
					[]ym.SharedImage{{FileID: "f1", Width: 1, Height: 2}}, nil)

				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, doer := serviceWithIdempotency(t)

			if err := tc.call(svc); err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if got := payloadIDOf(t, doer.Requests[0]); got != "" {
				t.Fatalf("%s sent an undocumented payload_id: %q", tc.name, got)
			}
		})
	}
}

// The endpoints that do document the key must keep sending it.
func TestDocumentedEndpointsStillSendPayloadID(t *testing.T) {
	t.Run("sendText", func(t *testing.T) {
		svc, doer := serviceWithIdempotency(t)

		if _, err := svc.SendToChat(context.Background(), "c", "hi", nil); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if payloadIDOf(t, doer.Requests[0]) == "" {
			t.Fatal("sendText must carry an idempotency key")
		}
	})

	t.Run("sendSticker", func(t *testing.T) {
		svc, doer := serviceWithIdempotency(t)

		if _, err := svc.SendSticker(context.Background(), ym.ChatTarget("c"), "s", "i", nil); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if payloadIDOf(t, doer.Requests[0]) == "" {
			t.Fatal("sendSticker must carry an idempotency key")
		}
	})

	t.Run("sendSystemMessage", func(t *testing.T) {
		svc, doer := serviceWithIdempotency(t)

		if _, err := svc.SendSystemMessage(context.Background(), ym.ChatTarget("c"), "x", nil); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if payloadIDOf(t, doer.Requests[0]) == "" {
			t.Fatal("sendSystemMessage must carry an idempotency key")
		}
	})
}

func TestSendTypingValidatesProcessingText(t *testing.T) {
	long := make([]rune, 101)
	for i := range long {
		long[i] = 'a'
	}

	cases := []struct {
		name    string
		content *ym.ProcessingContent
	}{
		{
			name:    "text display with no text",
			content: &ym.ProcessingContent{Display: ym.ProcessingDisplayText},
		},
		{
			name:    "text display over the limit",
			content: &ym.ProcessingContent{Display: ym.ProcessingDisplayText, Text: string(long)},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, doer := newTestService(t, `{"ok":true}`)

			err := svc.SendTyping(context.Background(), ym.ChatTarget("c"), &SendTypingOptions{
				Type:              ym.TypingProcessing,
				ProcessingContent: tc.content,
			})
			if err == nil {
				t.Fatal("expected a validation error")
			}
			if doer.CallCount() != 0 {
				t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
			}
		})
	}

	t.Run("default display needs no text", func(t *testing.T) {
		svc, doer := newTestService(t, `{"ok":true}`)

		err := svc.SendTyping(context.Background(), ym.ChatTarget("c"), &SendTypingOptions{
			Type:              ym.TypingProcessing,
			ProcessingContent: &ym.ProcessingContent{Display: ym.ProcessingDisplayDefault},
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if doer.CallCount() != 1 {
			t.Fatalf("expected the request to be sent, got %d calls", doer.CallCount())
		}
	})
}
