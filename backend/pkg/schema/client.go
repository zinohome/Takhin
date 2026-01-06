// Copyright 2025 Takhin Data, Inc.

package schema

import (
	"fmt"
	"sync"
)

// Client provides a high-level interface for schema registry operations
type Client struct {
	registry   *Registry
	serializers map[string]*Serializer
	deserializer *Deserializer
	mu         sync.RWMutex
}

// NewClient creates a new schema registry client
func NewClient(registryURL string, cfg *Config) (*Client, error) {
	// For now, we use the embedded registry
	// In production, this could be an HTTP client to remote registry
	registry, err := NewRegistry(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry: %w", err)
	}

	return &Client{
		registry:     registry,
		serializers:  make(map[string]*Serializer),
		deserializer: NewDeserializer(registry, true),
	}, nil
}

// GetSerializer gets or creates a serializer for a subject
func (c *Client) GetSerializer(subject string, schemaType SchemaType, autoRegister bool) *Serializer {
	c.mu.RLock()
	serializer, exists := c.serializers[subject]
	c.mu.RUnlock()

	if exists {
		return serializer
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if serializer, exists := c.serializers[subject]; exists {
		return serializer
	}

	serializer = NewSerializer(c.registry, subject, schemaType, autoRegister)
	c.serializers[subject] = serializer
	return serializer
}

// GetDeserializer returns the shared deserializer
func (c *Client) GetDeserializer() *Deserializer {
	return c.deserializer
}

// RegisterSchema registers a new schema
func (c *Client) RegisterSchema(subject, schema string, schemaType SchemaType) (*Schema, error) {
	return c.registry.RegisterSchema(subject, schema, schemaType, nil)
}

// GetLatestSchema retrieves the latest schema for a subject
func (c *Client) GetLatestSchema(subject string) (*Schema, error) {
	return c.registry.GetLatestSchema(subject)
}

// GetSchemaByID retrieves a schema by ID
func (c *Client) GetSchemaByID(id int) (*Schema, error) {
	return c.registry.GetSchemaByID(id)
}

// TestCompatibility tests schema compatibility
func (c *Client) TestCompatibility(subject, schema string, schemaType SchemaType) (bool, error) {
	// Test against latest version
	return c.registry.TestCompatibility(subject, schema, schemaType, 0)
}

// SetCompatibility sets the compatibility mode for a subject
func (c *Client) SetCompatibility(subject string, mode CompatibilityMode) error {
	return c.registry.SetCompatibility(subject, mode)
}

// GetCompatibility gets the compatibility mode for a subject
func (c *Client) GetCompatibility(subject string) (CompatibilityMode, error) {
	return c.registry.GetCompatibility(subject)
}

// Close closes the client
func (c *Client) Close() error {
	return c.registry.Close()
}

// ProducerConfig configures schema-aware producer
type ProducerConfig struct {
	Subject         string
	SchemaType      SchemaType
	Schema          string
	AutoRegister    bool
	ValidateSchema  bool
	CompatibilityMode CompatibilityMode
}

// ConsumerConfig configures schema-aware consumer
type ConsumerConfig struct {
	ValidateSchema bool
	CacheSchemas   bool
}

// SchemaAwareProducer wraps producer functionality with schema support
type SchemaAwareProducer struct {
	client       *Client
	config       ProducerConfig
	interceptor  *ProducerInterceptor
	mu           sync.RWMutex
}

// NewSchemaAwareProducer creates a new schema-aware producer
func NewSchemaAwareProducer(client *Client, config ProducerConfig) (*SchemaAwareProducer, error) {
	// Validate schema if provided
	if config.ValidateSchema && config.Schema != "" {
		if err := client.registry.validator.Validate(config.Schema, config.SchemaType); err != nil {
			return nil, fmt.Errorf("invalid schema: %w", err)
		}
	}

	// Register schema if auto-register is enabled
	if config.AutoRegister && config.Schema != "" {
		_, err := client.RegisterSchema(config.Subject, config.Schema, config.SchemaType)
		if err != nil {
			return nil, fmt.Errorf("failed to register schema: %w", err)
		}
	}

	interceptor := NewProducerInterceptor(client.registry, config.Subject, config.SchemaType)

	return &SchemaAwareProducer{
		client:      client,
		config:      config,
		interceptor: interceptor,
	}, nil
}

// Send serializes and returns wire-format data ready for Kafka produce
func (p *SchemaAwareProducer) Send(data []byte) ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.interceptor.Intercept(p.config.Schema, data)
}

// SendJSON serializes JSON object and returns wire-format data
func (p *SchemaAwareProducer) SendJSON(data interface{}) ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	serializer := p.client.GetSerializer(p.config.Subject, p.config.SchemaType, p.config.AutoRegister)
	return serializer.SerializeJSON(p.config.Schema, data)
}

// UpdateSchema updates the schema (e.g., for schema evolution)
func (p *SchemaAwareProducer) UpdateSchema(newSchema string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Test compatibility
	compatible, err := p.client.TestCompatibility(p.config.Subject, newSchema, p.config.SchemaType)
	if err != nil {
		return fmt.Errorf("compatibility test failed: %w", err)
	}

	if !compatible {
		return NewSchemaError(ErrCodeIncompatibleSchema, "new schema is not compatible")
	}

	// Register new version
	_, err = p.client.RegisterSchema(p.config.Subject, newSchema, p.config.SchemaType)
	if err != nil {
		return fmt.Errorf("failed to register new schema: %w", err)
	}

	p.config.Schema = newSchema
	return nil
}

// SchemaAwareConsumer wraps consumer functionality with schema support
type SchemaAwareConsumer struct {
	client      *Client
	config      ConsumerConfig
	interceptor *ConsumerInterceptor
	mu          sync.RWMutex
}

// NewSchemaAwareConsumer creates a new schema-aware consumer
func NewSchemaAwareConsumer(client *Client, config ConsumerConfig) *SchemaAwareConsumer {
	interceptor := NewConsumerInterceptor(client.registry, config.ValidateSchema)

	return &SchemaAwareConsumer{
		client:      client,
		config:      config,
		interceptor: interceptor,
	}
}

// Receive deserializes wire-format data from Kafka
func (c *SchemaAwareConsumer) Receive(wireData []byte) (int, []byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.interceptor.Intercept(wireData)
}

// ReceiveWithSchema deserializes and returns schema metadata
func (c *SchemaAwareConsumer) ReceiveWithSchema(wireData []byte) (*Schema, []byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.interceptor.InterceptWithSchema(wireData)
}

// ReceiveJSON deserializes JSON data
func (c *SchemaAwareConsumer) ReceiveJSON(wireData []byte, target interface{}) (*Schema, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.client.deserializer.DeserializeJSON(wireData, target)
}
