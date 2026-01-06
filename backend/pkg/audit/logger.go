// Copyright 2025 Takhin Data, Inc.

package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/takhin-data/takhin/pkg/logger"
)

// Logger handles audit log operations
type Logger struct {
	mu        sync.RWMutex
	writer    io.Writer
	config    Config
	logger    *logger.Logger
	store     *Store
	rotator   *Rotator
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// Config holds audit logger configuration
type Config struct {
	Enabled          bool   `koanf:"enabled"`
	OutputPath       string `koanf:"output.path"`
	MaxFileSize      int64  `koanf:"max.file.size"`      // Max size in bytes before rotation
	MaxBackups       int    `koanf:"max.backups"`        // Max number of backup files
	MaxAge           int    `koanf:"max.age"`            // Max age in days
	Compress         bool   `koanf:"compress"`           // Compress rotated files
	BufferSize       int    `koanf:"buffer.size"`        // Buffer size for async writes
	FlushIntervalMs  int    `koanf:"flush.interval.ms"`  // Flush interval in ms
	StoreEnabled     bool   `koanf:"store.enabled"`      // Enable queryable store
	StoreRetentionMs int64  `koanf:"store.retention.ms"` // Store retention period in ms
}

// NewLogger creates a new audit logger
func NewLogger(config Config) (*Logger, error) {
	if !config.Enabled {
		return &Logger{
			config: config,
			logger: logger.Default().WithComponent("audit"),
		}, nil
	}

	// Set defaults
	if config.OutputPath == "" {
		config.OutputPath = "/var/log/takhin/audit.log"
	}
	if config.MaxFileSize == 0 {
		config.MaxFileSize = 100 * 1024 * 1024 // 100MB
	}
	if config.MaxBackups == 0 {
		config.MaxBackups = 10
	}
	if config.MaxAge == 0 {
		config.MaxAge = 30 // 30 days
	}
	if config.BufferSize == 0 {
		config.BufferSize = 1000
	}
	if config.FlushIntervalMs == 0 {
		config.FlushIntervalMs = 1000 // 1 second
	}
	if config.StoreRetentionMs == 0 {
		config.StoreRetentionMs = 7 * 24 * 60 * 60 * 1000 // 7 days
	}

	// Ensure output directory exists
	dir := filepath.Dir(config.OutputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create audit log directory: %w", err)
	}

	// Create rotator
	rotator, err := NewRotator(RotatorConfig{
		Filename:   config.OutputPath,
		MaxSize:    config.MaxFileSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	})
	if err != nil {
		return nil, fmt.Errorf("create log rotator: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	l := &Logger{
		writer:  rotator,
		config:  config,
		logger:  logger.Default().WithComponent("audit"),
		rotator: rotator,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Initialize store if enabled
	if config.StoreEnabled {
		l.store = NewStore(config.StoreRetentionMs)
		l.wg.Add(1)
		go l.cleanupLoop()
	}

	return l, nil
}

// Log writes an audit event
func (l *Logger) Log(event *Event) error {
	if !l.config.Enabled {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Set defaults
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Severity == "" {
		event.Severity = SeverityInfo
	}

	// Write to file
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	data = append(data, '\n')
	if _, err := l.writer.Write(data); err != nil {
		l.logger.Error("failed to write audit log", "error", err)
		return fmt.Errorf("write audit log: %w", err)
	}

	// Store in memory if enabled
	if l.store != nil {
		l.store.Add(event)
	}

	return nil
}

// LogAuth logs an authentication event
func (l *Logger) LogAuth(principal, host, result, apiKey string, err error) error {
	eventType := EventTypeAuthSuccess
	severity := SeverityInfo
	errMsg := ""

	if err != nil {
		eventType = EventTypeAuthFailure
		severity = SeverityWarning
		errMsg = err.Error()
	}

	return l.Log(&Event{
		EventType: eventType,
		Severity:  severity,
		Principal: principal,
		Host:      host,
		Operation: "authenticate",
		Result:    result,
		Error:     errMsg,
		Metadata: map[string]interface{}{
			"api_key_prefix": maskAPIKey(apiKey),
		},
	})
}

// LogACL logs an ACL operation
func (l *Logger) LogACL(operation, principal, host, resourceType, resourceName string, result string, err error) error {
	var eventType EventType
	severity := SeverityInfo
	errMsg := ""

	switch operation {
	case "create":
		eventType = EventTypeACLCreate
	case "update":
		eventType = EventTypeACLUpdate
	case "delete":
		eventType = EventTypeACLDelete
	case "deny":
		eventType = EventTypeACLDeny
		severity = SeverityWarning
	default:
		eventType = EventType(fmt.Sprintf("acl.%s", operation))
	}

	if err != nil {
		severity = SeverityError
		errMsg = err.Error()
	}

	return l.Log(&Event{
		EventType:    eventType,
		Severity:     severity,
		Principal:    principal,
		Host:         host,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Operation:    operation,
		Result:       result,
		Error:        errMsg,
	})
}

// LogTopic logs a topic operation
func (l *Logger) LogTopic(operation, principal, host, topicName string, partitions int32, result string, err error) error {
	var eventType EventType
	severity := SeverityInfo
	errMsg := ""

	switch operation {
	case "create":
		eventType = EventTypeTopicCreate
	case "delete":
		eventType = EventTypeTopicDelete
	case "update":
		eventType = EventTypeTopicUpdate
	default:
		eventType = EventType(fmt.Sprintf("topic.%s", operation))
	}

	if err != nil {
		severity = SeverityError
		errMsg = err.Error()
	}

	return l.Log(&Event{
		EventType:    eventType,
		Severity:     severity,
		Principal:    principal,
		Host:         host,
		ResourceType: "topic",
		ResourceName: topicName,
		Operation:    operation,
		Result:       result,
		Error:        errMsg,
		Metadata: map[string]interface{}{
			"partitions": partitions,
		},
	})
}

// LogDataAccess logs data access events (produce/consume)
func (l *Logger) LogDataAccess(operation, principal, host, topicName string, partition int32, offset int64, size int64) error {
	var eventType EventType
	switch operation {
	case "produce":
		eventType = EventTypeDataProduce
	case "consume":
		eventType = EventTypeDataConsume
	case "delete":
		eventType = EventTypeDataDelete
	default:
		eventType = EventType(fmt.Sprintf("data.%s", operation))
	}

	return l.Log(&Event{
		EventType:    eventType,
		Severity:     SeverityInfo,
		Principal:    principal,
		Host:         host,
		ResourceType: "topic",
		ResourceName: topicName,
		Operation:    operation,
		Result:       "success",
		Metadata: map[string]interface{}{
			"partition": partition,
			"offset":    offset,
			"size":      size,
		},
	})
}

// Query queries audit logs with filters
func (l *Logger) Query(filter Filter) ([]*Event, error) {
	if !l.config.Enabled || !l.config.StoreEnabled || l.store == nil {
		return nil, fmt.Errorf("audit log store not enabled")
	}

	return l.store.Query(filter), nil
}

// Close closes the audit logger
func (l *Logger) Close() error {
	if !l.config.Enabled {
		return nil
	}

	l.cancel()
	l.wg.Wait()

	if l.rotator != nil {
		return l.rotator.Close()
	}

	return nil
}

// cleanupLoop periodically cleans up old entries from the store
func (l *Logger) cleanupLoop() {
	defer l.wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-l.ctx.Done():
			return
		case <-ticker.C:
			if l.store != nil {
				l.store.Cleanup()
			}
		}
	}
}

// maskAPIKey masks an API key for logging (shows only first 4 chars)
func maskAPIKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return key[:4] + "****"
}
