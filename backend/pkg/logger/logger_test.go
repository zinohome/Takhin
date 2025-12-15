// Copyright 2025 Takhin Data, Inc.

package logger

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "json format",
			config: Config{
				Level:  "info",
				Format: "json",
			},
		},
		{
			name: "text format",
			config: Config{
				Level:  "debug",
				Format: "text",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.config)
			assert.NotNil(t, logger)
			assert.NotNil(t, logger.Logger)
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		level string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"invalid", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			got := parseLevel(tt.level)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWithFields(t *testing.T) {
	logger := New(Config{Level: "info", Format: "json"})
	loggerWithFields := logger.WithFields("key1", "value1")
	assert.NotNil(t, loggerWithFields)
}
