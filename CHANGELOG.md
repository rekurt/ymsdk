# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Package-level documentation (`doc.go`) for all service subpackages
- Example tests for pkg.go.dev documentation
- Codecov integration for test coverage reporting
- CONTRIBUTING.md, CODE_OF_CONDUCT.md, SECURITY.md
- GitHub Issue and PR templates
- Features section in README (RU + EN)
- CI/CD/coverage badges in README

### Deprecated
- `files.SendFileOptions.Caption` — the Bot API `sendFile` method defines no
  `caption` parameter, so the value was silently discarded by the server. The
  field is no longer put on the wire and will be removed in a future release.

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

[Unreleased]: https://github.com/rekurt/ymsdk/compare/v0.0.2...HEAD
[0.0.2]: https://github.com/rekurt/ymsdk/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/rekurt/ymsdk/releases/tag/v0.0.1
