package polls

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func pollService(doer ym.HTTPDoer, disableAuto bool) *Service {
	return NewService(ym.NewClientWithHTTP(ym.Config{
		BaseURL:              "http://example.com",
		DisableAutoPayloadID: disableAuto,
	}, doer))
}

func sentPayloadID(t *testing.T, req *http.Request) string {
	t.Helper()

	raw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var parsed struct {
		PayloadID string `json:"payload_id"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("decode body %s: %v", raw, err)
	}

	return parsed.PayloadID
}

// createPoll documents payload_id, so a retried create must not produce two polls.
func TestCreateStampsPayloadID(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"message":{"message_id":1}}`),
	}}
	svc := pollService(doer, false)

	_, err := svc.Create(context.Background(), &CreatePollRequest{
		ChatID:  ym.Ptr(ym.ChatID("c1")),
		Title:   "Lunch?",
		Answers: []string{"yes", "no"},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got := sentPayloadID(t, doer.Requests[0]); got == "" {
		t.Fatal("expected an auto-generated payload_id on createPoll")
	}
}

func TestCreateKeepsCallerPayloadID(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"message":{"message_id":1}}`),
	}}
	svc := pollService(doer, false)

	req := &CreatePollRequest{
		ChatID:    ym.Ptr(ym.ChatID("c1")),
		Title:     "Lunch?",
		Answers:   []string{"yes", "no"},
		PayloadID: ym.Ptr("mine"),
	}
	if _, err := svc.Create(context.Background(), req); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got := sentPayloadID(t, doer.Requests[0]); got != "mine" {
		t.Fatalf("expected the caller's key, got %q", got)
	}
	// The caller's struct must come back unmodified.
	if req.PayloadID == nil || *req.PayloadID != "mine" {
		t.Fatalf("Create mutated the caller's request: %#v", req.PayloadID)
	}
}

func TestCreateRespectsPayloadIDOptOut(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"message":{"message_id":1}}`),
	}}
	svc := pollService(doer, true)

	_, err := svc.Create(context.Background(), &CreatePollRequest{
		ChatID:  ym.Ptr(ym.ChatID("c1")),
		Title:   "Lunch?",
		Answers: []string{"yes", "no"},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got := sentPayloadID(t, doer.Requests[0]); got != "" {
		t.Fatalf("expected no key, got %q", got)
	}
}
