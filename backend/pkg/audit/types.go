// Copyright 2025 Takhin Data, Inc.

package audit

import (
	"time"
)

// EventType represents the type of audit event
type EventType string

const (
	// Authentication events
	EventTypeAuthSuccess EventType = "auth.success"
	EventTypeAuthFailure EventType = "auth.failure"
	EventTypeAuthLogout  EventType = "auth.logout"

	// ACL events
	EventTypeACLCreate EventType = "acl.create"
	EventTypeACLUpdate EventType = "acl.update"
	EventTypeACLDelete EventType = "acl.delete"
	EventTypeACLDeny   EventType = "acl.deny"

	// Topic events
	EventTypeTopicCreate EventType = "topic.create"
	EventTypeTopicDelete EventType = "topic.delete"
	EventTypeTopicUpdate EventType = "topic.update"

	// Consumer group events
	EventTypeGroupCreate EventType = "group.create"
	EventTypeGroupDelete EventType = "group.delete"
	EventTypeGroupJoin   EventType = "group.join"
	EventTypeGroupLeave  EventType = "group.leave"

	// Configuration events
	EventTypeConfigChange EventType = "config.change"
	EventTypeConfigRead   EventType = "config.read"

	// Data access events
	EventTypeDataProduce EventType = "data.produce"
	EventTypeDataConsume EventType = "data.consume"
	EventTypeDataDelete  EventType = "data.delete"

	// System events
	EventTypeSystemStartup  EventType = "system.startup"
	EventTypeSystemShutdown EventType = "system.shutdown"
	EventTypeSystemError    EventType = "system.error"
)

// Severity represents the severity level of an audit event
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Event represents a single audit log event
type Event struct {
	// Core fields
	Timestamp time.Time `json:"timestamp"`
	EventID   string    `json:"event_id"`
	EventType EventType `json:"event_type"`
	Severity  Severity  `json:"severity"`

	// Actor information
	Principal string `json:"principal"`           // User/service principal
	Host      string `json:"host"`                // Source IP/hostname
	UserAgent string `json:"user_agent,omitempty"` // Client user agent

	// Resource information
	ResourceType string `json:"resource_type,omitempty"` // topic, group, acl, etc.
	ResourceName string `json:"resource_name,omitempty"` // Specific resource identifier

	// Operation details
	Operation string                 `json:"operation"`           // create, delete, read, write
	Result    string                 `json:"result"`              // success, failure, denied
	Metadata  map[string]interface{} `json:"metadata,omitempty"`  // Additional context
	Error     string                 `json:"error,omitempty"`     // Error message if failed

	// Request context
	RequestID   string `json:"request_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	APIVersion  string `json:"api_version,omitempty"`
	Duration    int64  `json:"duration_ms,omitempty"` // Operation duration in ms
}

// Filter represents query filters for audit logs
type Filter struct {
	StartTime    *time.Time
	EndTime      *time.Time
	EventTypes   []EventType
	Principals   []string
	ResourceType string
	ResourceName string
	Result       string
	Severity     Severity
	Limit        int
	Offset       int
}
