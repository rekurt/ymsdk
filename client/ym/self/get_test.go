package self

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func TestGet(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"id":"bot-1","login":"my-bot",
			"display_name":"My Bot","webhook_url":"https://example.com/hook",
			"organizations":[1,2],
			"settings":{"get_reactions":true,"get_members_changed":false}}`),
	}}
	svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

	bot, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if bot.ID != "bot-1" || bot.Login != "my-bot" || bot.DisplayName != "My Bot" {
		t.Fatalf("unexpected bot: %#v", bot)
	}
	if bot.WebhookURL == nil || *bot.WebhookURL != "https://example.com/hook" {
		t.Fatalf("unexpected webhook: %#v", bot.WebhookURL)
	}
	if bot.Settings == nil || bot.Settings.GetReactions == nil || !*bot.Settings.GetReactions {
		t.Fatalf("expected get_reactions to be reported as on: %#v", bot.Settings)
	}
	if bot.Settings.GetMembersChanged == nil || *bot.Settings.GetMembersChanged {
		t.Fatalf("expected get_members_changed to be reported as off: %#v", bot.Settings)
	}

	req := doer.Requests[0]
	if req.Method != http.MethodGet || req.URL.Path != ym.EndpointSelfGet {
		t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
	}
}

// Reaction and membership events are not delivered until the flags are on, so
// Update must be able to switch them.
func TestUpdateSendsSettings(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"id":"bot-1","login":"my-bot",
			"settings":{"get_reactions":true,"get_members_changed":true}}`),
	}}
	svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

	bot, err := svc.Update(context.Background(), &SelfUpdateRequest{
		Settings: &ym.BotSettings{
			GetReactions:      ym.Ptr(true),
			GetMembersChanged: ym.Ptr(true),
		},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if bot.Settings == nil || !*bot.Settings.GetMembersChanged {
		t.Fatalf("expected both flags reported on: %#v", bot.Settings)
	}
	if doer.Requests[0].URL.Path != ym.EndpointSelfUpdate {
		t.Fatalf("unexpected path: %s", doer.Requests[0].URL.Path)
	}
}

// id is the bot's identity and is documented as required; a zero-valued BotSelf
// with a nil error hides schema drift, as it did for the other single-object
// decoders.
func TestGetRejectsAResponseWithoutID(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"login":"my-bot"}`),
	}}
	svc := NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))

	bot, err := svc.Get(context.Background())
	if !errors.Is(err, ymerrors.ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got %v", err)
	}
	if bot != nil {
		t.Fatalf("expected no bot alongside the error, got %#v", bot)
	}
}
