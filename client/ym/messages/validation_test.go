package messages

import (
	"context"
	"errors"
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
