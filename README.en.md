# Yandex Messenger Go SDK (ymsdk)

[Русская версия](README.md)

[![CI](https://github.com/rekurt/ymsdk/actions/workflows/ci.yml/badge.svg)](https://github.com/rekurt/ymsdk/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/rekurt/ymsdk.svg)](https://pkg.go.dev/github.com/rekurt/ymsdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/rekurt/ymsdk)](https://goreportcard.com/report/github.com/rekurt/ymsdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rekurt/ymsdk)](go.mod)
[![codecov](https://codecov.io/gh/rekurt/ymsdk/branch/master/graph/badge.svg)](https://codecov.io/gh/rekurt/ymsdk)

Lightweight Go client for Yandex Messenger Bot API with typed models, built-in retry, and services for all 28 API methods. Docs: https://pkg.go.dev/github.com/rekurt/ymsdk

## Features

- **Type-safe models** — `ChatID`, `UserLogin`, `MessageID` and other distinct types prevent mix-ups at compile time
- **Safe retries** — exponential backoff with jitter, plus an automatic `payload_id` idempotency key on `sendText`, `sendSticker`, `sendSystemMessage` and `createPoll`, so a retried send cannot deliver the message twice. Every other send — the uploads (`sendFile`, `sendImage`, `sendGallery`) and the by-file-id resends (`shareFile`, `shareImage`, `shareGallery`) — has no idempotency key in the API, so retrying one can duplicate it
- **Rate limit handling** — automatic respect for API `Retry-After` headers
- **Service-oriented architecture** — separate packages for messages, chats, polls, updates, and users
- **Polling & Webhooks** — two update delivery modes
- **Debug logging** — structured logs via `zap` with HTTP request/response inspection
- **Minimal dependencies** — only `go.uber.org/zap`
- **Full API coverage** — all 28 Yandex Messenger Bot API methods

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
    └── self/           # Bot webhook_url management
middleware/             # zap-based logging
├── logging.go          # LogError, LogUpdateWithRawData, WithRequestID
├── debug.go            # DebugLogger with levels (Silent → Debug)
└── http_logger.go      # HTTP wrapper for request/response logging
```

## Services

| Service | Description |
|---------|-------------|
| `cs.Messages` | Text and edits, files, images, galleries, stickers, system messages, reactions, pinning, typing indicator, forwarding, delete, download |
| `cs.Chats` | Create chats and channels, list chats, chat info, members, membership management |
| `cs.Users` | Get chat_link / call_link by login |
| `cs.Polls` | Create polls, results, paginated voters, `GetAllVoters` |
| `cs.Updates` | `getUpdates`, resilient `Run` loop, deduplicating webhook handler |
| `cs.Self` | Bot info, `webhook_url`, `get_reactions` / `get_members_changed` flags |

### API coverage

All **28** methods described in the [Bot API documentation](https://yandex.ru/dev/messenger/doc/ru/) are implemented.

| Domain | Methods |
|--------|---------|
| Messages | `sendText` (edits via `message_id`), `sendFile`, `sendImage`, `sendGallery`, `sendSticker`, `sendSystemMessage`, `sendTyping`, `shareFile`, `shareImage`, `shareGallery`, `delete`, `pin`, `unpin`, `sendReaction`, `getReactions`, `getFile`, `getUpdates` |
| Chats | `create`, `get`, `getChat`, `getMembers`, `updateMembers` |
| Polls | `createPoll`, `getResults`, `getVoters` |
| Bot | `self/get`, `self/update` |
| Users | `getUserLink` |

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

## Using with LLM assistants

The repository ships a skill so that Claude, Codex, Cursor, Copilot, Gemini and
Windsurf write correct code against this SDK. It covers the API, and more
usefully the places where obvious-looking code compiles and then fails in
production: retries are off by default, `payload_id` carries idempotency,
`Update.Images` has the nested `[][]ym.Image` shape, webhooks get a one-second
budget, and `getUpdates` permanently erases updates below the offset.

| File | Purpose |
|------|---------|
| [`skills/ymsdk/SKILL.md`](skills/ymsdk/SKILL.md) | Skill for Claude Code and claude.ai |
| [`skills/ymsdk/references/reference.md`](skills/ymsdk/references/reference.md) | Full reference — the single source of truth |
| [`skills/ymsdk/references/recipes.md`](skills/ymsdk/references/recipes.md) | Complete programs: echo bot, button bot, webhook service |
| [`AGENTS.md`](AGENTS.md) | Codex, Cursor, Jules and anything else reading `AGENTS.md` |
| [`GEMINI.md`](GEMINI.md) | Gemini CLI |
| [`.cursor/rules/ymsdk.mdc`](.cursor/rules/ymsdk.mdc) | Cursor rules |
| [`.github/copilot-instructions.md`](.github/copilot-instructions.md) | GitHub Copilot |
| [`.windsurfrules`](.windsurfrules) | Windsurf |

To use it in your own project, copy the skill directory:

```bash
mkdir -p .claude/skills
cp -r "$(go env GOMODCACHE)"/github.com/rekurt/ymsdk@*/skills/ymsdk .claude/skills/
```

Every program in `recipes.md` is compile-checked, and the per-platform files
point at one shared reference so they cannot drift apart from the code.

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
