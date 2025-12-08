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

type Service struct {
	client *ym.Client
}

func NewService(client *ym.Client) *Service {
	return &Service{client: client}
}

type CreatePollRequest struct {
	ChatID                *ym.ChatID    `json:"chat_id,omitempty"`
	Login                 *ym.UserLogin `json:"login,omitempty"`
	Title                 string        `json:"title"`
	Answers               []string      `json:"answers"`
	MaxChoices            *int          `json:"max_choices,omitempty"`
	IsAnonymous           *bool         `json:"is_anonymous,omitempty"`
	PayloadID             *string       `json:"payload_id,omitempty"`
	ReplyMessageID        *ym.MessageID `json:"reply_message_id,omitempty"`
	DisableNotification   *bool         `json:"disable_notification,omitempty"`
	Important             *bool         `json:"important,omitempty"`
	DisableWebPagePreview *bool         `json:"disable_web_page_preview,omitempty"`
	ThreadID              *ym.ThreadID  `json:"thread_id,omitempty"`
}

func (s *Service) Create(ctx context.Context, req *CreatePollRequest) (*ym.Message, error) {
	if err := validateRecipient(req.ChatID, req.Login); err != nil {
		return nil, err
	}
	if req.Title == "" || len(req.Answers) < 2 || len(req.Answers) > 100 {
		return nil, errors.New("title required and answers must be between 2 and 100")
	}
	if req.MaxChoices != nil && *req.MaxChoices <= 0 {
		return nil, errors.New("max_choices must be > 0")
	}

	resp, err := s.client.DoRequest(ctx, http.MethodPost, "/bot/v1/messages/createPoll/", req)
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
			Endpoint:    "/bot/v1/messages/createPoll/",
		}
	}

	return parsed.Message, nil
}

type PollResultsParams struct {
	ChatID     *ym.ChatID
	Login      *ym.UserLogin
	MessageID  ym.MessageID
	InviteHash *string
	ThreadID   *ym.ThreadID
}

func (s *Service) GetResults(ctx context.Context, params PollResultsParams) (*ym.PollResult, error) {
	if err := validateRecipient(params.ChatID, params.Login); err != nil {
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

	path := "/bot/v1/polls/getResults/?" + q.Encode()
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
			Endpoint:    "/bot/v1/polls/getResults/",
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

func (s *Service) GetVotersPage(ctx context.Context, params PollVotersParams) (*ym.PollVotersPage, error) {
	if err := validateRecipient(params.ChatID, params.Login); err != nil {
		return nil, err
	}
	if params.MessageID == 0 || params.AnswerID == 0 {
		return nil, errors.New("message_id and answer_id are required")
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

	path := "/bot/v1/polls/getVoters/?" + q.Encode()
	resp, err := s.client.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed struct {
		OK          bool      `json:"ok"`
		AnswerID    int       `json:"answer_id"`
		VotedCount  int       `json:"voted_count"`
		Cursor      int64     `json:"cursor"`
		Votes       []ym.Vote `json:"votes"`
		Description string    `json:"description"`
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
			Endpoint:    "/bot/v1/polls/getVoters/",
		}
	}

	return &ym.PollVotersPage{
		AnswerID:   parsed.AnswerID,
		VotedCount: parsed.VotedCount,
		Cursor:     parsed.Cursor,
		Votes:      parsed.Votes,
	}, nil
}

func (s *Service) GetAllVoters(ctx context.Context, params PollVotersParams) ([]ym.Vote, error) {
	var all []ym.Vote
	for {
		page, err := s.GetVotersPage(ctx, params)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Votes...)
		if len(page.Votes) == 0 || page.Cursor <= 0 {
			break
		}
		params.Cursor = &page.Cursor
	}

	return all, nil
}

func validateRecipient(chatID *ym.ChatID, login *ym.UserLogin) error {
	if (chatID == nil || *chatID == "") && (login == nil || *login == "") {
		return errors.New("either chat_id or login is required")
	}
	if chatID != nil && *chatID != "" && login != nil && *login != "" {
		return errors.New("only one of chat_id or login must be set")
	}

	return nil
}
