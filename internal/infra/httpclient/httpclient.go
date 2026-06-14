// Package httpclient provides a tuned *http.Client and *http.Transport
// shared by all of nvs's outbound HTTP traffic (GitHub API, asset
// downloads, changelog fetch).
//
// The defaults shipped with net/http are sensible, but two
// adjustments are worth making for a CLI that issues many small
// sequential requests to the same host:
//
//  1. MaxIdleConnsPerHost: 100 instead of DefaultMaxIdleConnsPerHost
//     (= 2). The Neovim GitHub releases endpoint, the asset CDN, and
//     the mirror are all single-host, and we want the keep-alive pool
//     large enough that an idle connection is almost always available
//     between requests in a single command invocation.
//  2. IdleConnTimeout: 90s. The default is 0, meaning connections
//     stay in the pool indefinitely. Bounding it to 90s lets the
//     pool drain during long idle periods (e.g., between CLI
//     invocations from a shell prompt) so we do not hold sockets
//     open forever.
//
// Everything else is left at net/http's defaults: ForceAttemptHTTP2
// stays true, TLSHandshakeTimeout stays at 10s, etc.
package httpclient

import (
	"net/http"
	"time"
)

// DefaultTimeout is the per-request timeout used when a caller does
// not specify one. It is the same as the previous ad-hoc
// constants.ClientTimeoutSec default in the GitHub client.
const DefaultTimeout = 15 * time.Second

// maxIdleConns and idleConnTimeout are the tuning knobs applied on
// top of http.DefaultTransport. They live in named constants so the
// literal does not trip the magic-number linter.
const (
	maxIdleConns    = 100
	idleConnTimeout = 90 * time.Second
)

// sharedTransport is the singleton *http.Transport used by every
// client returned from NewClient. Sharing one transport lets all
// callers benefit from the same keep-alive pool.
var sharedTransport = func() *http.Transport {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		// http.DefaultTransport is documented as *http.Transport;
		// this branch only exists as a safety net so a future Go
		// release that changes the type does not panic on init.
		base = &http.Transport{}
	}

	cloned := base.Clone()
	// Match http.DefaultTransport's default; duplicated here for
	// clarity and so the value is visible at the call site.
	cloned.MaxIdleConns = maxIdleConns
	// http.DefaultTransport already sets MaxIdleConnsPerHost to
	// DefaultMaxIdleConnsPerHost (2); bump it up for sequential
	// single-host workloads.
	cloned.MaxIdleConnsPerHost = maxIdleConns
	// Bound how long an idle connection lingers in the pool so we
	// do not hold sockets open between unrelated CLI invocations.
	cloned.IdleConnTimeout = idleConnTimeout

	return cloned
}()

// NewClient returns an *http.Client that uses the shared, tuned
// transport and the supplied per-request timeout. Passing a zero
// timeout disables the timeout (matching http.Client's default
// behavior).
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: sharedTransport,
	}
}

// Transport returns the shared *http.Transport for callers that
// need to construct an *http.Client with custom fields (e.g., a
// CheckRedirect hook) and still want the tuned pool.
func Transport() *http.Transport {
	return sharedTransport
}
