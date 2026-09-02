package ymerrors

import "time"

// RetryStrategy configures automatic retry behavior for transient failures.
type RetryStrategy struct {
	// MaxAttempts is the total number of attempts including the initial request.
	// Default: 1 (no retries). Set to 3 for typical production use.
	MaxAttempts int `json:"max_attempts" yaml:"max_attempts"`
	// InitialBackoff is the delay before the first retry. Doubles on each subsequent attempt.
	// Default: 500ms.
	InitialBackoff time.Duration `json:"initial_backoff" yaml:"initial_backoff"`
	// MaxBackoff caps the exponential backoff growth.
	// Default: 10s.
	MaxBackoff time.Duration `json:"max_backoff" yaml:"max_backoff"`
	// RetryHTTP lists HTTP status codes that trigger a retry.
	// Default: [500, 502, 503, 504].
	RetryHTTP []int `json:"retry_http" yaml:"retry_http"`
	// RetryNetwork enables automatic retry on network-level errors (DNS, TCP).
	// Default: false.
	RetryNetwork bool `json:"retry_network" yaml:"retry_network"`
	// DisableJitter turns off backoff randomisation. Jitter is on by default so
	// that the zero value spreads retries; set this only when reproducible
	// timing matters, such as in tests.
	DisableJitter bool `json:"disable_jitter" yaml:"disable_jitter"`
}

// RateLimitHandling configures how the client reacts to HTTP 429 responses.
type RateLimitHandling struct {
	// UseRetryAfter respects the server's Retry-After header when present.
	UseRetryAfter bool `json:"use_retry_after" yaml:"use_retry_after"`
	// DefaultBackoff is the fallback delay when Retry-After is not provided.
	// Default: 1s.
	DefaultBackoff time.Duration `json:"default_backoff" yaml:"default_backoff"`
}

// ErrorHandlingConfig groups retry and rate-limit settings.
type ErrorHandlingConfig struct {
	RetryStrategy     RetryStrategy     `json:"retry_strategy"      yaml:"retry_strategy"`
	RateLimitHandling RateLimitHandling `json:"rate_limit_handling" yaml:"rate_limit_handling"`
	LoggingLevel      string            `json:"logging_level"       yaml:"logging_level"`
}

// UpdatesMode specifies how the bot receives updates from the API.
type UpdatesMode string

const (
	// UpdatesModePolling uses long-polling via getUpdates.
	UpdatesModePolling UpdatesMode = "polling"
	// UpdatesModeWebhook uses push-based webhook delivery.
	UpdatesModeWebhook UpdatesMode = "webhook"
)
