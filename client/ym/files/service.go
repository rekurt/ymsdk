package files

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

func sanitizeFilename(name string) string {
	return strings.NewReplacer(`"`, `\"`, `\`, `\\`).Replace(name)
}

// Service provides low-level file sending with raw byte payloads.
// For higher-level file operations with io.Reader, use the messages service.
type Service struct {
	client *ym.Client
}

// NewService creates a new files Service.
func NewService(client *ym.Client) *Service {
	return &Service{client: client}
}

// SendFileOptions holds optional parameters for file sending.
type SendFileOptions struct {
	Caption  string
	MimeType string
}

type sendFileResponse struct {
	OK      bool        `json:"ok"`
	Message *ym.Message `json:"message"`
}

// SendToChat sends a file to a chat by chat ID.
func (s *Service) SendToChat(
	ctx context.Context, chatID, filename, contentType string, data []byte, opts *SendFileOptions,
) (*ym.Message, error) {
	fields := map[string]string{
		"chat_id": chatID,
	}

	return s.send(ctx, fields, filename, contentType, data, opts)
}

// SendToLogin sends a file to a user by login.
func (s *Service) SendToLogin(
	ctx context.Context, login, filename, contentType string, data []byte, opts *SendFileOptions,
) (*ym.Message, error) {
	fields := map[string]string{
		"login": login,
	}

	return s.send(ctx, fields, filename, contentType, data, opts)
}

func (s *Service) send(
	ctx context.Context, fields map[string]string, filename, contentType string, data []byte, opts *SendFileOptions,
) (*ym.Message, error) {
	body, boundaryContentType, err := buildMultipartBody(fields, filename, contentType, data, opts)
	if err != nil {
		return nil, fmt.Errorf("yandex-messenger/files: build multipart: %w", err)
	}

	resp, err := s.client.DoMultipartRequest(ctx, http.MethodPost, "/bot/v1/messages/sendFile", boundaryContentType, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed sendFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("%w: decode sendFile response: %w", ymerrors.ErrInvalidResponse, err)
	}
	if !parsed.OK || parsed.Message == nil {
		return nil, fmt.Errorf(
			"%w: ok=%v message_present=%v", ymerrors.ErrInvalidResponse, parsed.OK, parsed.Message != nil,
		)
	}

	return parsed.Message, nil
}

func buildMultipartBody(
	fields map[string]string, filename, contentType string, data []byte, opts *SendFileOptions,
) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			return nil, "", err
		}
	}

	if opts != nil && opts.Caption != "" {
		if err := writer.WriteField("caption", opts.Caption); err != nil {
			return nil, "", err
		}
	}

	ct := contentType
	if opts != nil && opts.MimeType != "" {
		ct = opts.MimeType
	}

	headers := textproto.MIMEHeader{}
	headers.Set("Content-Disposition", fmt.Sprintf(`form-data; name="document"; filename="%s"`, sanitizeFilename(filename)))
	if ct != "" {
		headers.Set("Content-Type", ct)
	}
	part, err := writer.CreatePart(headers)
	if err != nil {
		return nil, "", err
	}
	if _, err := io.Copy(part, bytes.NewReader(data)); err != nil {
		return nil, "", err
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), writer.FormDataContentType(), nil
}
