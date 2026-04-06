# Yandex Messenger Go SDK (ymsdk)

[Русская версия](README.md)

Lightweight Go client for Yandex Messenger Bot API with typed models, built-in retry, and services for core API methods. Docs: https://pkg.go.dev/github.com/rekurt/ymsdk

## Installation

```bash
go get github.com/rekurt/ymsdk
```

## Quick start

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/messages"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

func main() {
	token := os.Getenv("YM_TOKEN")
	cl := ym.NewClient(ym.Config{
		Token: token,
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy:     ymerrors.RetryStrategy{MaxAttempts: 3, RetryNetwork: true},
			RateLimitHandling: ymerrors.RateLimitHandling{UseRetryAfter: true},
		},
	})

	msgSvc := messages.NewService(cl)
	msg, err := msgSvc.SendToChat(context.Background(), "chat-id", "hello", nil)
	if err != nil {
		handleErr(err)
		return
	}
	fmt.Println("sent message:", msg.ID)
}

func handleErr(err error) {
	var apiErr *ymerrors.APIError
	if errors.As(err, &apiErr) {
		fmt.Printf("API error kind=%d http=%d desc=%s\n", apiErr.Kind, apiErr.HTTPStatus, apiErr.Description)
		if errors.Is(err, ymerrors.ErrRateLimited) && apiErr.RetryAfter > 0 {
			fmt.Printf("retry after: %s\n", apiErr.RetryAfter)
		}
		return
	}
	fmt.Println("unexpected error:", err)
}
```

See `examples/basic_send`, `examples/poller`, `examples/poll_bot`, `examples/integration`.

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
    ├── messages/       # Text, files, images, galleries, delete
    ├── chats/          # Create chats, manage members
    ├── users/          # User chat/call deep links
    ├── polls/          # Polls, results, voters
    ├── updates/        # getUpdates and PollLoop
    ├── self/           # Bot webhook_url management
    └── files/          # Low-level file sending
middleware/             # zap-based logging
```

## Services

- `messages.Service` — text, files, images/galleries, delete, getFile.
- `chats.Service` — create chats/channels, update members/subscribers/admins.
- `users.Service` — fetch chat_link/call_link for a login.
- `polls.Service` — create polls, get results, list voters.
- `updates.Service` — getUpdates and `PollLoop`.
- `self.Service` — `self.update` for webhook_url.
- `middleware` — zap-based error logging helpers.
- Convenience aggregator: `client.YMClient` with prebuilt services (`client.New(cfg)`).

## Error handling

- API failures: `*ymerrors.APIError` (use `errors.As`).
- Rate limit: `errors.Is(err, ymerrors.ErrRateLimited)` + `RetryAfter`.
- Auth: `ErrInvalidToken` / `ErrUnauthorized`.
- Transport: `KindNetwork` / `net.Error` when `RetryNetwork` enabled.

## Configuration

`ym.Config`:

- `BaseURL` — API endpoint (defaults to production).
- `Token` — OAuth token.
- `ErrorHandling`:
  - `RetryStrategy`: `MaxAttempts`, `InitialBackoff`, `MaxBackoff`, `RetryHTTP`, `RetryNetwork`.
  - `RateLimitHandling`: `UseRetryAfter`, `DefaultBackoff`.
- `UpdatesMode`: `polling` / `webhook` (explicit mode flag).

## Examples

- `examples/basic_send` — send text to chat/login with error handling.
- `examples/poller` — polling loop respecting rate limits.
- `examples/poll_bot` — create a poll and process updates.
- `examples/integration` — end-to-end script hitting all SDK methods (configure via env vars).
- `examples/webhook` — minimal HTTP webhook receiver (webhook mode).

### Quick via aggregator

```go
import (
	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/polls"
)

cs := client.New(ym.Config{Token: "..."})
msg, _ := cs.Messages.SendToChat(ctx, "chat-id", "hi", nil)
_ = cs.Polls.Create(ctx, &polls.CreatePollRequest{
	ChatID:  ym.Ptr(ym.ChatID("chat-id")),
	Title:   "Q?",
	Answers: []string{"A", "B"},
})
```

Run integration example:

```bash
cd examples/integration
YM_TOKEN=... YM_CHAT_ID=... YM_LOGIN=... YM_FILE_PATH=... go run .
# or: YM_TOKEN=... ./run.sh
```

Run webhook example:
```bash
cd examples/webhook
YM_TOKEN=... YM_PORT=8080 go run .
```

## Versioning

This project follows [Semantic Versioning](https://semver.org/). To install a specific version:

```bash
go get github.com/rekurt/ymsdk@v0.1.0
```

## Tests

```bash
go test ./...
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
