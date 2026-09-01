package polls

import (
	"context"
	"encoding/json"
	"errors"
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

// createPoll accepts a keyboard, so the documented 100-button cap applies here
// too rather than only on the ordinary send paths.
func TestCreateEnforcesKeyboardLimit(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"message":{"message_id":1}}`),
	}}
	svc := pollService(doer, false)

	rows := make([][]ym.InlineSuggestButton, 0, 21)
	for range 21 {
		rows = append(rows, make([]ym.InlineSuggestButton, 5))
	}

	_, err := svc.Create(context.Background(), &CreatePollRequest{
		ChatID:         ym.Ptr(ym.ChatID("c1")),
		Title:          "Lunch?",
		Answers:        []string{"yes", "no"},
		SuggestButtons: &ym.SuggestButtons{Buttons: rows},
	})

	var limitErr *ym.LimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("expected a *ym.LimitError, got %T (%v)", err, err)
	}
	if doer.CallCount() != 0 {
		t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
	}
}

func TestGetVotersPageValidatesLimit(t *testing.T) {
	for _, limit := range []int{0, -1, ym.MaxPageLimit + 1} {
		doer := &testutil.FakeDoer{Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":true,"answer_id":0}`),
		}}
		svc := pollService(doer, false)

		_, err := svc.GetVotersPage(context.Background(), PollVotersParams{
			ChatID:    ym.Ptr(ym.ChatID("c1")),
			MessageID: 1,
			AnswerID:  1,
			Limit:     ym.Ptr(limit),
		})

		var limitErr *ym.LimitError
		if !errors.As(err, &limitErr) {
			t.Fatalf("limit %d: expected a *ym.LimitError, got %T (%v)", limit, err, err)
		}
		if doer.CallCount() != 0 {
			t.Fatalf("limit %d: invalid input must not reach the network", limit)
		}
	}
}

// The API numbers answers from zero, so the first option is answer_id 0.
// Treating zero as "missing" made its voters unreachable.
func TestGetVotersPageAcceptsTheFirstAnswer(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"answer_id":0,"voted_count":2,"votes":[]}`),
	}}
	svc := pollService(doer, false)

	page, err := svc.GetVotersPage(context.Background(), PollVotersParams{
		ChatID:    ym.Ptr(ym.ChatID("c1")),
		MessageID: 1,
		AnswerID:  0,
	})
	if err != nil {
		t.Fatalf("expected nil error for the first answer, got %v", err)
	}
	if page.VotedCount != 2 {
		t.Fatalf("unexpected page: %#v", page)
	}
	if got := doer.Requests[0].URL.Query().Get("answer_id"); got != "0" {
		t.Fatalf("answer_id: got %q, want 0", got)
	}
}

func TestGetVotersPageRejectsNegativeAnswer(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true}`),
	}}
	svc := pollService(doer, false)

	_, err := svc.GetVotersPage(context.Background(), PollVotersParams{
		ChatID:    ym.Ptr(ym.ChatID("c1")),
		MessageID: 1,
		AnswerID:  -1,
	})
	if err == nil {
		t.Fatal("expected a negative answer index to be rejected")
	}
	if doer.CallCount() != 0 {
		t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
	}
}

// A pointer to an empty string is not a usable key. omitempty looks at the
// pointer, not the value, so it would be serialised as "payload_id":"" and the
// retry protection would be silently absent — the send paths treat an empty
// string as unset, and this one should agree.
func TestCreateTreatsAnEmptyPayloadIDAsUnset(t *testing.T) {
	doer := &testutil.FakeDoer{Responses: []*http.Response{
		testutil.NewResponse(http.StatusOK, `{"ok":true,"message":{"message_id":1}}`),
	}}
	svc := pollService(doer, false)

	req := &CreatePollRequest{
		ChatID:    ym.Ptr(ym.ChatID("c1")),
		Title:     "Lunch?",
		Answers:   []string{"yes", "no"},
		PayloadID: ym.Ptr(""),
	}
	if _, err := svc.Create(context.Background(), req); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got := sentPayloadID(t, doer.Requests[0]); got == "" {
		t.Fatal("an empty payload_id was sent instead of a generated key")
	}
	// The caller's request must not be mutated.
	if req.PayloadID == nil || *req.PayloadID != "" {
		t.Fatalf("Create mutated the caller's request: %#v", req.PayloadID)
	}
}

func TestCreateReportsAnswerBoundsAsLimitError(t *testing.T) {
	cases := []struct {
		name    string
		answers []string
	}{
		{"too few", []string{"only"}},
		{"too many", make([]string, ym.MaxPollAnswers+1)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doer := &testutil.FakeDoer{Responses: []*http.Response{
				testutil.NewResponse(http.StatusOK, `{"ok":true,"message":{"message_id":1}}`),
			}}
			svc := pollService(doer, false)

			_, err := svc.Create(context.Background(), &CreatePollRequest{
				ChatID:  ym.Ptr(ym.ChatID("c1")),
				Title:   "Lunch?",
				Answers: tc.answers,
			})

			var limitErr *ym.LimitError
			if !errors.As(err, &limitErr) {
				t.Fatalf("expected a *ym.LimitError, got %T (%v)", err, err)
			}
			if limitErr.Field != "answers" {
				t.Fatalf("expected the answers field to be named, got %q", limitErr.Field)
			}
			if doer.CallCount() != 0 {
				t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
			}
		})
	}
}
