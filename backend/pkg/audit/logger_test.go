// Copyright 2025 Takhin Data, Inc.

package audit

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_Log(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	cfg := Config{
		Enabled:     true,
		OutputPath:  logPath,
		MaxFileSize: 1024 * 1024,
		MaxBackups:  5,
		MaxAge:      7,
		Compress:    false,
		StoreEnabled: true,
	}

	logger, err := NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Test basic logging
	event := &Event{
		EventType:    EventTypeTopicCreate,
		Severity:     SeverityInfo,
		Principal:    "admin",
		Host:         "localhost",
		ResourceType: "topic",
		ResourceName: "test-topic",
		Operation:    "create",
		Result:       "success",
	}

	err = logger.Log(event)
	assert.NoError(t, err)

	// Verify event was logged to file
	data, err := os.ReadFile(logPath)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "test-topic")
	assert.Contains(t, string(data), "topic.create")
}

func TestLogger_LogAuth(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	cfg := Config{
		Enabled:      true,
		OutputPath:   logPath,
		StoreEnabled: true,
	}

	logger, err := NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Test successful authentication
	err = logger.LogAuth("user1", "192.168.1.100", "success", "test-key-123", nil)
	assert.NoError(t, err)

	// Test failed authentication
	err = logger.LogAuth("user2", "192.168.1.101", "failure", "bad-key", errors.New("invalid credentials"))
	assert.NoError(t, err)

	// Query events
	events, err := logger.Query(Filter{
		EventTypes: []EventType{EventTypeAuthSuccess, EventTypeAuthFailure},
	})
	assert.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestLogger_LogACL(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	cfg := Config{
		Enabled:      true,
		OutputPath:   logPath,
		StoreEnabled: true,
	}

	logger, err := NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Test ACL create
	err = logger.LogACL("create", "admin", "localhost", "topic", "test-topic", "success", nil)
	assert.NoError(t, err)

	// Test ACL deny
	err = logger.LogACL("deny", "user1", "192.168.1.100", "topic", "restricted-topic", "denied", nil)
	assert.NoError(t, err)

	// Query ACL events
	events, err := logger.Query(Filter{
		EventTypes: []EventType{EventTypeACLCreate, EventTypeACLDeny},
	})
	assert.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestLogger_LogTopic(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	cfg := Config{
		Enabled:      true,
		OutputPath:   logPath,
		StoreEnabled: true,
	}

	logger, err := NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Test topic create
	err = logger.LogTopic("create", "admin", "localhost", "new-topic", 3, "success", nil)
	assert.NoError(t, err)

	// Test topic delete
	err = logger.LogTopic("delete", "admin", "localhost", "old-topic", 0, "success", nil)
	assert.NoError(t, err)

	// Query topic events
	events, err := logger.Query(Filter{
		ResourceType: "topic",
	})
	assert.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestLogger_LogDataAccess(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	cfg := Config{
		Enabled:      true,
		OutputPath:   logPath,
		StoreEnabled: true,
	}

	logger, err := NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Test produce
	err = logger.LogDataAccess("produce", "producer1", "192.168.1.100", "data-topic", 0, 1000, 2048)
	assert.NoError(t, err)

	// Test consume
	err = logger.LogDataAccess("consume", "consumer1", "192.168.1.101", "data-topic", 0, 1000, 2048)
	assert.NoError(t, err)

	// Query data access events
	events, err := logger.Query(Filter{
		EventTypes: []EventType{EventTypeDataProduce, EventTypeDataConsume},
	})
	assert.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestLogger_Query(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	cfg := Config{
		Enabled:      true,
		OutputPath:   logPath,
		StoreEnabled: true,
	}

	logger, err := NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Add multiple events
	for i := 0; i < 10; i++ {
		event := &Event{
			EventType:    EventTypeTopicCreate,
			Severity:     SeverityInfo,
			Principal:    "admin",
			Host:         "localhost",
			ResourceType: "topic",
			ResourceName: "topic-" + string(rune('0'+i)),
			Operation:    "create",
			Result:       "success",
		}
		err = logger.Log(event)
		assert.NoError(t, err)
	}

	// Query with limit
	events, err := logger.Query(Filter{
		Limit: 5,
	})
	assert.NoError(t, err)
	assert.Len(t, events, 5)

	// Query by principal
	events, err = logger.Query(Filter{
		Principals: []string{"admin"},
	})
	assert.NoError(t, err)
	assert.Len(t, events, 10)

	// Query by resource type
	events, err = logger.Query(Filter{
		ResourceType: "topic",
	})
	assert.NoError(t, err)
	assert.Len(t, events, 10)
}

func TestLogger_Disabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	// Logging should be a no-op
	event := &Event{
		EventType: EventTypeTopicCreate,
		Severity:  SeverityInfo,
		Principal: "admin",
	}

	err = logger.Log(event)
	assert.NoError(t, err)

	// Query should fail
	_, err = logger.Query(Filter{})
	assert.Error(t, err)
}

func TestStore_Cleanup(t *testing.T) {
	retentionMs := int64(100) // 100ms retention
	store := NewStore(retentionMs)

	// Add events with different ages
	for i := 0; i < 5; i++ {
		event := &Event{
			Timestamp: time.Now().Add(-time.Duration(i*50) * time.Millisecond),
			EventType: EventTypeTopicCreate,
			Principal: "admin",
		}
		store.Add(event)
	}

	assert.Equal(t, 5, store.Count())

	// Wait for some events to expire
	time.Sleep(150 * time.Millisecond)

	// Cleanup
	store.Cleanup()

	// Should have removed old events (those older than 100ms)
	assert.LessOrEqual(t, store.Count(), 3, "should have removed old events")
}

func TestRotator(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	cfg := RotatorConfig{
		Filename:   logPath,
		MaxSize:    100, // Small size to trigger rotation
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
	}

	rotator, err := NewRotator(cfg)
	require.NoError(t, err)
	defer rotator.Close()

	// Write data to trigger rotation
	data := make([]byte, 150)
	for i := range data {
		data[i] = 'A'
	}

	n, err := rotator.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)

	// Check that backup file was created
	files, err := os.ReadDir(tmpDir)
	assert.NoError(t, err)
	assert.Greater(t, len(files), 1, "should have created backup file")
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "normal key",
			key:      "test-key-12345",
			expected: "test****",
		},
		{
			name:     "short key",
			key:      "abc",
			expected: "****",
		},
		{
			name:     "empty key",
			key:      "",
			expected: "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskAPIKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}
