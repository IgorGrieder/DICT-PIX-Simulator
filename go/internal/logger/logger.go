package logger

import (
	otellog "go.opentelemetry.io/otel/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global logger instance
var Log *zap.Logger

// Init initializes the Zap logger with JSON output
func Init(env string, _ otellog.LoggerProvider) error {
	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: env == "development",
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.MillisDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	var err error
	Log, err = config.Build()
	if err != nil {
		return err
	}

	zap.ReplaceGlobals(Log)
	return nil
}

// Sync flushes any buffered log entries
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	if Log == nil {
		return
	}
	Log.WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	if Log == nil {
		return
	}
	Log.WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	if Log == nil {
		return
	}
	Log.WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	if Log == nil {
		return
	}
	Log.WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	if Log == nil {
		return
	}
	Log.WithOptions(zap.AddCallerSkip(1)).Fatal(msg, fields...)
}
