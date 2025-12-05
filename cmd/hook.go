package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/constants"
)

// hookCmd represents the "hook" command.
// It outputs shell hook code that enables automatic version switching when changing directories.
var hookCmd = &cobra.Command{
	Use:   "hook [shell]",
	Short: "Output shell hook for automatic version switching",
	Long: `Output shell integration code for automatic version switching.

Add this to your shell configuration file to enable automatic switching
when entering directories with .nvs-version files.

For bash (~/.bashrc):
  eval "$(nvs hook bash)"

For zsh (~/.zshrc):
  eval "$(nvs hook zsh)"

For fish (~/.config/fish/config.fish):
  nvs hook fish | source`,
	Args: cobra.MaximumNArgs(1),
	RunE: RunHook,
}

// RunHook executes the hook command.
func RunHook(_ *cobra.Command, args []string) error {
	var shell string

	if len(args) > 0 {
		shell = args[0]
	} else {
		shell = DetectShell()
		if shell == "" {
			return ErrCouldNotDetectShellSpecify
		}
	}

	shell = strings.ToLower(shell)
	logrus.Debugf("Generating hook for shell: %s", shell)

	var hookScript string

	switch shell {
	case constants.ShellBash, constants.ShellZsh:
		hookScript = constants.BashZshHook
	case constants.ShellFish:
		hookScript = constants.FishHook
	default:
		return fmt.Errorf(
			"%w: %s (supported: %s, %s, %s)",
			ErrUnsupportedShellHook,
			shell,
			constants.ShellBash,
			constants.ShellZsh,
			constants.ShellFish,
		)
	}

	_, err := fmt.Fprint(os.Stdout, hookScript)
	if err != nil {
		return fmt.Errorf("failed to write hook: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(hookCmd)
}
