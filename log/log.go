package log

import (
	"log/slog"
	"os"

	"github.com/mrjoelkamp/opkl-updater/config"
)

// Logger defines a set of methods for writing application logs
type Logger interface {
	Debug(args ...interface{})
	Error(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Fatal(args ...interface{})
}

var defaultLogger *slog.Logger

func init() {
	defaultLogger = newSlogLogger(config.Config())
}

// NewLogger returns a configured slog instance
func NewLogger(cfg config.Provider) *slog.Logger {
	return newSlogLogger(cfg)
}

func newSlogLogger(cfg config.Provider) *slog.Logger {

	opts := new(slog.HandlerOptions)

	switch cfg.GetString("loglevel") {
	case "debug":
		opts.Level = slog.LevelDebug
	case "warning":
		opts.Level = slog.LevelWarn
	case "info":
		opts.Level = slog.LevelInfo
	default:
		opts.Level = slog.LevelDebug
	}

	if cfg.GetBool("json_logs") {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

// Debug package-level convenience method.
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Error package-level convenience method.
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Info package-level convenience method.
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn package-level convenience method.
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Fatal package-level convenience method.
func Fatal(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
	os.Exit(1)
}
