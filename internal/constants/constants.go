// Package constants contains all the constants used in the application.
package constants

import "time"

const (
	// WindowsOS is the string for Windows OS.
	WindowsOS = "windows"

	// FilePerm is the file permission for created files.
	FilePerm = 0o644
	// DirPerm is the directory permission for created directories.
	DirPerm = 0o755

	// Stable is the stable version alias.
	Stable = "stable"
	// Nightly is the nightly version alias.
	Nightly = "nightly"

	// TimeoutMinutes is the timeout in minutes for installation.
	TimeoutMinutes = 30

	// ShortCommitLen is the number of characters to shorten commit hashes to.
	ShortCommitLen = 7

	// VersionFileName is the name of the version sync file.
	VersionFileName = ".nvs-version"

	// NightlyHistoryFile is the name of the nightly history file.
	NightlyHistoryFile = "nightly-history.json"
	// DefaultRollbackLimit is the default limit for rollback entries.
	DefaultRollbackLimit = 5

	// UnavailableDir is the string for unavailable directory.
	UnavailableDir = "Unavailable"

	// GitHubCompareURL is the URL for GitHub compare API.
	GitHubCompareURL = "https://api.github.com/repos/neovim/neovim/compare"
	// DefaultAPIBaseURL is the default API base URL.
	DefaultAPIBaseURL = "https://api.github.com"
	// DefaultGitHubBaseURL is the default GitHub base URL for downloads.
	DefaultGitHubBaseURL = "https://github.com"
	// ClientTimeoutSec is the client timeout in seconds.
	ClientTimeoutSec = 15
	// HTTPTimeoutSeconds is the timeout in seconds for HTTP requests.
	HTTPTimeoutSeconds = 30

	// ChangelogLimit is the limit for changelog entries.
	ChangelogLimit = 10
	// CommitHashLength is the length of a commit hash.
	CommitHashLength = 40
	// ShortHashLength is the length of a short hash.
	ShortHashLength = 8
	// DisplayHashLength is the length of hash to display.
	DisplayHashLength = 7
	// MessageTruncateLimit is the limit for truncating messages.
	MessageTruncateLimit = 70

	// CacheTTL is the time-to-live for cache entries.
	CacheTTL = 5 * time.Minute

	// ShellBash is the bash shell name.
	ShellBash = "bash"
	// ShellZsh is the zsh shell name.
	ShellZsh = "zsh"
	// ShellFish is the fish shell name.
	ShellFish = "fish"

	// BashZshHook is the hook script for bash and zsh.
	BashZshHook = "\n_nvs_find_version_file() {\n  local dir=\"$PWD\"\n  while [[ \"$dir\" != \"/\" ]]; do\n    if [[ -f \"$dir/.nvs-version\" ]]; then\n      echo \"$dir/.nvs-version\"\n      return\n    fi\n    dir=\"$(dirname \"$dir\")\"\n  done\n\n  # Check home directory\n  if [[ -f \"$HOME/.nvs-version\" ]]; then\n    echo \"$HOME/.nvs-version\"\n  fi\n}\n\n_nvs_hook() {\n  local nvs_version_file\n  nvs_version_file=\"$(_nvs_find_version_file)\"\n\n  if [[ -n \"$nvs_version_file\" ]]; then\n    local version\n    version=\"$(tr -d '[:space:]' < \"$nvs_version_file\")\"\n\n    # Only switch if version changed\n    if [[ \"$version\" != \"$_NVS_CURRENT_VERSION\" ]]; then\n      if nvs use \"$version\" --force >/dev/null 2>&1; then\n        export _NVS_CURRENT_VERSION=\"$version\"\n      fi\n    }\n\n# Add hook to PROMPT_COMMAND (bash) or directory-change hook (zsh)\nif [[ -n \"$BASH_VERSION\" ]]; then\n  if [[ ! \"$PROMPT_COMMAND\" =~ \"_nvs_hook\" ]]; then\n    PROMPT_COMMAND=\"_nvs_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}\"\n  fi\nelif [[ -n \"$ZSH_VERSION\" ]]; then\n  autoload -Uz add-zsh-hook\n  chpwd _nvs_hook\nfi\n\n# Run once on shell start"
	// FishHook is the hook script for fish.
	FishHook = "\nfunction _nvs_find_version_file\n  set -l dir \"$PWD\"\n  while test \"$dir\" != \"/\"\n    if test -f \"$dir/.nvs-version\"\n      echo \"$dir/.nvs-version\"\n      return\n    end\n    set dir (dirname -- \"$dir\")\n  end\n\n  # Check home directory\n  if test -f \"$HOME/.nvs-version\"\n    echo \"$HOME/.nvs-version\"\n  end\nfunction _nvs_hook --on-variable PWD\n  set -l nvs_version_file (_nvs_find_version_file)\n\n  if test -n \"$nvs_version_file\"\n    set -l nvs_version (string trim < \"$nvs_version_file\")\n\n    # Only switch if version changed\n    if test \"$nvs_version\" != \"$_NVS_CURRENT_VERSION\"\n      if nvs use \"$nvs_version\" --force >/dev/null 2>&1\n        set -g _NVS_CURRENT_VERSION=\"$nvs_version\"\n      end\n    # Run once on shell start"

	// TestVersion is the test version.
	TestVersion = "v1.0.0"
	// TestCommitHash is the test commit hash.
	TestCommitHash = "abc1234"

	// NvimBinaryName is the name of the nvim binary.
	NvimBinaryName = "nvim"
	// NvimBinaryNameWindows is the name of the nvim binary on Windows.
	NvimBinaryNameWindows = "nvim.exe"

	// ProgressBarWidth is the default width of the progress bar.
	ProgressBarWidth = 20
	// ProgressFilled is the character for filled portion of the bar.
	ProgressFilled = "█"
	// ProgressEmpty is the character for empty portion of the bar.
	ProgressEmpty = "░"
	// ProgressMax is the maximum percentage value.
	ProgressMax = 100
	// GoroutineNum is the number of goroutines for spinner updates.
	GoroutineNum = 2

	// Checkmark is the checkmark icon.
	Checkmark = "✓"
	// Cross is the cross icon.
	Cross = "✖"
	// Info is the info icon.
	Info = "ℹ"
	// Warn is the warn icon.
	Warn = "⚠"
	// Upgrade is the upgrade icon.
	Upgrade = "↑"
	// Prompt is the prompt icon.
	Prompt = "?"

	// SpinnerSpeed is the spinner speed.
	SpinnerSpeed = 100
	// MaxAttempts is the maximum number of attempts.
	MaxAttempts = 3

	// RepoURL is the repository URL.
	RepoURL = "https://github.com/neovim/neovim.git"
	// GlobalCacheURL is the URL for the global cache JSON file.
	GlobalCacheURL = "https://raw.githubusercontent.com/y3owk1n/nvs/main/versions.json"

	// ProgressComplete is the value for completed progress.
	ProgressComplete = 100

	// ProcessCheckTimeout is the timeout for process checks.
	ProcessCheckTimeout = 5 * time.Second

	// Arm64Arch is the arm64 architecture string.
	Arm64Arch = "arm64"

	// ProgressDiv is the progress division factor.
	ProgressDiv = 100
	// ProgressDone is the done progress.
	ProgressDone = 100
	// TickerInterval is the ticker interval.
	TickerInterval = 10
	// OutputChanSize is the output channel size.
	OutputChanSize = 10
	// NumReaders is the number of readers.
	NumReaders = 2
	// BufferSize is the buffer size for reading.
	BufferSize = 4096
	// Sha256HashLen is the length of SHA256 hash.
	Sha256HashLen = 64
	// DefaultTimeout is the default timeout for downloads.
	DefaultTimeout = 5 * time.Minute
	// BufSize is the buffer size for extraction.
	BufSize = 262144
	// ZipFormat is the zip format string.
	ZipFormat = "zip"
	// FileModeMask is the file mode mask.
	FileModeMask = 0o777

	// TempDirNamePartsMin is the minimum number of parts in temp directory name.
	TempDirNamePartsMin = 3
)
