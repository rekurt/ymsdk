package chats

import (
	"bytes"
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

// ErrChatIDRequired is returned when a chat-scoped query omits the chat.
var ErrChatIDRequired = errors.New("yandex-messenger: chat_id is required")

// ListParams holds optional parameters for listing chats.
type ListParams struct {
	// Limit caps the page size. Defaults to 100 server-side, maximum 1000.
	Limit *int
	// Offset continues pagination: pass the ID of the last chat from the
	// previous page.
	Offset ym.ChatID
}

// MembersParams holds parameters for listing the members of a chat.
type MembersParams struct {
	// ChatID is required.
	ChatID ym.ChatID
	// Role filters by participant role. Empty returns every role.
	Role ym.ChatMemberRole
	// Limit caps the page size, 1 to 1000. Defaults to 100 server-side.
	Limit *int
	// Offset continues pagination: pass the GUID of the last member from the
	// previous page.
	Offset string
}

// List returns one page of the chats and channels the bot belongs to.
//
// Pass the ID of the last chat returned as the next page's Offset, or use
// [Service.ListAll] to walk every page.
func (s *Service) List(ctx context.Context, params ListParams) ([]ym.ChatMetaData, error) {
	query := url.Values{}
	if err := setLimit(query, params.Limit); err != nil {
		return nil, err
	}
	if params.Offset != "" {
		query.Set("offset", string(params.Offset))
	}

	var out []ym.ChatMetaData
	if err := s.getData(ctx, ym.EndpointChatsGet, query, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// ListAll walks every page of the chat list and returns the combined result.
func (s *Service) ListAll(ctx context.Context, params ListParams) ([]ym.ChatMetaData, error) {
	var all []ym.ChatMetaData
	for {
		page, err := s.List(ctx, params)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}

		// Check the cursor before keeping the page: a server that repeats it has
		// sent the previous page again, so appending first would return those
		// items twice.
		next := page[len(page)-1].ID
		if next == params.Offset {
			break
		}

		all = append(all, page...)
		params.Offset = next
	}

	return all, nil
}

// GetChat returns detailed information about a chat or channel, including the
// reactions the chat permits.
func (s *Service) GetChat(ctx context.Context, chatID ym.ChatID) (*ym.ChatInfo, error) {
	if chatID == "" {
		return nil, ErrChatIDRequired
	}

	query := url.Values{"chat_id": {string(chatID)}}

	var out ym.ChatInfo
	if err := s.getRequiredData(ctx, ym.EndpointChatsGetChat, query, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// GetMembers returns one page of a chat's participants.
//
// Pass the GUID of the last member returned as the next page's Offset, or use
// [Service.GetAllMembers] to walk every page.
func (s *Service) GetMembers(ctx context.Context, params MembersParams) ([]ym.ChatMember, error) {
	if params.ChatID == "" {
		return nil, ErrChatIDRequired
	}

	query := url.Values{"chat_id": {string(params.ChatID)}}
	if err := setLimit(query, params.Limit); err != nil {
		return nil, err
	}
	if params.Role != "" {
		query.Set("role", string(params.Role))
	}
	if params.Offset != "" {
		query.Set("offset", params.Offset)
	}

	var out []ym.ChatMember
	if err := s.getData(ctx, ym.EndpointChatsGetMembers, query, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// GetAllMembers walks every page of a chat's member list.
func (s *Service) GetAllMembers(ctx context.Context, params MembersParams) ([]ym.ChatMember, error) {
	var all []ym.ChatMember
	for {
		page, err := s.GetMembers(ctx, params)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}

		// Check the cursor before keeping the page: a server that repeats it has
		// sent the previous page again, so appending first would return those
		// items twice.
		next := page[len(page)-1].GUID
		if next == params.Offset {
			break
		}

		all = append(all, page...)
		params.Offset = next
	}

	return all, nil
}

func setLimit(query url.Values, limit *int) error {
	if limit == nil {
		return nil
	}
	if err := ym.ValidatePageLimit(*limit); err != nil {
		return err
	}
	query.Set("limit", strconv.Itoa(*limit))

	return nil
}

// getData performs a GET whose response wraps the payload in a "data" field.
func (s *Service) getData(ctx context.Context, path string, query url.Values, out any) error {
	return s.get(ctx, path, query, out, false)
}

// getRequiredData is getData for the single-object queries, where an absent
// payload is malformed rather than empty.
func (s *Service) getRequiredData(ctx context.Context, path string, query url.Values, out any) error {
	return s.get(ctx, path, query, out, true)
}

func (s *Service) get(ctx context.Context, path string, query url.Values, out any, required bool) error {
	full := path
	if encoded := query.Encode(); encoded != "" {
		full += "?" + encoded
	}

	resp, err := s.client.DoRequest(ctx, http.MethodGet, full, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var parsed struct {
		OK          bool            `json:"ok"`
		Data        json.RawMessage `json:"data"`
		Description string          `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("%w: decode %s response: %w", ymerrors.ErrInvalidResponse, path, err)
	}
	if !parsed.OK {
		return &ymerrors.APIError{
			Kind:        ymerrors.KindBadRequest,
			HTTPStatus:  resp.StatusCode,
			Description: parsed.Description,
			Method:      http.MethodGet,
			Endpoint:    path,
		}
	}
	// A JSON null is four bytes, so a length check alone would let it through
	// and Unmarshal would quietly leave the destination zero-valued.
	if len(parsed.Data) == 0 || bytes.Equal(bytes.TrimSpace(parsed.Data), []byte("null")) {
		if required {
			return fmt.Errorf("%w: %s returned no data", ymerrors.ErrInvalidResponse, path)
		}

		// A list endpoint answering without data is simply empty.
		return nil
	}
	if err := json.Unmarshal(parsed.Data, out); err != nil {
		return fmt.Errorf("%w: decode %s data: %w", ymerrors.ErrInvalidResponse, path, err)
	}

	return nil
}
