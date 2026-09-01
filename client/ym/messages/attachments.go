package messages

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

const (
	maxGalleryImages  = 10
	maxActionButtons  = 6
	maxSuggestButtons = 100
	maxGalleryText    = 6000
)

// attachmentCommon holds the message parameters every multipart send shares.
// Keeping them in one place is what stops the per-method drift this package
// used to carry.
type attachmentCommon struct {
	ChatID              *ym.ChatID
	Login               *ym.UserLogin
	ThreadID            *ym.ThreadID
	MessageID           *ym.MessageID
	ReplyMessageID      *ym.MessageID
	ReplyQuote          string
	Forwards            []ym.Forward
	DisableNotification *bool
	Important           *bool
	SuggestButtons      *ym.SuggestButtons
	ActionButtons       *ym.ActionButtons
}

// validate enforces the constraints the Bot API documents for these parameters.
func (c attachmentCommon) validate() error {
	if err := ym.ValidateRecipient(c.ChatID, c.Login); err != nil {
		return err
	}
	if c.ReplyQuote != "" && c.ReplyMessageID == nil {
		return errors.New("reply_quote requires reply_message_id")
	}
	if len(c.Forwards) > 0 && c.ReplyMessageID != nil {
		return errors.New("forwards cannot be combined with reply_message_id")
	}
	if c.ActionButtons != nil && len(c.ActionButtons.Buttons) > maxActionButtons {
		return fmt.Errorf("action buttons limit exceeded: %d (max %d)", len(c.ActionButtons.Buttons), maxActionButtons)
	}
	if c.SuggestButtons != nil {
		total := 0
		for _, row := range c.SuggestButtons.Buttons {
			total += len(row)
		}
		if total > maxSuggestButtons {
			return fmt.Errorf("suggest buttons limit exceeded: %d (max %d)", total, maxSuggestButtons)
		}
	}

	return nil
}

// writeCommonFields writes every shared parameter that is set. Unset optional
// parameters are omitted rather than sent empty, so the server applies its own
// documented defaults.
func writeCommonFields(writer *multipart.Writer, c attachmentCommon) error {
	if c.ChatID != nil {
		if err := writer.WriteField("chat_id", string(*c.ChatID)); err != nil {
			return err
		}
	}
	if c.Login != nil {
		if err := writer.WriteField("login", string(*c.Login)); err != nil {
			return err
		}
	}
	if c.ThreadID != nil {
		if err := writer.WriteField("thread_id", strconv.FormatInt(int64(*c.ThreadID), 10)); err != nil {
			return err
		}
	}
	for _, f := range []struct {
		name string
		id   *ym.MessageID
	}{{"message_id", c.MessageID}, {"reply_message_id", c.ReplyMessageID}} {
		if f.id != nil {
			if err := writer.WriteField(f.name, strconv.FormatInt(int64(*f.id), 10)); err != nil {
				return err
			}
		}
	}
	if c.ReplyQuote != "" {
		if err := writer.WriteField("reply_quote", c.ReplyQuote); err != nil {
			return err
		}
	}
	for _, f := range []struct {
		name string
		val  *bool
	}{{"disable_notification", c.DisableNotification}, {"important", c.Important}} {
		if f.val != nil {
			if err := writer.WriteField(f.name, strconv.FormatBool(*f.val)); err != nil {
				return err
			}
		}
	}
	for _, f := range []struct {
		name string
		val  any
	}{{"forwards", c.Forwards}, {"suggest_buttons", c.SuggestButtons}, {"action_buttons", c.ActionButtons}} {
		if err := writeJSONField(writer, f.name, f.val); err != nil {
			return err
		}
	}

	return nil
}

// writeJSONField marshals val into a form field, skipping nil and empty values.
func writeJSONField(writer *multipart.Writer, name string, val any) error {
	switch v := val.(type) {
	case []ym.Forward:
		if len(v) == 0 {
			return nil
		}
	case *ym.SuggestButtons:
		if v == nil {
			return nil
		}
	case *ym.ActionButtons:
		if v == nil {
			return nil
		}
	}
	encoded, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", name, err)
	}

	return writer.WriteField(name, string(encoded))
}

// sanitizeFilename escapes special characters in a filename for Content-Disposition headers.
func sanitizeFilename(name string) string {
	return strings.NewReplacer(`"`, `\"`, `\`, `\\`).Replace(name)
}

// SendFileRequest contains parameters for sending a file attachment.
// Exactly one of ChatID or Login must be set.
type SendFileRequest struct {
	ChatID   *ym.ChatID
	Login    *ym.UserLogin
	Document io.Reader
	Filename string

	// MimeType overrides the Content-Type of the uploaded document part.
	// Empty leaves the part without an explicit Content-Type.
	MimeType string

	ThreadID            *ym.ThreadID
	MessageID           *ym.MessageID
	ReplyMessageID      *ym.MessageID
	ReplyQuote          string
	Forwards            []ym.Forward
	DisableNotification *bool
	Important           *bool
	SuggestButtons      *ym.SuggestButtons
	ActionButtons       *ym.ActionButtons
}

// FileMeta holds metadata about a downloaded file.
type FileMeta struct {
	FileID        string
	ContentType   string
	ContentLength int64
}

// SendImageRequest contains parameters for sending an image attachment.
// Exactly one of ChatID or Login must be set.
type SendImageRequest struct {
	ChatID   *ym.ChatID
	Login    *ym.UserLogin
	Image    io.Reader
	Filename string

	ThreadID            *ym.ThreadID
	MessageID           *ym.MessageID
	ReplyMessageID      *ym.MessageID
	ReplyQuote          string
	Forwards            []ym.Forward
	DisableNotification *bool
	Important           *bool
	SuggestButtons      *ym.SuggestButtons
	ActionButtons       *ym.ActionButtons
}

// FilePart represents a single file in a gallery upload.
type FilePart struct {
	Reader   io.Reader
	Filename string
}

// SendGalleryRequest contains parameters for sending a gallery of images.
// Exactly one of ChatID or Login must be set.
type SendGalleryRequest struct {
	ChatID *ym.ChatID
	Login  *ym.UserLogin
	Images []FilePart
	Text   string

	ThreadID            *ym.ThreadID
	MessageID           *ym.MessageID
	ReplyMessageID      *ym.MessageID
	ReplyQuote          string
	Forwards            []ym.Forward
	DisableNotification *bool
	Important           *bool
	SuggestButtons      *ym.SuggestButtons
	ActionButtons       *ym.ActionButtons
}

// DeleteMessageRequest contains parameters for deleting a message.
// Exactly one of ChatID or Login must be set.
type DeleteMessageRequest struct {
	ChatID    *ym.ChatID    `json:"chat_id,omitempty"`
	Login     *ym.UserLogin `json:"login,omitempty"`
	MessageID ym.MessageID  `json:"message_id"`
	ThreadID  *ym.ThreadID  `json:"thread_id,omitempty"`
}

// SendFile uploads and sends a file attachment via multipart/form-data.
func (s *Service) SendFile(ctx context.Context, req *SendFileRequest) (*ym.Message, error) {
	common := req.common()
	if err := common.validate(); err != nil {
		return nil, err
	}
	if req.Document == nil || req.Filename == "" {
		return nil, errors.New("document and filename are required")
	}
	payload, contentType, err := buildSingleFilePayload(common, "document", req.Filename, req.MimeType, req.Document)
	if err != nil {
		return nil, err
	}

	parsed, err := s.doMultipart(ctx, "/bot/v1/messages/sendFile/", contentType, payload)
	if err != nil {
		return nil, err
	}
	msg := &ym.Message{ID: parsed.MessageID}
	if parsed.FileID != "" {
		msg.Document = &ym.File{ID: parsed.FileID}
	}

	return msg, nil
}

func (r *SendFileRequest) common() attachmentCommon {
	return attachmentCommon{
		ChatID: r.ChatID, Login: r.Login, ThreadID: r.ThreadID,
		MessageID: r.MessageID, ReplyMessageID: r.ReplyMessageID, ReplyQuote: r.ReplyQuote,
		Forwards: r.Forwards, DisableNotification: r.DisableNotification, Important: r.Important,
		SuggestButtons: r.SuggestButtons, ActionButtons: r.ActionButtons,
	}
}

// SendImage uploads and sends an image attachment via multipart/form-data.
func (s *Service) SendImage(ctx context.Context, req *SendImageRequest) (*ym.Message, error) {
	common := req.common()
	if err := common.validate(); err != nil {
		return nil, err
	}
	if req.Image == nil || req.Filename == "" {
		return nil, errors.New("image and filename are required")
	}
	payload, contentType, err := buildSingleFilePayload(common, "image", req.Filename, "", req.Image)
	if err != nil {
		return nil, err
	}

	parsed, err := s.doMultipart(ctx, "/bot/v1/messages/sendImage/", contentType, payload)
	if err != nil {
		return nil, err
	}
	msg := &ym.Message{ID: parsed.MessageID}
	if parsed.FileID != "" {
		msg.Image = &ym.Image{FileID: parsed.FileID, Width: parsed.Width, Height: parsed.Height}
	}

	return msg, nil
}

func (r *SendImageRequest) common() attachmentCommon {
	return attachmentCommon{
		ChatID: r.ChatID, Login: r.Login, ThreadID: r.ThreadID,
		MessageID: r.MessageID, ReplyMessageID: r.ReplyMessageID, ReplyQuote: r.ReplyQuote,
		Forwards: r.Forwards, DisableNotification: r.DisableNotification, Important: r.Important,
		SuggestButtons: r.SuggestButtons, ActionButtons: r.ActionButtons,
	}
}

// SendGallery uploads and sends multiple images as a gallery.
func (s *Service) SendGallery(ctx context.Context, req *SendGalleryRequest) (*ym.Message, error) {
	common := req.common()
	if err := common.validate(); err != nil {
		return nil, err
	}
	if len(req.Images) == 0 {
		return nil, errors.New("at least one image is required")
	}
	if len(req.Images) > maxGalleryImages {
		return nil, fmt.Errorf("gallery images limit exceeded: %d (max %d)", len(req.Images), maxGalleryImages)
	}
	if len([]rune(req.Text)) > maxGalleryText {
		return nil, fmt.Errorf("gallery text limit exceeded: %d (max %d)", len([]rune(req.Text)), maxGalleryText)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writeCommonFields(writer, common); err != nil {
		return nil, err
	}
	if req.Text != "" {
		if err := writer.WriteField("text", req.Text); err != nil {
			return nil, err
		}
	}
	for i, img := range req.Images {
		if img.Reader == nil || img.Filename == "" {
			return nil, fmt.Errorf("image %d missing reader or filename", i)
		}
		headers := textproto.MIMEHeader{}
		headers.Set("Content-Disposition", fmt.Sprintf(`form-data; name="images"; filename="%s"`, sanitizeFilename(img.Filename)))
		part, err := writer.CreatePart(headers)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(part, img.Reader); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	parsed, err := s.doMultipart(ctx, "/bot/v1/messages/sendGallery/", writer.FormDataContentType(), buf.Bytes())
	if err != nil {
		return nil, err
	}

	return &ym.Message{ID: parsed.MessageID, Gallery: parsed.Images}, nil
}

func (r *SendGalleryRequest) common() attachmentCommon {
	return attachmentCommon{
		ChatID: r.ChatID, Login: r.Login, ThreadID: r.ThreadID,
		MessageID: r.MessageID, ReplyMessageID: r.ReplyMessageID, ReplyQuote: r.ReplyQuote,
		Forwards: r.Forwards, DisableNotification: r.DisableNotification, Important: r.Important,
		SuggestButtons: r.SuggestButtons, ActionButtons: r.ActionButtons,
	}
}

// Delete removes a message from a chat.
func (s *Service) Delete(ctx context.Context, req *DeleteMessageRequest) error {
	if err := ym.ValidateRecipient(req.ChatID, req.Login); err != nil {
		return err
	}
	if req.MessageID == 0 {
		return errors.New("message_id is required")
	}
	resp, err := s.client.DoRequest(ctx, http.MethodPost, "/bot/v1/messages/delete/", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var parsed struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("%w: decode delete response: %w", ymerrors.ErrInvalidResponse, err)
	}
	if !parsed.OK {
		return &ymerrors.APIError{
			Kind:        ymerrors.KindBadRequest,
			HTTPStatus:  resp.StatusCode,
			Description: parsed.Description,
			Method:      http.MethodPost,
			Endpoint:    "/bot/v1/messages/delete/",
		}
	}

	return nil
}

// GetFile downloads a file by its ID. The caller must close the returned ReadCloser.
func (s *Service) GetFile(ctx context.Context, fileID string) (io.ReadCloser, *FileMeta, error) {
	if fileID == "" {
		return nil, nil, errors.New("file_id is required")
	}
	path := "/bot/v1/messages/getFile/?file_id=" + url.QueryEscape(fileID)
	resp, err := s.client.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	meta := &FileMeta{
		FileID:        fileID,
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: resp.ContentLength,
	}

	if strings.HasPrefix(meta.ContentType, "application/json") {
		defer resp.Body.Close()
		var parsed struct {
			OK          bool   `json:"ok"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return nil, nil, fmt.Errorf("%w: decode getFile response: %w", ymerrors.ErrInvalidResponse, err)
		}
		if !parsed.OK {
			return nil, nil, &ymerrors.APIError{
				Kind:        ymerrors.KindBadRequest,
				HTTPStatus:  resp.StatusCode,
				Description: parsed.Description,
				Method:      http.MethodGet,
				Endpoint:    "/bot/v1/messages/getFile/",
			}
		}
	}

	return resp.Body, meta, nil
}

func buildSingleFilePayload(
	common attachmentCommon, field, filename, mimeType string, reader io.Reader,
) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writeCommonFields(writer, common); err != nil {
		return nil, "", err
	}
	headers := textproto.MIMEHeader{}
	headers.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, field, sanitizeFilename(filename)))
	if mimeType != "" {
		headers.Set("Content-Type", mimeType)
	}
	part, err := writer.CreatePart(headers)
	if err != nil {
		return nil, "", err
	}
	if _, err := io.Copy(part, reader); err != nil {
		return nil, "", err
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), writer.FormDataContentType(), nil
}

// multipartResponse is the flat body every multipart send method answers with.
// sendFile adds file_id, sendImage adds file_id/width/height, and sendGallery
// adds an images array.
type multipartResponse struct {
	OK        bool         `json:"ok"`
	MessageID ym.MessageID `json:"message_id"`
	FileID    string       `json:"file_id"`
	Width     int          `json:"width"`
	Height    int          `json:"height"`
	Images    []ym.Image   `json:"images"`
}

func (s *Service) doMultipart(ctx context.Context, path, contentType string, payload []byte) (*multipartResponse, error) {
	resp, err := s.client.DoMultipartRequest(ctx, http.MethodPost, path, contentType, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed multipartResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("%w: decode multipart response: %w", ymerrors.ErrInvalidResponse, err)
	}
	if !parsed.OK {
		return nil, fmt.Errorf("%w: ok=false", ymerrors.ErrInvalidResponse)
	}
	if parsed.MessageID == 0 {
		return nil, fmt.Errorf("%w: message_id missing", ymerrors.ErrInvalidResponse)
	}

	return &parsed, nil
}
