package ymerrors

import "time"

type RetryStrategy struct {
	MaxAttempts    int           `json:"max_attempts"    yaml:"max_attempts"`
	InitialBackoff time.Duration `json:"initial_backoff" yaml:"initial_backoff"`
	MaxBackoff     time.Duration `json:"max_backoff"     yaml:"max_backoff"`
	RetryHTTP      []int         `json:"retry_http"      yaml:"retry_http"`
	RetryNetwork   bool          `json:"retry_network"   yaml:"retry_network"`
}

type RateLimitHandling struct {
	UseRetryAfter  bool          `json:"use_retry_after" yaml:"use_retry_after"`
	DefaultBackoff time.Duration `json:"default_backoff" yaml:"default_backoff"`
}

type ErrorHandlingConfig struct {
	RetryStrategy     RetryStrategy     `json:"retry_strategy"      yaml:"retry_strategy"`
	RateLimitHandling RateLimitHandling `json:"rate_limit_handling" yaml:"rate_limit_handling"`
	LoggingLevel      string            `json:"logging_level"       yaml:"logging_level"`
}

type UpdatesMode string

const (
	UpdatesModePolling UpdatesMode = "polling"
	UpdatesModeWebhook UpdatesMode = "webhook"
)
