package builder_test

import (
	"bytes"
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
	stdoutStr string
}

const (
	gitCmd      = "git"
	whichCmd    = "which"
	gitTool     = "git"
	makeTool    = "make"
	cmakeTool   = "cmake"
	gettextTool = "gettext"
	ninjaTool   = "ninja"
	curlTool    = "curl"
)

func (m *mockCommand) Run() error {
	// Simulate writing to stdout if it's a buffer
	if buf, ok := m.stdout.(*bytes.Buffer); ok && m.stdoutStr != "" {
		buf.WriteString(m.stdoutStr)
	}

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
		if name == gitCmd && len(args) > 0 && args[0] == "clone" {
			return &mockCommand{runErr: cloneErr}
		}
		// Mock successful tool checks
		if name == whichCmd &&
			(args[0] == gitTool || args[0] == makeTool || args[0] == cmakeTool || args[0] == gettextTool || args[0] == ninjaTool || args[0] == curlTool) {
			return &mockCommand{}
		}

		return &mockCommand{}
	}

	b := builder.New(mockExec)
	ctx := context.Background()

	_, err := b.BuildFromCommit(ctx, "abc1234", t.TempDir(), nil)
	if err == nil {
		t.Error("BuildFromCommit() expected error for clone failure, got nil")
	}
}

// TestBuildFromCommit_ProgressReporting tests that progress is reported correctly.
func TestBuildFromCommit_ProgressReporting(t *testing.T) {
	var progressCalls []struct {
		phase   string
		percent int
	}

	progressFunc := func(phase string, percent int) {
		progressCalls = append(progressCalls, struct {
			phase   string
			percent int
		}{phase, percent})
	}

	mockExec := func(ctx context.Context, name string, args ...string) builder.Commander {
		// Mock git rev-parse to return a valid commit hash
		if name == gitCmd && len(args) > 0 && args[0] == "rev-parse" {
			return &mockCommand{stdoutStr: "abc1234567890"}
		}
		// Mock successful tool checks
		if name == whichCmd &&
			(args[0] == gitTool || args[0] == makeTool || args[0] == cmakeTool || args[0] == gettextTool || args[0] == ninjaTool || args[0] == curlTool) {
			return &mockCommand{}
		}

		return &mockCommand{}
	}

	b := builder.New(mockExec)
	ctx := context.Background()

	// This will fail later, but we check the progress calls up to that point
	_, _ = b.BuildFromCommit(ctx, "abc1234", t.TempDir(), progressFunc)

	// Check that progress calls use -1 for indeterminate phases
	for progressIndex, call := range progressCalls {
		if strings.Contains(call.phase, "Build complete") {
			if call.percent != 100 {
				t.Errorf(
					"Progress call %d (%s): expected percent 100 for completion, got %d",
					progressIndex,
					call.phase,
					call.percent,
				)
			}
		} else {
			if call.percent != -1 {
				t.Errorf(
					"Progress call %d (%s): expected percent -1 for indeterminate, got %d",
					progressIndex,
					call.phase,
					call.percent,
				)
			}
			// Check that phase is one of the expected ones (may include elapsed time)
			basePhase := strings.Split(call.phase, " (")[0]
			if basePhase != "Cloning repository" && basePhase != "Checking out commit" &&
				basePhase != "Building Neovim" && basePhase != "Installing Neovim" {
				t.Errorf("Progress call %d: unexpected phase %q", progressIndex, call.phase)
			}
		}
	}

	// Ensure we have at least some progress calls
	if len(progressCalls) == 0 {
		t.Error("Expected some progress calls, got none")
	}
}

// TestBuildFromCommit_CheckoutFailure tests build failure when git checkout fails.
func TestBuildFromCommit_CheckoutFailure(t *testing.T) {
	checkoutErr := errCheckoutFailed

	mockExec := func(ctx context.Context, name string, args ...string) builder.Commander {
		// First call is clone (succeed), second is checkout (fail)
		if name == gitCmd && len(args) > 0 && args[0] == "checkout" {
			return &mockCommand{runErr: checkoutErr}
		}
		// Mock successful tool checks
		if name == whichCmd &&
			(args[0] == gitTool || args[0] == makeTool || args[0] == cmakeTool || args[0] == gettextTool || args[0] == ninjaTool || args[0] == curlTool) {
			return &mockCommand{}
		}

		return &mockCommand{}
	}

	b := builder.New(mockExec)
	ctx := context.Background()

	_, err := b.BuildFromCommit(ctx, "abc1234", t.TempDir(), nil)
	if err == nil {
		t.Error("BuildFromCommit() expected error for checkout failure, got nil")
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
