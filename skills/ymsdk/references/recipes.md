# ymsdk recipes

Working programs for the three shapes most bot tasks take, plus attachment
handling. Each needs `YM_TOKEN` in the environment; the webhook service also
requires `YM_WEBHOOK_SECRET` and `YM_WEBHOOK_PATH`, since nothing else
authenticates a delivery.

## Contents

1. [Echo bot (polling)](#echo-bot-polling)
2. [Bot with buttons](#bot-with-buttons)
3. [Webhook service](#webhook-service)
4. [Attachments](#attachments)

## Echo bot (polling)

Shows the resilient loop, guarded update fields, and escaping text on the way
back out.

```go
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/messages"
	"github.com/rekurt/ymsdk/client/ym/updates"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cs := client.New(ym.Config{
		Token: os.Getenv("YM_TOKEN"),
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			// Without this the client makes one attempt and any 502 fails the send.
			RetryStrategy: ymerrors.RetryStrategy{
				MaxAttempts:    3,
				InitialBackoff: 500 * time.Millisecond,
				MaxBackoff:     10 * time.Second,
				RetryNetwork:   true,
			},
			RateLimitHandling: ymerrors.RateLimitHandling{UseRetryAfter: true},
		},
	})

	err := cs.Updates.Run(ctx, updates.RunOptions{
		Limit: ym.Ptr(100),
		// One bad update should not end the bot.
		OnHandlerError: func(u ym.Update, err error) updates.ErrorAction {
			log.Printf("update %d failed: %v", u.UpdateID, err)

			return updates.ActionContinue
		},
	}, func(ctx context.Context, u ym.Update) error {
		// Reaction events, membership changes and button presses arrive here
		// too, with no text and often no chat.
		if u.Chat == nil || u.Text == "" {
			return nil
		}

		opts := &messages.SendMessageOptions{
			ReplyToMessageID: ym.Ptr(u.MessageID),
		}
		// The messenger renders ** and __ as markup, so echoed text is escaped.
		reply := "echo: " + ym.EscapeMarkdown(u.Text)

		_, err := cs.Messages.SendText(ctx, ym.ChatTarget(u.Chat.ID), reply, opts)

		return err
	})

	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("bot stopped: %v", err)
	}
}
```

## Bot with buttons

A `server_action` press arrives as an update carrying `BotRequest` and no text,
so that branch has to come before the text branch.

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"

	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/messages"
	"github.com/rekurt/ymsdk/client/ym/updates"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

var keyboard = &ym.SuggestButtons{
	Buttons: [][]ym.InlineSuggestButton{{
		{
			ID:    "approve",
			Title: "Approve",
			Directives: []ym.Directive{{
				Type:    ym.DirectiveServerAction,
				Name:    "approve",
				Payload: json.RawMessage(`{"request_id":42}`),
			}},
		},
		{
			ID:    "docs",
			Title: "Open docs",
			Directives: []ym.Directive{{
				Type: ym.DirectiveOpenURI,
				URI:  "https://example.com/docs",
			}},
		},
	}},
}

func handle(ctx context.Context, cs *client.YMClient, u ym.Update) error {
	// Button presses carry no text, so check them first.
	if u.BotRequest != nil && u.BotRequest.ServerAction != nil {
		action := u.BotRequest.ServerAction
		log.Printf("pressed %q with payload %s", action.Name, action.Payload)

		if u.Chat == nil {
			return nil
		}
		_, err := cs.Messages.SendText(ctx, ym.ChatTarget(u.Chat.ID),
			"Recorded: "+ym.Code(action.Name), nil)

		return err
	}

	if u.Chat == nil || u.Text == "" {
		return nil
	}

	_, err := cs.Messages.SendText(ctx, ym.ChatTarget(u.Chat.ID), "Choose an option:",
		&messages.SendMessageOptions{SuggestButtons: keyboard})

	return err
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cs := client.New(ym.Config{
		Token: os.Getenv("YM_TOKEN"),
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 3},
		},
	})

	err := cs.Updates.Run(ctx, updates.RunOptions{}, func(ctx context.Context, u ym.Update) error {
		return handle(ctx, cs, u)
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("bot stopped: %v", err)
	}
}
```

## Webhook service

The API allows 100 ms to connect and 1 s to respond, and redelivers on
timeout. `NewWebhookHandler` acknowledges immediately, runs the handler on a
worker pool, and drops the repeats that at-least-once delivery guarantees.

```go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/rekurt/ymsdk/client"
	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/updates"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cs := client.New(ym.Config{
		Token:       os.Getenv("YM_TOKEN"),
		UpdatesMode: ymerrors.UpdatesModeWebhook,
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 3},
		},
	})

	// Nothing about a delivery is signed and no custom headers arrive, so these
	// two values are the only thing standing between the bot and anyone who can
	// reach the port. Without them a forged update makes the bot send an
	// authenticated message to a chat of the caller's choosing, so refuse to
	// start rather than serve an open endpoint.
	webhookSecret := mustEnv("YM_WEBHOOK_SECRET")
	webhookPath := mustEnv("YM_WEBHOOK_PATH")

	hook := updates.NewWebhookHandler(
		func(ctx context.Context, u ym.Update) error {
			if u.Chat == nil || u.Text == "" {
				return nil
			}
			// Sending a reply takes far longer than the API's 1s budget, which
			// is exactly why this runs off the request path.
			_, err := cs.Messages.SendText(ctx, ym.ChatTarget(u.Chat.ID), "got it", nil)

			return err
		},
		updates.WebhookOptions{
			Secret:  webhookSecret,
			Workers: 8,
			OnError: func(err error) { log.Printf("webhook: %v", err) },
		},
	)

	mux := http.NewServeMux()
	mux.Handle("/hook/"+webhookPath, hook)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Shutting the server down makes ListenAndServe return at once, so main has
	// to wait for the drain — otherwise the process exits while accepted
	// updates are still in flight.
	drained := make(chan struct{})

	go func() {
		defer close(drained)

		<-ctx.Done()

		// Two deadlines, not one: a slow request can consume the whole HTTP
		// shutdown budget, and reusing that exhausted context would make the
		// drain return at once, dropping already-acknowledged updates.
		httpCtx, cancelHTTP := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelHTTP()
		_ = srv.Shutdown(httpCtx)

		drainCtx, cancelDrain := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelDrain()
		// Drain updates accepted but not yet processed.
		_ = hook.Shutdown(drainCtx)
	}()

	// Register the endpoint once, then leave it in place. The secret has to be
	// part of the registered URL — the handler checks it on every delivery, so
	// a URL without it gets 403 for legitimate traffic:
	//
	//   hookURL := "https://example.com/hook/" + webhookPath +
	//       "?secret=" + url.QueryEscape(webhookSecret)
	//   cs.Self.Update(ctx, &self.SelfUpdateRequest{WebhookURL: ym.Ptr(hookURL)})

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}

	<-drained
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is required", key)
	}

	return v
}
```

## Attachments

```go
func handleAttachments(ctx context.Context, cs *client.YMClient, u ym.Update) error {
	switch {
	case len(u.Images) > 0:
		// Each image arrives as a list of size variants; take the original of each.
		for _, img := range u.OriginalImages() {
			body, meta, err := cs.Messages.GetFile(ctx, img.FileID)
			if err != nil {
				return err
			}
			defer body.Close()
			_ = meta
		}

	case u.Document != nil:
		// Decoded from the API's "file" field.
		body, _, err := cs.Messages.GetFile(ctx, u.Document.ID)
		if err != nil {
			return err
		}
		defer body.Close()
	}

	return nil
}
```

Resending something already stored skips the upload entirely:

```go
_, err := cs.Messages.ShareImage(ctx, target,
	ym.SharedImage{FileID: img.FileID, Width: img.Width, Height: img.Height}, nil)
```
