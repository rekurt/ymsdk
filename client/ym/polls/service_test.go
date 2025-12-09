package polls

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func TestCreatePollValidation(t *testing.T) {
	svc := NewService(nil)
	_, err := svc.Create(context.Background(), &CreatePollRequest{Title: "t", Answers: []string{"a"}})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestCreatePollSuccess(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":true,"message":{"message_id":1,"chat":{"id":"c1","type":"private"},"from":{"login":"u1"},"text":"poll"}}`),
		},
	}
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, doer)
	svc := NewService(client)
	msg, err := svc.Create(context.Background(), &CreatePollRequest{
		ChatID:  ptrChat("c1"),
		Title:   "title",
		Answers: []string{"a", "b"},
	})
	if err != nil || msg == nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetResultsError(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":false,"description":"not found"}`),
		},
	}
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, doer)
	svc := NewService(client)
	_, err := svc.GetResults(context.Background(), PollResultsParams{
		ChatID:    ptrChat("c1"),
		MessageID: 1,
	})
	var apiErr *ymerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected api error")
	}
}

func ptrChat(id ym.ChatID) *ym.ChatID {
	return &id
}
