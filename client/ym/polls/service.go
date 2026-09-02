package polls

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

// Service provides methods for creating and querying polls in Yandex Messenger.
type Service struct {
	client *ym.Client
}

// NewService creates a new polls Service.
func NewService(client *ym.Client) *Service {
	return &Service{client: client}
}

// CreatePollRequest contains parameters for creating a new poll.
// Exactly one of ChatID or Login must be set.
type CreatePollRequest struct {
	ChatID                *ym.ChatID         `json:"chat_id,omitempty"`
	Login                 *ym.UserLogin      `json:"login,omitempty"`
	Title                 string             `json:"title"`
	Answers               []string           `json:"answers"`
	MaxChoices            *int               `json:"max_choices,omitempty"`
	IsAnonymous           *bool              `json:"is_anonymous,omitempty"`
	PayloadID             *string            `json:"payload_id,omitempty"`
	ReplyMessageID        *ym.MessageID      `json:"reply_message_id,omitempty"`
	DisableNotification   *bool              `json:"disable_notification,omitempty"`
	Important             *bool              `json:"important,omitempty"`
	DisableWebPagePreview *bool              `json:"disable_web_page_preview,omitempty"`
	ThreadID              *ym.ThreadID       `json:"thread_id,omitempty"`
	SuggestButtons        *ym.SuggestButtons `json:"suggest_buttons,omitempty"`
}

// Create sends a new poll to a chat or user.
func (s *Service) Create(ctx context.Context, req *CreatePollRequest) (*ym.Message, error) {
	if err := ym.ValidateTarget(targetFromPointers(req.ChatID, req.Login)); err != nil {
		return nil, err
	}
	if req.Title == "" {
		return nil, errors.New("poll title is required")
	}
	// Reported as a limit so callers can match it like every other documented
	// bound, and separately from the title so the message says which is wrong.
	answerErr := ym.ValidateRange("answers", len(req.Answers), ym.MinPollAnswers, ym.MaxPollAnswers)
	if answerErr != nil {
		return nil, answerErr
	}
	if req.MaxChoices != nil && *req.MaxChoices <= 0 {
		return nil, errors.New("max_choices must be > 0")
	}
	// createPoll accepts a keyboard, so the documented cap applies here too.
	if err := ym.ValidateSuggestButtons(req.SuggestButtons); err != nil {
		return nil, err
	}

	// createPoll documents payload_id, so a retried create collapses into one
	// poll instead of two. Copy the request rather than mutating the caller's.
	//
	// An empty string counts as unset, matching the send paths: omitempty looks
	// at the pointer rather than the value, so a pointer to "" would be sent as
	// an empty key and leave the retry unprotected.
	body := *req
	if (body.PayloadID == nil || *body.PayloadID == "") && s.client.AutoPayloadID() {
		body.PayloadID = ym.Ptr(ym.NewPayloadID())
	}

	resp, err := s.client.DoRequest(ctx, http.MethodPost, ym.EndpointMessagesCreatePoll, &body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed struct {
		OK      bool        `json:"ok"`
		Message *ym.Message `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("%w: decode create poll response: %w", ymerrors.ErrInvalidResponse, err)
	}
	if !parsed.OK || parsed.Message == nil {
		return nil, &ymerrors.APIError{
			Kind:        ymerrors.KindBadRequest,
			HTTPStatus:  resp.StatusCode,
			Description: "create poll failed",
			Method:      http.MethodPost,
			Endpoint:    ym.EndpointMessagesCreatePoll,
		}
	}

	return parsed.Message, nil
}

// PollResultsParams contains parameters for fetching poll results.
type PollResultsParams struct {
	ChatID     *ym.ChatID
	Login      *ym.UserLogin
	MessageID  ym.MessageID
	InviteHash *string
	ThreadID   *ym.ThreadID
}

// GetResults returns aggregated voting results for a poll.
func (s *Service) GetResults(ctx context.Context, params PollResultsParams) (*ym.PollResult, error) {
	if err := ym.ValidateTarget(targetFromPointers(params.ChatID, params.Login)); err != nil {
		return nil, err
	}
	if params.MessageID == 0 {
		return nil, errors.New("message_id is required")
	}

	q := url.Values{}
	if params.ChatID != nil {
		q.Set("chat_id", string(*params.ChatID))
	}
	if params.Login != nil {
		q.Set("login", string(*params.Login))
	}
	q.Set("message_id", strconv.FormatInt(int64(params.MessageID), 10))
	if params.InviteHash != nil {
		q.Set("invite_hash", *params.InviteHash)
	}
	if params.ThreadID != nil {
		q.Set("thread_id", strconv.FormatInt(int64(*params.ThreadID), 10))
	}

	path := ym.EndpointPollsGetResults + "?" + q.Encode()
	resp, err := s.client.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed struct {
		OK          bool           `json:"ok"`
		VotedCount  int            `json:"voted_count"`
		Answers     map[string]int `json:"answers"`
		Description string         `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("%w: decode getResults response: %w", ymerrors.ErrInvalidResponse, err)
	}
	if !parsed.OK {
		return nil, &ymerrors.APIError{
			Kind:        ymerrors.KindBadRequest,
			HTTPStatus:  resp.StatusCode,
			Description: parsed.Description,
			Method:      http.MethodGet,
			Endpoint:    ym.EndpointPollsGetResults,
		}
	}
	answerMap := make(map[int]int, len(parsed.Answers))
	for k, v := range parsed.Answers {
		if id, err := strconv.Atoi(k); err == nil {
			answerMap[id] = v
		}
	}

	return &ym.PollResult{
		VotedCount: parsed.VotedCount,
		Answers:    answerMap,
	}, nil
}

// PollVotersParams contains parameters for fetching individual voters of a poll answer.
type PollVotersParams struct {
	ChatID     *ym.ChatID
	Login      *ym.UserLogin
	MessageID  ym.MessageID
	InviteHash *string
	AnswerID   int
	Limit      *int
	Cursor     *int64
	ThreadID   *ym.ThreadID
}

// GetVotersPage returns a single page of voters for a given poll answer.
func (s *Service) GetVotersPage(ctx context.Context, params PollVotersParams) (*ym.PollVotersPage, error) {
	if err := ym.ValidateTarget(targetFromPointers(params.ChatID, params.Login)); err != nil {
		return nil, err
	}
	if params.MessageID == 0 {
		return nil, errors.New("message_id is required")
	}
	// The API numbers answers from zero, so answer 0 is the first option rather
	// than a missing value. Only a negative index is meaningless.
	if params.AnswerID < 0 {
		return nil, fmt.Errorf("yandex-messenger: answer_id %d must not be negative", params.AnswerID)
	}
	if params.Limit != nil {
		if err := ym.ValidatePageLimit(*params.Limit); err != nil {
			return nil, err
		}
	}

	q := url.Values{}
	if params.ChatID != nil {
		q.Set("chat_id", string(*params.ChatID))
	}
	if params.Login != nil {
		q.Set("login", string(*params.Login))
	}
	q.Set("message_id", strconv.FormatInt(int64(params.MessageID), 10))
	q.Set("answer_id", strconv.Itoa(params.AnswerID))
	if params.InviteHash != nil {
		q.Set("invite_hash", *params.InviteHash)
	}
	if params.Limit != nil {
		q.Set("limit", strconv.Itoa(*params.Limit))
	}
	if params.Cursor != nil {
		q.Set("cursor", strconv.FormatInt(*params.Cursor, 10))
	}
	if params.ThreadID != nil {
		q.Set("thread_id", strconv.FormatInt(int64(*params.ThreadID), 10))
	}

	path := ym.EndpointPollsGetVoters + "?" + q.Encode()
	resp, err := s.client.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed struct {
		OK          bool          `json:"ok"`
		AnswerID    int           `json:"answer_id"`
		VotedCount  int           `json:"voted_count"`
		Cursor      ym.PollCursor `json:"cursor"`
		Votes       []ym.Vote     `json:"votes"`
		Description string        `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("%w: decode getVoters response: %w", ymerrors.ErrInvalidResponse, err)
	}
	if !parsed.OK {
		return nil, &ymerrors.APIError{
			Kind:        ymerrors.KindBadRequest,
			HTTPStatus:  resp.StatusCode,
			Description: parsed.Description,
			Method:      http.MethodGet,
			Endpoint:    ym.EndpointPollsGetVoters,
		}
	}

	return &ym.PollVotersPage{
		AnswerID:   parsed.AnswerID,
		VotedCount: parsed.VotedCount,
		Cursor:     parsed.Cursor,
		Votes:      parsed.Votes,
	}, nil
}

// GetAllVoters iterates through all pages and returns every voter for a poll answer.
func (s *Service) GetAllVoters(ctx context.Context, params PollVotersParams) ([]ym.Vote, error) {
	var all []ym.Vote
	for {
		page, err := s.GetVotersPage(ctx, params)
		if err != nil {
			return nil, err
		}
		if len(page.Votes) == 0 || page.Cursor.Next <= 0 {
			all = append(all, page.Votes...)

			break
		}

		// A cursor that does not advance means the server sent the same page
		// again: keeping it would duplicate those votes and the walk would
		// never end.
		if params.Cursor != nil && page.Cursor.Next == *params.Cursor {
			break
		}

		all = append(all, page.Votes...)
		next := page.Cursor.Next
		params.Cursor = &next
	}

	return all, nil
}

// targetFromPointers adapts the pointer-based recipient fields of the request
// structs to a [ym.Target].
func targetFromPointers(chatID *ym.ChatID, login *ym.UserLogin) ym.Target {
	var t ym.Target
	if chatID != nil {
		t.ChatID = *chatID
	}
	if login != nil {
		t.Login = *login
	}

	return t
}
