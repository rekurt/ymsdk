package chats

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func TestCreateValidation(t *testing.T) {
	err := validateCreate(&ChatCreateRequest{Name: "", Description: "d"})
	if err == nil {
		t.Fatalf("expected validation error")
	}

	err = validateCreate(&ChatCreateRequest{Name: "n", Channel: true, Members: []ym.UserRef{{Login: "u"}}})
	if err == nil {
		t.Fatalf("expected members invalid for channel")
	}
}

func TestCreateSuccess(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":true,"chat_id":"c1"}`),
		},
	}
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, doer)

	svc := NewService(client)
	chat, err := svc.Create(context.Background(), &ChatCreateRequest{Name: "name", Description: "d"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chat == nil || chat.ID == "" {
		t.Fatalf("chat not filled")
	}
}

func TestUpdateMembersError(t *testing.T) {
	doer := &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":false,"description":"denied"}`),
		},
	}
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, doer)
	svc := NewService(client)
	err := svc.UpdateMembers(context.Background(), &ChatUpdateMembersRequest{
		ChatID:  "c1",
		Members: []ym.UserRef{{Login: "u1"}},
	})
	var apiErr *ymerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected api error")
	}
}
