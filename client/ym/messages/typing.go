package messages

import (
	"context"
	"errors"

	"github.com/rekurt/ymsdk/client/ym"
)

// ErrProcessingContentRequired is returned when a processing indicator is
// requested without the content that describes how to render it.
var ErrProcessingContentRequired = errors.New(
	"yandex-messenger: processing_content is required when the typing type is processing",
)

// SendTypingOptions holds optional parameters for the typing indicator.
type SendTypingOptions struct {
	// Type selects the indicator. Defaults to [ym.TypingText].
	// [ym.TypingProcessing] is only available in private chats and requires
	// ProcessingContent.
	Type ym.TypingType
	// Timeout is how long the indicator stays visible, 1 to 60 seconds.
	// Defaults to 3 server-side.
	Timeout *int
	// ThreadID scopes the indicator to a thread.
	ThreadID *ym.ThreadID
	// ProcessingContent configures a processing indicator.
	ProcessingContent *ym.ProcessingContent
}

type sendTypingRequest struct {
	ym.Target
	Type              ym.TypingType         `json:"type,omitempty"`
	Timeout           *int                  `json:"timeout,omitempty"`
	ThreadID          *ym.ThreadID          `json:"thread_id,omitempty"`
	ProcessingContent *ym.ProcessingContent `json:"processing_content,omitempty"`
}

// SendTyping shows a typing or processing indicator in the target chat.
//
// Call it before a slow reply so the user sees activity. The indicator clears
// itself after the timeout, 3 seconds by default.
func (s *Service) SendTyping(ctx context.Context, target ym.Target, opts *SendTypingOptions) error {
	if err := ym.ValidateTarget(target); err != nil {
		return err
	}

	req := sendTypingRequest{Target: target}
	if opts != nil {
		if err := validateTypingOptions(opts); err != nil {
			return err
		}
		req.Type = opts.Type
		req.Timeout = opts.Timeout
		req.ThreadID = opts.ThreadID
		req.ProcessingContent = opts.ProcessingContent
	}

	return s.postForOK(ctx, ym.EndpointMessagesSendTyping, req)
}

func validateTypingOptions(opts *SendTypingOptions) error {
	if opts.Type == ym.TypingProcessing && opts.ProcessingContent == nil {
		return ErrProcessingContentRequired
	}
	// A text display without text, or with more than the documented 100
	// characters, is a request the API can only reject.
	if pc := opts.ProcessingContent; pc != nil && pc.Display == ym.ProcessingDisplayText {
		err := ym.ValidateRange("processing text length", len([]rune(pc.Text)),
			ym.MinProcessingTextLength, ym.MaxProcessingTextLength)
		if err != nil {
			return err
		}
	}
	if opts.Timeout != nil {
		return ym.ValidateRange("typing timeout", *opts.Timeout,
			ym.MinTypingTimeout, ym.MaxTypingTimeout)
	}

	return nil
}
