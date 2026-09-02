package updates

import (
	"context"
	"net/http"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func serviceReturning(t *testing.T, payload string) *Service {
	t.Helper()

	doer := &testutil.FakeDoer{Responses: []*http.Response{testutil.NewResponse(http.StatusOK, payload)}}

	return NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer))
}

// Payloads below are the documented update examples verbatim, so a schema drift
// in the SDK's types shows up here rather than in production.
func TestDecodesDocumentedUpdatePayloads(t *testing.T) {
	// The API sends each image as a list of size variants, so images is an
	// array of arrays. Decoding it into a flat slice used to fail, and because
	// a decode error rejects the whole response, a single incoming picture cost
	// the bot every update in the batch.
	t.Run("image update", func(t *testing.T) {
		svc := serviceReturning(t, `{"ok":true,"updates":[{
			"message_id":1702329451492005,"timestamp":1702329451,"update_id":1571251,
			"chat":{"type":"private"},
			"from":{"id":"g","display_name":"Ivan","login":"ivan","robot":false},
			"images":[[
				{"file_id":"f?size=small","width":150,"height":11},
				{"file_id":"f?size=middle","width":250,"height":18},
				{"file_id":"f","width":1048,"height":78,"size":20362,"name":"a.jpeg"}]]}]}`)

		upds, _, err := svc.GetUpdates(context.Background(), GetUpdatesParams{})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(upds) != 1 {
			t.Fatalf("expected 1 update, got %d", len(upds))
		}
		if got := len(upds[0].Images); got != 1 {
			t.Fatalf("expected 1 image, got %d", got)
		}
		if got := len(upds[0].Images[0]); got != 3 {
			t.Fatalf("expected 3 size variants, got %d", got)
		}

		originals := upds[0].OriginalImages()
		if len(originals) != 1 {
			t.Fatalf("expected 1 original, got %d", len(originals))
		}
		if originals[0].Name != "a.jpeg" || originals[0].Width != 1048 {
			t.Fatalf("expected the full-size variant, got %#v", originals[0])
		}
	})

	// The API names the attachment "file"; the SDK read "document", so files
	// silently arrived as nil.
	t.Run("file update", func(t *testing.T) {
		svc := serviceReturning(t, `{"ok":true,"updates":[{
			"message_id":1702329844441005,"timestamp":1702329844,"update_id":1571253,
			"chat":{"type":"private"},"from":{"id":"g","login":"ivan"},
			"file":{"id":"fid","name":"data.txt","size":20}}]}`)

		upds, _, err := svc.GetUpdates(context.Background(), GetUpdatesParams{})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if upds[0].Document == nil {
			t.Fatal("the attached file did not deserialise")
		}
		if upds[0].Document.Name != "data.txt" || upds[0].Document.Size != 20 {
			t.Fatalf("unexpected file: %#v", upds[0].Document)
		}
	})

	t.Run("membership update", func(t *testing.T) {
		svc := serviceReturning(t, `{"ok":true,"updates":[{
			"update_id":1571254,"message_id":1702329900000005,"timestamp":1702329900,
			"chat":{"type":"group","id":"c1"},
			"chat_members_update":{"new_chat_members":[
				{"id":"g","display_name":"Ivan Ivanov","login":"ivan_ivanov","robot":false}]}}]}`)

		upds, _, err := svc.GetUpdates(context.Background(), GetUpdatesParams{})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		cmu := upds[0].ChatMembersUpdate
		if cmu == nil || len(cmu.NewChatMembers) != 1 {
			t.Fatalf("membership event did not deserialise: %#v", cmu)
		}
		if cmu.NewChatMembers[0].Login != "ivan_ivanov" {
			t.Fatalf("unexpected member: %#v", cmu.NewChatMembers[0])
		}
	})

	t.Run("reaction update", func(t *testing.T) {
		svc := serviceReturning(t, `{"ok":true,"updates":[{
			"update_id":1571255,"chat":{"type":"group","id":"c1"},
			"reaction":{"message_id":17,"reaction":{"type":"default_reaction","name":"like"},"action":"add"}}]}`)

		upds, _, err := svc.GetUpdates(context.Background(), GetUpdatesParams{})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		ev := upds[0].Reaction
		if ev == nil {
			t.Fatal("reaction event did not deserialise")
		}
		if ev.Action != ym.ReactionAdded || ev.Reaction.Name != "like" || ev.MessageID != 17 {
			t.Fatalf("unexpected reaction event: %#v", ev)
		}
	})

	t.Run("reply and forwards", func(t *testing.T) {
		svc := serviceReturning(t, `{"ok":true,"updates":[{
			"update_id":1571256,"chat":{"type":"group","id":"c1"},"text":"see above",
			"reply_to_message":{"update_id":1571200,"text":"original"},
			"forwarded_messages":[{"update_id":1571100,"text":"forwarded"}]}]}`)

		upds, _, err := svc.GetUpdates(context.Background(), GetUpdatesParams{})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if upds[0].ReplyToMessage == nil || upds[0].ReplyToMessage.Text != "original" {
			t.Fatalf("reply did not deserialise: %#v", upds[0].ReplyToMessage)
		}
		if len(upds[0].ForwardedMessages) != 1 || upds[0].ForwardedMessages[0].Text != "forwarded" {
			t.Fatalf("forwards did not deserialise: %#v", upds[0].ForwardedMessages)
		}
	})
}

func TestOriginalImages(t *testing.T) {
	t.Run("nil update yields nothing", func(t *testing.T) {
		var u *ym.Update
		if got := u.OriginalImages(); got != nil {
			t.Fatalf("expected nil, got %#v", got)
		}
	})

	t.Run("falls back to the widest variant when none is marked", func(t *testing.T) {
		u := &ym.Update{Images: [][]ym.Image{{
			{FileID: "a", Width: 150},
			{FileID: "b", Width: 900},
			{FileID: "c", Width: 250},
		}}}
		got := u.OriginalImages()
		if len(got) != 1 || got[0].FileID != "b" {
			t.Fatalf("expected the widest variant, got %#v", got)
		}
	})

	t.Run("skips empty variant lists", func(t *testing.T) {
		u := &ym.Update{Images: [][]ym.Image{{}, {{FileID: "a", Width: 10}}}}
		if got := u.OriginalImages(); len(got) != 1 || got[0].FileID != "a" {
			t.Fatalf("unexpected result: %#v", got)
		}
	})
}
