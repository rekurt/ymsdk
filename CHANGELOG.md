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
- `messages.SendFileRequest.MimeType` — overrides the `Content-Type` of the
  uploaded `document` part, replacing `files.SendFileOptions.MimeType`
- Full documented parameter coverage for `sendText`, `sendFile`, `sendImage` and
  `sendGallery`: `message_id` (edit), `reply_message_id`, `reply_quote`,
  `forwards`, `disable_notification`, `important` and `action_buttons`. Every
  send method now accepts the same shared set instead of a per-method subset.
- `ym.Forward`, `ym.ActionButtons`, `ym.ActionButton`, `ym.ActionButtonIcon` and
  the `ym.Icon*` / `ym.ActionButtonIconType` constants
- Send methods now surface the parts of the response they used to drop:
  `sendFile` returns `file_id` in `Message.Document`, `sendImage` returns
  `file_id` plus dimensions in `Message.Image`, and `sendGallery` returns the
  per-image results in `Message.Gallery`
- Client-side enforcement of documented constraints, so an invalid combination
  fails before a request is spent: `reply_quote` requires `reply_message_id`,
  `forwards` cannot be combined with `reply_message_id`, at most 6 action
  buttons, at most 100 suggest buttons, gallery text at most 6000 characters
- `ym.SuggestButtons` now emits the button array in the shape its layout
  requires: a flat array for `layout: "false"` and a nested one for
  `layout: "true"`. Rows carry no meaning in the flat layout, so they are
  concatenated in order rather than dropped. `UnmarshalJSON` accepts both
  shapes, so a value survives a round trip through its own output.
- `ym.SuggestLayoutFlat` and `ym.SuggestLayoutRows` constants — the Bot API
  spells the layout as the strings `"false"` and `"true"`, not as booleans

### Removed
- **BREAKING**: `client/ym/files` package and the `YMClient.Files` field. The
  service parsed a `{"ok":true,"message":{...}}` response that the Bot API never
  sends — `sendFile` answers with flat `{"ok":true,"message_id":N,"file_id":"..."}` —
  so `files.SendToChat` and `files.SendToLogin` could never succeed against the
  live API. Its tests passed only against a fabricated response shape. The
  package also carried an undocumented `caption` field that the server silently
  discarded; removing the package supersedes the deprecation from a8c89f8. Use
  `messages.SendFile`, which now also carries the `MimeType` override the files
  service provided.

### Fixed
- Dead fallback lookup in `getRequestID`. `http.Header.Get` canonicalises its
  argument, so `"X-Request-Id"` and `"X-Request-ID"` address the same entry and
  the second lookup could never find what the first one missed. Collapsed to one
  lookup; behaviour is unchanged. Also clears the `canonicalheader` lint finding.
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
