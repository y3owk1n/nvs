package platform

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/y3owk1n/nvs/internal/constants"
)

// GetNvimConfigBaseDir determines the canonical configuration directory
// used by Neovim according to its runtime path conventions.
//
// Resolution order:
//
//  1. If the environment variable XDG_CONFIG_HOME is set, Neovim looks under
//     $XDG_CONFIG_HOME/nvim. This is the highest-precedence override.
//     Example (Linux/macOS):
//     XDG_CONFIG_HOME="$HOME/.xdg"
//     Example (Windows, PowerShell):
//     $env:XDG_CONFIG_HOME="C:\\xdg"
//
//  2. If XDG_CONFIG_HOME is not set, Neovim falls back to a platform-specific
//     default:
//
//     • Linux/macOS → $HOME/.config
//     Example: "/home/alice/.config"
//     "/Users/alice/.config"
//
//     • Windows → %LOCALAPPDATA%
//     Example: "C:\\Users\\alice\\AppData\\Local"
//
//  3. If LOCALAPPDATA is not set on Windows, this function falls back to
//     $HOME/.config/nvim for consistency with other platforms.
//
// Returns:
//   - The absolute path to the Neovim configuration directory.
//   - An error if the user's home directory cannot be determined when required.
//
// Notes:
//   - This function does *not* consider tool-specific overrides such as
//     NVS_CONFIG_DIR, because it is intended to strictly reflect Neovim's
//     own search path rules.
//   - Callers should ensure that the returned directory exists before use;
//     Neovim itself will create it lazily if needed.
func GetNvimConfigBaseDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return xdg, nil
	}

	if runtime.GOOS == constants.WindowsOS {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return local, nil
		}
		// fallback to home/.config if LOCALAPPDATA is missing
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config"), nil
}
