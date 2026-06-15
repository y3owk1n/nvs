package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/app/config"
	"github.com/y3owk1n/nvs/internal/app/versionsvc"
	"github.com/y3owk1n/nvs/internal/constants"
	"github.com/y3owk1n/nvs/internal/infra/archive"
	"github.com/y3owk1n/nvs/internal/infra/builder"
	"github.com/y3owk1n/nvs/internal/infra/downloader"
	"github.com/y3owk1n/nvs/internal/infra/filesystem"
	"github.com/y3owk1n/nvs/internal/infra/github"
	"github.com/y3owk1n/nvs/internal/infra/installer"
	"github.com/y3owk1n/nvs/internal/log"
	"github.com/y3owk1n/nvs/internal/ui/style"
)

var (
	// verbose raises the developer log to debug level. It is a
	// shortcut for NVS_LOG=debug; the env var wins when both
	// are set, on the principle that the more specific signal
	// (a named level) overrides the less specific one (a
	// boolean toggle).
	verbose bool

	// ctx is the global context used by the CLI.
	// cancel cancels the context, e.g. on interrupt signals.
	ctx, cancel = context.WithCancel(context.Background())

	// Services (initialized in InitConfig).
	versionService *versionsvc.Service
	configService  *config.Service

	// Configuration paths (initialized in InitConfig).
	versionsDir   string
	cacheFilePath string
	globalBinDir  string

	// errInvalidGitHubMirror is returned when the GitHub mirror URL is invalid.
	errInvalidGitHubMirror = errors.New(
		"invalid GitHub mirror URL: must be a valid absolute URL with http:// or https://",
	)
	errInvalidGitHubMirrorHost = errors.New(
		"invalid GitHub mirror URL: must include a valid host",
	)

	// Version of nvs, defaults to "v0.0.0" but may be set during build time.
	Version = "v0.0.0"
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"Enable verbose logging (shortcut for NVS_LOG=debug)")
}

// Execute initializes the configuration, sets up global flags, and executes the root command.
// Example usage:
//
//	func main() {
//	    if err := cmd.Execute(); err != nil {
//	        os.Exit(1)
//	    }
//	}
func Execute() error {
	// Use PersistentPreRunE to ensure flags are parsed before InitConfig runs,
	// and errors are propagated properly through cobra's error handling.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return InitConfig()
	}

	// Always cancel the global context when Execute returns, so
	// the signal-handler goroutine started by InitConfig (which
	// is still blocked on <-sigCh) is unblocked and can exit
	// cleanly. Without this defer, 'nvs <subcommand>' that
	// completes without receiving SIGINT would leak the
	// goroutine and the channel until the process exits —
	// technically harmless because the OS reaps everything on
	// exit, but visible to -race, to pprof, and to any future
	// 'nvs <subcommand>' that wants to fork/exec a child and
	// needs the parent state to be clean.
	defer cancel()

	// Execute the root command with the global context.
	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		return err
	}

	return nil
}

var signalOnce sync.Once

// InitConfig is called automatically on command initialization.
// It sets up logging levels, handles OS signals for graceful shutdown, and initializes services.
func InitConfig() error {
	var err error

	// Initialize the developer logger first so every step
	// below can emit traces.
	//
	// Resolution order for the level:
	//   1. NVS_LOG env var (named level, most specific)
	//   2. -v flag      (boolean shortcut for debug)
	//   3. default       (WarnLevel — silent in normal use)
	//
	// NVS_LOG_FILE, if set, tees all output to that file so
	// the terminal UI (spinners, panels) is not polluted by
	// debug traces even when the user wants them.
	level := log.WarnLevel
	if verbose {
		level = log.DebugLevel
	}

	if envLevel := os.Getenv("NVS_LOG"); envLevel != "" {
		parsed, parseErr := log.ParseLevel(envLevel)
		if parseErr != nil {
			_, _ = fmt.Fprintf(os.Stderr,
				"nvs: invalid NVS_LOG=%q, defaulting to warn: %v\n",
				envLevel, parseErr,
			)
		} else {
			level = parsed
		}
	}

	logErr := log.Init(log.Options{
		Level:    level,
		FilePath: os.Getenv("NVS_LOG_FILE"),
		NoColor:  !style.ColorEnabled(),
	})
	if logErr != nil {
		return fmt.Errorf("initialize logger: %w", logErr)
	}

	if verbose {
		log.Debug("verbose mode enabled")
	}

	// Set up a signal handler to cancel the global context on an interrupt signal.
	signalOnce.Do(func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)

		go func() {
			<-sigCh

			_, err := fmt.Fprintln(os.Stdout)
			if err != nil {
				log.Warn("failed to write to stdout on signal", "err", err)
			}

			log.Debug("interrupt received, canceling operations")
			signal.Stop(sigCh)
			cancel()
		}()
	})

	// Determine the base configuration directory.
	var baseConfigDir string
	if custom := os.Getenv("NVS_CONFIG_DIR"); custom != "" {
		baseConfigDir = custom
		log.Debug("using custom config directory", "dir", baseConfigDir, "source", "NVS_CONFIG_DIR")
	} else {
		var (
			configDir string
			configErr error
		)

		configDir, configErr = os.UserConfigDir()
		if configErr == nil {
			baseConfigDir = filepath.Join(configDir, "nvs")
			log.Debug("using system config directory", "dir", baseConfigDir)
		} else {
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				return fmt.Errorf("failed to get user home directory: %w", homeErr)
			}

			baseConfigDir = filepath.Join(home, ".nvs")
			log.Debug("falling back to home directory for config", "dir", baseConfigDir)
		}
	}

	// Ensure the configuration directory exists.
	err = os.MkdirAll(baseConfigDir, constants.DirPerm)
	if err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	log.Debug("config directory ensured", "dir", baseConfigDir)

	// Set the directory for installed versions.
	versionsDir = filepath.Join(baseConfigDir, "versions")

	err = os.MkdirAll(versionsDir, constants.DirPerm)
	if err != nil {
		return fmt.Errorf("failed to create versions directory: %w", err)
	}

	log.Debug("versions directory ensured", "dir", versionsDir)

	// Determine the base cache directory.
	var baseCacheDir string
	if custom := os.Getenv("NVS_CACHE_DIR"); custom != "" {
		baseCacheDir = custom
		log.Debug("using custom cache directory", "dir", baseCacheDir, "source", "NVS_CACHE_DIR")
	} else {
		var (
			cacheDir string
			cacheErr error
		)

		cacheDir, cacheErr = os.UserCacheDir()
		if cacheErr == nil {
			baseCacheDir = filepath.Join(cacheDir, "nvs")
			log.Debug("using system cache directory", "dir", baseCacheDir)
		} else {
			baseCacheDir = filepath.Join(baseConfigDir, "cache")
			log.Debug("falling back to config directory for cache", "dir", baseCacheDir)
		}
	}
	// Ensure the cache directory exists.
	err = os.MkdirAll(baseCacheDir, constants.DirPerm)
	if err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cacheFilePath = filepath.Join(baseCacheDir, "releases.json")
	log.Debug("cache directory ensured", "dir", baseCacheDir)
	log.Debug("cache file path set", "path", cacheFilePath)

	// Determine the base binary directory.
	var baseBinDir string
	if custom := os.Getenv("NVS_BIN_DIR"); custom != "" {
		baseBinDir = custom
		log.Debug("using custom binary directory", "dir", baseBinDir, "source", "NVS_BIN_DIR")
	} else {
		if runtime.GOOS == constants.WindowsOS {
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				return fmt.Errorf("failed to get user home directory: %w", homeErr)
			}

			baseBinDir = filepath.Join(home, "AppData", "Local", "Programs")
			log.Debug("using Windows binary directory", "dir", baseBinDir)
		} else {
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				return fmt.Errorf("failed to get user home directory: %w", homeErr)
			}

			baseBinDir = filepath.Join(home, ".local", "bin")
			log.Debug("using default binary directory", "dir", baseBinDir)
		}
	}
	// Ensure the binary directory exists.
	err = os.MkdirAll(baseBinDir, constants.DirPerm)
	if err != nil {
		return fmt.Errorf("failed to create binary directory: %w", err)
	}

	globalBinDir = baseBinDir
	log.Debug("global binary directory ensured", "dir", globalBinDir)

	// Read GitHub mirror URL from environment
	githubMirror := os.Getenv("NVS_GITHUB_MIRROR")

	var normalizedMirrorURL string
	if githubMirror != "" {
		parsedURL, err := url.Parse(githubMirror)
		if err != nil {
			return fmt.Errorf("failed to parse GitHub mirror URL: %w", err)
		}

		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return errInvalidGitHubMirror
		}

		if parsedURL.Host == "" {
			return errInvalidGitHubMirrorHost
		}

		normalizedMirrorURL = strings.TrimRight(parsedURL.String(), "/")
		log.Debug("using GitHub mirror", "url", normalizedMirrorURL)
	}

	// Read global cache setting from environment
	envValue := os.Getenv("NVS_USE_GLOBAL_CACHE")

	useGlobalCache := strings.EqualFold(envValue, "true") || envValue == "1"
	if useGlobalCache {
		log.Debug("global cache enabled")
	}

	// Initialize services
	githubClient := github.NewClient(
		cacheFilePath,
		constants.CacheTTL,
		"0.5.0",
		normalizedMirrorURL,
		useGlobalCache,
	)
	versionManager := filesystem.New(&filesystem.Config{
		VersionsDir:  versionsDir,
		GlobalBinDir: globalBinDir,
	})

	// Installer components
	dl := downloader.New()
	extractor := archive.New()
	srcBuilder := builder.New(nil) // nil for default exec command

	installService := installer.New(dl, extractor, srcBuilder)

	versionService, err = versionsvc.New(
		githubClient,
		versionManager,
		installService,
		&versionsvc.Config{
			VersionsDir:    versionsDir,
			CacheFilePath:  cacheFilePath,
			GlobalBinDir:   globalBinDir,
			MirrorURL:      normalizedMirrorURL,
			UseGlobalCache: useGlobalCache,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create version service: %w", err)
	}

	configService = config.New()

	log.Debug("services initialized")

	return nil
}

// GetVersionsDir returns the versions directory path.
// This is a compatibility function during migration.
func GetVersionsDir() string {
	return versionsDir
}

// GetCacheFilePath returns the cache file path.
// This is a compatibility function during migration.
func GetCacheFilePath() string {
	return cacheFilePath
}

// GetGlobalBinDir returns the global binary directory path.
// This is a compatibility function during migration.
func GetGlobalBinDir() string {
	return globalBinDir
}

// GetVersionService returns the version service instance.
func GetVersionService() *versionsvc.Service {
	return versionService
}

// SetVersionServiceForTesting sets the version service for testing.
// This should only be used in tests.
func SetVersionServiceForTesting(service *versionsvc.Service) {
	if os.Getenv("NVS_TEST_MODE") == "" {
		panic("SetVersionServiceForTesting should only be called in tests")
	}

	versionService = service
}

// GetConfigService returns the config service instance.
func GetConfigService() *config.Service {
	return configService
}

// rootCmd is the base command for the CLI.
// It holds the main description and version of the tool.
// This command is the entry point for subcommands such as "install", "list", "reset", etc.
//
// Example usage (from command-line):
//
//	nvs --help
var rootCmd = &cobra.Command{
	Use:     "nvs",
	Short:   "Neovim version switcher",
	Long:    "A CLI tool to install, switch, list, uninstall, and reset Neovim versions.",
	Version: Version,
}
