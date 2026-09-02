package messages

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
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
			if !errors.Is(err, ymerrors.ErrImageDimensionsRequired) {
				t.Fatalf("expected ymerrors.ErrImageDimensionsRequired, got %v", err)
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
			want: ymerrors.ErrReplyQuoteNeedsReply,
		},
		{
			name: "forwards combined with a reply",
			opts: &SendMessageOptions{
				ReplyToMessageID: ym.Ptr(ym.MessageID(7)),
				Forwards:         []ym.Forward{{ChatID: "src", MessageIDs: []ym.MessageID{1}}},
			},
			want: ymerrors.ErrForwardsWithReply,
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
	if !errors.Is(err, ymerrors.ErrMessageIDRequired) {
		t.Fatalf("expected ymerrors.ErrMessageIDRequired, got %v", err)
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

// nil means "remove the reaction"; a non-nil one has to identify a reaction,
// since the API requires both fields.
func TestSendReactionRejectsAnIncompleteReaction(t *testing.T) {
	cases := []struct {
		name     string
		reaction *ym.Reaction
	}{
		{"empty name", ym.DefaultReaction("")},
		{"empty type", &ym.Reaction{Name: "like"}},
		{"both empty", &ym.Reaction{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, doer := newTestService(t, `{"ok":true}`)

			err := svc.SendReaction(context.Background(), ym.ChatTarget("c"), 1, tc.reaction, nil)
			if !errors.Is(err, ymerrors.ErrIncompleteReaction) {
				t.Fatalf("expected ymerrors.ErrIncompleteReaction, got %v", err)
			}
			if doer.CallCount() != 0 {
				t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
			}
		})
	}

	t.Run("nil still removes the reaction", func(t *testing.T) {
		svc, doer := newTestService(t, `{"ok":true}`)

		if err := svc.SendReaction(context.Background(), ym.ChatTarget("c"), 1, nil, nil); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if doer.CallCount() != 1 {
			t.Fatalf("expected the removal to be sent, got %d calls", doer.CallCount())
		}
	})
}

// Zero means "no message" throughout this package, and EditText guards its
// scalar argument — but the same value arriving through the option fields was
// serialised and left for the API to reject.
func TestSendRejectsZeroMessageIDsInOptions(t *testing.T) {
	cases := []struct {
		name string
		opts *SendMessageOptions
	}{
		{"zero MessageID", &SendMessageOptions{MessageID: ym.Ptr(ym.MessageID(0))}},
		{"zero ReplyToMessageID", &SendMessageOptions{ReplyToMessageID: ym.Ptr(ym.MessageID(0))}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, doer := newTestService(t, `{"ok":true,"message_id":1}`)

			_, err := svc.SendToChat(context.Background(), "c", "hi", tc.opts)
			if !errors.Is(err, ymerrors.ErrMessageIDRequired) {
				t.Fatalf("expected ymerrors.ErrMessageIDRequired, got %v", err)
			}
			if doer.CallCount() != 0 {
				t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
			}
		})
	}
}

// A send response without message_id is malformed: returning a Message whose ID
// is zero lets the caller record an unusable id or believe a broken response
// succeeded.
func TestSendRejectsAResponseWithoutMessageID(t *testing.T) {
	cases := []struct {
		name string
		call func(*Service) error
	}{
		{
			name: "SendText",
			call: func(s *Service) error {
				_, err := s.SendToChat(context.Background(), "c", "hi", nil)

				return err
			},
		},
		{
			name: "SendSticker",
			call: func(s *Service) error {
				_, err := s.SendSticker(context.Background(), ym.ChatTarget("c"), "set", "st", nil)

				return err
			},
		},
		{
			name: "ShareFile",
			call: func(s *Service) error {
				_, err := s.ShareFile(context.Background(), ym.ChatTarget("c"), ym.SharedFile{FileID: "f"}, nil)

				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newTestService(t, `{"ok":true}`)

			if err := tc.call(svc); !errors.Is(err, ymerrors.ErrInvalidResponse) {
				t.Fatalf("expected ErrInvalidResponse, got %v", err)
			}
		})
	}
}

// The API marks the system message text required, and unlike an ordinary send
// there is no attachment that could give an empty body meaning.
func TestSendSystemMessageRequiresText(t *testing.T) {
	svc, doer := newTestService(t, `{"ok":true,"message_id":1}`)

	_, err := svc.SendSystemMessage(context.Background(), ym.ChatTarget("c"), "", nil)
	if !errors.Is(err, ymerrors.ErrTextRequired) {
		t.Fatalf("expected ymerrors.ErrTextRequired, got %v", err)
	}
	if doer.CallCount() != 0 {
		t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
	}
}

// pin documents message_id in its response; a zero one is unusable.
func TestPinRejectsAResponseWithoutMessageID(t *testing.T) {
	svc, _ := newTestService(t, `{"ok":true}`)

	res, err := svc.Pin(context.Background(), ym.ChatTarget("c"), 7, nil)
	if !errors.Is(err, ymerrors.ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got %v", err)
	}
	if res != nil {
		t.Fatalf("expected no result alongside the error, got %#v", res)
	}
}

// The API requires the display discriminator; the zero value of
// ProcessingContent leaves it empty, which is the shape a caller writes by
// accident most easily.
func TestSendTypingRequiresAProcessingDisplay(t *testing.T) {
	svc, doer := newTestService(t, `{"ok":true}`)

	err := svc.SendTyping(context.Background(), ym.ChatTarget("c"), &SendTypingOptions{
		Type:              ym.TypingProcessing,
		ProcessingContent: &ym.ProcessingContent{},
	})
	if !errors.Is(err, ymerrors.ErrProcessingDisplayRequired) {
		t.Fatalf("expected ymerrors.ErrProcessingDisplayRequired, got %v", err)
	}
	if doer.CallCount() != 0 {
		t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
	}
}

// The error already states that only "default" and "text" are valid, but the
// check compared against the empty string alone, so an unsupported value was
// still sent. A request parameter is the caller's to get right — unlike a
// response discriminator, where an unfamiliar value may simply be newer.
func TestSendTypingRejectsAnUnsupportedProcessingDisplay(t *testing.T) {
	svc, doer := newTestService(t, `{"ok":true}`)

	err := svc.SendTyping(context.Background(), ym.ChatTarget("c"), &SendTypingOptions{
		Type: ym.TypingProcessing,
		ProcessingContent: &ym.ProcessingContent{
			Display: ym.ProcessingDisplay("bogus"),
			Text:    "hi",
		},
	})
	if !errors.Is(err, ymerrors.ErrProcessingDisplayRequired) {
		t.Fatalf("expected ymerrors.ErrProcessingDisplayRequired, got %v", err)
	}
	if doer.CallCount() != 0 {
		t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
	}
}

// The two documented values keep working.
func TestSendTypingAcceptsBothDocumentedDisplays(t *testing.T) {
	cases := []*ym.ProcessingContent{
		{Display: ym.ProcessingDisplayDefault},
		{Display: ym.ProcessingDisplayText, Text: "Thinking…"},
	}

	for _, pc := range cases {
		svc, doer := newTestService(t, `{"ok":true}`)

		err := svc.SendTyping(context.Background(), ym.ChatTarget("c"), &SendTypingOptions{
			Type:              ym.TypingProcessing,
			ProcessingContent: pc,
		})
		if err != nil {
			t.Fatalf("display %q: expected nil error, got %v", pc.Display, err)
		}
		if doer.CallCount() != 1 {
			t.Fatalf("display %q: expected the request to be sent", pc.Display)
		}
	}
}

// The same gap the display discriminator had, one field over: type is a request
// parameter with exactly two documented values, and an arbitrary one was passed
// straight through to the API as the discriminator.
func TestSendTypingRejectsAnUnsupportedType(t *testing.T) {
	svc, doer := newTestService(t, `{"ok":true}`)

	err := svc.SendTyping(context.Background(), ym.ChatTarget("c"), &SendTypingOptions{
		Type: ym.TypingType("bogus"),
	})
	if !errors.Is(err, ymerrors.ErrTypingTypeInvalid) {
		t.Fatalf("expected ymerrors.ErrTypingTypeInvalid, got %v", err)
	}
	if doer.CallCount() != 0 {
		t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
	}
}

// The zero value means "let the server apply its default" and is omitted from
// the payload, so it must keep working; both documented values must too.
func TestSendTypingAcceptsTheDefaultAndBothDocumentedTypes(t *testing.T) {
	cases := []*SendTypingOptions{
		{},
		{Type: ym.TypingText},
		{Type: ym.TypingProcessing, ProcessingContent: &ym.ProcessingContent{
			Display: ym.ProcessingDisplayDefault,
		}},
	}

	for _, opts := range cases {
		svc, doer := newTestService(t, `{"ok":true}`)

		if err := svc.SendTyping(context.Background(), ym.ChatTarget("c"), opts); err != nil {
			t.Fatalf("type %q: expected nil error, got %v", opts.Type, err)
		}
		if doer.CallCount() != 1 {
			t.Fatalf("type %q: expected the request to be sent", opts.Type)
		}
	}
}

// A processing type still needs its content: adding the type check must not
// short-circuit the check that already guarded that pairing.
func TestSendTypingStillRequiresProcessingContent(t *testing.T) {
	svc, _ := newTestService(t, `{"ok":true}`)

	err := svc.SendTyping(context.Background(), ym.ChatTarget("c"), &SendTypingOptions{
		Type: ym.TypingProcessing,
	})
	if !errors.Is(err, ymerrors.ErrProcessingContentRequired) {
		t.Fatalf("expected ymerrors.ErrProcessingContentRequired, got %v", err)
	}
}
