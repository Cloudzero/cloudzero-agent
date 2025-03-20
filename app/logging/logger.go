package logging

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/cloudzero/cloudzero-insights-controller/app/build"
	"github.com/rs/zerolog"
)

type internalLogger struct {
	level   zerolog.Level
	sinks   []io.Writer
	hooks   []zerolog.Hook
	version string
}

type LoggerOpt = func(logger *internalLogger) error

// WithLevel parses the log level for the logger
func WithLevel(level string) LoggerOpt {
	return func(logger *internalLogger) error {
		// parse the level
		logLevel, err := zerolog.ParseLevel(level)
		if err != nil {
			return fmt.Errorf("failed to parse the log level: %w", err)
		}
		logger.level = logLevel
		return nil
	}
}

// WithSink attaches a sink to the logger. This can be called multiple times
func WithSink(sink io.Writer) LoggerOpt {
	return func(logger *internalLogger) error {
		logger.sinks = append(logger.sinks, sink)
		return nil
	}
}

// WithHook attaches a hook to the logger. This can be called multiple times
func WithHook(hook zerolog.Hook) LoggerOpt {
	return func(logger *internalLogger) error {
		logger.hooks = append(logger.hooks, hook)
		return nil
	}
}

// WithVersion overrides the default version fetched from the `build` library
func WithVersion(version string) LoggerOpt {
	return func(logger *internalLogger) error {
		logger.version = version
		return nil
	}
}

// NewLogger creates a new zerolog logger with the requested options
func NewLogger(opts ...LoggerOpt) (*zerolog.Logger, error) {
	ilogger := &internalLogger{
		level: zerolog.InfoLevel,
		sinks: make([]io.Writer, 0),
	}

	// apply the opts
	for _, opt := range opts {
		if err := opt(ilogger); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// add a default version
	if ilogger.version == "" {
		ilogger.version = build.GetVersion()
	}

	// add a default sink
	if len(ilogger.sinks) == 0 {
		ilogger.sinks = append(ilogger.sinks, os.Stdout)
	}

	// create a multi-sink
	multiSink := io.MultiWriter(ilogger.sinks...)

	// create the logger
	var zlogger zerolog.Logger
	zlogger = zerolog.New(multiSink)
	zlogger = zlogger.Level(ilogger.level).With().
		Str("version", ilogger.version).
		Timestamp().
		Caller().
		Logger()

	// add hooks
	for _, hook := range ilogger.hooks {
		zlogger = zlogger.Hook(hook)
	}

	// set as default context logger
	zerolog.DefaultContextLogger = &zlogger

	return &zlogger, nil
}

// bind the zerologger to the current context
func BindDefaultLoggerToContext(ctx context.Context) context.Context {
	return zerolog.DefaultContextLogger.WithContext(ctx)
}
