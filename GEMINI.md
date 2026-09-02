# GEMINI.md

Instructions for Gemini CLI working in this repository. Identical in substance
to `AGENTS.md`.

## What this is

`github.com/rekurt/ymsdk` — a Go client for the Yandex Messenger Bot API,
covering all 28 documented endpoints. One dependency (`go.uber.org/zap`), Go 1.25+.

## Commands

```bash
make check        # lint + race tests + build + vet — run before every commit
make test-race    # go test -race -count=1 ./...
make test-cover   # coverage report
make lint         # golangci-lint with .golangci.yml (50+ linters)
```

`make check` is what CI runs. Both lint and tests must pass to merge.

## API semantics

**Read [`skills/ymsdk/references/reference.md`](skills/ymsdk/references/reference.md)
before writing code that calls the API.** It is the single source of truth for
this SDK — every other agent-instruction file in this repository points at it
rather than restating it. [`recipes.md`](skills/ymsdk/references/recipes.md)
alongside it holds complete, compiling programs for the common bot shapes.

The seven things that plausible-looking code gets wrong:

1. **Retries are off by default.** `MaxAttempts` is 1 unless you set it.
2. **Do not disable `payload_id`.** It is what stops a retried send from
   delivering the message twice — on `sendText`, `sendSticker`,
   `sendSystemMessage` and `createPoll`. Uploads and the `share*` resends
   accept no such key, so retrying one of those can duplicate.
3. **Guard update fields.** Reaction, membership and button-press updates carry
   no text and often no `Chat` or `From`.
4. **`Update.Images` is `[][]ym.Image`** — outer per image, inner per size
   variant. Use `OriginalImages()`. An incoming file is `Update.Document`.
5. **Webhooks get 1 second.** Never work inline; use `updates.NewWebhookHandler`.
   Delivery is at-least-once, so deduplication is required.
6. **`getUpdates` erases everything below the offset, permanently.**
7. **Escape user text before echoing** with `ym.EscapeMarkdown` or `ym.Code`.

## Repository conventions

- **Endpoint paths** live in `client/ym/endpoints.go`. Never write a path
  literal — they drifted apart when services each kept their own.
- **Services** take a `*ym.Client` and expose focused methods. One package per
  API domain under `client/ym/`.
- **Errors**: return `*ymerrors.APIError` for API failures; match sentinels
  with `errors.Is`. New sentinels go in `client/ym/ymerrors/errors.go`.
- **Limits** from the API docs belong in `client/ym/limits.go` and are checked
  before the request goes out, not after it fails.
- **Tests** are table-driven, colocated, and use `internal/testutil.FakeDoer`.
  Payloads copied verbatim from the API docs are preferred over invented ones —
  that is how the `Update` schema mismatches were caught.
- **Imports** are grouped standard library → external → `github.com/rekurt/`.
- Line length 180; `dupl` triggers at 150 tokens, so factor shared logic out
  rather than copying it.

## Using the SDK from another project

Copy `skills/ymsdk/` into that project's `.claude/skills/` (or the equivalent
skills directory) so the assistant working there has the same reference.
