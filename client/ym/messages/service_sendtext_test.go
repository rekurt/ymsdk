package messages

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func decodeJSONBody(t *testing.T, req *http.Request) map[string]any {
	t.Helper()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	return parsed
}

// sendText documents message_id, reply_quote, forwards and action_buttons too;
// leaving them out of one send method is how the two sendFile paths drifted.
func TestSendTextSendsAllDocumentedFields(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":1}`),
	}}
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL:       "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1}},
	}, doer)

	_, err := NewService(client).SendToChat(context.Background(), "c1", "hi", &SendMessageOptions{
		MessageID:        ym.Ptr(ym.MessageID(22)),
		ReplyToMessageID: ym.Ptr(ym.MessageID(33)),
		ReplyQuote:       "quoted",
		ActionButtons: &ym.ActionButtons{Buttons: []ym.ActionButton{{
			Title: "Like",
			Icon:  ym.ActionButtonIcon{Type: ym.ActionButtonIconType, Value: ym.IconLike},
		}}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := decodeJSONBody(t, doer.Requests[0])
	for _, name := range []string{"message_id", "reply_message_id", "reply_quote", "action_buttons"} {
		if _, ok := body[name]; !ok {
			t.Errorf("field %q missing from sendText body", name)
		}
	}
}

func TestSendTextSendsForwards(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"message_id":1}`),
	}}
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL:       "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1}},
	}, doer)

	_, err := NewService(client).SendToChat(context.Background(), "c1", "hi", &SendMessageOptions{
		Forwards: []ym.Forward{{ChatID: "src", MessageIDs: []ym.MessageID{1, 2}}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := decodeJSONBody(t, doer.Requests[0])["forwards"]; !ok {
		t.Error("forwards missing from sendText body")
	}
}
