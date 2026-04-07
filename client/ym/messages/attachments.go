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
	"strings"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

const maxGalleryImages = 10

// sanitizeFilename escapes special characters in a filename for Content-Disposition headers.
func sanitizeFilename(name string) string {
	return strings.NewReplacer(`"`, `\"`, `\`, `\\`).Replace(name)
}

// SendFileRequest contains parameters for sending a file attachment.
// Exactly one of ChatID or Login must be set.
type SendFileRequest struct {
	ChatID         *ym.ChatID
	Login          *ym.UserLogin
	ThreadID       *ym.ThreadID
	Document       io.Reader
	Filename       string
	SuggestButtons *ym.SuggestButtons
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
	ChatID         *ym.ChatID
	Login          *ym.UserLogin
	ThreadID       *ym.ThreadID
	Image          io.Reader
	Filename       string
	SuggestButtons *ym.SuggestButtons
}

// FilePart represents a single file in a gallery upload.
type FilePart struct {
	Reader   io.Reader
	Filename string
}

// SendGalleryRequest contains parameters for sending a gallery of images.
// Exactly one of ChatID or Login must be set.
type SendGalleryRequest struct {
	ChatID         *ym.ChatID
	Login          *ym.UserLogin
	ThreadID       *ym.ThreadID
	Images         []FilePart
	Text           string
	SuggestButtons *ym.SuggestButtons
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
	if err := ym.ValidateRecipient(req.ChatID, req.Login); err != nil {
		return nil, err
	}
	if req.Document == nil || req.Filename == "" {
		return nil, errors.New("document and filename are required")
	}
	payload, contentType, err := buildSingleFilePayload(
		req.ChatID, req.Login, req.ThreadID, "document", req.Filename, req.Document, req.SuggestButtons,
	)
	if err != nil {
		return nil, err
	}

	return s.doMultipart(ctx, "/bot/v1/messages/sendFile/", contentType, payload)
}

// SendImage uploads and sends an image attachment via multipart/form-data.
func (s *Service) SendImage(ctx context.Context, req *SendImageRequest) (*ym.Message, error) {
	if err := ym.ValidateRecipient(req.ChatID, req.Login); err != nil {
		return nil, err
	}
	if req.Image == nil || req.Filename == "" {
		return nil, errors.New("image and filename are required")
	}
	payload, contentType, err := buildSingleFilePayload(
		req.ChatID, req.Login, req.ThreadID, "image", req.Filename, req.Image, req.SuggestButtons,
	)
	if err != nil {
		return nil, err
	}

	return s.doMultipart(ctx, "/bot/v1/messages/sendImage/", contentType, payload)
}

// SendGallery uploads and sends multiple images as a gallery.
func (s *Service) SendGallery(ctx context.Context, req *SendGalleryRequest) (*ym.Message, error) {
	if err := ym.ValidateRecipient(req.ChatID, req.Login); err != nil {
		return nil, err
	}
	if len(req.Images) == 0 {
		return nil, errors.New("at least one image is required")
	}
	if len(req.Images) > maxGalleryImages {
		return nil, fmt.Errorf("gallery images limit exceeded: %d (max %d)", len(req.Images), maxGalleryImages)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if req.ChatID != nil {
		if err := writer.WriteField("chat_id", string(*req.ChatID)); err != nil {
			return nil, err
		}
	}
	if req.Login != nil {
		if err := writer.WriteField("login", string(*req.Login)); err != nil {
			return nil, err
		}
	}
	if req.ThreadID != nil {
		if err := writer.WriteField("thread_id", fmt.Sprintf("%d", *req.ThreadID)); err != nil {
			return nil, err
		}
	}
	if req.Text != "" {
		if err := writer.WriteField("text", req.Text); err != nil {
			return nil, err
		}
	}
	if req.SuggestButtons != nil {
		sb, err := json.Marshal(req.SuggestButtons)
		if err != nil {
			return nil, fmt.Errorf("marshal suggest_buttons: %w", err)
		}
		if err := writer.WriteField("suggest_buttons", string(sb)); err != nil {
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

	return s.doMultipart(ctx, "/bot/v1/messages/sendGallery/", writer.FormDataContentType(), buf.Bytes())
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
	chatID *ym.ChatID, login *ym.UserLogin, threadID *ym.ThreadID,
	field, filename string, reader io.Reader, suggestButtons *ym.SuggestButtons,
) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if chatID != nil {
		if err := writer.WriteField("chat_id", string(*chatID)); err != nil {
			return nil, "", err
		}
	}
	if login != nil {
		if err := writer.WriteField("login", string(*login)); err != nil {
			return nil, "", err
		}
	}
	if threadID != nil {
		if err := writer.WriteField("thread_id", fmt.Sprintf("%d", *threadID)); err != nil {
			return nil, "", err
		}
	}
	if suggestButtons != nil {
		sb, err := json.Marshal(suggestButtons)
		if err != nil {
			return nil, "", fmt.Errorf("marshal suggest_buttons: %w", err)
		}
		if err := writer.WriteField("suggest_buttons", string(sb)); err != nil {
			return nil, "", err
		}
	}
	headers := textproto.MIMEHeader{}
	headers.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, field, sanitizeFilename(filename)))
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

func (s *Service) doMultipart(ctx context.Context, path, contentType string, payload []byte) (*ym.Message, error) {
	resp, err := s.client.DoMultipartRequest(ctx, http.MethodPost, path, contentType, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed struct {
		OK        bool         `json:"ok"`
		MessageID ym.MessageID `json:"message_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("%w: decode multipart response: %w", ymerrors.ErrInvalidResponse, err)
	}
	if !parsed.OK {
		return nil, fmt.Errorf("%w: ok=false", ymerrors.ErrInvalidResponse)
	}
	if parsed.MessageID != 0 {
		return &ym.Message{ID: parsed.MessageID}, nil
	}

	return nil, fmt.Errorf("%w: message_id missing", ymerrors.ErrInvalidResponse)
}
