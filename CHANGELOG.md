# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Complete API coverage: all 28 documented Bot API endpoints. New in this
  release — `messages/pin`, `unpin`, `sendReaction`, `getReactions`,
  `sendSticker`, `sendSystemMessage`, `sendTyping`, `shareFile`, `shareImage`,
  `shareGallery`, `chats/get`, `chats/getChat`, `chats/getMembers`, `self/get`
- Message editing via `EditText` and `SendMessageOptions.MessageID`, plus the
  previously missing `reply_quote`, `forwards` and `action_buttons` parameters
- `ym.Target` with `ChatTarget`, `LoginTarget` and `UserIDTarget`, covering the
  `user_id` recipient form the SDK could not express
- Automatic `payload_id` idempotency keys, so a retried send cannot deliver the
  same message twice; opt out with `Config.DisableAutoPayloadID`
- `updates.Run`, which backs off and retries failed polls instead of ending the
  loop, with per-error policies and optional panic recovery
- `updates.NewWebhookHandler`: acknowledges within the API's 1s budget,
  processes on a worker pool, and drops the repeats that at-least-once delivery
  guarantees
- Types the update schema needs: `Reaction`, `ReactionEvent`, `ReactionsPage`,
  `ChatMembersUpdate`, `ChatInfo`, `ChatMetaData`, `ChatMember`, `Forward`,
  `ActionButtons`, `BotSettings`
- Text formatting helpers — `EscapeMarkdown`, `Bold`, `Italic`, `Strikethrough`,
  `Underline`, `Code`, `CodeBlock`, `Link`
- Local validation of the documented API limits, reported as `*ym.LimitError`
- Retry back-off jitter, a `User-Agent` identifying the SDK, and
  `ym.SleepContext`
- Endpoint path constants in `client/ym/endpoints.go`
- LLM skill for using the SDK: `skills/ymsdk/`, plus `AGENTS.md`, `GEMINI.md`,
  Cursor, Copilot and Windsurf adapters

### Fixed
- **Retries could duplicate messages.** The API documents `payload_id` as an
  idempotency key; the client never sent one, so a `sendText` retried after a
  timeout or a 500 delivered the message twice
- **A single incoming image broke the bot.** `Update.Images` was declared as a
  flat `[]Image` while the API sends `Image[][]` — one list of size variants per
  image. Decoding failed, and since one decode error rejects the whole response,
  every update in that batch was lost; `PollLoop` then stopped the bot
- **Incoming files were silently dropped.** The API field is `file`; the SDK
  read `document`, so `Update.Document` was always nil
- **A single API error killed the bot.** `PollLoop` returned on the first error
  of any kind
- **Cancellation was ignored during back-off.** `time.Sleep` kept a shutting-down
  bot blocked for the full delay, up to `MaxBackoff`
- **The webhook example could not have worked.** It required a header the API
  never sends, decoded a single update where the API sends a batch, and replied
  inline well past the 1s budget
- **Header injection through filenames.** `sanitizeFilename` left CR and LF
  intact, so a filename could inject MIME headers into a multipart part
- **`GetFile` could return a drained, closed body** that read as an empty file
- **A refused webhook delivery was acknowledged and lost.** The update was
  recorded in the dedup window before the enqueue succeeded, so the redelivery
  the 503 asked for was skipped as a duplicate and answered 200. Refused
  updates are now rolled back out of the window
- **`Shutdown` could panic a serving goroutine** with `send on closed channel`
  when it raced an in-flight delivery; new deliveries are now refused before
  the queue is closed
- **`ActionRetry` on a handler error never retried.** It behaved like
  `ActionContinue`, and the advancing offset then put the update permanently
  out of reach. It now re-invokes the handler on the same update, bounded by
  `RunOptions.MaxHandlerRetries`
- `self.Update` reported `KindBadRequest` regardless of the actual HTTP status
- `Update` was missing `reply_to_message`, `forwarded_messages`,
  `chat_members_update` and `reaction`

### Changed
- `DoRequest` and `DoMultipartRequest` now share one implementation; the `dupl`
  linter threshold is back to its default of 150
- Endpoint paths are constants — `sendFile` had been sent both with and without
  a trailing slash from two packages
- README and CLAUDE.md claimed full API coverage while half the endpoints were
  missing; the claim is now accurate and backed by a per-domain table

### Deprecated
- `PollLoop` — use `updates.Run`. The wrapper keeps the stop-on-any-error
  behaviour and now honours context cancellation
- `ValidateRecipient` — use `ValidateTarget`, which understands `user_id`
- `HttpDoer` — use `HTTPDoer`; the old name remains as a type alias
- `Update.Image` and `Update.Forward` — the API never populated either; read
  `Update.Images` / `OriginalImages()` and `Update.ForwardedMessages`

## [0.1.1] - 2026-04-08

### Added
- Package-level documentation (`doc.go`) for all service subpackages
- Example tests for pkg.go.dev documentation
- Codecov integration for test coverage reporting
- CONTRIBUTING.md, CODE_OF_CONDUCT.md, SECURITY.md
- GitHub Issue and PR templates
- Features section in README (RU + EN)
- CI/CD/coverage badges in README

### Fixed
- `Update.ToMessage` nil dereference on missing fields
- Webhook example request validation hardened

## [0.0.2] - 2025-12-08

### Added
- MIT License
- Go version upgrade to 1.25.1
- pkg.go.dev documentation links

## [0.0.1] - 2025-12-08

### Added
- Initial release of Yandex Messenger Bot API Go SDK
- Core HTTP client with retry logic and exponential backoff
- Service-oriented architecture: messages, chats, polls, updates, users, files, self
- Type-safe models: `ChatID`, `UserLogin`, `MessageID`, `ThreadID`, `PollID`
- Error handling with `APIError`, sentinel errors, rate limit support
- Polling and webhook update modes
- Debug logging middleware with zap integration
- 6 runnable examples: basic_send, poller, poll_bot, webhook, integration, debug_logger

[Unreleased]: https://github.com/rekurt/ymsdk/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/rekurt/ymsdk/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/rekurt/ymsdk/compare/v0.0.2...v0.1.0
[0.0.2]: https://github.com/rekurt/ymsdk/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/rekurt/ymsdk/releases/tag/v0.0.1
