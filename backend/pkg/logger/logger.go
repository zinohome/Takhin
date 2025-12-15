// Copyright 2025 Takhin Data, Inc.

package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type Logger struct {
	*slog.Logger
}

type Config struct {
	Level  string
	Format string
}

func New(cfg Config) *Logger {
	level := parseLevel(cfg.Level)

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	switch strings.ToLower(cfg.Format) {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	return &Logger{Logger: logger}
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{Logger: l.Logger.With()}
}

func (l *Logger) WithFields(fields ...any) *Logger {
	return &Logger{Logger: l.Logger.With(fields...)}
}

func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{Logger: l.Logger.With("component", component)}
}

func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{Logger: l.Logger.With("request_id", requestID)}
}

func (l *Logger) Fatal(msg string, args ...any) {
	l.Logger.Error(msg, args...)
	os.Exit(1)
}

var defaultLogger = New(Config{
	Level:  "info",
	Format: "json",
})

func SetDefault(logger *Logger) {
	defaultLogger = logger
	slog.SetDefault(logger.Logger)
}

func Default() *Logger {
	return defaultLogger
}

func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

func Fatal(msg string, args ...any) {
	defaultLogger.Fatal(msg, args...)
}
