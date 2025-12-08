package messages

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

type SendFileRequest struct {
	ChatID   *ym.ChatID
	Login    *ym.UserLogin
	ThreadID *ym.ThreadID
	Document io.Reader
	Filename string
}

type FileMeta struct {
	FileID        string
	ContentType   string
	ContentLength int64
}

type SendImageRequest struct {
	ChatID   *ym.ChatID
	Login    *ym.UserLogin
	ThreadID *ym.ThreadID
	Image    io.Reader
	Filename string
}

type FilePart struct {
	Reader   io.Reader
	Filename string
}

type SendGalleryRequest struct {
	ChatID   *ym.ChatID
	Login    *ym.UserLogin
	ThreadID *ym.ThreadID
	Images   []FilePart
}

type DeleteMessageRequest struct {
	ChatID    *ym.ChatID    `json:"chat_id,omitempty"`
	Login     *ym.UserLogin `json:"login,omitempty"`
	MessageID ym.MessageID  `json:"message_id"`
	ThreadID  *ym.ThreadID  `json:"thread_id,omitempty"`
}

func (s *Service) SendFile(ctx context.Context, req *SendFileRequest) (*ym.Message, error) {
	if err := validateRecipient(req.ChatID, req.Login); err != nil {
		return nil, err
	}
	if req.Document == nil || req.Filename == "" {
		return nil, errors.New("document and filename are required")
	}
	payload, contentType, err := buildSingleFilePayload(
		req.ChatID, req.Login, req.ThreadID, "document", req.Filename, req.Document,
	)
	if err != nil {
		return nil, err
	}

	return s.doMultipart(ctx, "/bot/v1/messages/sendFile/", contentType, payload)
}

func (s *Service) SendImage(ctx context.Context, req *SendImageRequest) (*ym.Message, error) {
	if err := validateRecipient(req.ChatID, req.Login); err != nil {
		return nil, err
	}
	if req.Image == nil || req.Filename == "" {
		return nil, errors.New("image and filename are required")
	}
	payload, contentType, err := buildSingleFilePayload(
		req.ChatID, req.Login, req.ThreadID, "image", req.Filename, req.Image,
	)
	if err != nil {
		return nil, err
	}

	return s.doMultipart(ctx, "/bot/v1/messages/sendImage/", contentType, payload)
}

func (s *Service) SendGallery(ctx context.Context, req *SendGalleryRequest) (*ym.Message, error) {
	if err := validateRecipient(req.ChatID, req.Login); err != nil {
		return nil, err
	}
	if len(req.Images) == 0 {
		return nil, errors.New("at least one image is required")
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
	for i, img := range req.Images {
		if img.Reader == nil || img.Filename == "" {
			return nil, fmt.Errorf("image %d missing reader or filename", i)
		}
		headers := textproto.MIMEHeader{}
		headers.Set("Content-Disposition", fmt.Sprintf(`form-data; name="images"; filename="%s"`, img.Filename))
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

func (s *Service) Delete(ctx context.Context, req *DeleteMessageRequest) error {
	if err := validateRecipient(req.ChatID, req.Login); err != nil {
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

func validateRecipient(chatID *ym.ChatID, login *ym.UserLogin) error {
	if (chatID == nil || *chatID == "") && (login == nil || *login == "") {
		return errors.New("either chat_id or login is required")
	}
	if chatID != nil && *chatID != "" && login != nil && *login != "" {
		return errors.New("only one of chat_id or login must be set")
	}

	return nil
}

func buildSingleFilePayload(
	chatID *ym.ChatID, login *ym.UserLogin, threadID *ym.ThreadID, field, filename string, reader io.Reader,
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
	headers := textproto.MIMEHeader{}
	headers.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, field, filename))
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
	cfg := s.client.Config()
	retryCfg := cfg.ErrorHandling.RetryStrategy
	rateCfg := cfg.ErrorHandling.RateLimitHandling

	attempts := retryCfg.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}
	backoff := retryCfg.InitialBackoff
	if backoff <= 0 {
		backoff = 500 * time.Millisecond
	}

	baseUrl := strings.TrimRight(cfg.BaseURL, "/") + path

	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseUrl, bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("yandex-messenger/messages: build request: %w", err)
		}

		if cfg.Token != "" {
			req.Header.Set("Authorization", "OAuth "+cfg.Token)
		}
		req.Header.Set("Content-Type", contentType)

		resp, doErr := s.client.HTTPDoer().Do(req)
		if doErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, fmt.Errorf("yandex-messenger/messages: %w for %s", ctxErr, path)
			}
			var netErr net.Error
			if errors.As(doErr, &netErr) && retryCfg.RetryNetwork && attempt < attempts {
				time.Sleep(backoff)
				backoff = nextBackoffFiles(backoff, retryCfg.MaxBackoff)

				continue
			}

			return nil, fmt.Errorf("yandex-messenger/messages: %w for %s", doErr, path)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var parsed struct {
				OK        bool         `json:"ok"`
				Message   *ym.Message  `json:"message"`
				MessageID ym.MessageID `json:"message_id"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
				resp.Body.Close()

				return nil, fmt.Errorf("%w: decode multipart response: %w", ymerrors.ErrInvalidResponse, err)
			}

			resp.Body.Close()

			if parsed.Message != nil {
				return parsed.Message, nil
			}
			if parsed.MessageID != 0 {
				return &ym.Message{ID: parsed.MessageID}, nil
			}

			return nil, fmt.Errorf("%w: ok=%v message missing", ymerrors.ErrInvalidResponse, parsed.OK)
		}

		apiErr, parseErr := s.client.NewAPIError(http.MethodPost, path, resp)
		if parseErr != nil {
			return nil, parseErr
		}

		if apiErr.Kind == ymerrors.KindRateLimited && attempt < attempts {
			sleep := rateCfg.DefaultBackoff
			if rateCfg.UseRetryAfter && apiErr.RetryAfter > 0 {
				sleep = apiErr.RetryAfter
			}
			time.Sleep(sleep)

			continue
		}
		if shouldRetryHTTPFiles(apiErr.HTTPStatus, retryCfg.RetryHTTP) && attempt < attempts {
			time.Sleep(backoff)
			backoff = nextBackoffFiles(backoff, retryCfg.MaxBackoff)

			continue
		}

		return nil, apiErr
	}

	return nil, fmt.Errorf("yandex-messenger/messages: retries exhausted for %s", path)
}

func nextBackoffFiles(current, maximum time.Duration) time.Duration {
	if current <= 0 {
		current = 500 * time.Millisecond
	}
	next := current * 2
	if maximum > 0 && next > maximum {
		return maximum
	}

	return next
}

func shouldRetryHTTPFiles(status int, list []int) bool {
	for _, s := range list {
		if status == s {
			return true
		}
	}

	return false
}
