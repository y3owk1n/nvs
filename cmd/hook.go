package cmd

import (
	"fmt"
	"os"

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
const (
	bashZshHook = "\n_nvs_hook() {\n  local nvs_version_file\n  nvs_version_file=\"$(_nvs_find_version_file)\"\n  \n  if [[ -n \"$nvs_version_file\" ]]; then\n    local version\n    version=\"$(cat \"$nvs_version_file\" | tr -d '[:space:]')\"\n    \n    # Only switch if version changed\n    if [[ \"$version\" != \"$_NVS_CURRENT_VERSION\" ]]; then\n      if nvs use \"$version\" --force >/dev/null 2>&1; then\n        export _NVS_CURRENT_VERSION=\"$version\"\n      fi\n    }\n\n_nvs_find_version_file() {\n  local dir=\"$PWD\"\n  while [[ \"$dir\" != \"/\" ]]; do\n    if [[ -f \"$dir/.nvs-version\" ]]; then\n      echo \"$dir/.nvs-version\"\n      return\n    fi\n    dir=\"$(dirname \"$dir\")\"\n  done\n  \n  # Check home directory\n  if [[ -f \"$HOME/.nvs-version\" ]]; then\n    echo \"$HOME/.nvs-version\"\n  fi\n}\n\n# Add hook to PROMPT_COMMAND (bash) or precmd (zsh)\nif [[ -n \"$BASH_VERSION\" ]]; then\n  if [[ ! \"$PROMPT_COMMAND\" =~ \"_nvs_hook\" ]]; then\n    PROMPT_COMMAND=\"_nvs_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}\"\n  fi\nelif [[ -n \"$ZSH_VERSION\" ]]; then\n  autoload -Uz add-zsh-hook\n  chpwd _nvs_hook\n  # Run once on shell start\n  _nvs_hook"

	fishHook = "\nfunction _nvs_hook --on-variable PWD\n  set -l nvs_version_file (_nvs_find_version_file)\n  \n  if test -n \"$nvs_version_file\"\n    set -l version (cat \"$nvs_version_file\" | string trim)\n    \n    # Only switch if version changed\n    if test \"$version\" != \"$_NVS_CURRENT_VERSION\"\n      if nvs use \"$version\" --force >/dev/null 2>&1\n        set -g _NVS_CURRENT_VERSION \"$version\"\n      end\n    function _nvs_find_version_file\n  set -l dir $PWD\n  while test \"$dir\" != \"/\"\n    if test -f \"$dir/.nvs-version\"\n      echo \"$dir/.nvs-version\"\n      return\n    end\n    set dir (dirname \"$dir\")\n  end\n  \n  # Check home directory\n  if test -f \"$HOME/.nvs-version\"\n    echo \"$HOME/.nvs-version\"\n  end\n# Run once on shell start"
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

	logrus.Debugf("Generating hook for shell: %s", shell)

	var hookScript string

	switch shell {
	case ShellBash, ShellZsh, "sh":
		hookScript = bashZshHook
	case ShellFish:
		hookScript = fishHook
	default:
		return fmt.Errorf("%w: %s (supported: %s, %s, %s)", ErrUnsupportedShellHook, shell, ShellBash, ShellZsh, ShellFish)
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
