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

	"github.com/rekurt/ymsdk/config"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/messages"
	"github.com/rekurt/ymsdk/client/ymerrors"
)

func main() {
	token := os.Getenv("YM_TOKEN")
	client := ym.NewClient(ym.Config{
		Token: token,
		ErrorHandling: config.ErrorHandlingConfig{
			RetryStrategy: config.RetryStrategy{MaxAttempts: 3, RetryNetwork: true},
			RateLimitHandling: config.RateLimitHandling{UseRetryAfter: true},
		},
	})

	msgSvc := messages.NewService(client)
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

## Services

- `messages.Service` — text, files, images/galleries, delete, getFile.
- `chats.Service` — create chats/channels, update members/subscribers/admins.
- `users.Service` — fetch chat_link/call_link for a login.
- `polls.Service` — create polls, get results, list voters.
- `updates.Service` — getUpdates and `PollLoop`.
- `self.Service` — `self.update` for webhook_url.
- `middleware` — zap-based error logging helpers.
- Convenience aggregator: `sdk.ClientSet` with prebuilt services (`sdk.New(cfg)`).

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
import "github.com/rekurt/ymsdk/client"

cs := sdk.New(ym.Config{Token: "..."})
msg, _ := cs.Messages.SendToChat(ctx, "chat-id", "hi", nil)
_ = cs.Polls.Create(ctx, &polls.CreatePollRequest{ChatID: ptr("chat-id"), Title: "Q?", Answers: []string{"A","B"}})
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

## Tests

```bash
go test ./...
```
