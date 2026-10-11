package httpclient_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/y3owk1n/nvs/internal/infra/httpclient"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{"zero timeout (no limit)", 0},
		{"five seconds", 5 * time.Second},
		{"thirty seconds", 30 * time.Second},
		{"one minute", time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := httpclient.NewClient(tt.timeout)

			if client == nil {
				t.Fatal("NewClient returned nil")
			}

			if client.Timeout != tt.timeout {
				t.Errorf("client.Timeout = %v, want %v", client.Timeout, tt.timeout)
			}

			if client.Transport == nil {
				t.Fatal("client.Transport is nil")
			}

			if client.Transport != httpclient.Transport() {
				t.Error("client.Transport is not the shared transport")
			}
		})
	}
}

func TestNewClient_ReturnsUsableClient(t *testing.T) {
	t.Parallel()

	client := httpclient.NewClient(5 * time.Second)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport is %T, want *http.Transport", client.Transport)
	}

	if transport.MaxIdleConnsPerHost != 100 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 100", transport.MaxIdleConnsPerHost)
	}

	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout = %v, want 90s", transport.IdleConnTimeout)
	}
}

func TestTransport(t *testing.T) {
	t.Parallel()

	transport := httpclient.Transport()

	if transport == nil {
		t.Fatal("Transport() returned nil")
	}

	if transport.MaxIdleConnsPerHost != 100 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 100", transport.MaxIdleConnsPerHost)
	}

	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout = %v, want 90s", transport.IdleConnTimeout)
	}
}

func TestTransport_Singleton(t *testing.T) {
	t.Parallel()

	first := httpclient.Transport()
	second := httpclient.Transport()

	if first != second {
		t.Error("Transport() returned different pointers; expected a shared singleton")
	}
}
