package builder_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/y3owk1n/nvs/internal/infra/builder"
)

var (
	errCloneFailed    = errors.New("clone failed")
	errCheckoutFailed = errors.New("checkout failed")
)

// mockCommand implements builder.Commander for testing.
type mockCommand struct {
	runErr    error
	dir       string
	stdout    any
	stderr    any
	stdoutBuf *strings.Reader
	stderrBuf *strings.Reader
}

func (m *mockCommand) Run() error {
	return m.runErr
}

func (m *mockCommand) SetDir(dir string) {
	m.dir = dir
}

func (m *mockCommand) SetStdout(stdout any) {
	m.stdout = stdout
}

func (m *mockCommand) SetStderr(stderr any) {
	m.stderr = stderr
}

func (m *mockCommand) StdoutPipe() (any, error) {
	if m.stdoutBuf == nil {
		m.stdoutBuf = strings.NewReader("")
	}

	return m.stdoutBuf, nil
}

func (m *mockCommand) StderrPipe() (any, error) {
	if m.stderrBuf == nil {
		m.stderrBuf = strings.NewReader("")
	}

	return m.stderrBuf, nil
}

// TestNew tests the New constructor.
func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		execFunc builder.ExecCommandFunc
		wantNil  bool
	}{
		{
			name:     "with nil execFunc uses default",
			execFunc: nil,
			wantNil:  false,
		},
		{
			name: "with custom execFunc",
			execFunc: func(ctx context.Context, name string, args ...string) builder.Commander {
				return &mockCommand{}
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.New(tt.execFunc)
			if (got == nil) != tt.wantNil {
				t.Errorf("New() returned nil = %v, want nil = %v", got == nil, tt.wantNil)
			}
		})
	}
}

// TestBuildFromCommit_CloneFailure tests build failure when git clone fails.
func TestBuildFromCommit_CloneFailure(t *testing.T) {
	cloneErr := errCloneFailed

	mockExec := func(ctx context.Context, name string, args ...string) builder.Commander {
		// Simulate git clone failure
		if name == "git" && len(args) > 0 && args[0] == "clone" {
			return &mockCommand{runErr: cloneErr}
		}

		return &mockCommand{}
	}

	b := builder.New(mockExec)
	ctx := context.Background()

	err := b.BuildFromCommit(ctx, "abc1234", t.TempDir())
	if err == nil {
		t.Error("BuildFromCommit() expected error for clone failure, got nil")
	}

	if !errors.Is(err, cloneErr) {
		t.Errorf("BuildFromCommit() error = %v, want to contain %v", err, cloneErr)
	}
}

// TestBuildFromCommit_CheckoutFailure tests build failure when git checkout fails.
func TestBuildFromCommit_CheckoutFailure(t *testing.T) {
	checkoutErr := errCheckoutFailed

	mockExec := func(ctx context.Context, name string, args ...string) builder.Commander {
		// First call is clone (succeed), second is checkout (fail)
		if name == "git" && len(args) > 0 && args[0] == "checkout" {
			return &mockCommand{runErr: checkoutErr}
		}

		return &mockCommand{}
	}

	b := builder.New(mockExec)
	ctx := context.Background()

	err := b.BuildFromCommit(ctx, "abc1234", t.TempDir())
	if err == nil {
		t.Error("BuildFromCommit() expected error for checkout failure, got nil")
	}

	if !errors.Is(err, checkoutErr) {
		t.Errorf("BuildFromCommit() error = %v, want to contain %v", err, checkoutErr)
	}
}

// TestSourceBuilder_Interface ensures SourceBuilder can be used with Commander interface.
func TestSourceBuilder_Interface(t *testing.T) {
	// This test verifies the interface contracts are satisfied
	var _ builder.Commander = &mockCommand{}

	// Test that ExecCommandFunc type is correct
	var execFn builder.ExecCommandFunc = func(ctx context.Context, name string, args ...string) builder.Commander {
		return &mockCommand{}
	}

	b := builder.New(execFn)
	if b == nil {
		t.Error("New() returned nil with valid execFunc")
	}
}
