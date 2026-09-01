package messages

import (
	"context"
	"errors"
	"fmt"

	"github.com/rekurt/ymsdk/client/ym"
)

// ErrFileIDRequired is returned when a share operation omits the file identifier.
var ErrFileIDRequired = errors.New("yandex-messenger: file_id is required")

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
		return nil, ErrFileIDRequired
	}

	req := shareFileRequest{
		messageEnvelope: s.envelope(target, shareMessageOptions(opts)),
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
	if img.FileID == "" {
		return nil, ErrFileIDRequired
	}

	req := shareImageRequest{
		messageEnvelope: s.envelope(target, shareMessageOptions(opts)),
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
	req.messageEnvelope = s.envelope(target, base)

	return s.postForMessage(ctx, ym.EndpointMessagesShareGallery, req)
}

func validateSharedImages(images []ym.SharedImage) error {
	if len(images) == 0 {
		return errors.New("yandex-messenger: at least one image is required")
	}
	if len(images) > maxGalleryImages {
		return fmt.Errorf(
			"yandex-messenger: gallery images limit exceeded: %d (max %d)",
			len(images), maxGalleryImages,
		)
	}
	for i, img := range images {
		if img.FileID == "" {
			return fmt.Errorf("yandex-messenger: image %d: %w", i, ErrFileIDRequired)
		}
	}

	return nil
}

func shareMessageOptions(opts *ShareOptions) *SendMessageOptions {
	if opts == nil {
		return nil
	}

	return &opts.SendMessageOptions
}
