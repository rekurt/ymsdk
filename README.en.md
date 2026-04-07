# Yandex Messenger Go SDK (ymsdk)

[Русская версия](README.md)

Lightweight Go client for Yandex Messenger Bot API with typed models, built-in retry, and services for core API methods. Docs: https://pkg.go.dev/github.com/rekurt/ymsdk

## Installation

```bash
go get github.com/rekurt/ymsdk
```

## Quick start

### Via aggregator (recommended)

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

func main() {
	cs := client.New(ym.Config{
		Token: os.Getenv("YM_TOKEN"),
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy:     ymerrors.RetryStrategy{MaxAttempts: 3, RetryNetwork: true},
			RateLimitHandling: ymerrors.RateLimitHandling{UseRetryAfter: true},
		},
	})

	msg, err := cs.Messages.SendToChat(context.Background(), "chat-id", "hello", nil)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("sent message:", msg.ID)
}
```

### Via individual services

```go
cl := ym.NewClient(ym.Config{Token: os.Getenv("YM_TOKEN")})
msgSvc := messages.NewService(cl)
pollSvc := polls.NewService(cl)

msg, _ := msgSvc.SendToChat(ctx, "chat-id", "hello", nil)
```

## Architecture

```
client/
├── sdk.go              # YMClient — aggregator with all services
└── ym/                 # Core SDK
    ├── client.go       # HTTP client with retry/rate-limit logic
    ├── types.go        # Shared types (Chat, Message, Update, …)
    ├── ptr.go          # ym.Ptr[T] helper for optional fields
    ├── validate.go     # Shared recipient validation
    ├── ymerrors/       # Error types and configuration
    ├── messages/       # Text, files, images, galleries, delete, getFile
    ├── chats/          # Create chats/channels, manage members
    ├── users/          # User chat/call deep links
    ├── polls/          # Polls: create, results, voters, GetAllVoters
    ├── updates/        # getUpdates, GetUpdates, and PollLoop
    ├── self/           # Bot webhook_url management
    └── files/          # Low-level file sending (byte[])
middleware/             # zap-based logging
├── logging.go          # LogError, LogUpdateWithRawData, WithRequestID
├── debug.go            # DebugLogger with levels (Silent → Debug)
└── http_logger.go      # HTTP wrapper for request/response logging
```

## Services

| Service | Description |
|---------|-------------|
| `cs.Messages` | Text messages, files, images, galleries, delete, file download |
| `cs.Chats` | Create chats/channels, add/remove members, subscribers, admins |
| `cs.Users` | Get chat_link / call_link by login |
| `cs.Polls` | Create polls, results, paginated voters, GetAllVoters |
| `cs.Updates` | getUpdates (raw + typed), PollLoop for continuous polling |
| `cs.Self` | self.update for webhook_url configuration |
| `cs.Files` | Low-level file sending via byte[] |

Convenience aggregator `client.YMClient` with prebuilt services:
- `client.New(cfg)` — create with new HTTP client
- `client.Wrap(cl)` — wrap existing `ym.Client`

## Error handling

```go
var apiErr *ymerrors.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("kind=%d http=%d desc=%s request_id=%s\n",
        apiErr.Kind, apiErr.HTTPStatus, apiErr.Description, apiErr.RequestID)

    if errors.Is(err, ymerrors.ErrRateLimited) && apiErr.RetryAfter > 0 {
        time.Sleep(apiErr.RetryAfter)
    }
}
```

- API failures: `*ymerrors.APIError` (use `errors.As`).
- Rate limit: `errors.Is(err, ymerrors.ErrRateLimited)` + `RetryAfter`.
- Auth: `ErrInvalidToken` (403) / `ErrUnauthorized` (401).
- Transport: `KindNetwork` (5xx) / `net.Error` when `RetryNetwork` enabled.

## Configuration

```go
cfg := ym.Config{
    BaseURL: "",  // defaults to production endpoint
    Token:   os.Getenv("YM_TOKEN"),
    ErrorHandling: ymerrors.ErrorHandlingConfig{
        RetryStrategy: ymerrors.RetryStrategy{
            MaxAttempts:    3,
            InitialBackoff: 500 * time.Millisecond,
            MaxBackoff:     10 * time.Second,
            RetryNetwork:   true,
            RetryHTTP:      []int{500, 502, 503, 504},
        },
        RateLimitHandling: ymerrors.RateLimitHandling{
            UseRetryAfter:  true,
            DefaultBackoff: time.Second,
        },
    },
    UpdatesMode: ymerrors.UpdatesModePolling, // "polling" or "webhook"
}
```

## Debug logging

Inspect raw HTTP request/response bodies with middleware:

```go
logger, _ := zap.NewDevelopmentConfig().Build()
debugLogger := middleware.NewDebugLogger(logger, middleware.LogLevelDebug)
loggedHTTP := middleware.NewHTTPLogger(&http.Client{Timeout: 15 * time.Second}, debugLogger)

ymClient := ym.NewClientWithHTTP(cfg, loggedHTTP)
cs := client.Wrap(ymClient)
```

See `middleware/README.md` and `examples/debug_logger` for details.

## Examples

| Example | Description |
|---------|-------------|
| `examples/basic_send` | Send text to chat/login, reply-to, mark-important, error handling |
| `examples/poller` | Continuous polling via PollLoop, handles text/files/stickers/forwards |
| `examples/poll_bot` | Create poll, GetResults, GetAllVoters, read updates |
| `examples/webhook` | HTTP webhook receiver with secret validation, graceful shutdown, echo bot |
| `examples/debug_logger` | HTTP request/response logging, handling updates without messages |
| `examples/integration` | End-to-end script exercising all SDK methods (configure via env) |

### Running examples

```bash
# Send a message
cd examples/basic_send
YM_TOKEN=... go run . -chat "chat-id" -text "hello"

# Poll for updates
cd examples/poller
YM_TOKEN=... go run .

# Poll bot
cd examples/poll_bot
YM_TOKEN=... YM_CHAT_ID=... go run .

# Webhook server
cd examples/webhook
YM_TOKEN=... YM_WEBHOOK_SECRET=... YM_PORT=8080 go run .

# Debug logging
cd examples/debug_logger
YM_TOKEN=... go run .

# Full integration
cd examples/integration
YM_TOKEN=... YM_CHAT_ID=... YM_LOGIN=... go run .
```

## Versioning

This project follows [Semantic Versioning](https://semver.org/). To install a specific version:

```bash
go get github.com/rekurt/ymsdk@v0.1.0
```

## Tests

```bash
# Run all tests
go test ./...

# Lint (50+ linters)
golangci-lint run --config .golangci.yml
```
