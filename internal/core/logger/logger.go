// Package logger provides structured logging using zap.
//
// Example:
//
//	logger, _ := logger.New(logger.Config{
//		Level:     "debug",
//		Format:    "console",
//		Component: "cli",
//	})
//	defer logger.Sync()
//
//	logger.Info("Processing file")
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Level     string // debug, info, warn, error
	Format    string // json, console
	Component string // core, external, cli ...etc
}

// New creates a new logger with debug information
func New(cfg Config) (*zap.Logger, error) {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
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
	}

	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, err
	}

	var core zapcore.Core
	if cfg.Format == "json" {
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			level,
		)
	} else {
		core = zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			level,
		)
	}

	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.Fields(zap.String("component", cfg.Component)),
	)

	return logger, nil
}
