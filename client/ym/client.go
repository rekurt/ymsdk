package ym

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

const defaultBaseURL = "https://botapi.messenger.yandex.net"

type HttpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Config struct {
	BaseURL       string
	Token         string
	UpdatesMode   ymerrors.UpdatesMode
	ErrorHandling ymerrors.ErrorHandlingConfig
}

type Client struct {
	http HttpDoer
	cfg  Config
}

func NewClient(cfg Config) *Client {
	httpClient := &http.Client{Timeout: 15 * time.Second}

	return NewClientWithHTTP(cfg, httpClient)
}

func NewClientWithHTTP(cfg Config, httpClient HttpDoer) *Client {
	cfg = applyDefaults(cfg)

	return &Client{
		http: httpClient,
		cfg:  cfg,
	}
}

func (c *Client) DoRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("yandex-messenger/client: marshal request body: %w", err)
		}
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + path
	retryCfg := c.cfg.ErrorHandling.RetryStrategy
	rateCfg := c.cfg.ErrorHandling.RateLimitHandling

	attempts := retryCfg.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}
	backoff := retryCfg.InitialBackoff
	if backoff <= 0 {
		backoff = 500 * time.Millisecond
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		var bodyReader io.Reader
		if payload != nil {
			bodyReader = bytes.NewReader(payload)
		}

		req, reqErr := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if reqErr != nil {
			return nil, fmt.Errorf("yandex-messenger/client: build request: %w", reqErr)
		}
		if c.cfg.Token != "" {
			req.Header.Set("Authorization", "OAuth "+c.cfg.Token)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, doErr := c.http.Do(req)
		if doErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, fmt.Errorf("yandex-messenger/client: %w for %s %s", ctxErr, method, path)
			}
			var netErr net.Error
			if errors.As(doErr, &netErr) && retryCfg.RetryNetwork && attempt < attempts {
				time.Sleep(backoff)
				backoff = nextBackoff(backoff, retryCfg.MaxBackoff)

				continue
			}

			return nil, fmt.Errorf("yandex-messenger/client: %w for %s %s", doErr, method, path)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		apiErr, parseErr := c.newAPIError(method, path, resp)
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

		if shouldRetryHTTP(apiErr.HTTPStatus, retryCfg.RetryHTTP) && attempt < attempts {
			time.Sleep(backoff)
			backoff = nextBackoff(backoff, retryCfg.MaxBackoff)

			continue
		}

		return nil, apiErr
	}

	return nil, fmt.Errorf("yandex-messenger/client: retries exhausted for %s %s", method, path)
}

func (c *Client) newAPIError(method, path string, resp *http.Response) (*ymerrors.APIError, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("yandex-messenger/client: read response body: %w", err)
	}

	var parsed struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
		Code        int    `json:"code"`
	}
	_ = json.Unmarshal(body, &parsed)

	kind := ymerrors.KindUnknown
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		kind = ymerrors.KindRateLimited
	case http.StatusUnauthorized:
		kind = ymerrors.KindUnauthorized
	case http.StatusForbidden:
		kind = ymerrors.KindInvalidToken
	case http.StatusBadRequest:
		kind = ymerrors.KindBadRequest
	default:
		if resp.StatusCode >= 500 {
			kind = ymerrors.KindNetwork
		}
	}

	retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
	description := strings.TrimSpace(parsed.Description)
	if description == "" {
		description = strings.TrimSpace(string(body))
	}
	if description == "" {
		description = http.StatusText(resp.StatusCode)
	}
	if len(description) > 512 {
		description = description[:512]
	}

	return &ymerrors.APIError{
		Kind:        kind,
		Code:        parsed.Code,
		HTTPStatus:  resp.StatusCode,
		Description: description,
		RequestID:   getRequestID(resp.Header),
		Method:      method,
		Endpoint:    path,
		RetryAfter:  retryAfter,
	}, nil
}

// Config returns a copy of client configuration.
func (c *Client) Config() Config {
	return c.cfg
}

// HTTPDoer exposes the underlying HTTP transport used by the client.
func (c *Client) HTTPDoer() HttpDoer {
	return c.http
}

// NewAPIError wraps newAPIError for external users that need to parse raw responses.
func (c *Client) NewAPIError(method, path string, resp *http.Response) (*ymerrors.APIError, error) {
	return c.newAPIError(method, path, resp)
}

func applyDefaults(cfg Config) Config {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}

	rs := cfg.ErrorHandling.RetryStrategy
	if rs.MaxAttempts < 1 {
		rs.MaxAttempts = 1
	}
	if rs.InitialBackoff <= 0 {
		rs.InitialBackoff = 500 * time.Millisecond
	}
	if rs.MaxBackoff <= 0 {
		rs.MaxBackoff = 10 * time.Second
	}
	if rs.RetryHTTP == nil {
		rs.RetryHTTP = []int{
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		}
	}
	cfg.ErrorHandling.RetryStrategy = rs

	rl := cfg.ErrorHandling.RateLimitHandling
	if rl.DefaultBackoff <= 0 {
		rl.DefaultBackoff = time.Second
	}
	cfg.ErrorHandling.RateLimitHandling = rl

	return cfg
}

func nextBackoff(current, maximum time.Duration) time.Duration {
	if current <= 0 {
		current = 500 * time.Millisecond
	}
	next := current * 2
	if maximum > 0 && next > maximum {
		return maximum
	}

	return next
}

func shouldRetryHTTP(status int, list []int) bool {
	for _, s := range list {
		if status == s {
			return true
		}
	}

	return false
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	if secs, err := strconv.Atoi(value); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}

	return 0
}

func getRequestID(h http.Header) string {
	if h == nil {
		return ""
	}
	if id := h.Get("X-Request-Id"); id != "" {
		return id
	}

	return h.Get("X-Request-ID")
}
