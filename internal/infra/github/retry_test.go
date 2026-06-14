//nolint:testpackage
package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/infra/httpclient"
)

// staticError is used to construct a non-context, non-nil error in
// the isRetriableNetError table tests.
type staticError struct {
	msg string
}

func (err *staticError) Error() string {
	return err.msg
}

// doWithRetryForTest wraps doWithRetry and ensures any non-nil
// response body is closed, satisfying the bodyclose linter for
// error-path tests.
func doWithRetryForTest(ctx context.Context, client *Client, url string) error {
	resp, err := client.doWithRetry(ctx, url)
	if resp != nil {
		_ = resp.Body.Close()
	}

	return err
}

func TestDoWithRetry_SuccessOnFirstAttempt(t *testing.T) {
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			calls++

			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte("ok"))
		},
	))
	defer server.Close()

	client := &Client{httpClient: httpclient.NewClient(5 * time.Second)}

	resp, err := client.doWithRetry(t.Context(), server.URL)
	if err != nil {
		t.Fatalf("doWithRetry error: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestDoWithRetry_RetriesOn5xx(t *testing.T) {
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			calls++
			if calls < 3 {
				writer.WriteHeader(http.StatusBadGateway)

				return
			}

			writer.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := &Client{httpClient: httpclient.NewClient(5 * time.Second)}

	resp, err := client.doWithRetry(t.Context(), server.URL)
	if err != nil {
		t.Fatalf("doWithRetry error: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestDoWithRetry_RespectsRetryAfter(t *testing.T) {
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			calls++
			if calls == 1 {
				writer.Header().Set("Retry-After", "0")
				writer.WriteHeader(http.StatusTooManyRequests)

				return
			}

			writer.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	client := &Client{httpClient: httpclient.NewClient(5 * time.Second)}

	start := time.Now()

	resp, err := client.doWithRetry(t.Context(), server.URL)
	if err != nil {
		t.Fatalf("doWithRetry error: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	elapsed := time.Since(start)

	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}

	if elapsed > 100*time.Millisecond {
		t.Errorf("elapsed = %s; expected < 100ms with Retry-After: 0", elapsed)
	}
}

func TestDoWithRetry_GivesUpAfterMaxAttempts(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			writer.WriteHeader(http.StatusServiceUnavailable)
		},
	))
	defer server.Close()

	client := &Client{httpClient: httpclient.NewClient(5 * time.Second)}

	err := doWithRetryForTest(t.Context(), client, server.URL)
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}

	if got := calls.Load(); got < 2 {
		t.Errorf("calls = %d, want at least 2 (1 + MaxGitHubRetries)", got)
	}
}

func TestDoWithRetry_DoesNotRetry4xxOtherThan429(t *testing.T) {
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			calls++

			writer.WriteHeader(http.StatusNotFound)
		},
	))
	defer server.Close()

	client := &Client{httpClient: httpclient.NewClient(5 * time.Second)}

	resp, err := client.doWithRetry(t.Context(), server.URL)
	if err != nil {
		t.Fatalf("doWithRetry error: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}

	if calls != 1 {
		t.Errorf("calls = %d, want 1 (no retry on 4xx)", calls)
	}
}

func TestDoWithRetry_AbortsOnContextCancel(t *testing.T) {
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			calls++

			writer.Header().Set("Retry-After", "60")
			writer.WriteHeader(http.StatusTooManyRequests)
		},
	))
	defer server.Close()

	client := &Client{httpClient: httpclient.NewClient(5 * time.Second)}

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel before calling

	err := doWithRetryForTest(ctx, client, server.URL)
	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want wrap of context.Canceled", err)
	}
}

func TestRetryAfterDelay_RetryAfterSeconds(t *testing.T) {
	resp := httptest.NewRecorder()
	resp.Header().Set("Retry-After", strconv.Itoa(5))

	got := retryAfterDelay(resp.Result(), time.Hour)
	want := 5 * time.Second

	if got != want {
		t.Errorf("retryAfterDelay = %s, want %s", got, want)
	}
}

func TestRetryAfterDelay_RetryAfterZero(t *testing.T) {
	resp := httptest.NewRecorder()
	resp.Header().Set("Retry-After", "0")

	got := retryAfterDelay(resp.Result(), time.Hour)
	if got != 0 {
		t.Errorf("retryAfterDelay = %s, want 0 (retry immediately)", got)
	}
}

func TestRetryAfterDelay_RatelimitReset(t *testing.T) {
	resp := httptest.NewRecorder()

	resetAt := time.Now().Add(2 * time.Second).Unix()
	resp.Header().Set("X-Ratelimit-Reset", strconv.FormatInt(resetAt, 10))

	got := retryAfterDelay(resp.Result(), time.Hour)
	if got <= 0 || got > 3*time.Second {
		t.Errorf("retryAfterDelay = %s, want ~2s", got)
	}
}

func TestRetryAfterDelay_NoHeader(t *testing.T) {
	resp := httptest.NewRecorder()

	fallback := 999 * time.Millisecond

	got := retryAfterDelay(resp.Result(), fallback)
	if got != fallback {
		t.Errorf("retryAfterDelay = %s, want fallback %s", got, fallback)
	}
}

func TestSleepWithCtx_RespectsCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	start := time.Now()

	sleepWithCtx(ctx, 10*time.Second)

	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("sleepWithCtx took %s with canceled context; want < 100ms", elapsed)
	}
}

func TestSleepWithCtx_NoOpForZero(t *testing.T) {
	start := time.Now()

	sleepWithCtx(t.Context(), 0)

	if elapsed := time.Since(start); elapsed > 10*time.Millisecond {
		t.Errorf("sleepWithCtx(0) took %s; want ~0", elapsed)
	}
}

func TestIsRetriableNetError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"context canceled", context.Canceled, false},
		{"context deadline", context.DeadlineExceeded, false},
		{"generic", &staticError{msg: "connection reset"}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRetriableNetError(tc.err); got != tc.want {
				t.Errorf("isRetriableNetError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestRetriableStatus(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{301, false},
		{404, false},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
		{600, false},
	}

	for _, tc := range tests {
		if got := retriableStatus(tc.code); got != tc.want {
			t.Errorf("retriableStatus(%d) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

// TestMaxGitHubResponseBytes_BoundsDecode ensures the body limit is
// a sane value. 32 MiB at the time of writing.
func TestMaxGitHubResponseBytes_BoundsDecode(t *testing.T) {
	if constants.MaxGitHubResponseBytes < 1<<20 {
		t.Errorf(
			"MaxGitHubResponseBytes = %d, want at least 1 MiB",
			constants.MaxGitHubResponseBytes,
		)
	}
}
