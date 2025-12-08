package updates

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

type Service struct {
	client *ym.Client
}

func NewService(client *ym.Client) *Service {
	return &Service{client: client}
}

type getUpdatesResponse struct {
	OK         bool        `json:"ok"`
	Updates    []ym.Update `json:"updates"`
	NextOffset int64       `json:"next_offset"`
}

type GetUpdatesParams struct {
	Limit  *int
	Offset *int64
}

func (s *Service) Get(ctx context.Context, limit int, offset string) ([]ym.Update, string, error) {
	path := "/bot/v1/messages/getUpdates"
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	if offset != "" {
		query.Set("offset", offset)
	}
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	resp, err := s.client.DoRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	var parsed getUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, "", fmt.Errorf("%w: decode getUpdates response: %w", ymerrors.ErrInvalidResponse, err)
	}
	if !parsed.OK {
		return nil, "", fmt.Errorf("%w: ok=false", ymerrors.ErrInvalidResponse)
	}

	return parsed.Updates, strconv.FormatInt(parsed.NextOffset, 10), nil
}

func (s *Service) GetUpdates(ctx context.Context, params GetUpdatesParams) ([]ym.Update, int64, error) {
	limit := 0
	if params.Limit != nil {
		limit = *params.Limit
	}
	offsetStr := ""
	if params.Offset != nil {
		offsetStr = strconv.FormatInt(*params.Offset, 10)
	}
	updates, next, err := s.Get(ctx, limit, offsetStr)
	if err != nil {
		return nil, 0, err
	}
	var nextOffset int64
	if next != "" {
		if v, err := strconv.ParseInt(next, 10, 64); err == nil {
			nextOffset = v
		}
	}
	if nextOffset == 0 {
		nextOffset = calculateNextOffset(updates, params.Offset)
	}

	return updates, nextOffset, nil
}

func calculateNextOffset(updates []ym.Update, current *int64) int64 {
	var maxID int64
	if current != nil {
		maxID = *current
	}
	for _, u := range updates {
		if u.UpdateID >= maxID {
			maxID = u.UpdateID + 1
		}
	}

	return maxID
}

func (s *Service) PollLoop(
	ctx context.Context, params GetUpdatesParams, handler func(context.Context, ym.Update) error,
) error {
	offset := params.Offset
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		upds, nextOffset, err := s.GetUpdates(ctx, GetUpdatesParams{Limit: params.Limit, Offset: offset})
		if err != nil {
			return err
		}
		for _, u := range upds {
			if err := handler(ctx, u); err != nil {
				return err
			}
		}
		offset = &nextOffset
		time.Sleep(time.Second)
	}
}
