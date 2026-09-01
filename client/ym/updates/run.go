package updates

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rekurt/ymsdk/client/ym"
	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

// Handler processes a single update. Returning an error hands control to
// [RunOptions.OnHandlerError].
type Handler func(context.Context, ym.Update) error

// ErrorAction tells [Service.Run] how to proceed after an error.
type ErrorAction int

const (
	// ActionStop ends the loop and returns the error to the caller.
	ActionStop ErrorAction = iota
	// ActionContinue moves on as if nothing had happened.
	ActionContinue
	// ActionRetry waits with exponential back-off and tries again.
	ActionRetry
)

const (
	defaultPollInterval   = time.Second
	defaultMaxBackoff     = 30 * time.Second
	initialPollBackoff    = time.Second
	defaultHandlerRetries = 3
)

// RunOptions configures [Service.Run].
type RunOptions struct {
	// Limit caps the number of updates per poll. Defaults to 100 server-side,
	// maximum 1000.
	Limit *int
	// Offset is the first update to fetch. Zero starts from the oldest update
	// still available.
	Offset *int64
	// Interval is the pause between polls. Defaults to one second.
	Interval time.Duration
	// MaxBackoff caps the back-off applied after a failed poll. Defaults to 30s.
	MaxBackoff time.Duration

	// OnPollError decides what to do when fetching updates fails.
	//
	// The default retries what a later attempt might survive — network trouble,
	// rate limits, 5xx — and stops on failures that will repeat forever, such
	// as a revoked token or a malformed request. Supply a policy to override
	// either half.
	OnPollError func(error) ErrorAction
	// OnHandlerError decides what to do when the handler fails. The default is
	// [ActionStop], which surfaces the bug rather than dropping the update
	// silently. [ActionContinue] moves on to the next update — the failed one
	// is then carried out of reach by the advancing offset. [ActionRetry]
	// re-invokes the handler on the same update with back-off, up to
	// MaxHandlerRetries times.
	OnHandlerError func(ym.Update, error) ErrorAction
	// MaxHandlerRetries bounds the extra attempts an update gets while
	// OnHandlerError keeps returning [ActionRetry]. Defaults to 3. Once they
	// are spent the error is returned: a caller who asked for retries did not
	// ask for the failure to be swallowed.
	MaxHandlerRetries int
	// OnPanic, when set, recovers panics raised by the handler and reports
	// them. Left nil, a panicking handler crashes the process as usual.
	OnPanic func(ym.Update, any)
}

// Run polls for updates and dispatches each one to handler until the context
// is cancelled.
//
// Unlike [Service.PollLoop] it survives transient failures: a failed poll backs
// off and retries instead of ending the loop. Every wait honours the context,
// so cancelling returns promptly rather than after the current interval.
//
// Note that fetching updates with an offset permanently erases every update
// below it. Updates the handler rejected are therefore only redelivered while
// the offset has not advanced past them.
func (s *Service) Run(ctx context.Context, opts RunOptions, handler Handler) error {
	// A limit the API cannot accept would 400 forever under the default retry
	// policy, so it fails here rather than becoming a hot loop.
	if opts.Limit != nil {
		if err := ym.ValidatePageLimit(*opts.Limit); err != nil {
			return err
		}
	}

	interval := opts.Interval
	if interval <= 0 {
		interval = defaultPollInterval
	}
	maxBackoff := opts.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = defaultMaxBackoff
	}

	offset := opts.Offset
	backoff := startingBackoff(maxBackoff)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		upds, nextOffset, err := s.GetUpdates(ctx, GetUpdatesParams{Limit: opts.Limit, Offset: offset})
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			switch pollErrorAction(opts, err) {
			case ActionStop:
				return err
			case ActionContinue:
				backoff = initialPollBackoff
			case ActionRetry:
				if waitErr := ym.SleepContext(ctx, backoff); waitErr != nil {
					return waitErr
				}
				backoff = ym.NextBackoff(backoff, maxBackoff)
			}

			continue
		}
		backoff = initialPollBackoff

		stop, handlerErr := s.dispatch(ctx, opts, upds, handler)
		if handlerErr != nil {
			return handlerErr
		}
		if stop {
			return ctx.Err()
		}

		offset = &nextOffset

		if err := ym.SleepContext(ctx, interval); err != nil {
			return err
		}
	}
}

// dispatch hands each update to the handler. It reports whether the loop should
// stop, and any error that must be returned to the caller.
func (s *Service) dispatch(
	ctx context.Context, opts RunOptions, upds []ym.Update, handler Handler,
) (bool, error) {
	for _, u := range upds {
		if ctx.Err() != nil {
			return true, nil
		}

		if err := s.deliver(ctx, opts, u, handler); err != nil {
			return false, err
		}
	}

	return false, nil
}

// deliver hands one update to the handler and applies the caller's error policy.
//
// [ActionRetry] re-invokes the handler on the same update rather than moving
// on, because moving on lets the batch offset advance past the failure and the
// update is then unreachable — getUpdates erases everything below the offset.
func (s *Service) deliver(ctx context.Context, opts RunOptions, u ym.Update, handler Handler) error {
	maxRetries := opts.MaxHandlerRetries
	if maxRetries <= 0 {
		maxRetries = defaultHandlerRetries
	}
	maxBackoff := opts.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = defaultMaxBackoff
	}
	backoff := startingBackoff(maxBackoff)

	for attempt := 0; ; attempt++ {
		err := invokeHandler(ctx, opts, u, handler)
		if err == nil {
			return nil
		}

		action := handlerErrorAction(opts, u, err)
		if action == ActionContinue {
			return nil
		}
		if action != ActionRetry || attempt >= maxRetries {
			return err
		}

		if waitErr := ym.SleepContext(ctx, backoff); waitErr != nil {
			return waitErr
		}
		backoff = ym.NextBackoff(backoff, maxBackoff)
	}
}

// invokeHandler calls handler, converting a panic into an error when the caller
// asked for panics to be recovered.
func invokeHandler(ctx context.Context, opts RunOptions, u ym.Update, handler Handler) (err error) {
	if opts.OnPanic != nil {
		defer func() {
			if r := recover(); r != nil {
				opts.OnPanic(u, r)
				err = fmt.Errorf("yandex-messenger/updates: handler panicked on update %d: %v", u.UpdateID, r)
			}
		}()
	}

	return handler(ctx, u)
}

func pollErrorAction(opts RunOptions, err error) ErrorAction {
	if opts.OnPollError == nil {
		return defaultPollErrorAction(err)
	}

	return opts.OnPollError(err)
}

// defaultPollErrorAction retries what a later attempt might survive and stops
// on what it cannot.
//
// Retrying everything would keep a bot alive through a transient 502, but it
// would also bury permanent failures: a revoked token answers 401 on every
// attempt, so the loop would spin at MaxBackoff forever and never let the
// caller or a supervisor learn that the bot is misconfigured.
func defaultPollErrorAction(err error) ErrorAction {
	switch {
	case errors.Is(err, ymerrors.ErrUnauthorized),
		errors.Is(err, ymerrors.ErrInvalidToken),
		errors.Is(err, ymerrors.ErrBadRequest),
		errors.Is(err, ymerrors.ErrNotFound),
		errors.Is(err, ymerrors.ErrConflict),
		errors.Is(err, ymerrors.ErrPayloadTooLarge):
		return ActionStop
	default:
		// Network trouble, rate limits and 5xx are worth another attempt.
		return ActionRetry
	}
}

func handlerErrorAction(opts RunOptions, u ym.Update, err error) ErrorAction {
	if opts.OnHandlerError == nil {
		return ActionStop
	}

	return opts.OnHandlerError(u, err)
}

// startingBackoff is the first delay before a retry, never longer than the
// ceiling the caller configured. MaxBackoff is meant to bound every wait, not
// only the ones exponential growth would produce.
func startingBackoff(maxBackoff time.Duration) time.Duration {
	if maxBackoff > 0 && maxBackoff < initialPollBackoff {
		return maxBackoff
	}

	return initialPollBackoff
}
