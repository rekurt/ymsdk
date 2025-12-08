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

func TestUpdateWebhookSuccess(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":true,"id":"bot1","display_name":"Bot","webhook_url":"https://example.com","organizations":[1],"login":"bot"}`),
		},
	})
	svc := NewService(client)
	webhook := "https://example.com"
	self, err := svc.Update(context.Background(), &SelfUpdateRequest{WebhookURL: &webhook})
	if err != nil || self == nil || self.WebhookURL == nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateWebhookAPIError(t *testing.T) {
	client := ym.NewClientWithHTTP(ym.Config{
		BaseURL: "http://example.com",
		ErrorHandling: ymerrors.ErrorHandlingConfig{
			RetryStrategy: ymerrors.RetryStrategy{MaxAttempts: 1},
		},
	}, &testutil.FakeDoer{
		Responses: []*http.Response{
			testutil.NewResponse(http.StatusOK, `{"ok":false,"description":"denied"}`),
		},
	})
	svc := NewService(client)
	_, err := svc.Update(context.Background(), &SelfUpdateRequest{})
	var apiErr *ymerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected api error")
	}
}
