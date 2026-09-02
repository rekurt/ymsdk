package messages

import (
	"context"
	"errors"
	"fmt"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

// ShareOptions holds parameters for resending a single already-uploaded
// attachment. It extends [SendMessageOptions] with the attachment's filename.
type ShareOptions struct {
	SendMessageOptions
	// Filename overrides the name shown to the recipient.
	Filename string
}

// ShareGalleryOptions holds parameters for resending an album of
// already-uploaded images.
type ShareGalleryOptions struct {
	SendMessageOptions
	// Text is an optional caption, at most 6000 characters.
	Text string
}

type shareFileRequest struct {
	messageEnvelope
	Document ym.SharedFile `json:"document"`
	Filename string        `json:"filename,omitempty"`
}

type shareImageRequest struct {
	messageEnvelope
	Image    ym.SharedImage `json:"image"`
	Filename string         `json:"filename,omitempty"`
}

type shareGalleryRequest struct {
	messageEnvelope
	Images []ym.SharedImage `json:"images"`
	Text   string           `json:"text,omitempty"`
}

// ShareFile sends a file that is already stored in the messenger, identified by
// its file_id, without uploading the bytes again.
//
// The identifier comes from the File in an update or from the response to an
// earlier upload.
func (s *Service) ShareFile(
	ctx context.Context, target ym.Target, doc ym.SharedFile, opts *ShareOptions,
) (*ym.Message, error) {
	if err := ym.ValidateTarget(target); err != nil {
		return nil, err
	}
	if doc.FileID == "" {
		return nil, ymerrors.ErrFileIDRequired
	}
	if err := validateSend("", shareMessageOptions(opts)); err != nil {
		return nil, err
	}

	req := shareFileRequest{
		messageEnvelope: s.shareEnvelope(target, shareMessageOptions(opts)),
		Document:        doc,
	}
	if opts != nil {
		req.Filename = opts.Filename
	}

	return s.postForMessage(ctx, ym.EndpointMessagesShareFile, req)
}

// ShareImage sends an image that is already stored in the messenger.
//
// Width and height are required by the API; take them from the [ym.Image] in an
// update or from an earlier upload's response.
func (s *Service) ShareImage(
	ctx context.Context, target ym.Target, img ym.SharedImage, opts *ShareOptions,
) (*ym.Message, error) {
	if err := ym.ValidateTarget(target); err != nil {
		return nil, err
	}
	if err := validateSharedImage(img); err != nil {
		return nil, err
	}
	if err := validateSend("", shareMessageOptions(opts)); err != nil {
		return nil, err
	}

	req := shareImageRequest{
		messageEnvelope: s.shareEnvelope(target, shareMessageOptions(opts)),
		Image:           img,
	}
	if opts != nil {
		req.Filename = opts.Filename
	}

	return s.postForMessage(ctx, ym.EndpointMessagesShareImage, req)
}

// ShareGallery sends an album of images that are already stored in the
// messenger. Between 1 and 10 images.
func (s *Service) ShareGallery(
	ctx context.Context, target ym.Target, images []ym.SharedImage, opts *ShareGalleryOptions,
) (*ym.Message, error) {
	if err := ym.ValidateTarget(target); err != nil {
		return nil, err
	}
	if err := validateSharedImages(images); err != nil {
		return nil, err
	}

	var base *SendMessageOptions
	req := shareGalleryRequest{Images: images}
	if opts != nil {
		base = &opts.SendMessageOptions
		req.Text = opts.Text
	}
	if err := validateSend(req.Text, base); err != nil {
		return nil, err
	}
	req.messageEnvelope = s.shareEnvelope(target, base)

	return s.postForMessage(ctx, ym.EndpointMessagesShareGallery, req)
}

func validateSharedImages(images []ym.SharedImage) error {
	if len(images) == 0 {
		return errors.New("yandex-messenger: at least one image is required")
	}
	if err := ym.ValidateCount("images", len(images), ym.MaxGalleryImages); err != nil {
		return err
	}
	for i, img := range images {
		if err := validateSharedImage(img); err != nil {
			return fmt.Errorf("yandex-messenger: image %d: %w", i, err)
		}
	}

	return nil
}

// validateSharedImage checks the fields the API marks required on a shared
// image. Width and height come from the original upload or update; sending
// zeroes produces a request the server can only reject.
func validateSharedImage(img ym.SharedImage) error {
	if img.FileID == "" {
		return ymerrors.ErrFileIDRequired
	}
	if img.Width <= 0 || img.Height <= 0 {
		return ymerrors.ErrImageDimensionsRequired
	}

	return nil
}

func shareMessageOptions(opts *ShareOptions) *SendMessageOptions {
	if opts == nil {
		return nil
	}

	return &opts.SendMessageOptions
}
