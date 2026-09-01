package ym

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

const defaultBaseURL = "https://botapi.messenger.yandex.net"

// Version is the SDK release this build corresponds to. It is reported to the
// API in the default User-Agent.
const Version = "0.2.0"

const defaultUserAgent = "ymsdk/" + Version + " (+https://github.com/rekurt/ymsdk)"

// HTTPDoer is an interface for executing HTTP requests, typically satisfied by *http.Client.
// Implementations must be safe for concurrent use if the Client is shared across goroutines.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// HttpDoer is the former name of [HTTPDoer].
//
// Deprecated: use [HTTPDoer]. This alias keeps existing code compiling and will
// be removed in a future major release.
type HttpDoer = HTTPDoer

// Config holds configuration for the Yandex Messenger API client.
type Config struct {
	// BaseURL overrides the API endpoint. Defaults to the production host.
	BaseURL string
	// Token is the bot's OAuth token.
	Token string
	// UserAgent overrides the User-Agent header. Defaults to "ymsdk/<version>".
	UserAgent string
	// UpdatesMode records whether the bot consumes updates by polling or webhook.
	UpdatesMode ymerrors.UpdatesMode
	// ErrorHandling configures retries and rate-limit back-off.
	ErrorHandling ymerrors.ErrorHandlingConfig
	// DisableAutoPayloadID turns off automatic generation of the payload_id
	// idempotency key.
	//
	// The API treats two requests carrying the same payload_id as duplicates.
	// Because a retried request replays the identical body, an automatically
	// generated key makes retries safe: a sendText that times out and is retried
	// delivers one message, not two. Leave the zero value in place unless you
	// supply payload_id yourself.
	DisableAutoPayloadID bool
}

// Client is the core HTTP client for the Yandex Messenger Bot API.
// It handles request execution, retries, and rate limit back-off.
type Client struct {
	http HTTPDoer
	cfg  Config
}

// NewClient creates a new Client with a default HTTP transport (15 s timeout).
func NewClient(cfg Config) *Client {
	httpClient := &http.Client{Timeout: 15 * time.Second}

	return NewClientWithHTTP(cfg, httpClient)
}

// NewClientWithHTTP creates a new Client with a caller-provided HTTP transport.
func NewClientWithHTTP(cfg Config, httpClient HTTPDoer) *Client {
	cfg = applyDefaults(cfg)

	return &Client{
		http: httpClient,
		cfg:  cfg,
	}
}

// DoRequest sends a JSON request to the Yandex Messenger API with automatic
// retry and rate-limit handling according to the client configuration.
// A nil body sends no payload and no Content-Type header.
// On success (2xx), the caller is responsible for closing the returned response body.
func (c *Client) DoRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	req := request{method: method, path: path}
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("yandex-messenger/client: marshal request body: %w", err)
		}
		req.body = payload
		req.contentType = "application/json"
	}

	return c.do(ctx, req)
}

// DoMultipartRequest sends an HTTP request with a pre-built body and content type,
// applying the same retry and rate-limit logic as DoRequest.
// On success (2xx), the caller is responsible for closing the returned response body.
func (c *Client) DoMultipartRequest(ctx context.Context, method, path, contentType string, body []byte) (*http.Response, error) {
	return c.do(ctx, request{
		method:      method,
		path:        path,
		body:        body,
		contentType: contentType,
	})
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
	case http.StatusNotFound:
		kind = ymerrors.KindNotFound
	case http.StatusConflict:
		kind = ymerrors.KindConflict
	case http.StatusRequestEntityTooLarge:
		kind = ymerrors.KindPayloadTooLarge
	default:
		if resp.StatusCode >= http.StatusInternalServerError {
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
func (c *Client) HTTPDoer() HTTPDoer {
	return c.http
}

// AutoPayloadID reports whether services should generate a payload_id
// idempotency key for requests that do not carry one.
func (c *Client) AutoPayloadID() bool {
	return !c.cfg.DisableAutoPayloadID
}

// NewAPIError wraps newAPIError for external users that need to parse raw responses.
func (c *Client) NewAPIError(method, path string, resp *http.Response) (*ymerrors.APIError, error) {
	return c.newAPIError(method, path, resp)
}

func applyDefaults(cfg Config) Config {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = defaultUserAgent
	}

	rs := cfg.ErrorHandling.RetryStrategy
	if rs.MaxAttempts < 1 {
		rs.MaxAttempts = 1
	}
	if rs.InitialBackoff <= 0 {
		rs.InitialBackoff = defaultInitialBackoff
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

// NextBackoff doubles the current backoff duration, capping at maximum.
// Used internally for retry logic; exported for use by multipart upload services.
func NextBackoff(current, maximum time.Duration) time.Duration {
	if current <= 0 {
		current = defaultInitialBackoff
	}
	next := current * 2
	if maximum > 0 && next > maximum {
		return maximum
	}

	return next
}

// ShouldRetryHTTP checks if the given HTTP status code is in the retryable list.
func ShouldRetryHTTP(status int, list []int) bool {
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
	if t, err := time.Parse(time.RFC1123, value); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
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
