// Copyright 2025 Takhin Data, Inc.

package schema

import (
	"fmt"
	"time"
)

// SchemaType represents the type of schema format
type SchemaType string

const (
	SchemaTypeAvro     SchemaType = "AVRO"
	SchemaTypeJSON     SchemaType = "JSON"
	SchemaTypeProtobuf SchemaType = "PROTOBUF"
)

// CompatibilityMode defines how schema compatibility is checked
type CompatibilityMode string

const (
	CompatibilityNone            CompatibilityMode = "NONE"
	CompatibilityBackward        CompatibilityMode = "BACKWARD"
	CompatibilityBackwardTransit CompatibilityMode = "BACKWARD_TRANSITIVE"
	CompatibilityForward         CompatibilityMode = "FORWARD"
	CompatibilityForwardTransit  CompatibilityMode = "FORWARD_TRANSITIVE"
	CompatibilityFull            CompatibilityMode = "FULL"
	CompatibilityFullTransit     CompatibilityMode = "FULL_TRANSITIVE"
)

// Schema represents a versioned schema
type Schema struct {
	ID         int               `json:"id"`
	Subject    string            `json:"subject"`
	Version    int               `json:"version"`
	SchemaType SchemaType        `json:"schemaType"`
	Schema     string            `json:"schema"`
	References []SchemaReference `json:"references,omitempty"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}

// SchemaReference represents a reference to another schema
type SchemaReference struct {
	Name    string `json:"name"`
	Subject string `json:"subject"`
	Version int    `json:"version"`
}

// SubjectVersion uniquely identifies a schema version
type SubjectVersion struct {
	Subject string
	Version int
}

// SchemaMetadata contains schema metadata without the full schema content
type SchemaMetadata struct {
	ID         int        `json:"id"`
	Subject    string     `json:"subject"`
	Version    int        `json:"version"`
	SchemaType SchemaType `json:"schemaType"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// SchemaError represents schema registry errors
type SchemaError struct {
	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
}

func (e *SchemaError) Error() string {
	return fmt.Sprintf("schema error %d: %s", e.ErrorCode, e.Message)
}

// Error codes matching Confluent Schema Registry
const (
	ErrCodeSubjectNotFound           = 40401
	ErrCodeVersionNotFound           = 40402
	ErrCodeSchemaNotFound            = 40403
	ErrCodeIncompatibleSchema        = 409
	ErrCodeInvalidSchema             = 42201
	ErrCodeInvalidVersion            = 42202
	ErrCodeInvalidCompatibilityLevel = 42203
	ErrCodeSubjectLevelNotFound      = 40408
	ErrCodeSchemaReferencesNotFound  = 42205
)

// NewSchemaError creates a new schema error
func NewSchemaError(code int, message string) *SchemaError {
	return &SchemaError{ErrorCode: code, Message: message}
}

// Config represents schema registry configuration
type Config struct {
	DataDir              string            `koanf:"data.dir"`
	DefaultCompatibility CompatibilityMode `koanf:"default.compatibility"`
	MaxVersions          int               `koanf:"max.versions"`
	CacheSize            int               `koanf:"cache.size"`
}
