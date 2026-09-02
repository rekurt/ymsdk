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
- **`GetVotersPage` could not decode a real response.** The documented payload
  sends `"cursor": {"next": N}`, but `PollVotersPage.Cursor` was declared as an
  int64, so decoding failed and both it and `GetAllVoters` were unusable against
  the live API. The existing test passed only because its fixture invented a
  flat cursor. `Cursor` is now a `ym.PollCursor` with a `Next` field
- **`GetReactions` accepted a response with no `reactions_type`.** Callers must
  branch on it to know which field carries the answer, so an absent
  discriminator is malformed rather than empty. An unfamiliar value is still
  passed through, since the API may add a third shape
- **`GetFile` still built its path from a literal.** The behaviour matched, so
  no test could see it — url.Parse strips the query, leaving the same Path — but
  the endpoint had two definitions free to drift apart. A source-level test now
  fails on any endpoint literal outside endpoints.go
- **`SendSystemMessage` sent an empty text.** The API marks it required and,
  unlike an ordinary send, the endpoint carries no attachment that could give an
  empty body meaning
- **`Pin` reported success without a message id**, which the separate decoder
  there had escaped the guard added to the shared one
- **A send response without `message_id` was accepted.** The endpoints that
  return only that field handed back a Message whose ID was zero and a nil
  error, so a caller could store an unusable id or read a malformed response as
  a success
- **A webhook batch entry without `update_id` was admitted.** Deduplication
  keys on that id alone, so several such entries collapsed into one delivery
  and the rest disappeared behind a 200; the bare-update path already discarded
  the same value
- **`{"ok":true,"data":null}` slipped past the missing-payload guard.** A JSON
  null is four bytes, so the length check passed and Unmarshal left the chat
  zero-valued
- **The examples and recipes only listened for SIGINT.** Container supervisors
  send SIGTERM, so routine shutdown killed the process outright and updates
  already acknowledged with 200 were lost before the drain could run
- **A successful poll reset the back-off to a hardcoded second**, so a caller
  who set MaxBackoff below that kept it only until the first success and every
  later retry ignored the setting. Both reset sites now go through the same
  helper as the initial value, which is the only place the raw constant is
  reachable from
- **A zero message id passed through the send options was serialised.**
  `EditText` guarded its scalar argument, but `SendMessageOptions.MessageID`
  and `ReplyToMessageID` set to a pointer to zero went out as `"message_id":0`,
  which zero means "no message" everywhere else in the package
- **Action buttons were sent without a title or icon**, both of which the API
  marks required, so the request could only be refused
- **`GetChat` returned an empty result for a response with no `data`.** For a
  single-object query that is a malformed response, not an empty one, and
  returning a zero-valued ChatInfo with a nil error hid schema drift. List
  endpoints keep treating an absent payload as empty
- **`ListAll` and `GetAllMembers` returned a repeated page twice.** The page was
  appended before the cursor was compared, so the exhaustion case the loop
  exists to handle duplicated its own last page. The cursor is checked first now
- **The webhook example echoed an unescaped filename.** Text was escaped but the
  document branch was not, and a filename is chosen by whoever uploaded it, so
  crafted markup rendered as bot-authored content. Every string taken from an
  update is escaped now
- **A slow `OnError` could spend the webhook's reply budget.** Request-path
  errors were reported before the status was written, so a callback doing
  blocking I/O — a remote log write, say — delayed the response past the API's
  one-second limit. The API then saw a timeout instead of the final 4xx or the
  retryable 503 and redelivered something already settled. The response is
  written and flushed first now
- **A concurrent redelivery could be acknowledged with nothing queued.**
  The webhook recorded an update id, then tried to enqueue, then rolled the
  record back on failure. In the gap a second delivery of the same update saw
  the record, was answered 200 — final for the API — and the first copy then
  failed and undid the record, so nothing processed the update. Under test 31 of
  64 concurrent callers hit that window. Admission is now one critical section:
  the id is recorded only once a copy is queued
- **The webhook example shared one deadline between the HTTP shutdown and the
  drain**, so a slow request could consume it and leave the drain no time,
  dropping already-acknowledged updates. Each gets its own now, in the recipe too
- **The legacy `updates.Service.Get` sent an unvalidated limit.** It is a
  separate entry point from `GetUpdates` and `Run` in the same file, so it
  bypassed the validation added to them
- **A poll's answer count reported a plain error** that also folded in the
  title check, so callers could neither match it with `errors.As` nor tell
  which of the two was wrong
- **`Link` could be closed early by a crafted label.** Escaping the delimiters
  without escaping the backslash first turned a label ending in one into a
  doubled backslash, which reads as an escaped backslash and leaves the bracket
  after it live: `Link("safe\\](https://evil.example)", "https://ok.example")`
  rendered a link to the attacker's URL rather than the caller's
- **`SendReaction` sent reactions with empty identifiers.** The API requires
  both `type` and `name`, so `ym.DefaultReaction("")` produced a request that
  could only be rejected; nil still means removal
- **A poll created with `PayloadID: ym.Ptr("")` sent an empty idempotency key.**
  `omitempty` looks at the pointer rather than the value, so the field went out
  empty and the retry protection was silently absent. An empty string now
  counts as unset, as it already did on the send paths
- **`GetVotersPage` sent an unvalidated page limit**, the only paginated method
  that still did
- **Voters for a poll's first answer were unreachable.** The API numbers answers
  from zero, but `answer_id == 0` was rejected as a missing value, so the first
  option's voters could never be fetched. Zero is accepted now and only a
  negative index is refused
- **Several documented limits reported ad-hoc errors** instead of the promised
  `*ym.LimitError`, so `errors.As` worked for some and not others: the shared
  gallery size, the typing timeout and processing-text bounds, and the
  membership caps in `UpdateMembers` — which also did not say which list was
  over. The typing bounds moved to `client/ym/limits.go` alongside the rest,
  and a new `ym.ValidateRange` backs them
- **The webhook recipe registered a URL without its secret.** With `Secret` set
  the handler rejects every delivery that lacks `?secret=`, so a bot copied
  from the recipe would have 403'd all legitimate traffic
- **The upload endpoints skipped the documented limits.** `SendFile`,
  `SendImage` and `SendGallery` build their own multipart bodies rather than
  going through SendMessageOptions, so an oversized keyboard or an over-long
  gallery caption was serialised and sent; `polls.Create` did the same with a
  keyboard. All of them now share one `validateLimits`, and the gallery image
  count reports a `*ym.LimitError` like every other limit
- **The header-injection fix reached only one of two multipart builders.**
  `files.SendToChat` and `files.SendToLogin` kept their own sanitizer, which
  escaped quotes and backslashes but left CR and LF intact, so those two calls
  stayed vulnerable. Both builders now share `ym.SanitizeFilename`
- **A webhook batch that would not decode was acknowledged and dropped.**
  The single-update fallback parsed the envelope itself as an empty update, so
  ServeHTTP answered 200, OnError never fired, and every update in the batch
  vanished. Batch payloads now report their decode failure
- **The polling examples overrode the transient-only default** with an
  unconditional `ActionRetry`, teaching exactly the hot loop the default was
  changed to prevent. `DefaultPollErrorAction` is now exported so a caller who
  only wants to log can delegate the decision
- **The chat length limits were declared but never checked.** MaxChatNameLength
  and MaxChatDescriptionLength existed as constants while `validateCreate`
  looked at neither, so the documented local enforcement did not happen. The
  membership counts in the same function also reported ad-hoc errors rather
  than the promised `*ym.LimitError`; both now go through ValidateLength and
  ValidateCount
- **The webhook recipe in the skill served an open endpoint.** It read the
  secret and the path with optional `os.Getenv` while the example had already
  been fixed to require them — and the recipe is the copy meant to be pasted
  into other projects. Both values are now mandatory
- **The webhook example echoed unescaped user text**, breaking the SDK's own
  rule, and set `reply_message_id` even when `YM_REPLY_CHAT` redirected the
  reply to another chat, which the API rejects because the message must belong
  to the target chat
- **An unclassified poll failure spun in a hot loop.** The default policy
  stopped only on an enumerated list of permanent API errors and retried
  everything else, so a body that would not decode or a caller's own transport
  failing produced 1250 retries in two seconds under test. The default now
  whitelists what is known to clear on its own — transport failures recognised
  through net.Error, rate limits and 5xx — and stops on the rest
- **`ValidatePageLimit` returned a plain error** while the docs promised
  `*ym.LimitError` for every documented violation, so page limits could not be
  matched with errors.As like the rest. LimitError also gained a Min field so a
  range violation reads correctly
- **`EditText` accepted a zero message id** and serialised it, although zero
  means "no message" in every other message-scoped method
- **The webhook example could exit before its drain finished.** Shutting the
  server down makes ListenAndServe return at once, so main raced the goroutine
  that drains accepted updates — losing exactly the work the early
  acknowledgement promised. The recipe in the skill had the same shape
- **A revoked token looped forever.** The default poll policy retried every
  failure, so a permanent 401, 403 or 400 was retried at MaxBackoff
  indefinitely and never reached the caller. The default now retries only what
  a later attempt might survive — network trouble, rate limits, 5xx — and stops
  on permanent failures
- **`MaxBackoff` did not bound the first retry.** The initial delay was a
  hardcoded second, so tuning MaxBackoff down had no effect until the second
  attempt
- **`reply_quote` without `reply_message_id`, and `forwards` combined with a
  reply**, were serialised and sent even though the API documents both as
  invalid; they now fail locally
- **The `share*` endpoints sent an undocumented `payload_id`.** They reused the
  send envelope, so every resend carried a key the API documents only for
  sendText, sendSticker, sendSystemMessage and createPoll — risking rejection
  and, worse, making a retry look idempotent when nothing deduplicates it
- **`SendTyping` accepted a processing indicator with empty or over-long text**,
  which the API documents as 1 to 100 characters
- **`Run` forwarded an out-of-range `limit`.** The resulting 400 fed the default
  retry policy, turning an impossible request into an endless hot loop
- **A transport returning neither a response nor an error panicked the client**
  with a nil dereference instead of reporting a transport failure
- **`polls.Create` sent no idempotency key** although createPoll documents
  `payload_id`, so a retried create could produce two polls
- **`GetReactions` shipped an out-of-range `limit` to the server** instead of
  reporting a `*ym.LimitError` locally, unlike every other paginated method
- **Shared images were accepted without dimensions.** The API requires width
  and height on `shareImage` and `shareGallery`; zeroes produced a request the
  server could only reject
- **The webhook example accepted unauthenticated deliveries.** `YM_WEBHOOK_SECRET`
  was optional while the route was fixed and guessable, so a forged update
  would have made the bot send an authenticated reply to a chat of the
  caller's choosing. The secret is now required
- **The poller example never reached its reaction and membership branches.**
  A guard requiring chat and sender ran first, and those updates carry neither
- **`ActionRetry` on a handler error never retried.** It behaved like
  `ActionContinue`, and the advancing offset then put the update permanently
  out of reach. It now re-invokes the handler on the same update, bounded by
  `RunOptions.MaxHandlerRetries`
- `self.Update` reported `KindBadRequest` regardless of the actual HTTP status
- `Update` was missing `reply_to_message`, `forwarded_messages`,
  `chat_members_update` and `reaction`

### Changed
- The idempotency guarantee is now stated precisely: `payload_id` is documented
  for `sendText`, `sendSticker`, `sendSystemMessage` and `createPoll` only.
  Multipart uploads accept no such key, so retrying one can duplicate it —
  previously the docs implied every retried send was safe
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
