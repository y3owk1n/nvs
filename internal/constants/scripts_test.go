package constants_test

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/y3owk1n/nvs/internal/constants"
)

// TestHookScript_Parses verifies that every shell hook script
// embedded into the binary via //go:embed parses cleanly with the
// target shell's `-n` flag. This runs as part of `go test ./...` on
// every platform where the relevant shell is available, and exists
// specifically to prevent the regression that truncated BashZshHook /
// FishHook as Go string literals: with the scripts living in real
// .sh / .fish files, any hand-editing truncation shows up immediately
// in a diff and fails this test before it can reach a user.
func TestHookScript_Parses(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("skipping shell parse check on windows")
	}

	cases := []struct {
		name     string
		shell    string
		shellBin string
	}{
		{name: "bash", shell: constants.ShellBash, shellBin: "bash"},
		{name: "zsh", shell: constants.ShellZsh, shellBin: "zsh"},
		{name: "fish", shell: constants.ShellFish, shellBin: "fish"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			bin, err := exec.LookPath(testCase.shellBin)
			if err != nil {
				t.Skipf("%s not available in PATH: %v", testCase.shellBin, err)
			}

			script, err := constants.HookScript(testCase.shell)
			if err != nil {
				t.Fatalf("HookScript(%q) returned error: %v", testCase.shell, err)
			}

			if strings.TrimSpace(script) == "" {
				t.Fatalf("HookScript(%q) returned empty content", testCase.shell)
			}

			cmd := exec.CommandContext(context.Background(), bin, "-n")
			cmd.Stdin = strings.NewReader(script)

			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s -n rejected embedded %s hook: %v\n--- output ---\n%s",
					testCase.shellBin, testCase.name, err, out)
			}
		})
	}
}

// TestHookScript_UnsupportedShell verifies the dispatch function
// surfaces a useful error for unrecognized shell names rather than
// silently returning an empty string (which would cause `nvs hook
// some-bad-shell` to write zero bytes to stdout and confuse the
// caller's `eval "$(...)"`).
func TestHookScript_UnsupportedShell(t *testing.T) {
	t.Parallel()

	for _, shell := range []string{"", "tcsh", "powershell", "BASH", "Fish"} {
		t.Run("shell="+shell, func(t *testing.T) {
			t.Parallel()

			_, err := constants.HookScript(shell)
			if err == nil {
				t.Fatalf("HookScript(%q) expected error, got nil", shell)
			}
		})
	}
}
