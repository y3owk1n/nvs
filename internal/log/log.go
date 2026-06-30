// Package log is the nvs developer-facing logger. It wraps
// github.com/charmbracelet/log so the rest of the codebase has
// a small, stable surface to write debug, warn, and error
// traces without committing to a specific backend.
//
// User-facing output (the lines a `nvs <subcommand>` end user
// sees) does NOT go through this package — it goes through
// internal/ui/message. The split is deliberate:
//
//	internal/ui/message  -> "what the user reads"  (always on)
//	internal/log         -> "what the developer reads" (off by default)
//
// By default the logger is at WarnLevel and writes to stderr.
// `nvs -v` (or NVS_LOG=debug) raises it to DebugLevel; NVS_LOG_FILE
// redirects all developer traces to a file so the terminal UI
// (spinners, panels) stays clean even while debugging.
//
// The styling (color of each level prefix) is wired to the nvs
// palette in internal/ui/style, so the developer log looks like
// it belongs to the same product as the rest of the CLI.
//
// Package name "log" is the conventional name for a logger in
// the Go ecosystem; the import path (internal/log) makes the
// dependency clear at the call site.
package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	charmlog "github.com/charmbracelet/log"
	"github.com/y3owk1n/nvs/internal/ui/style"
)

// logFileMode is the permission used for files opened by
// NVS_LOG_FILE. Owner read/write only is the right default for
// trace logs that may contain repository paths or partial
// command lines.
const logFileMode os.FileMode = 0o600

// ErrUnknownLogLevel is returned by ParseLevel when the input
// does not match any recognized level name. The error message
// includes the offending input; use errors.Is to detect the
// condition without comparing strings.
var ErrUnknownLogLevel = errors.New("unknown log level")

// Level mirrors charmlog.Level so callers do not have to
// import the upstream package.
type Level = charmlog.Level

// Re-exported levels.
const (
	DebugLevel = charmlog.DebugLevel
	InfoLevel  = charmlog.InfoLevel
	WarnLevel  = charmlog.WarnLevel
	ErrorLevel = charmlog.ErrorLevel
	FatalLevel = charmlog.FatalLevel
)

// Options configure the logger at Init time.
type Options struct {
	// Level is the minimum level that will be emitted.
	// Zero value is WarnLevel (the production default).
	Level Level

	// Output is the destination writer. If nil, os.Stderr.
	Output io.Writer

	// FilePath, if non-empty, opens the file in append mode
	// and tees output to it. Used for NVS_LOG_FILE so a
	// developer can capture debug traces without polluting
	// the terminal UI. Mutually compatible with Output:
	// when both are set, output is duplicated to both.
	FilePath string

	// NoColor disables ANSI color escapes regardless of TTY
	// detection. Pulled from style.ColorEnabled() at Init
	// time; exposed here for testability.
	NoColor bool
}

var (
	mutex   sync.RWMutex
	current *charmlog.Logger
	closer  io.Closer
)

// Init builds the package-level logger from opts. It is safe
// to call multiple times; each call replaces the previous
// logger and closes any file opened by a prior call.
//
// Callers should invoke Init exactly once early in cmd
// initialization (after flag parsing). Sub-packages can then
// use the package-level Debug/Info/... helpers without any
// further wiring.
func Init(opts Options) error {
	mutex.Lock()
	defer mutex.Unlock()

	// Close the previously opened file, if any.
	if closer != nil {
		_ = closer.Close()
		closer = nil
	}

	out := opts.Output
	if out == nil {
		out = os.Stderr
	}

	// Tee to a file when requested. The file is opened in
	// append mode with restrictive permissions; if the open
	// fails we surface the error to the caller so the user
	// sees why NVS_LOG_FILE was ignored. We deliberately do
	// not silently fall back: a misconfigured log path is
	// the kind of bug you want to learn about loudly.
	if opts.FilePath != "" {
		file, err := os.OpenFile(
			opts.FilePath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			logFileMode,
		)
		if err != nil {
			return fmt.Errorf("open log file %q: %w", opts.FilePath, err)
		}

		closer = file
		out = io.MultiWriter(out, file)
	}

	logger := charmlog.NewWithOptions(out, charmlog.Options{
		ReportTimestamp: true,
		TimeFormat:      "15:04:05",
		Level:           opts.Level,
		Prefix:          "",
	})

	if !opts.NoColor {
		logger.SetStyles(styles())
	} else {
		// A neutral style set with all colors stripped, so
		// the level prefixes still align but no ANSI escapes
		// reach the writer (correct for NO_COLOR and for
		// file output without a TTY).
		logger.SetStyles(charmlog.DefaultStyles())
		logger.SetColorProfile(0) // termenv.Ascii
	}

	current = logger

	return nil
}

// Close releases any resources held by the logger (currently:
// the NVS_LOG_FILE handle, if any). Safe to call multiple
// times. Intended to be deferred from main.
func Close() error {
	mutex.Lock()
	defer mutex.Unlock()

	if closer == nil {
		return nil
	}

	err := closer.Close()
	closer = nil

	return err
}

// SetLevel adjusts the level of the active logger at runtime.
// Useful in tests; production code should set the level once
// via Init.
func SetLevel(level Level) {
	mutex.RLock()
	defer mutex.RUnlock()

	if current != nil {
		current.SetLevel(level)
	}
}

// GetLevel reports the active logger's current level.
func GetLevel() Level {
	mutex.RLock()
	defer mutex.RUnlock()

	if current == nil {
		return WarnLevel
	}

	return current.GetLevel()
}

// ParseLevel converts a level name ("debug", "info", "warn",
// "error", "fatal") into a Level. Case-insensitive. Unknown
// values return WarnLevel wrapped ErrUnknownLogLevel so the
// caller can decide whether to surface or silently fall back.
func ParseLevel(name string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "debug", "trace":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn", "warning", "":
		return WarnLevel, nil
	case "error", "err":
		return ErrorLevel, nil
	case "fatal":
		return FatalLevel, nil
	default:
		return WarnLevel, fmt.Errorf("%w: %q", ErrUnknownLogLevel, name)
	}
}

// logger returns the active logger, initializing a default one
// on first use. This makes the package usable before Init is
// called (e.g. from a test that imports a package which logs
// in its init), at the cost of one lazy default.
func logger() *charmlog.Logger {
	mutex.RLock()

	if current != nil {
		defer mutex.RUnlock()

		return current
	}

	mutex.RUnlock()

	mutex.Lock()
	defer mutex.Unlock()

	if current == nil {
		current = charmlog.NewWithOptions(os.Stderr, charmlog.Options{
			ReportTimestamp: true,
			TimeFormat:      "15:04:05",
			Level:           WarnLevel,
		})
		current.SetStyles(styles())
	}

	return current
}

// Debug logs a structured debug message. Keyvals are
// alternating key/value pairs (key1, val1, key2, val2, ...).
func Debug(msg string, keyvals ...any) {
	logger().Helper()
	logger().Debug(msg, keyvals...)
}

// Info logs a structured info message.
func Info(msg string, keyvals ...any) {
	logger().Helper()
	logger().Info(msg, keyvals...)
}

// Warn logs a structured warning.
func Warn(msg string, keyvals ...any) {
	logger().Helper()
	logger().Warn(msg, keyvals...)
}

// Error logs a structured error.
func Error(msg string, keyvals ...any) {
	logger().Helper()
	logger().Error(msg, keyvals...)
}

// Fatal logs a structured fatal message and exits the
// process. Reserved for unrecoverable errors during startup.
func Fatal(msg string, keyvals ...any) {
	logger().Helper()
	logger().Fatal(msg, keyvals...)
}

// Debugf is the printf-style counterpart of Debug, provided to
// ease migration from logrus and to keep call sites short when
// there's no natural key/value pair to add.
func Debugf(format string, args ...any) {
	logger().Helper()
	logger().Debugf(format, args...)
}

// Infof is the printf-style counterpart of Info.
func Infof(format string, args ...any) {
	logger().Helper()
	logger().Infof(format, args...)
}

// Warnf is the printf-style counterpart of Warn.
func Warnf(format string, args ...any) {
	logger().Helper()
	logger().Warnf(format, args...)
}

// Errorf is the printf-style counterpart of Error.
func Errorf(format string, args ...any) {
	logger().Helper()
	logger().Errorf(format, args...)
}

// Fatalf is the printf-style counterpart of Fatal.
func Fatalf(format string, args ...any) {
	logger().Helper()
	logger().Fatalf(format, args...)
}

// With returns a child logger that carries the supplied
// key/value pairs on every record. Useful for adding a stable
// context (e.g. "version=stable") to a sequence of calls.
func With(keyvals ...any) *charmlog.Logger {
	return logger().With(keyvals...)
}

// styles returns the level-prefix styles wired to the nvs
// palette. Kept private because the choice of colors is part
// of the design system, not part of the logging contract.
func styles() *charmlog.Styles {
	palette := style.Default()
	styles := charmlog.DefaultStyles()

	// Charm log defaults are good but use generic colors.
	// Rebind each level to our palette so a verbose run feels
	// like part of the same product as the rest of the CLI.
	styles.Levels[charmlog.DebugLevel] = styles.Levels[charmlog.DebugLevel].
		SetString("DEBU").
		Foreground(palette.Muted)

	styles.Levels[charmlog.InfoLevel] = styles.Levels[charmlog.InfoLevel].
		SetString("INFO").
		Foreground(palette.Accent)

	styles.Levels[charmlog.WarnLevel] = styles.Levels[charmlog.WarnLevel].
		SetString("WARN").
		Foreground(palette.Warning)

	styles.Levels[charmlog.ErrorLevel] = styles.Levels[charmlog.ErrorLevel].
		SetString("ERRO").
		Foreground(palette.Error)

	styles.Levels[charmlog.FatalLevel] = styles.Levels[charmlog.FatalLevel].
		SetString("FATA").
		Foreground(palette.Error).
		Bold(true)

	return styles
}
