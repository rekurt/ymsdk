package messages

import (
	"context"
	"errors"

	"github.com/rekurt/ymsdk/client/ym"
)

// ErrStickerRequired is returned when a sticker is sent without identifying it.
var ErrStickerRequired = errors.New("yandex-messenger: sticker_set_id and sticker_id are required")

type sendStickerRequest struct {
	messageEnvelope
	StickerSetID string `json:"sticker_set_id"`
	StickerID    string `json:"sticker_id"`
}

// SendSticker sends a sticker.
//
// Both identifiers come from a [ym.Sticker] received in an update: SetID and ID.
func (s *Service) SendSticker(
	ctx context.Context, target ym.Target, setID, stickerID string, opts *SendMessageOptions,
) (*ym.Message, error) {
	if err := ym.ValidateTarget(target); err != nil {
		return nil, err
	}
	if setID == "" || stickerID == "" {
		return nil, ErrStickerRequired
	}

	req := sendStickerRequest{
		messageEnvelope: s.envelope(target, opts),
		StickerSetID:    setID,
		StickerID:       stickerID,
	}

	return s.postForMessage(ctx, ym.EndpointMessagesSendSticker, req)
}
