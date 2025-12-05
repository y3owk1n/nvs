package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
	RunE: runHook,
}

// Shell hook scripts for different shells.
//
//nolint:dupword
const (
	bashZshHook = `
_nvs_find_version_file() {
  local dir="$PWD"
  while [[ "$dir" != "/" ]]; do
    if [[ -f "$dir/.nvs-version" ]]; then
      echo "$dir/.nvs-version"
      return
    fi
    dir="$(dirname "$dir")"
  done

  # Check home directory
  if [[ -f "$HOME/.nvs-version" ]]; then
    echo "$HOME/.nvs-version"
  fi
}

_nvs_hook() {
  local nvs_version_file
  nvs_version_file="$(_nvs_find_version_file)"

  if [[ -n "$nvs_version_file" ]]; then
    local version
    version="$(cat "$nvs_version_file" | tr -d '[:space:]')"

    # Only switch if version changed
    if [[ "$version" != "$_NVS_CURRENT_VERSION" ]]; then
      if nvs use "$version" --force >/dev/null 2>&1; then
        export _NVS_CURRENT_VERSION="$version"
      fi
    fi
  fi
}

# Add hook to PROMPT_COMMAND (bash) or directory-change hook (zsh)
if [[ -n "$BASH_VERSION" ]]; then
  if [[ ! "$PROMPT_COMMAND" =~ "_nvs_hook" ]]; then
    PROMPT_COMMAND="_nvs_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
  fi
elif [[ -n "$ZSH_VERSION" ]]; then
  autoload -Uz add-zsh-hook
  add-zsh-hook chpwd _nvs_hook
  # Run once on shell start
  _nvs_hook
fi
`

	fishHook = `
function _nvs_find_version_file
  set -l dir $PWD
  while test "$dir" != "/"
    if test -f "$dir/.nvs-version"
      echo "$dir/.nvs-version"
      return
    end
    set dir (dirname "$dir")
  end

  # Check home directory
  if test -f "$HOME/.nvs-version"
    echo "$HOME/.nvs-version"
  end
end

function _nvs_hook --on-variable PWD
  set -l nvs_version_file (_nvs_find_version_file)

  if test -n "$nvs_version_file"
    set -l version (string trim (cat "$nvs_version_file"))

    # Only switch if version changed
    if test "$version" != "$_NVS_CURRENT_VERSION"
      if nvs use "$version" --force >/dev/null 2>&1
        set -g _NVS_CURRENT_VERSION "$version"
      end
    end
  end
end

# Run once on shell start
_nvs_hook
`
)

func runHook(cmd *cobra.Command, args []string) error {
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
	case ShellBash, ShellZsh:
		hookScript = bashZshHook
	case ShellFish:
		hookScript = fishHook
	default:
		return fmt.Errorf(
			"%w: %s (supported: %s, %s, %s)",
			ErrUnsupportedShellHook,
			shell,
			ShellBash,
			ShellZsh,
			ShellFish,
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
