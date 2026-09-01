package ym

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rekurt/ymsdk/client/ym/ymerrors"
)

const defaultInitialBackoff = 500 * time.Millisecond

// request describes a single logical API call. The body is pre-marshalled so
// that every retry attempt replays the exact same bytes — this is what makes
// the API's payload_id idempotency key effective across retries.
type request struct {
	method string
	path   string
	// query is appended to path. Callers that already embedded a query string
	// in path leave this nil.
	query url.Values
	// body is nil for requests without a payload, which also suppresses the
	// Content-Type header.
	body []byte
	// contentType is only set as a header when non-empty, so GET requests do
	// not advertise a JSON body they don't have.
	contentType string
}

// do executes r, retrying according to the client's retry and rate-limit
// configuration. It is the single implementation behind both [Client.DoRequest]
// and [Client.DoMultipartRequest].
//
// On success (2xx) the caller owns the returned response body and must close it.
func (c *Client) do(ctx context.Context, r request) (*http.Response, error) {
	target := c.buildURL(r.path, r.query)
	retryCfg := c.cfg.ErrorHandling.RetryStrategy
	rateCfg := c.cfg.ErrorHandling.RateLimitHandling

	attempts := retryCfg.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}
	backoff := retryCfg.InitialBackoff
	if backoff <= 0 {
		backoff = defaultInitialBackoff
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		hasRetriesLeft := attempt < attempts

		req, reqErr := c.newHTTPRequest(ctx, r, target)
		if reqErr != nil {
			return nil, reqErr
		}

		resp, doErr := c.http.Do(req)
		if doErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, c.wrapErr(ctxErr, r)
			}
			var netErr net.Error
			if errors.As(doErr, &netErr) && retryCfg.RetryNetwork && hasRetriesLeft {
				if err := sleepCtx(ctx, applyJitter(backoff, retryCfg.DisableJitter)); err != nil {
					return nil, c.wrapErr(err, r)
				}
				backoff = NextBackoff(backoff, retryCfg.MaxBackoff)

				continue
			}

			return nil, c.wrapErr(doErr, r)
		}

		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			return resp, nil
		}

		apiErr, parseErr := c.newAPIError(r.method, r.path, resp)
		if parseErr != nil {
			return nil, parseErr
		}

		// Rate-limit delays are not jittered: when the server sends Retry-After
		// it has told us exactly how long to wait.
		if apiErr.Kind == ymerrors.KindRateLimited && hasRetriesLeft {
			if err := sleepCtx(ctx, rateLimitDelay(apiErr, rateCfg)); err != nil {
				return nil, c.wrapErr(err, r)
			}

			continue
		}

		if ShouldRetryHTTP(apiErr.HTTPStatus, retryCfg.RetryHTTP) && hasRetriesLeft {
			if err := sleepCtx(ctx, applyJitter(backoff, retryCfg.DisableJitter)); err != nil {
				return nil, c.wrapErr(err, r)
			}
			backoff = NextBackoff(backoff, retryCfg.MaxBackoff)

			continue
		}

		return nil, apiErr
	}

	return nil, fmt.Errorf("yandex-messenger/client: retries exhausted for %s %s", r.method, r.path)
}

func (c *Client) newHTTPRequest(ctx context.Context, r request, target string) (*http.Request, error) {
	var bodyReader io.Reader
	if r.body != nil {
		bodyReader = bytes.NewReader(r.body)
	}

	req, err := http.NewRequestWithContext(ctx, r.method, target, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("yandex-messenger/client: build request: %w", err)
	}
	if c.cfg.Token != "" {
		req.Header.Set("Authorization", "OAuth "+c.cfg.Token)
	}
	if r.contentType != "" {
		req.Header.Set("Content-Type", r.contentType)
	}
	if c.cfg.UserAgent != "" {
		req.Header.Set("User-Agent", c.cfg.UserAgent)
	}

	return req, nil
}

func (c *Client) buildURL(path string, query url.Values) string {
	target := strings.TrimRight(c.cfg.BaseURL, "/") + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	return target
}

func (c *Client) wrapErr(err error, r request) error {
	return fmt.Errorf("yandex-messenger/client: %w for %s %s", err, r.method, r.path)
}

// sleepCtx waits for d, returning early with the context's error if the caller
// cancels first. Plain time.Sleep would keep a shutting-down bot blocked for
// the full backoff — up to MaxBackoff, 10s by default.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// applyJitter spreads a backoff over [d/2, d] so that clients that failed
// together do not retry in lockstep. Returns d unchanged when disabled.
func applyJitter(d time.Duration, disabled bool) time.Duration {
	if disabled || d <= 0 {
		return d
	}
	half := int64(d / 2)

	//nolint:gosec // decorrelates retry timing; not a security primitive
	return time.Duration(half + rand.Int64N(half+1))
}

func rateLimitDelay(apiErr *ymerrors.APIError, cfg ymerrors.RateLimitHandling) time.Duration {
	if cfg.UseRetryAfter && apiErr.RetryAfter > 0 {
		return apiErr.RetryAfter
	}

	return cfg.DefaultBackoff
}
