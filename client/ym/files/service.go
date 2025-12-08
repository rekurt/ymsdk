package files

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
	"strings"
	"time"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

type Service struct {
	client *ym.Client
}

func NewService(client *ym.Client) *Service {
	return &Service{client: client}
}

type SendFileOptions struct {
	Caption  string
	MimeType string
}

type sendFileResponse struct {
	OK      bool        `json:"ok"`
	Message *ym.Message `json:"message"`
}

func (s *Service) SendToChat(
	ctx context.Context, chatID, filename, contentType string, data []byte, opts *SendFileOptions,
) (*ym.Message, error) {
	fields := map[string]string{
		"chat_id": chatID,
	}

	return s.send(ctx, fields, filename, contentType, data, opts)
}

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

	resp, err := s.doMultipartWithRetry(ctx, http.MethodPost, "/bot/v1/messages/sendFile", boundaryContentType, body)
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
	headers.Set("Content-Disposition", fmt.Sprintf(`form-data; name="document"; filename="%s"`, filename))
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

func (s *Service) doMultipartWithRetry(
	ctx context.Context, method, path, contentType string, body []byte,
) (*http.Response, error) {
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

	url := strings.TrimRight(cfg.BaseURL, "/") + path

	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("yandex-messenger/files: build request: %w", err)
		}
		if token := cfg.Token; token != "" {
			req.Header.Set("Authorization", "OAuth "+token)
		}
		req.Header.Set("Content-Type", contentType)

		resp, doErr := s.client.HTTPDoer().Do(req)
		if doErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, fmt.Errorf("yandex-messenger/files: %w for %s %s", ctxErr, method, path)
			}
			var netErr net.Error
			if errors.As(doErr, &netErr) && retryCfg.RetryNetwork && attempt < attempts {
				time.Sleep(backoff)
				backoff = nextBackoffFiles(backoff, retryCfg.MaxBackoff)

				continue
			}

			return nil, fmt.Errorf("yandex-messenger/files: %w for %s %s", doErr, method, path)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		apiErr, parseErr := s.client.NewAPIError(method, path, resp)
		if parseErr != nil {
			return nil, parseErr
		}

		if apiErr.Kind == ymerrors.KindRateLimited && attempt < attempts {
			sleep := rateCfg.DefaultBackoff
			if rateCfg.UseRetryAfter && apiErr.RetryAfter > 0 {
				sleep = apiErr.RetryAfter
			}
			if sleep <= 0 {
				sleep = rateCfg.DefaultBackoff
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

	return nil, fmt.Errorf("yandex-messenger/files: retries exhausted for %s %s", method, path)
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
