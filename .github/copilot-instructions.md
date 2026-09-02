# GitHub Copilot instructions

This repository is `github.com/rekurt/ymsdk`, a Go client for the Yandex
Messenger Bot API covering all 28 documented endpoints.

Full reference: [`skills/ymsdk/references/reference.md`](../skills/ymsdk/references/reference.md).
Working programs: [`skills/ymsdk/references/recipes.md`](../skills/ymsdk/references/recipes.md).

## When suggesting code that calls this SDK

These are the mistakes that compile cleanly and break in production:

- `RetryStrategy.MaxAttempts` defaults to **1** — no retries. Set it to 3 in
  anything production-shaped.
- Leave `payload_id` generation on. It is the API's idempotency key and the
  only reason retrying a send is safe — on `sendText`, `sendSticker`,
  `sendSystemMessage` and `createPoll`. Uploads and the `share*` resends accept
  no such key, so retrying one of those can duplicate.
- Check `u.Chat != nil` and `u.From != nil` before dereferencing: reaction
  events, membership changes and button presses have neither.
- `Update.Images` is `[][]ym.Image` — one inner slice of size variants per
  image. Use `u.OriginalImages()`.
- An incoming file is `u.Document` (the API field is `file`).
- Webhook handlers must respond within 1 second, so do the work off the request
  path via `updates.NewWebhookHandler`; delivery is at-least-once, so keep
  deduplication on.
- `getUpdates` permanently erases updates below the offset.
- Prefer `updates.Run` over the deprecated `PollLoop`.
- Escape user text with `ym.EscapeMarkdown` before echoing it.

## When editing the SDK itself

- Run `make check` (lint + race tests + build + vet) before committing.
- Endpoint paths come from `client/ym/endpoints.go`; never write a literal.
- Tests are table-driven, use `internal/testutil.FakeDoer`, and prefer payloads
  copied verbatim from the API docs.
- Limits from the docs belong in `client/ym/limits.go`.
