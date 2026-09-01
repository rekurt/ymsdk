package messages

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
)

// The other paginated endpoints reject an out-of-range limit locally; this one
// should not be the exception that spends a round trip to learn the same thing.
func TestGetReactionsRejectsOutOfRangeLimit(t *testing.T) {
	for _, limit := range []int{0, -1, ym.MaxPageLimit + 1} {
		svc, doer := newTestService(t, `{"ok":true,"reactions_type":"public"}`)

		_, err := svc.GetReactions(context.Background(), ym.ChatTarget("c"), 1,
			&GetReactionsOptions{Limit: ym.Ptr(limit)})
		if err == nil {
			t.Fatalf("limit %d: expected a range error", limit)
		}
		if doer.CallCount() != 0 {
			t.Fatalf("limit %d: invalid input must not reach the network", limit)
		}
	}
}

func TestGetReactionsAcceptsValidLimit(t *testing.T) {
	svc, doer := newTestService(t, `{"ok":true,"reactions_type":"public"}`)

	_, err := svc.GetReactions(context.Background(), ym.ChatTarget("c"), 1,
		&GetReactionsOptions{Limit: ym.Ptr(ym.MaxPageLimit)})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if doer.CallCount() != 1 {
		t.Fatalf("expected the request to be sent, got %d calls", doer.CallCount())
	}
}

// The API requires width and height on a shared image, so a SharedImage built
// from a file id alone is a deterministic 400 waiting to happen.
func TestShareRejectsImagesWithoutDimensions(t *testing.T) {
	cases := []struct {
		name string
		call func(*Service) error
	}{
		{
			name: "ShareImage without dimensions",
			call: func(s *Service) error {
				_, err := s.ShareImage(context.Background(), ym.ChatTarget("c"),
					ym.SharedImage{FileID: "f1"}, nil)

				return err
			},
		},
		{
			name: "ShareImage with a zero height",
			call: func(s *Service) error {
				_, err := s.ShareImage(context.Background(), ym.ChatTarget("c"),
					ym.SharedImage{FileID: "f1", Width: 10}, nil)

				return err
			},
		},
		{
			name: "ShareGallery with a dimensionless image",
			call: func(s *Service) error {
				_, err := s.ShareGallery(context.Background(), ym.ChatTarget("c"),
					[]ym.SharedImage{{FileID: "f1", Width: 1, Height: 1}, {FileID: "f2"}}, nil)

				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, doer := newTestService(t, `{"ok":true,"message_id":1}`)

			err := tc.call(svc)
			if !errors.Is(err, ErrImageDimensionsRequired) {
				t.Fatalf("expected ErrImageDimensionsRequired, got %v", err)
			}
			if doer.CallCount() != 0 {
				t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
			}
		})
	}
}

func TestShareAcceptsImagesWithDimensions(t *testing.T) {
	svc, doer := newTestService(t, `{"ok":true,"message_id":1}`)

	_, err := svc.ShareImage(context.Background(), ym.ChatTarget("c"),
		ym.SharedImage{FileID: "f1", Width: 10, Height: 20}, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if doer.CallCount() != 1 {
		t.Fatalf("expected the request to be sent, got %d calls", doer.CallCount())
	}
}

// The API documents reply_quote as requiring reply_message_id, and forwards as
// mutually exclusive with it. Sending either combination is a request that can
// only fail, so it should fail here instead.
func TestSendRejectsIncompatibleReplyAndForwardOptions(t *testing.T) {
	cases := []struct {
		name string
		opts *SendMessageOptions
		want error
	}{
		{
			name: "quote without a reply target",
			opts: &SendMessageOptions{ReplyQuote: "fragment"},
			want: ErrReplyQuoteNeedsReply,
		},
		{
			name: "forwards combined with a reply",
			opts: &SendMessageOptions{
				ReplyToMessageID: ym.Ptr(ym.MessageID(7)),
				Forwards:         []ym.Forward{{ChatID: "src", MessageIDs: []ym.MessageID{1}}},
			},
			want: ErrForwardsWithReply,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, doer := newTestService(t, `{"ok":true,"message_id":1}`)

			_, err := svc.SendToChat(context.Background(), "c", "hi", tc.opts)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if doer.CallCount() != 0 {
				t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
			}
		})
	}
}

func TestSendAcceptsValidReplyAndForwardOptions(t *testing.T) {
	cases := []struct {
		name string
		opts *SendMessageOptions
	}{
		{
			name: "quote with a reply target",
			opts: &SendMessageOptions{
				ReplyToMessageID: ym.Ptr(ym.MessageID(7)),
				ReplyQuote:       "fragment",
			},
		},
		{
			name: "forwards on their own",
			opts: &SendMessageOptions{
				Forwards: []ym.Forward{{ChatID: "src", MessageIDs: []ym.MessageID{1}}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, doer := newTestService(t, `{"ok":true,"message_id":1}`)

			if _, err := svc.SendToChat(context.Background(), "c", "hi", tc.opts); err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if doer.CallCount() != 1 {
				t.Fatalf("expected the request to be sent, got %d calls", doer.CallCount())
			}
		})
	}
}

// Zero means "no message" everywhere else in this package; EditText should not
// be the one place that serialises it and lets the API reject the request.
func TestEditTextRejectsAZeroMessageID(t *testing.T) {
	svc, doer := newTestService(t, `{"ok":true,"message_id":1}`)

	_, err := svc.EditText(context.Background(), ym.ChatTarget("c"), 0, "text", nil)
	if !errors.Is(err, ErrMessageIDRequired) {
		t.Fatalf("expected ErrMessageIDRequired, got %v", err)
	}
	if doer.CallCount() != 0 {
		t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
	}
}

// oversizedKeyboard builds a suggest keyboard past the documented 100-button cap.
func oversizedKeyboard() *ym.SuggestButtons {
	rows := make([][]ym.InlineSuggestButton, 0, 21)
	for range 21 {
		rows = append(rows, make([]ym.InlineSuggestButton, 5))
	}

	return &ym.SuggestButtons{Buttons: rows}
}

// The upload endpoints build their own multipart bodies rather than going
// through SendMessageOptions, and were skipping the limits every other send
// path enforces — so the documented local check did not apply to them.
func TestUploadsEnforceDocumentedLimits(t *testing.T) {
	chat := ym.Ptr(ym.ChatID("c1"))

	cases := []struct {
		name string
		call func(*Service) error
	}{
		{
			name: "SendFile with an oversized keyboard",
			call: func(s *Service) error {
				_, err := s.SendFile(context.Background(), &SendFileRequest{
					ChatID: chat, Document: strings.NewReader("x"), Filename: "a.txt",
					SuggestButtons: oversizedKeyboard(),
				})

				return err
			},
		},
		{
			name: "SendImage with an oversized keyboard",
			call: func(s *Service) error {
				_, err := s.SendImage(context.Background(), &SendImageRequest{
					ChatID: chat, Image: strings.NewReader("x"), Filename: "a.png",
					SuggestButtons: oversizedKeyboard(),
				})

				return err
			},
		},
		{
			name: "SendGallery with an oversized keyboard",
			call: func(s *Service) error {
				_, err := s.SendGallery(context.Background(), &SendGalleryRequest{
					ChatID:         chat,
					Images:         []FilePart{{Reader: strings.NewReader("x"), Filename: "a.png"}},
					SuggestButtons: oversizedKeyboard(),
				})

				return err
			},
		},
		{
			name: "SendGallery with an over-long caption",
			call: func(s *Service) error {
				_, err := s.SendGallery(context.Background(), &SendGalleryRequest{
					ChatID: chat,
					Images: []FilePart{{Reader: strings.NewReader("x"), Filename: "a.png"}},
					Text:   strings.Repeat("a", ym.MaxTextLength+1),
				})

				return err
			},
		},
		{
			name: "SendGallery over the image limit",
			call: func(s *Service) error {
				parts := make([]FilePart, ym.MaxGalleryImages+1)
				for i := range parts {
					parts[i] = FilePart{Reader: strings.NewReader("x"), Filename: "a.png"}
				}
				_, err := s.SendGallery(context.Background(), &SendGalleryRequest{ChatID: chat, Images: parts})

				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, doer := newTestService(t, `{"ok":true,"message_id":1}`)

			err := tc.call(svc)

			var limitErr *ym.LimitError
			if !errors.As(err, &limitErr) {
				t.Fatalf("expected a *ym.LimitError, got %T (%v)", err, err)
			}
			if doer.CallCount() != 0 {
				t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
			}
		})
	}
}

// Documented limits are supposed to be reachable with errors.As no matter which
// endpoint enforces them. These three reported ad-hoc errors instead.
func TestDocumentedLimitsAllReportLimitError(t *testing.T) {
	cases := []struct {
		name string
		call func(*Service) error
	}{
		{
			name: "ShareGallery over the image limit",
			call: func(s *Service) error {
				images := make([]ym.SharedImage, ym.MaxGalleryImages+1)
				for i := range images {
					images[i] = ym.SharedImage{FileID: "f", Width: 1, Height: 1}
				}
				_, err := s.ShareGallery(context.Background(), ym.ChatTarget("c"), images, nil)

				return err
			},
		},
		{
			name: "processing text below the range",
			call: func(s *Service) error {
				return s.SendTyping(context.Background(), ym.ChatTarget("c"), &SendTypingOptions{
					Type:              ym.TypingProcessing,
					ProcessingContent: &ym.ProcessingContent{Display: ym.ProcessingDisplayText},
				})
			},
		},
		{
			name: "processing text above the range",
			call: func(s *Service) error {
				return s.SendTyping(context.Background(), ym.ChatTarget("c"), &SendTypingOptions{
					Type: ym.TypingProcessing,
					ProcessingContent: &ym.ProcessingContent{
						Display: ym.ProcessingDisplayText,
						Text:    strings.Repeat("a", ym.MaxProcessingTextLength+1),
					},
				})
			},
		},
		{
			name: "typing timeout out of range",
			call: func(s *Service) error {
				return s.SendTyping(context.Background(), ym.ChatTarget("c"), &SendTypingOptions{
					Timeout: ym.Ptr(ym.MaxTypingTimeout + 1),
				})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, doer := newTestService(t, `{"ok":true,"message_id":1}`)

			err := tc.call(svc)

			var limitErr *ym.LimitError
			if !errors.As(err, &limitErr) {
				t.Fatalf("expected a *ym.LimitError, got %T (%v)", err, err)
			}
			if doer.CallCount() != 0 {
				t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
			}
		})
	}
}
