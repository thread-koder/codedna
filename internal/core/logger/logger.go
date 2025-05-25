// Package logger provides structured logging using zap.
//
// Example:
//
//	logger, _ := logger.New(logger.Config{
//		Level:     "debug",
//		Format:    "console",
//		Component: "cli",
//		Output:    "stdout",
//		File:      "logs/codedna.log",
//	})
//	defer logger.Sync()
//
//	logger.Info("Processing file")
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Level     string // debug, info, warn, error
	Format    string // json, console
	Component string // core, external, cli ...etc
	Output    string // stdout, file, both
	File      string // file path
}

func New(cfg Config) (*zap.Logger, error) {
	component := cfg.Component
	if component == "" {
		component = "unknown"
	}

	zapCfg := zap.Config{
		Level:       zap.NewAtomicLevelAt(getZapLevel(cfg.Level)),
		Development: false,
		Encoding:    getEncoderFormat(cfg.Format),
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      getOutputPaths(cfg.Output, cfg.File),
		ErrorOutputPaths: []string{"stderr"},
		InitialFields:    map[string]any{"component": component},
	}

	return zapCfg.Build()
}

func getOutputPaths(output, file string) []string {
	switch output {
	case "stdout":
		return []string{"stdout"}
	case "file":
		return []string{file}
	case "both":
		return []string{"stdout", file}
	default:
		return []string{"stdout"}
	}
}

func getZapLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func getEncoderFormat(format string) string {
	switch format {
	case "json":
		return "json"
	case "console":
		return "console"
	default:
		return "console"
	}
}
