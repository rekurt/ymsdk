package ymerrors

import "time"

// RetryStrategy configures automatic retry behavior for transient failures.
type RetryStrategy struct {
	MaxAttempts    int           `json:"max_attempts"    yaml:"max_attempts"`
	InitialBackoff time.Duration `json:"initial_backoff" yaml:"initial_backoff"`
	MaxBackoff     time.Duration `json:"max_backoff"     yaml:"max_backoff"`
	RetryHTTP      []int         `json:"retry_http"      yaml:"retry_http"`
	RetryNetwork   bool          `json:"retry_network"   yaml:"retry_network"`
}

// RateLimitHandling configures how the client reacts to HTTP 429 responses.
type RateLimitHandling struct {
	UseRetryAfter  bool          `json:"use_retry_after" yaml:"use_retry_after"`
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
