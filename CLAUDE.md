# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a lightweight Go SDK (ymsdk) providing a type-safe client for the Yandex Messenger Bot API. The SDK features built-in retry logic with exponential backoff, comprehensive error handling with rate limit support, and service-based architecture for all major API operations.

**Repository**: https://github.com/rekurt/ymsdk
**Documentation**: https://pkg.go.dev/github.com/rekurt/ymsdk
**License**: MIT (2024 rekurt)

## Common Development Commands

```bash
# Run all tests
go test ./...

# Run linting (enforces 50+ linters with strict rules)
golangci-lint run --config .golangci.yml

# Run a single test file
go test ./client/ym/messages -run TestSomething

# Run integration example (requires YM_TOKEN environment variable)
cd examples/integration && YM_TOKEN=<token> ./run.sh

# Run webhook example
cd examples/webhook && YM_TOKEN=<token> YM_PORT=8080 go run .

# Run basic example
cd examples/basic_send && go run .
```

## Repository Structure

- **`/client/sdk.go`** - YMClient aggregator (convenient facade with all services initialized)
- **`/client/ym/client.go`** - Core HTTP client with retry logic and error handling
- **`/client/ym/<service>/`** - Service packages for each API domain:
  - `messages/` - Send/delete messages, galleries, file operations
  - `chats/` - Chat/channel creation and member management
  - `users/` - User information via login
  - `polls/` - Poll creation and management
  - `updates/` - Polling and webhook update handling
  - `self/` - Self endpoint (webhook configuration)
  - `files/` - File operations
  - `ymerrors/` - Error types and handling
- **`/internal/testutil/`** - Test utilities (FakeDoer mock, HTTP helpers)
- **`/middleware/`** - Zap-based structured error logging
- **`/examples/`** - 5 runnable examples (basic_send, poller, poll_bot, webhook, integration)

## Architecture & Design Patterns

### Service Architecture
Each API domain (messages, chats, polls, etc.) is a separate service package. Services accept a `*ym.Client` dependency and implement focused operations:

```go
type Service struct {
    client *ym.Client
}

func NewService(client *ym.Client) *Service { ... }
```

The `sdk.YMClient` aggregator provides convenient access to all services with single initialization.

### Error Handling
- **APIError**: Typed error with `ErrorKind`, `HTTPStatus`, `Description`, and `RetryAfter` duration
- **Sentinel Errors**: `ErrRateLimited`, `ErrInvalidToken`, `ErrUnauthorized`, `ErrNetwork`
- **Error Inspection**: Use `errors.As()` to extract `*ymerrors.APIError` details
- **Rate Limiting**: Check `errors.Is(err, ymerrors.ErrRateLimited)` and respect `RetryAfter`

### Retry Strategy
The HTTP client implements automatic retry with exponential backoff:
- **Config**: `RetryStrategy` with `MaxAttempts`, `InitialBackoff`, `MaxBackoff`, `RetryNetwork`, `RetryHTTP` flags
- **Rate Limit Handling**: `RateLimitHandling` with `UseRetryAfter` (respects API's `Retry-After` header)
- **Network Errors**: Optional retry for connection failures via `RetryNetwork` flag

### Type Safety
Distinct type aliases prevent ID mix-ups:
- `ChatID` for chat identifiers
- `UserLogin` for user handles
- `MessageID`, `ThreadID`, `PollID` for respective entities
- All JSON marshal/unmarshal supported for API serialization

### Configuration
All settings go through `ym.Config`:
```go
type Config struct {
    BaseURL       string // defaults to production endpoint
    Token         string // OAuth token
    ErrorHandling config.ErrorHandlingConfig // retry and rate limit settings
    UpdatesMode   config.UpdatesMode // "polling" or "webhook"
}
```

## Code Quality Standards

### Linting Rules
Enforced via `.golangci.yml` with 50+ linters. Key settings:
- **Line Length**: 180 characters max (lll linter)
- **Duplication**: 300 token threshold for code clones (dupl linter)
- **Complexity**: Disabled by default (not enforced); govet with fieldalignment disabled
- **Imports**: Ordered by: standard library → external → `github.com/rekurt/` packages (gci formatter)
- **Struct Tags**: Aligned and sorted (json → yaml → toml → etc.) with strict style

### Test Patterns
- **Table-Driven Tests**: Standard pattern used throughout
- **Mocking**: `FakeDoer` in `/internal/testutil/fake_doer.go` mocks HTTP responses
- **Test Organization**: Tests colocated in `*_test.go` files alongside implementation
- **Retry Tests**: Verify exponential backoff, retry-after respect, and network error handling

### CI/CD (`.github/workflows/ci.yml`)
Runs on push to `main`/`master` and PRs:
1. Lint check: `golangci-lint run` with `.golangci.yml` config
2. Test: `go test ./...`

Both must pass for merges.

## Dependencies

**Single external dependency**: `go.uber.org/zap v1.27.1` (structured logging)

## Key Files to Know

- **`client/ym/client.go`** - HTTP client with retry/backoff/error handling
- **`client/ym/ymerrors/errors.go`** - Error type definitions and constructors
- **`client/sdk.go`** - Service aggregator (entry point for users)
- **`middleware/logging.go`** - Error logging with Zap integration
- **`middleware/debug.go`** - Debug logger with log levels and HTTP inspection
- **`middleware/http_logger.go`** - HTTP wrapper to log raw request/response bodies
- **`examples/integration/main.go`** - Comprehensive example exercising all API methods
- **`examples/debug_logger/main.go`** - Example with full HTTP debugging and logging
- **`.golangci.yml`** - Linting configuration (read for code standards)

## Testing & Debugging

### Unit Testing
- Use `FakeDoer` from `internal/testutil` to mock HTTP responses
- Run single test: `go test ./client/ym/messages -run TestSendToChat`
- Run linting before commit: `golangci-lint run --config .golangci.yml`
- Check error types with `errors.As(&apiErr)` instead of type assertion

### Debug Logging

Enable detailed HTTP and update logging for development:

```go
// Set up Zap logger with DEBUG level
cfg := zap.NewDevelopmentConfig()
cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
logger, _ := cfg.Build()

// Create debug logger
debugLogger := middleware.NewDebugLogger(logger, middleware.LogLevelDebug)

// Wrap HTTP client to log request/response bodies
loggedClient := middleware.NewHTTPLogger(httpClient, debugLogger)

// Use with SDK
ymClient := ym.NewClientWithHTTP(cfg, loggedClient)
```

See `examples/debug_logger/main.go` and `middleware/README.md` for complete examples.

### Handling Updates Without Messages

The `Update` type has an optional `Message` field. Check before using:

```go
err := cs.Updates.PollLoop(ctx, params, func(ctx context.Context, update ym.Update) error {
    if update.Message == nil {
        // Not all updates include message data (e.g., edit/delete events)
        logger.Warn("Update without message", zap.Int64("update_id", update.UpdateID))
        return nil
    }

    // Process message
    return nil
})
```
