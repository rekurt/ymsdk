# Contributing to ymsdk

Thank you for your interest in contributing! This guide will help you get started.

## Prerequisites

- Go 1.25+
- [golangci-lint](https://golangci-lint.run/) v6+

## Getting started

```bash
git clone https://github.com/rekurt/ymsdk.git
cd ymsdk
go mod download
```

## Running tests

```bash
make test
# or directly:
go test ./...
```

## Running linter

```bash
make lint
# or directly:
golangci-lint run --config .golangci.yml
```

## Code style

- Follow existing patterns and conventions in the codebase.
- Add godoc comments on all exported types, functions, and constants.
- Use `ym.Ptr[T]` helper instead of writing custom pointer functions.
- Wrap errors using `fmt.Errorf("...: %w", err)` with sentinel errors from `ymerrors`.
- Keep line length under 180 characters (enforced by linter).
- Run `golangci-lint` before submitting — CI will reject PRs that don't pass.

## Package structure

| Package | Purpose |
|---------|---------|
| `client` | `YMClient` aggregator with all services |
| `client/ym` | Core `Client`, shared types, and configuration |
| `client/ym/ymerrors` | Error types (`APIError`, `ErrorKind`) and config structs |
| `client/ym/messages` | Send text, files, images, galleries; delete messages |
| `client/ym/chats` | Create chats/channels, manage members |
| `client/ym/users` | User deep-link retrieval |
| `client/ym/polls` | Poll creation and results |
| `client/ym/updates` | Polling for bot updates |
| `client/ym/self` | Bot self-management (webhook URL) |
| `client/ym/files` | Low-level file upload |
| `middleware` | zap-based logging utilities |
| `internal/testutil` | Test helpers (mock HTTP client) |

## Pull request process

1. Fork the repo and create a feature branch from `main`.
2. Make your changes with clear, focused commits.
3. Ensure `make test` and `make lint` pass locally.
4. Open a PR against `main` with a description of what changed and why.
5. Address any review feedback.

## Releasing

Releases are automated via GitHub Actions. When a version tag (`v*`) is pushed,
the CI verifies the build and creates a GitHub Release. Use `make release-patch`
or `make release-minor` to tag and push a new version.
