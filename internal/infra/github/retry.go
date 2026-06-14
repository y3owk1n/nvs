package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/y3owk1n/nvs/internal/constants"
)

// retriableStatus reports whether an HTTP status is worth retrying.
// 429 (rate limit) and 5xx (server errors) are; everything else is
// treated as terminal.
func retriableStatus(code int) bool {
	return code == http.StatusTooManyRequests ||
		(code >= 500 && code < 600)
}

// retryAfterDelay returns how long to wait before the next attempt
// based on the server's Retry-After / X-Ratelimit-Reset headers. If
// the server did not provide a hint, the fallback delay is used.
// A Retry-After of 0 is honored as "retry immediately".
func retryAfterDelay(resp *http.Response, fallback time.Duration) time.Duration {
	const maxHonored = time.Minute

	headerVal := resp.Header.Get("Retry-After")
	if headerVal != "" {
		// Retry-After is in seconds (HTTP-date is also allowed but rare).
		secs, err := strconv.Atoi(headerVal)
		if err == nil && secs >= 0 {
			delay := time.Duration(secs) * time.Second
			if delay <= maxHonored {
				return delay
			}
		}
	}

	headerVal = resp.Header.Get("X-Ratelimit-Reset")
	if headerVal != "" {
		// X-Ratelimit-Reset is a Unix timestamp (seconds).
		resetUnix, err := strconv.ParseInt(headerVal, 10, 64)
		if err == nil {
			resetAt := time.Unix(resetUnix, 0)

			delay := time.Until(resetAt)
			if delay > 0 && delay <= maxHonored {
				return delay
			}
		}
	}

	return fallback
}

// isRetriableNetError reports whether a network error is worth
// retrying. Connection resets, refused connections, and DNS errors
// are retriable; context cancellation is not.
func isRetriableNetError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	return true
}

// sleepWithCtx sleeps for d, returning early if ctx is canceled.
func sleepWithCtx(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}

	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-t.C:
	case <-ctx.Done():
	}
}

// errExhaustedRetries is the sentinel error used when all retry
// attempts have been used.
var errExhaustedRetries = errors.New("exhausted retries")

// doWithRetry executes a GET with bounded exponential backoff for
// transient failures (network errors, 429, 5xx). On 429 the
// server's Retry-After / X-Ratelimit-Reset headers are respected.
// The user-agent and the vnd.github+json Accept header are set on
// every attempt.
func (c *Client) doWithRetry(ctx context.Context, url string) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= constants.MaxGitHubRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", "nvs")
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, doErr := c.httpClient.Do(req)
		if doErr != nil {
			lastErr = doErr
			if !isRetriableNetError(doErr) {
				return nil, fmt.Errorf("failed to fetch: %w", doErr)
			}

			logrus.Debugf("GitHub fetch attempt %d failed: %v", attempt+1, doErr)
		} else {
			if !retriableStatus(resp.StatusCode) {
				return resp, nil
			}

			delay := retryAfterDelay(
				resp,
				constants.GitHubInitialBackoff<<attempt,
			)
			logrus.Debugf(
				"GitHub fetch attempt %d returned %d; sleeping %s",
				attempt+1,
				resp.StatusCode,
				delay,
			)
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("%w: %d", ErrAPIRequestFailed, resp.StatusCode)

			sleepWithCtx(ctx, delay)

			continue
		}

		if attempt == constants.MaxGitHubRetries {
			break
		}

		delay := constants.GitHubInitialBackoff << attempt
		sleepWithCtx(ctx, delay)
	}

	if lastErr == nil {
		lastErr = errExhaustedRetries
	}

	ctxErr := ctx.Err()
	if ctxErr != nil {
		return nil, fmt.Errorf("context done: %w", ctxErr)
	}

	return nil, fmt.Errorf("exhausted %d retries: %w", constants.MaxGitHubRetries+1, lastErr)
}
