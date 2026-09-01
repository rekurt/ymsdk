---
name: ymsdk
description: Write, review or debug Go code that talks to the Yandex Messenger Bot API through github.com/rekurt/ymsdk. Use this whenever the task involves a Yandex Messenger bot, Яндекс Мессенджер бот, botapi.messenger.yandex.net, the ymsdk module, or any Go code that sends messages, uploads files, manages chats, runs polls, handles reactions, or receives updates by polling or webhook from Yandex Messenger — even when the user does not name the SDK, and even when they only ask to "fix" or "review" such code. The SDK has several sharp edges (idempotency keys, a nested image shape, a 1-second webhook budget, destructive update offsets) where the obvious code compiles but breaks in production, so consult this before writing rather than after.
---

# ymsdk: Yandex Messenger bots in Go

`github.com/rekurt/ymsdk` covers all 28 documented Bot API endpoints. This
skill exists because several of them behave in ways that plausible-looking Go
code gets wrong — the compiler will not catch these, and neither will a quick
manual test.

## How to use this skill

For anything beyond a one-line change, read
[`references/reference.md`](references/reference.md) first. It is the complete
API surface with signatures and the reasoning behind each pitfall. The
recipes in [`references/recipes.md`](references/recipes.md) are working
programs for the three shapes most tasks take: an echo bot, a bot with
buttons, and a webhook service.

`references/reference.md` is the single source of truth for this SDK. The
repository's `AGENTS.md`, `GEMINI.md`, Cursor rule, Copilot instructions and
Windsurf rules all point at it rather than restating it, so there is exactly
one document to keep in step with the code.

## The seven things to get right

Everything else in the SDK behaves the way you would expect. These do not.

**1. Turn retries on.** `MaxAttempts` defaults to 1, so out of the box a
single 502 fails the call. Production configs want 3.

**2. Leave `payload_id` alone, and know where it does not reach.** The API
deduplicates requests carrying the same `payload_id`, and the SDK stamps one on
`sendText`, `sendSticker`, `sendSystemMessage` and `createPoll` — the four
endpoints that document it — so a retried request cannot post twice. Setting
`DisableAutoPayloadID` without supplying your own key silently makes those
retries unsafe. Multipart uploads (`sendFile`, `sendImage`, `sendGallery`) have
no idempotency key in the API at all, so a retried upload can duplicate; keep
uploads short or send them with `MaxAttempts: 1`.

**3. Guard every update field before reading it.** Reaction events, membership
changes and button presses are updates with no text and frequently no `Chat`
or `From`. `upd.Chat.ID` on one of those panics.

**4. `Update.Images` is `[][]ym.Image`.** Outer slice: one entry per image.
Inner slice: that image's size variants. Call `upd.OriginalImages()` for one
full-size image each — flattening by hand yields four copies of every picture.
An incoming file is `upd.Document`, decoded from the API's `file` field.

**5. Never do work inline in a webhook.** The API allows 1 second to respond
and retries on timeout, so an inline reply produces duplicate deliveries. Use
`updates.NewWebhookHandler`: it answers immediately, works on a pool, and drops
repeats. Delivery is at-least-once, so deduplication is required, not optional.

**6. `getUpdates` destroys what it passes.** Every update below the offset is
erased permanently. Advance only after handling the batch. Use `updates.Run`,
not the deprecated `PollLoop`, which dies on the first error.

**7. Escape user text you echo.** `**`, `__`, `~~`, `++` and backticks are
markup. `ym.EscapeMarkdown` neutralises them; `ym.Code` suppresses formatting
entirely and is the safer choice for arbitrary input.

## Shape of a bot

```go
cs := client.New(ym.Config{
    Token: os.Getenv("YM_TOKEN"),
    ErrorHandling: ymerrors.ErrorHandlingConfig{
        RetryStrategy: ymerrors.RetryStrategy{
            MaxAttempts: 3, InitialBackoff: 500 * time.Millisecond,
            MaxBackoff: 10 * time.Second, RetryNetwork: true,
        },
        RateLimitHandling: ymerrors.RateLimitHandling{UseRetryAfter: true},
    },
})

err := cs.Updates.Run(ctx, updates.RunOptions{
    OnHandlerError: func(u ym.Update, err error) updates.ErrorAction {
        log.Printf("update %d: %v", u.UpdateID, err)
        return updates.ActionContinue
    },
}, func(ctx context.Context, u ym.Update) error {
    if u.Chat == nil || u.Text == "" {
        return nil
    }
    _, err := cs.Messages.SendToChat(ctx, u.Chat.ID, "echo: "+ym.EscapeMarkdown(u.Text), nil)
    return err
})
```

Recipients go through `ym.Target`, and exactly one field may be set:
`ym.ChatTarget(id)`, `ym.LoginTarget(login)`, `ym.UserIDTarget(uuid)`.

## Errors

`errors.Is` matches the sentinels in `ymerrors`; `errors.As` reaches the
`*ymerrors.APIError` carrying `HTTPStatus`, `Description`, `RequestID` and
`RetryAfter`.

```go
var apiErr *ymerrors.APIError
if errors.Is(err, ymerrors.ErrRateLimited) && errors.As(err, &apiErr) {
    time.Sleep(apiErr.RetryAfter)
}
```

`ErrInvalidToken` and `ErrUnauthorized` are fatal — retrying will not help.

## Before you finish

Reviewing or writing bot code, check these, since each has a failure mode that
only shows up under load or on the unhappy path:

- retries enabled, and `payload_id` not disabled;
- every `upd.Chat`, `upd.From` and attachment field guarded before use;
- images read through `OriginalImages()`;
- webhook work off the request path, with deduplication left on;
- update offsets advanced only after successful handling;
- user-supplied text escaped before it goes back out;
- `get_reactions` / `get_members_changed` switched on via `cs.Self.Update` if
  the bot expects those events at all.
