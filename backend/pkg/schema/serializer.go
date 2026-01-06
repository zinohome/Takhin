// Copyright 2025 Takhin Data, Inc.

package schema

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

const (
	magicByte byte = 0x0
)

// Serializer serializes data with schema registry integration
type Serializer struct {
	registry   *Registry
	subject    string
	schemaType SchemaType
	autoRegister bool
}

// NewSerializer creates a new serializer
func NewSerializer(registry *Registry, subject string, schemaType SchemaType, autoRegister bool) *Serializer {
	return &Serializer{
		registry:     registry,
		subject:      subject,
		schemaType:   schemaType,
		autoRegister: autoRegister,
	}
}

// Serialize serializes data with schema ID embedded
// Wire format: [magic_byte (1)][schema_id (4)][data (N)]
func (s *Serializer) Serialize(schema string, data []byte) ([]byte, error) {
	var schemaID int

	if s.autoRegister {
		// Check if schema already exists first
		latestSchema, err := s.registry.GetLatestSchema(s.subject)
		if err == nil && latestSchema.Schema == schema {
			// Schema already exists and matches - reuse it
			schemaID = latestSchema.ID
		} else {
			// Register new schema
			registeredSchema, err := s.registry.RegisterSchema(s.subject, schema, s.schemaType, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to register schema: %w", err)
			}
			schemaID = registeredSchema.ID
		}
	} else {
		// Get existing schema
		latestSchema, err := s.registry.GetLatestSchema(s.subject)
		if err != nil {
			return nil, fmt.Errorf("failed to get schema: %w", err)
		}
		schemaID = latestSchema.ID
	}

	// Build wire format
	result := make([]byte, 5+len(data))
	result[0] = magicByte
	binary.BigEndian.PutUint32(result[1:5], uint32(schemaID))
	copy(result[5:], data)

	return result, nil
}

// SerializeAvro serializes Avro data with automatic schema registration
func (s *Serializer) SerializeAvro(schema string, data interface{}) ([]byte, error) {
	// Serialize data to JSON (simplified - in production use proper Avro encoding)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	return s.Serialize(schema, jsonData)
}

// SerializeJSON serializes JSON data with schema
func (s *Serializer) SerializeJSON(schema string, data interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	return s.Serialize(schema, jsonData)
}

// Deserializer deserializes data with schema validation
type Deserializer struct {
	registry *Registry
	validate bool
}

// NewDeserializer creates a new deserializer
func NewDeserializer(registry *Registry, validate bool) *Deserializer {
	return &Deserializer{
		registry: registry,
		validate: validate,
	}
}

// Deserialize deserializes data and extracts schema ID
// Returns: schema ID, data, error
func (d *Deserializer) Deserialize(wireData []byte) (int, []byte, error) {
	if len(wireData) < 5 {
		return 0, nil, fmt.Errorf("data too short: expected at least 5 bytes, got %d", len(wireData))
	}

	// Check magic byte
	if wireData[0] != magicByte {
		return 0, nil, fmt.Errorf("invalid magic byte: expected %d, got %d", magicByte, wireData[0])
	}

	// Extract schema ID
	schemaID := int(binary.BigEndian.Uint32(wireData[1:5]))

	// Validate schema exists if validation enabled
	if d.validate {
		_, err := d.registry.GetSchemaByID(schemaID)
		if err != nil {
			return 0, nil, fmt.Errorf("schema validation failed: %w", err)
		}
	}

	// Extract data
	data := wireData[5:]

	return schemaID, data, nil
}

// DeserializeWithSchema deserializes and returns both schema and data
func (d *Deserializer) DeserializeWithSchema(wireData []byte) (*Schema, []byte, error) {
	schemaID, data, err := d.Deserialize(wireData)
	if err != nil {
		return nil, nil, err
	}

	schema, err := d.registry.GetSchemaByID(schemaID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get schema: %w", err)
	}

	return schema, data, nil
}

// DeserializeJSON deserializes JSON data with schema
func (d *Deserializer) DeserializeJSON(wireData []byte, target interface{}) (*Schema, error) {
	schema, data, err := d.DeserializeWithSchema(wireData)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, target); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return schema, nil
}

// ProducerInterceptor intercepts produce requests for schema registration
type ProducerInterceptor struct {
	serializer *Serializer
	enabled    bool
}

// NewProducerInterceptor creates a new producer interceptor
func NewProducerInterceptor(registry *Registry, subject string, schemaType SchemaType) *ProducerInterceptor {
	return &ProducerInterceptor{
		serializer: NewSerializer(registry, subject, schemaType, true),
		enabled:    true,
	}
}

// Intercept intercepts and transforms message data
func (p *ProducerInterceptor) Intercept(schema string, data []byte) ([]byte, error) {
	if !p.enabled {
		return data, nil
	}

	return p.serializer.Serialize(schema, data)
}

// SetEnabled enables or disables the interceptor
func (p *ProducerInterceptor) SetEnabled(enabled bool) {
	p.enabled = enabled
}

// ConsumerInterceptor intercepts fetch responses for schema validation
type ConsumerInterceptor struct {
	deserializer *Deserializer
	enabled      bool
}

// NewConsumerInterceptor creates a new consumer interceptor
func NewConsumerInterceptor(registry *Registry, validateSchema bool) *ConsumerInterceptor {
	return &ConsumerInterceptor{
		deserializer: NewDeserializer(registry, validateSchema),
		enabled:      true,
	}
}

// Intercept intercepts and validates message data
func (c *ConsumerInterceptor) Intercept(data []byte) (int, []byte, error) {
	if !c.enabled {
		return 0, data, nil
	}

	return c.deserializer.Deserialize(data)
}

// InterceptWithSchema intercepts and returns schema with data
func (c *ConsumerInterceptor) InterceptWithSchema(data []byte) (*Schema, []byte, error) {
	if !c.enabled {
		return nil, data, nil
	}

	return c.deserializer.DeserializeWithSchema(data)
}

// SetEnabled enables or disables the interceptor
func (c *ConsumerInterceptor) SetEnabled(enabled bool) {
	c.enabled = enabled
}
