package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/y3owk1n/nvs/internal/app/config"
	appversion "github.com/y3owk1n/nvs/internal/app/version"
	"github.com/y3owk1n/nvs/internal/infra/archive"
	"github.com/y3owk1n/nvs/internal/infra/builder"
	"github.com/y3owk1n/nvs/internal/infra/downloader"
	"github.com/y3owk1n/nvs/internal/infra/filesystem"
	"github.com/y3owk1n/nvs/internal/infra/github"
	"github.com/y3owk1n/nvs/internal/infra/installer"
)

const (
	windows  = "windows"
	dirPerm  = 0o755
	cacheTTL = 5 * time.Minute
)

var (
	// verbose controls the log level.
	verbose bool

	// ctx is the global context used by the CLI.
	// cancel cancels the context, e.g. on interrupt signals.
	ctx, cancel = context.WithCancel(context.Background())

	// Services (initialized in InitConfig).
	versionService *appversion.Service
	configService  *config.Service

	// Configuration paths (initialized in InitConfig).
	versionsDir   string
	cacheFilePath string
	globalBinDir  string

	// Version of nvs, defaults to "v0.0.0" but may be set during build time.
	Version = "v0.0.0"
)

// Execute initializes the configuration, sets up global flags, and executes the root command.
// Example usage:
//
//	func main() {
//	    if err := cmd.Execute(); err != nil {
//	        os.Exit(1)
//	    }
//	}
func Execute() error {
	// Initialize configuration before running any commands.
	cobra.OnInitialize(InitConfig)

	// Set a persistent flag for verbose logging.
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

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
func InitConfig() {
	var err error

	// Set logging level based on the verbose flag.
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("Verbose mode enabled")
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	// Set up a signal handler to cancel the global context on an interrupt signal.
	signalOnce.Do(func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)

		go func() {
			<-sigCh

			_, err := fmt.Fprintln(os.Stdout)
			if err != nil {
				logrus.Warnf("Failed to write to stdout: %v", err)
			}

			logrus.Debug("Interrupt received, canceling operations...")
			cancel()
			os.Exit(1)
		}()
	})

	// Determine the base configuration directory.
	var baseConfigDir string
	if custom := os.Getenv("NVS_CONFIG_DIR"); custom != "" {
		baseConfigDir = custom
		logrus.Debugf("Using custom config directory from NVS_CONFIG_DIR: %s", baseConfigDir)
	} else {
		var (
			configDir string
			configErr error
		)

		configDir, configErr = os.UserConfigDir()
		if configErr == nil {
			baseConfigDir = filepath.Join(configDir, "nvs")
			logrus.Debugf("Using system config directory: %s", baseConfigDir)
		} else {
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				logrus.Fatalf("Failed to get user home directory: %v", homeErr)
			}

			baseConfigDir = filepath.Join(home, ".nvs")
			logrus.Debugf("Falling back to home directory for config: %s", baseConfigDir)
		}
	}

	// Ensure the configuration directory exists.
	err = os.MkdirAll(baseConfigDir, dirPerm)
	if err != nil {
		logrus.Fatalf("Failed to create config directory: %v", err)
	}

	logrus.Debugf("Config directory ensured: %s", baseConfigDir)

	// Set the directory for installed versions.
	versionsDir = filepath.Join(baseConfigDir, "versions")

	err = os.MkdirAll(versionsDir, dirPerm)
	if err != nil {
		logrus.Fatalf("Failed to create versions directory: %v", err)
	}

	logrus.Debugf("Versions directory ensured: %s", versionsDir)

	// Determine the base cache directory.
	var baseCacheDir string
	if custom := os.Getenv("NVS_CACHE_DIR"); custom != "" {
		baseCacheDir = custom
		logrus.Debugf("Using custom cache directory from NVS_CACHE_DIR: %s", baseCacheDir)
	} else {
		var (
			cacheDir string
			cacheErr error
		)

		cacheDir, cacheErr = os.UserCacheDir()
		if cacheErr == nil {
			baseCacheDir = filepath.Join(cacheDir, "nvs")
			logrus.Debugf("Using system cache directory: %s", baseCacheDir)
		} else {
			baseCacheDir = filepath.Join(baseConfigDir, "cache")
			logrus.Debugf("Falling back to config directory for cache: %s", baseCacheDir)
		}
	}
	// Ensure the cache directory exists.
	err = os.MkdirAll(baseCacheDir, dirPerm)
	if err != nil {
		logrus.Fatalf("Failed to create cache directory: %v", err)
	}

	cacheFilePath = filepath.Join(baseCacheDir, "releases.json")
	logrus.Debugf("Cache directory ensured: %s", baseCacheDir)
	logrus.Debugf("Cache file path set: %s", cacheFilePath)

	// Determine the base binary directory.
	var baseBinDir string
	if custom := os.Getenv("NVS_BIN_DIR"); custom != "" {
		baseBinDir = custom
		logrus.Debugf("Using custom binary directory from NVS_BIN_DIR: %s", baseBinDir)
	} else {
		if runtime.GOOS == windows {
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				logrus.Fatalf("Failed to get user home directory: %v", homeErr)
			}

			baseBinDir = filepath.Join(home, "AppData", "Local", "Programs")
			logrus.Debugf("Using Windows binary directory: %s", baseBinDir)
		} else {
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				logrus.Fatalf("Failed to get user home directory: %v", homeErr)
			}

			baseBinDir = filepath.Join(home, ".local", "bin")
			logrus.Debugf("Using default binary directory: %s", baseBinDir)
		}
	}
	// Ensure the binary directory exists.
	err = os.MkdirAll(baseBinDir, dirPerm)
	if err != nil {
		logrus.Fatalf("Failed to create binary directory: %v", err)
	}

	globalBinDir = baseBinDir
	logrus.Debugf("Global binary directory ensured: %s", globalBinDir)

	// Initialize services
	githubClient := github.NewClient(cacheFilePath, cacheTTL, "0.5.0")
	versionManager := filesystem.New(&filesystem.Config{
		VersionsDir:  versionsDir,
		GlobalBinDir: globalBinDir,
	})

	// Installer components
	dl := downloader.New()
	extractor := archive.New()
	srcBuilder := builder.New(nil) // nil for default exec command

	installService := installer.New(dl, extractor, srcBuilder)

	versionService, err = appversion.New(
		githubClient,
		versionManager,
		installService,
		&appversion.Config{
			VersionsDir:   versionsDir,
			CacheFilePath: cacheFilePath,
			GlobalBinDir:  globalBinDir,
		},
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create version service: %v", err))
	}

	configService = config.New()

	logrus.Debug("Services initialized")
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
func GetVersionService() *appversion.Service {
	return versionService
}

// SetVersionServiceForTesting sets the version service for testing.
// This should only be used in tests.
func SetVersionServiceForTesting(service *appversion.Service) {
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
