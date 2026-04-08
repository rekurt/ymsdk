// Package ymerrors defines error types and sentinel errors for the Yandex
// Messenger Bot API SDK.
//
// The primary type is [APIError], which carries structured information about
// API failures including HTTP status, error kind, description, and an
// optional [APIError.RetryAfter] duration for rate-limited responses.
//
// Sentinel errors allow quick classification with [errors.Is]:
//
//	if errors.Is(err, ymerrors.ErrRateLimited) {
//	    // back off and retry
//	}
//
// Available sentinels: [ErrRateLimited], [ErrInvalidToken],
// [ErrUnauthorized], [ErrNetwork].
//
// The package also provides [RetryStrategy] and [RateLimitHandling]
// configuration types used by the HTTP client's automatic retry logic.
package ymerrors
