package messages

import (
	"context"

	"github.com/rekurt/ymsdk/client/ym"
)

// ReactionOptions holds optional parameters for setting a reaction.
type ReactionOptions struct {
	// ThreadID scopes the operation to a thread.
	ThreadID *ym.ThreadID
}

type sendReactionRequest struct {
	ym.Target
	MessageID ym.MessageID `json:"message_id"`
	ThreadID  *ym.ThreadID `json:"thread_id,omitempty"`
	Reaction  *ym.Reaction `json:"reaction,omitempty"`
}

// GetReactionsOptions holds optional parameters for reading reactions.
type GetReactionsOptions struct {
	// ThreadID scopes the query to a thread.
	ThreadID *ym.ThreadID
	// Limit caps the number of reactions returned. Public chats only.
	// Defaults to 100 server-side, maximum 1000.
	Limit *int
	// Offset continues pagination from the last reaction seen. Public chats only.
	Offset *int64
}

type getReactionsRequest struct {
	ym.Target
	MessageID ym.MessageID `json:"message_id"`
	ThreadID  *ym.ThreadID `json:"thread_id,omitempty"`
	Limit     *int         `json:"limit,omitempty"`
	Offset    *int64       `json:"offset,omitempty"`
}

// SendReaction sets the bot's reaction on a message.
//
// Passing a nil reaction removes whatever reaction the bot had set. The
// operation is idempotent: setting the same reaction twice, or removing an
// absent one, both succeed.
//
// The reactions a chat permits are listed in ChatInfo.AvailableReactions.
func (s *Service) SendReaction(
	ctx context.Context, target ym.Target, messageID ym.MessageID, reaction *ym.Reaction, opts *ReactionOptions,
) error {
	if err := ym.ValidateTarget(target); err != nil {
		return err
	}
	if messageID == 0 {
		return ErrMessageIDRequired
	}

	req := sendReactionRequest{Target: target, MessageID: messageID, Reaction: reaction}
	if opts != nil {
		req.ThreadID = opts.ThreadID
	}

	return s.postForOK(ctx, ym.EndpointMessagesSendReaction, req)
}

// RemoveReaction clears the bot's reaction from a message.
func (s *Service) RemoveReaction(
	ctx context.Context, target ym.Target, messageID ym.MessageID, opts *ReactionOptions,
) error {
	return s.SendReaction(ctx, target, messageID, nil, opts)
}

// GetReactions returns the reactions on a message.
//
// The response shape depends on the chat: private and group chats return
// reactions with their authors in List, while channels return anonymous
// tallies in Counts. Check [ym.ReactionsPage.Type] before reading either.
func (s *Service) GetReactions(
	ctx context.Context, target ym.Target, messageID ym.MessageID, opts *GetReactionsOptions,
) (*ym.ReactionsPage, error) {
	if err := ym.ValidateTarget(target); err != nil {
		return nil, err
	}
	if messageID == 0 {
		return nil, ErrMessageIDRequired
	}

	req := getReactionsRequest{Target: target, MessageID: messageID}
	if opts != nil {
		req.ThreadID = opts.ThreadID
		req.Limit = opts.Limit
		req.Offset = opts.Offset
	}

	var parsed struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
		ym.ReactionsPage
	}
	if err := s.post(ctx, ym.EndpointMessagesGetReactions, req, &parsed); err != nil {
		return nil, err
	}
	if !parsed.OK {
		return nil, okFalseError(ym.EndpointMessagesGetReactions, parsed.Description)
	}

	page := parsed.ReactionsPage

	return &page, nil
}
