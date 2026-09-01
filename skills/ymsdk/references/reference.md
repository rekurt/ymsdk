# ymsdk reference

Complete reference for `github.com/rekurt/ymsdk`, a Go client for the Yandex
Messenger Bot API. Written for coding assistants: it states the whole API
surface and, more importantly, the places where a reasonable-looking guess
produces broken code.

- Module: `github.com/rekurt/ymsdk` · Go 1.25+ · one dependency (`go.uber.org/zap`)
- Upstream API docs: <https://yandex.ru/dev/messenger/doc/ru/>
- Coverage: all 28 documented endpoints

## Contents

1. [Setup](#setup)
2. [Traps](#traps) — read this before writing code
3. [Addressing a recipient](#addressing-a-recipient)
4. [Messages](#messages)
5. [Attachments](#attachments)
6. [Reactions, pinning, typing](#reactions-pinning-typing)
7. [Chats](#chats)
8. [Polls, users, bot settings](#polls-users-bot-settings)
9. [Receiving updates](#receiving-updates)
10. [Errors and retries](#errors-and-retries)
11. [Text formatting](#text-formatting)
12. [Limits](#limits)
13. [Testing](#testing)

## Setup

```go
import (
    "github.com/rekurt/ymsdk/client"
    "github.com/rekurt/ymsdk/client/ym"
    "github.com/rekurt/ymsdk/client/ym/ymerrors"
)

cs := client.New(ym.Config{
    Token: os.Getenv("YM_TOKEN"),
    ErrorHandling: ymerrors.ErrorHandlingConfig{
        RetryStrategy: ymerrors.RetryStrategy{
            MaxAttempts:    3,   // default is 1, i.e. no retries at all
            InitialBackoff: 500 * time.Millisecond,
            MaxBackoff:     10 * time.Second,
            RetryNetwork:   true,
        },
        RateLimitHandling: ymerrors.RateLimitHandling{
            UseRetryAfter:  true,
            DefaultBackoff: time.Second,
        },
    },
})
```

`client.New` returns a `*client.YMClient` holding every service:
`cs.Messages`, `cs.Chats`, `cs.Users`, `cs.Polls`, `cs.Updates`, `cs.Self`,
`cs.Files`, plus the underlying `cs.Client`. `client.Wrap(cl)` does the same
around an existing `*ym.Client`.

For a custom transport — proxies, instrumentation, tests — use
`ym.NewClientWithHTTP(cfg, doer)` with any `ym.HTTPDoer`.

## Traps

These are the mistakes that produce code which compiles, looks right, and
fails in production.

**Retries are off by default.** `MaxAttempts` defaults to 1. Set it to 3 or
more, or a single transient 502 fails the call.

**Do not disable `payload_id`.** The API treats two requests with the same
`payload_id` as duplicates. The SDK generates one per send, and every retry
replays the identical body, which is exactly what stops a retried `sendText`
from posting the message twice. `Config.DisableAutoPayloadID` exists for
callers who supply their own key — turning it on without supplying one makes
retries unsafe.

**Not every update is a message.** Reaction events, membership changes and
button callbacks arrive with no text and often no `Chat` or `From`. Check the
field you are about to read:

```go
if upd.Chat == nil || upd.From == nil {
    return nil // not a message-shaped update
}
```

**`Update.Images` is `[][]ym.Image`.** The outer slice is one entry per image;
the inner slice is that image's size variants (small, middle, middle-400,
original). To get one full-size image each, call `upd.OriginalImages()`.
Treating it as a flat list will not compile — and flattening it by hand gives
four copies of every picture.

**An incoming file is `upd.Document`**, decoded from the API's `file` field.

**Webhook handlers must not do work inline.** The API allows 100 ms to connect
and 1 s to respond. Sending a reply takes longer than that, so an inline
handler times out and the API retries — delivering the message again. Use
`updates.NewWebhookHandler`, which acknowledges immediately, processes on a
worker pool, and drops repeats. Delivery is at-least-once, so deduplication is
not optional.

**Fetching updates with an offset destroys them.** `getUpdates` permanently
erases every update whose ID is below the offset. Advance the offset only
after the batch has been handled.

**Prefer `updates.Run` over `PollLoop`.** `PollLoop` is deprecated and stops
on the first error of any kind, so one 500 takes the bot down.

**Escape user text before echoing it.** `**`, `__`, `~~`, `++` and backticks
are markup. Use `ym.EscapeMarkdown` or wrap in `ym.Code`.

**Enable the optional update types before expecting them.** Reaction and
membership events are not delivered until the bot's `get_reactions` and
`get_members_changed` flags are on. See [bot settings](#polls-users-bot-settings).

## Addressing a recipient

Most methods take a `ym.Target`. Exactly one field may be set; the API rejects
requests naming more than one.

```go
ym.ChatTarget("0/0/4f24b544-…")     // group chat or channel
ym.LoginTarget("user@example.org")  // private chat by login
ym.UserIDTarget("447c35f4-…")       // private chat by UUID
```

`ym.ValidateTarget(t)` returns `ym.ErrNoTarget` or `ym.ErrAmbiguousTarget`; the
services call it for you. `ym.ValidateRecipient` is the deprecated pointer-pair
form and does not understand `user_id`.

## Messages

```go
// The chat/login convenience wrappers cover the common cases.
msg, err := cs.Messages.SendToChat(ctx, "chat-id", "hello", nil)
msg, err := cs.Messages.SendToLogin(ctx, "user@example.org", "hello", nil)

// SendText is the general form and the only one that reaches user_id.
msg, err := cs.Messages.SendText(ctx, ym.ChatTarget("chat-id"), "hello", &messages.SendMessageOptions{
    ReplyToMessageID: ym.Ptr(ym.MessageID(123)),
    ReplyQuote:       "the part being answered",
    ThreadID:         ym.Ptr(ym.ThreadID(456)),
    Important:        ym.Ptr(true),
    SuggestButtons:   keyboard,
})
```

The response carries only `message_id`; the returned `*ym.Message` has `ID`
set and the rest zero. That is the API's shape, not a gap in the SDK.

**Editing.** The API models an edit as `sendText` with `message_id`:

```go
_, err := cs.Messages.EditText(ctx, ym.ChatTarget("chat-id"), messageID, "corrected", nil)
```

**Forwarding.** `Forwards` cannot be combined with a reply:

```go
_, err := cs.Messages.SendText(ctx, target, "", &messages.SendMessageOptions{
    Forwards: []ym.Forward{{ChatID: "source-chat", MessageIDs: []ym.MessageID{1, 2}}},
})
```

**Buttons.** `SuggestButtons` is a keyboard (≤100 buttons); `ActionButtons` is
a row of action buttons (≤6). `InlineKeyboard`/`ym.Button` are deprecated
upstream.

```go
keyboard := &ym.SuggestButtons{Buttons: [][]ym.InlineSuggestButton{{
    {ID: "yes", Title: "Yes", Directives: []ym.Directive{{
        Type: ym.DirectiveServerAction, Name: "confirm",
        Payload: json.RawMessage(`{"id":42}`),
    }}},
}}}
```

A `server_action` press arrives as an update with `BotRequest.ServerAction` set
and no message text.

**Other message kinds.**

```go
cs.Messages.SendSticker(ctx, target, setID, stickerID, nil)      // ids come from ym.Sticker
cs.Messages.SendSystemMessage(ctx, target, "deployed", nil)      // no sender shown
cs.Messages.Delete(ctx, &messages.DeleteMessageRequest{ChatID: ym.Ptr(ym.ChatID("c")), MessageID: id})
```

## Attachments

Uploading streams bytes; sharing reuses a `file_id` and skips the upload.

```go
// Upload
cs.Messages.SendFile(ctx, &messages.SendFileRequest{
    ChatID: ym.Ptr(ym.ChatID("chat-id")), Document: reader, Filename: "report.pdf",
})
cs.Messages.SendImage(ctx, &messages.SendImageRequest{ChatID: …, Image: reader, Filename: "a.png"})
cs.Messages.SendGallery(ctx, &messages.SendGalleryRequest{ChatID: …, Images: parts}) // 1..10

// Re-send something already stored
cs.Messages.ShareFile(ctx, target, ym.SharedFile{FileID: id}, nil)
cs.Messages.ShareImage(ctx, target, ym.SharedImage{FileID: id, Width: w, Height: h}, nil)
cs.Messages.ShareGallery(ctx, target, []ym.SharedImage{…}, nil)

// Download — the caller closes the reader
body, meta, err := cs.Messages.GetFile(ctx, fileID)
defer body.Close()
```

`ShareImage` requires width and height; take them from the `ym.Image` in the
update or the earlier upload. Uploads are buffered in memory so a retry can
replay them, so stream very large files yourself rather than through a retrying
client.

## Reactions, pinning, typing

```go
cs.Messages.SendReaction(ctx, target, messageID, ym.DefaultReaction("like"), nil)
cs.Messages.RemoveReaction(ctx, target, messageID, nil)   // same as a nil reaction

page, err := cs.Messages.GetReactions(ctx, target, messageID, nil)
switch page.Type {
case ym.ReactionsPublic:  // private and group chats: reactions with authors
    for _, e := range page.List { _ = e.User.Login }
case ym.ReactionsPrivate: // channels: anonymous tallies
    for _, c := range page.Counts { _ = c.Count }
}

cs.Messages.Pin(ctx, target, messageID, nil)
cs.Messages.Unpin(ctx, target, messageID, nil)

cs.Messages.SendTyping(ctx, target, nil) // "typing…", clears after 3s
cs.Messages.SendTyping(ctx, target, &messages.SendTypingOptions{
    Type:              ym.TypingProcessing, // private chats only
    ProcessingContent: &ym.ProcessingContent{Display: ym.ProcessingDisplayText, Text: "Thinking…"},
})
```

Which reactions a chat allows is in `ChatInfo.AvailableReactions`.

## Chats

```go
chat, err := cs.Chats.Create(ctx, &chats.ChatCreateRequest{
    Name: "Team", Description: "…", Channel: false,
    Members: []ym.UserRef{{Login: "a@example.org"}},
})

page, err := cs.Chats.List(ctx, chats.ListParams{Limit: ym.Ptr(100)})
all,  err := cs.Chats.ListAll(ctx, chats.ListParams{})   // walks every page

info, err := cs.Chats.GetChat(ctx, "chat-id")            // includes AvailableReactions

members, err := cs.Chats.GetMembers(ctx, chats.MembersParams{ChatID: "chat-id", Role: ym.RoleAdmin})
everyone, err := cs.Chats.GetAllMembers(ctx, chats.MembersParams{ChatID: "chat-id"})

err = cs.Chats.UpdateMembers(ctx, &chats.ChatUpdateMembersRequest{
    ChatID: "chat-id",
    Members: []ym.UserRef{{Login: "new@example.org"}},
    Remove:  []ym.UserRef{{Login: "old@example.org"}},
})
```

Pagination cursors are the last item's `ID` (chats) or `GUID` (members). The
`*All` helpers handle this and stop if the server repeats a cursor.

## Polls, users, bot settings

```go
msg, err := cs.Polls.Create(ctx, &polls.CreatePollRequest{ /* ChatID, Title, Answers */ })
res, err := cs.Polls.GetResults(ctx, polls.PollResultsParams{ /* ChatID, MessageID */ })
votes, err := cs.Polls.GetAllVoters(ctx, polls.PollVotersParams{ /* … */ })

link, err := cs.Users.GetUserLink(ctx, "user@example.org") // ChatLink, CallLink

bot, err := cs.Self.Get(ctx)
_, err = cs.Self.Update(ctx, &self.SelfUpdateRequest{
    WebhookURL: ym.Ptr("https://example.com/hook/<unguessable>"),
    Settings: &ym.BotSettings{
        GetReactions:      ym.Ptr(true), // without this, reaction events never arrive
        GetMembersChanged: ym.Ptr(true), // nor membership events
    },
})
```

## Receiving updates

### Polling

```go
err := cs.Updates.Run(ctx, updates.RunOptions{
    Limit: ym.Ptr(100),
    OnPollError: func(err error) updates.ErrorAction {
        log.Printf("poll failed: %v", err)
        return updates.ActionRetry // the default: back off and carry on
    },
    OnHandlerError: func(u ym.Update, err error) updates.ErrorAction {
        log.Printf("update %d failed: %v", u.UpdateID, err)
        return updates.ActionContinue // the default is ActionStop
    },
    OnPanic: func(u ym.Update, r any) { log.Printf("panic: %v", r) },
}, func(ctx context.Context, u ym.Update) error {
    return handle(ctx, u)
})
```

`Run` returns `context.Canceled` on shutdown and honours cancellation while
waiting. `OnPanic` opts into recovering a panicking handler.

Actions, and what each costs:

| Action | Poll error | Handler error |
|---|---|---|
| `ActionRetry` | back off and poll again (**default**) | re-invoke the handler on the same update, up to `MaxHandlerRetries` (3), then return the error |
| `ActionContinue` | poll again immediately | move to the next update — the failed one is then carried out of reach by the advancing offset |
| `ActionStop` | return the error | return the error (**default**) |

`ActionContinue` on a handler error accepts losing that update: `getUpdates`
erases everything below the offset, so once the batch advances it is gone.
Prefer `ActionRetry` when the work matters.

### Webhook

```go
hook := updates.NewWebhookHandler(handle, updates.WebhookOptions{
    Secret:       os.Getenv("YM_WEBHOOK_SECRET"), // checked against ?secret=
    Workers:      8,
    Queue:        256,
    DedupeWindow: 4096,
    OnError:      func(err error) { log.Printf("webhook: %v", err) },
})
mux.Handle("/hook/<unguessable>", hook)
…
hook.Shutdown(ctx) // drains accepted updates
```

Nothing about a delivery is signed and no custom headers are sent, so the
webhook URL itself is the credential — keep the path unguessable. The handler
answers 4xx for a bad secret or unparsable body (final for the API) and 503
when it cannot accept the update — a saturated queue, or a shutdown in
progress. A refused update is removed from the dedup window so the redelivery
it asks for is actually processed rather than mistaken for a duplicate.

`Shutdown` stops accepting new deliveries before draining, so it is safe to
call while requests are in flight.

### Reading an update

```go
switch {
case u.Reaction != nil:
    // u.Reaction.Action is ym.ReactionAdded or ym.ReactionRemoved
case u.ChatMembersUpdate != nil:
    // NewChatMembers / RemovedChatMembers
case u.BotRequest != nil:
    // a button was pressed; BotRequest.ServerAction carries Name and Payload
case len(u.Images) > 0:
    for _, img := range u.OriginalImages() { _ = img.FileID }
case u.Document != nil:
    _ = u.Document.Name
case len(u.ForwardedMessages) > 0:
case u.ReplyToMessage != nil:
case u.Text != "":
}
```

## Errors and retries

```go
_, err := cs.Messages.SendToChat(ctx, chatID, text, nil)

var apiErr *ymerrors.APIError
switch {
case errors.Is(err, ymerrors.ErrRateLimited):
    if errors.As(err, &apiErr) { time.Sleep(apiErr.RetryAfter) }
case errors.Is(err, ymerrors.ErrInvalidToken),
     errors.Is(err, ymerrors.ErrUnauthorized):
    // fatal: the token is wrong
case errors.As(err, &apiErr):
    log.Printf("%s %s -> %d: %s", apiErr.Method, apiErr.Endpoint, apiErr.HTTPStatus, apiErr.Description)
}
```

Sentinels: `ErrRateLimited`, `ErrInvalidToken`, `ErrUnauthorized`,
`ErrBadRequest`, `ErrNotFound`, `ErrConflict`, `ErrPayloadTooLarge`,
`ErrRequestTimeout`, `ErrNetworkError`, `ErrInvalidResponse`.

`APIError` carries `Kind`, `Code`, `HTTPStatus`, `Description`, `RequestID`,
`Method`, `Endpoint` and `RetryAfter`. A 200 whose body says `ok:false` matches
both `ErrInvalidResponse` and `*APIError`.

Retry behaviour comes from `RetryStrategy`: `MaxAttempts` (default 1),
`InitialBackoff`, `MaxBackoff`, `RetryHTTP` (default 500/502/503/504),
`RetryNetwork`, `DisableJitter`. Back-off is jittered over [d/2, d] unless
disabled, and rate-limit waits honour `Retry-After` when `UseRetryAfter` is set.

## Text formatting

The messenger renders `**bold**`, `__italic__`, `~~strike~~`, `++underline++`,
`` `code` ``, fenced blocks and `[text](url)`.

```go
ym.Bold("important")
ym.Italic("aside")
ym.Code(userInput)                 // safest for arbitrary input
ym.CodeBlock("go", src)
ym.Link("docs", "https://example.com")
ym.EscapeMarkdown(userInput)       // neutralises the markers
```

The API documents the markup but no escape syntax, so `EscapeMarkdown` applies
the usual backslash convention as a best effort. When correctness matters more
than styling, `ym.Code` suppresses formatting outright.

## Limits

Checked locally before a request goes out; violations return `*ym.LimitError`.

| Constant | Value | Applies to |
|---|---|---|
| `ym.MaxTextLength` | 6000 | message text, in runes |
| `ym.MaxSuggestButtons` | 100 | suggest keyboard |
| `ym.MaxActionButtons` | 6 | action button row |
| `ym.MaxDirectivesPerButton` | 3 | directives per button |
| `ym.MaxButtonFieldLength` | 255 | button id and title |
| `ym.MaxGalleryImages` | 10 | album size |
| `ym.MaxPageLimit` | 1000 | paginated endpoints |
| `ym.MaxChatNameLength` | 200 | chat name |
| `ym.MaxChatDescriptionLength` | 500 | chat description |
| `ym.MaxChatAdmins` | 100 | admins per request |
| `ym.MaxChatMembers` | 500 | members or subscribers per request |

Image uploads are additionally capped at 3 GB per month per bot, which the SDK
cannot check.

## Testing

`internal/testutil.FakeDoer` replays queued responses and records requests:

```go
doer := &testutil.FakeDoer{Responses: []*http.Response{
    testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":1}`),
}}
cl := ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer)

_, err := messages.NewService(cl).SendToChat(context.Background(), "c1", "hi", nil)
// doer.Requests[0] holds the method, URL and body
```

Set `DisableAutoPayloadID: true` when asserting on exact request bodies, and
`RetryStrategy{DisableJitter: true}` for deterministic timing.
