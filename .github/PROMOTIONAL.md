# Promotional Materials for ymsdk

Ready-to-post content for promoting ymsdk across platforms.

---

## Reddit r/golang Post

**Title:** ymsdk — Lightweight Go SDK for Yandex Messenger Bot API

**Body:**

Hi everyone! I'd like to share **ymsdk** — a Go SDK for the Yandex Messenger Bot API.

### Why another messenger SDK?

Yandex Messenger is widely used across CIS companies, but had no proper Go SDK with modern patterns. ymsdk fills this gap with:

- **Type-safe models** — `ChatID`, `UserLogin`, `MessageID` prevent mix-ups at compile time
- **Automatic retry** — exponential backoff with configurable strategy
- **Rate limit handling** — respects `Retry-After` headers automatically
- **Service-oriented architecture** — separate packages for messages, chats, polls, updates, files, users
- **Minimal deps** — only `go.uber.org/zap` for structured logging
- **Debug middleware** — full HTTP request/response inspection

### Quick example

```go
cs := client.New(ym.Config{Token: os.Getenv("YM_TOKEN")})
msg, err := cs.Messages.SendToChat(ctx, chatID, "Hello!", nil)
```

Links:
- GitHub: https://github.com/rekurt/ymsdk
- pkg.go.dev: https://pkg.go.dev/github.com/rekurt/ymsdk
- Go Report Card: https://goreportcard.com/report/github.com/rekurt/ymsdk

Feedback and contributions welcome!

---

## Dev.to / Habr Article Draft

**Title:** Building a Type-Safe Go SDK for Yandex Messenger Bot API

**Tags:** go, golang, sdk, yandex, bot, messenger

**Outline:**

1. **Introduction** — Why Yandex Messenger needs a Go SDK
2. **Architecture** — Service-oriented design, dependency injection via `*ym.Client`
3. **Type Safety** — How distinct type aliases (`ChatID`, `UserLogin`) prevent bugs
4. **Retry & Rate Limiting** — Exponential backoff implementation, `Retry-After` handling
5. **Error Handling** — `APIError` with `ErrorKind`, sentinel errors, `errors.Is/As` patterns
6. **Debug Logging** — Middleware architecture with zap, HTTP body inspection
7. **Getting Started** — Code examples with aggregator and individual services
8. **Testing** — `FakeDoer` mock pattern, table-driven tests
9. **Conclusion** — Links, call for contributions

---

## Twitter/X Post

ymsdk — lightweight Go SDK for Yandex Messenger Bot API 🤖

✅ Type-safe models
✅ Auto retry with exponential backoff
✅ Rate limit handling
✅ Service-oriented architecture
✅ MIT licensed

github.com/rekurt/ymsdk

#golang #go #sdk #yandex #bot #opensource

---

## Telegram Post (for Go communities)

🚀 **ymsdk** — Go SDK для Yandex Messenger Bot API

Типобезопасные модели, автоматический retry с exponential backoff, обработка rate limit, сервис-ориентированная архитектура.

```go
cs := client.New(ym.Config{Token: os.Getenv("YM_TOKEN")})
msg, _ := cs.Messages.SendToChat(ctx, chatID, "Hello!", nil)
```

🔗 GitHub: github.com/rekurt/ymsdk
📖 Docs: pkg.go.dev/github.com/rekurt/ymsdk

MIT • Minimal deps • 50+ linters CI

---

## LinkedIn Post

I'm excited to share **ymsdk** — an open-source Go SDK for the Yandex Messenger Bot API.

As Yandex Messenger grows in adoption across enterprises, developers need reliable tools to build integrations. ymsdk provides:

• Type-safe API with compile-time safety
• Automatic retry with exponential backoff
• Rate limit handling respecting Retry-After headers
• Service-oriented architecture for clean separation
• Comprehensive error handling with structured error types
• Debug middleware for HTTP inspection

The SDK follows Go community best practices: minimal dependencies, table-driven tests, 50+ linter rules, and full pkg.go.dev documentation.

Check it out: https://github.com/rekurt/ymsdk

#golang #opensource #sdk #yandex #development
