package logger

import (
	"go.opentelemetry.io/contrib/bridges/otelzap"
	otellog "go.opentelemetry.io/otel/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global logger instance
var Log *zap.Logger

// Init initializes the Zap logger with JSON output and optional OTEL integration
// If logProvider is not nil, logs will be exported to OpenTelemetry as well as stdout
func Init(env string, logProvider otellog.LoggerProvider) error {
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

	// Build the base logger for stdout
	stdoutLogger, err := config.Build()
	if err != nil {
		return err
	}

	stdoutCore := stdoutLogger.Core()

	// Create otelzap core for OTEL export
	otelCore := otelzap.NewCore("dict-simulator",
		otelzap.WithLoggerProvider(logProvider),
		otelzap.WithVersion("1.0.0"),
	)

	// Combine both cores: logs go to stdout AND OTEL
	combinedCore := zapcore.NewTee(stdoutCore, otelCore)
	Log = zap.New(combinedCore, zap.AddCaller())

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
	Log.WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	Log.WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	Log.WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	Log.WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	Log.WithOptions(zap.AddCallerSkip(1)).Fatal(msg, fields...)
}
