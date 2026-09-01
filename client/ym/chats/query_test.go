package chats

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/internal/testutil"
)

func serviceWith(t *testing.T, bodies ...string) (*Service, *testutil.FakeDoer) {
	t.Helper()

	responses := make([]*http.Response, 0, len(bodies))
	for _, b := range bodies {
		responses = append(responses, testutil.NewResponse(http.StatusOK, b))
	}
	doer := &testutil.FakeDoer{Responses: responses}

	return NewService(ym.NewClientWithHTTP(ym.Config{BaseURL: "http://example.com"}, doer)), doer
}

func TestList(t *testing.T) {
	svc, doer := serviceWith(t, `{"ok":true,"data":[
		{"type":"group","id":"c1","title":"Work","description":"d"},
		{"type":"private","id":"c2","username":"vasya@example.org"}]}`)

	chats, err := svc.List(context.Background(), ListParams{Limit: ym.Ptr(50)})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(chats) != 2 {
		t.Fatalf("expected 2 chats, got %d", len(chats))
	}
	if chats[0].Title != "Work" || chats[1].Username != "vasya@example.org" {
		t.Fatalf("unexpected chats: %#v", chats)
	}

	req := doer.Requests[0]
	if req.Method != http.MethodGet {
		t.Fatalf("expected GET, got %s", req.Method)
	}
	if req.URL.Path != ym.EndpointChatsGet {
		t.Fatalf("unexpected path: %s", req.URL.Path)
	}
	if got := req.URL.Query().Get("limit"); got != "50" {
		t.Fatalf("limit: got %q", got)
	}
	// A GET carries no body, so it must not advertise one.
	if ct := req.Header.Get("Content-Type"); ct != "" {
		t.Fatalf("expected no Content-Type on a GET, got %q", ct)
	}
}

func TestGetChat(t *testing.T) {
	svc, doer := serviceWith(t, `{"ok":true,"data":{"type":"group","id":"c1","name":"Work",
		"invite_link":"https://example/invite",
		"available_reactions":[{"type":"default_reaction","name":"like"}]}}`)

	info, err := svc.GetChat(context.Background(), "c1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if info.Name != "Work" || info.InviteLink != "https://example/invite" {
		t.Fatalf("unexpected chat info: %#v", info)
	}
	if len(info.AvailableReactions) != 1 || info.AvailableReactions[0].Name != "like" {
		t.Fatalf("unexpected reactions: %#v", info.AvailableReactions)
	}
	if got := doer.Requests[0].URL.Query().Get("chat_id"); got != "c1" {
		t.Fatalf("chat_id: got %q", got)
	}
}

func TestGetMembers(t *testing.T) {
	svc, doer := serviceWith(t, `{"ok":true,"data":[
		{"guid":"g1","login":"anya@example.org","role":"admin","is_bot":false},
		{"guid":"g2","login":"vasya@example.org","role":"member","is_bot":false}]}`)

	members, err := svc.GetMembers(context.Background(), MembersParams{
		ChatID: "c1",
		Role:   ym.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(members) != 2 || members[0].Role != ym.RoleAdmin || members[0].GUID != "g1" {
		t.Fatalf("unexpected members: %#v", members)
	}
	q := doer.Requests[0].URL.Query()
	if q.Get("chat_id") != "c1" || q.Get("role") != "admin" {
		t.Fatalf("unexpected query: %s", doer.Requests[0].URL.RawQuery)
	}
}

func TestPagination(t *testing.T) {
	t.Run("ListAll follows the cursor and stops", func(t *testing.T) {
		svc, doer := serviceWith(t,
			`{"ok":true,"data":[{"type":"group","id":"c1"},{"type":"group","id":"c2"}]}`,
			`{"ok":true,"data":[{"type":"group","id":"c3"}]}`,
			`{"ok":true,"data":[]}`,
		)

		all, err := svc.ListAll(context.Background(), ListParams{})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(all) != 3 {
			t.Fatalf("expected 3 chats, got %d", len(all))
		}
		if got := doer.Requests[1].URL.Query().Get("offset"); got != "c2" {
			t.Fatalf("second page offset: got %q, want c2", got)
		}
	})

	// A server that keeps returning the same last item must not spin forever.
	t.Run("ListAll stops on a repeated cursor", func(t *testing.T) {
		svc, _ := serviceWith(t,
			`{"ok":true,"data":[{"type":"group","id":"c1"}]}`,
			`{"ok":true,"data":[{"type":"group","id":"c1"}]}`,
		)

		all, err := svc.ListAll(context.Background(), ListParams{})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(all) != 2 {
			t.Fatalf("expected the walk to stop after the repeat, got %d", len(all))
		}
	})

	t.Run("GetAllMembers follows the guid cursor", func(t *testing.T) {
		svc, doer := serviceWith(t,
			`{"ok":true,"data":[{"guid":"g1","role":"member","is_bot":false}]}`,
			`{"ok":true,"data":[]}`,
		)

		all, err := svc.GetAllMembers(context.Background(), MembersParams{ChatID: "c1"})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(all) != 1 {
			t.Fatalf("expected 1 member, got %d", len(all))
		}
		if got := doer.Requests[1].URL.Query().Get("offset"); got != "g1" {
			t.Fatalf("second page offset: got %q, want g1", got)
		}
	})
}

func TestQueryValidation(t *testing.T) {
	cases := []struct {
		name string
		call func(*Service) error
		want error
	}{
		{
			name: "GetChat without an id",
			call: func(s *Service) error { _, err := s.GetChat(context.Background(), ""); return err },
			want: ErrChatIDRequired,
		},
		{
			name: "GetMembers without an id",
			call: func(s *Service) error {
				_, err := s.GetMembers(context.Background(), MembersParams{})

				return err
			},
			want: ErrChatIDRequired,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, doer := serviceWith(t, `{"ok":true}`)
			if err := tc.call(svc); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if doer.CallCount() != 0 {
				t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
			}
		})
	}

	t.Run("limit out of range", func(t *testing.T) {
		svc, doer := serviceWith(t, `{"ok":true}`)
		if _, err := svc.List(context.Background(), ListParams{Limit: ym.Ptr(1001)}); err == nil {
			t.Fatal("expected a range error")
		}
		if doer.CallCount() != 0 {
			t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
		}
	})
}

// The reference documents MaxChatNameLength and MaxChatDescriptionLength as
// checked locally, and promises a *ym.LimitError for every documented
// violation. Declaring the constants without wiring them up made both claims
// false and left the caller with a deterministic API rejection instead.
func TestCreateEnforcesChatTextLimits(t *testing.T) {
	cases := []struct {
		name  string
		req   *ChatCreateRequest
		field string
	}{
		{
			name:  "name over the limit",
			req:   &ChatCreateRequest{Name: strings.Repeat("a", ym.MaxChatNameLength+1)},
			field: "name",
		},
		{
			name: "description over the limit",
			req: &ChatCreateRequest{
				Name:        "ok",
				Description: strings.Repeat("a", ym.MaxChatDescriptionLength+1),
			},
			field: "description",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, doer := serviceWith(t, `{"ok":true,"chat_id":"c1"}`)

			_, err := svc.Create(context.Background(), tc.req)

			var limitErr *ym.LimitError
			if !errors.As(err, &limitErr) {
				t.Fatalf("expected a *ym.LimitError, got %T (%v)", err, err)
			}
			if limitErr.Field != tc.field {
				t.Fatalf("expected the %q field, got %q", tc.field, limitErr.Field)
			}
			if doer.CallCount() != 0 {
				t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
			}
		})
	}

	t.Run("values at the limit are accepted", func(t *testing.T) {
		svc, doer := serviceWith(t, `{"ok":true,"chat_id":"c1"}`)

		_, err := svc.Create(context.Background(), &ChatCreateRequest{
			Name:        strings.Repeat("a", ym.MaxChatNameLength),
			Description: strings.Repeat("b", ym.MaxChatDescriptionLength),
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if doer.CallCount() != 1 {
			t.Fatalf("expected the request to be sent, got %d calls", doer.CallCount())
		}
	})
}

// UpdateMembers enforced its documented caps with a single opaque error that
// named neither the offending list nor the limit, and could not be matched with
// errors.As like every other limit violation.
func TestUpdateMembersReportsLimitError(t *testing.T) {
	svc, doer := serviceWith(t, `{"ok":true}`)

	members := make([]ym.UserRef, ym.MaxChatMembers+1)
	for i := range members {
		members[i] = ym.UserRef{Login: ym.UserLogin(strings.Repeat("a", i%5+1) + string(rune('a'+i%26)))}
	}

	err := svc.UpdateMembers(context.Background(), &ChatUpdateMembersRequest{
		ChatID:  "c1",
		Members: members,
	})

	var limitErr *ym.LimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("expected a *ym.LimitError, got %T (%v)", err, err)
	}
	if limitErr.Field != "members" {
		t.Fatalf("expected the offending list to be named, got %q", limitErr.Field)
	}
	if doer.CallCount() != 0 {
		t.Fatalf("invalid input must not reach the network, got %d calls", doer.CallCount())
	}
}
