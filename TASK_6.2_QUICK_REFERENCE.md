# Task 6.2: Schema Registry Integration - Quick Reference

## Overview

Schema Registry integration provides automatic schema registration for producers and schema validation for consumers with Confluent-compatible wire format.

---

## Quick Start

### 1. Initialize Client

```go
import "github.com/takhin-data/takhin/pkg/schema"

cfg := &schema.Config{
    DataDir:              "/var/lib/schemas",
    DefaultCompatibility: schema.CompatibilityBackward,
    CacheSize:            1000,
}

client, err := schema.NewClient("", cfg)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### 2. Create Schema-Aware Producer

```go
producer, err := schema.NewSchemaAwareProducer(client, schema.ProducerConfig{
    Subject:      "users-value",
    SchemaType:   schema.SchemaTypeAvro,
    Schema:       `{"type":"record","name":"User","fields":[...]}`,
    AutoRegister: true,
})

// Send data - schema auto-registers on first send
wireData, err := producer.SendJSON(userData)
// Send wireData via Kafka
```

### 3. Create Schema-Aware Consumer

```go
consumer := schema.NewSchemaAwareConsumer(client, schema.ConsumerConfig{
    ValidateSchema: true,
})

// Receive data from Kafka
var user User
receivedSchema, err := consumer.ReceiveJSON(wireData, &user)
fmt.Printf("Schema v%d: %+v\n", receivedSchema.Version, user)
```

---

## Wire Format

All messages use Confluent-compatible wire format:

```
[0x00][Schema ID (4 bytes, big-endian)][Serialized Data]
```

**Example:**
```
0x00 0x00 0x00 0x00 0x01 {"name":"Alice"}
 ^    ^-----------------^  ^--------------^
 |           |                    |
Magic    Schema ID            Data
```

---

## API Reference

### Client

```go
// Create client
client, err := schema.NewClient(registryURL, cfg)

// Register schema manually
schema, err := client.RegisterSchema(subject, schemaStr, schemaType)

// Get latest schema
schema, err := client.GetLatestSchema(subject)

// Get schema by ID
schema, err := client.GetSchemaByID(schemaID)

// Test compatibility
compatible, err := client.TestCompatibility(subject, newSchema, schemaType)

// Set compatibility mode
err := client.SetCompatibility(subject, schema.CompatibilityFull)
```

### Producer

```go
// Create producer
producer, err := schema.NewSchemaAwareProducer(client, schema.ProducerConfig{
    Subject:         "topic-value",
    SchemaType:      schema.SchemaTypeAvro,
    Schema:          schemaStr,
    AutoRegister:    true,
    ValidateSchema:  true,
})

// Send raw bytes
wireData, err := producer.Send(data)

// Send JSON object
wireData, err := producer.SendJSON(object)

// Update schema (with compatibility check)
err := producer.UpdateSchema(newSchema)
```

### Consumer

```go
// Create consumer
consumer := schema.NewSchemaAwareConsumer(client, schema.ConsumerConfig{
    ValidateSchema: true,
    CacheSchemas:   true,
})

// Receive raw data
schemaID, data, err := consumer.Receive(wireData)

// Receive with schema metadata
schema, data, err := consumer.ReceiveWithSchema(wireData)

// Receive JSON
var result MyType
schema, err := consumer.ReceiveJSON(wireData, &result)
```

### Low-Level Serialization

```go
// Manual serializer
serializer := schema.NewSerializer(registry, subject, schemaType, autoRegister)
wireData, err := serializer.Serialize(schemaStr, data)
wireData, err := serializer.SerializeJSON(schemaStr, jsonObject)

// Manual deserializer
deserializer := schema.NewDeserializer(registry, validate)
schemaID, data, err := deserializer.Deserialize(wireData)
schema, data, err := deserializer.DeserializeWithSchema(wireData)
```

---

## Schema Types

```go
schema.SchemaTypeAvro      // Avro schema
schema.SchemaTypeJSON      // JSON Schema
schema.SchemaTypeProtobuf  // Protobuf schema
```

---

## Compatibility Modes

```go
schema.CompatibilityNone              // No checks
schema.CompatibilityBackward          // New schema reads old data
schema.CompatibilityBackwardTransit   // Backward check all versions
schema.CompatibilityForward           // Old schema reads new data
schema.CompatibilityForwardTransit    // Forward check all versions
schema.CompatibilityFull              // Both backward and forward
schema.CompatibilityFullTransit       // Full check all versions
```

---

## Common Patterns

### Pattern 1: Auto-Registration Producer

```go
producer, _ := schema.NewSchemaAwareProducer(client, schema.ProducerConfig{
    Subject:      "orders-value",
    SchemaType:   schema.SchemaTypeAvro,
    Schema:       orderSchema,
    AutoRegister: true,  // Schema registers automatically
})

// First send registers schema
wireData, _ := producer.SendJSON(order)
```

### Pattern 2: Manual Schema Management

```go
// Register schema explicitly
registeredSchema, _ := client.RegisterSchema("products-value", productSchema, schema.SchemaTypeJSON)

// Producer uses existing schema
producer, _ := schema.NewSchemaAwareProducer(client, schema.ProducerConfig{
    Subject:      "products-value",
    SchemaType:   schema.SchemaTypeJSON,
    Schema:       productSchema,
    AutoRegister: false,  // Expect schema already registered
})
```

### Pattern 3: Schema Evolution

```go
// Start with v1
producer, _ := schema.NewSchemaAwareProducer(client, schema.ProducerConfig{
    Subject:      "events-value",
    SchemaType:   schema.SchemaTypeAvro,
    Schema:       schemaV1,
    AutoRegister: true,
})

// Send with v1
producer.SendJSON(eventV1)

// Evolve to v2 (checks compatibility)
err := producer.UpdateSchema(schemaV2)
if err != nil {
    log.Fatal("Schema incompatible:", err)
}

// Now sends with v2
producer.SendJSON(eventV2)
```

### Pattern 4: Validation Optional Consumer

```go
// With validation (production)
consumer := schema.NewSchemaAwareConsumer(client, schema.ConsumerConfig{
    ValidateSchema: true,  // Validates schema exists
})

// Without validation (development)
consumer := schema.NewSchemaAwareConsumer(client, schema.ConsumerConfig{
    ValidateSchema: false,  // Skip validation for speed
})
```

### Pattern 5: Schema Compatibility Testing

```go
// Test before registering
compatible, err := client.TestCompatibility("users-value", newSchema, schema.SchemaTypeAvro)
if !compatible {
    log.Fatal("Schema would break backward compatibility")
}

// Safe to register
client.RegisterSchema("users-value", newSchema, schema.SchemaTypeAvro)
```

---

## Configuration

### Producer Config

```go
type ProducerConfig struct {
    Subject         string              // Schema subject (e.g., "topic-value")
    SchemaType      SchemaType          // AVRO, JSON, or PROTOBUF
    Schema          string              // Schema definition
    AutoRegister    bool                // Auto-register on first send
    ValidateSchema  bool                // Validate before registration
    CompatibilityMode CompatibilityMode // Override default compatibility
}
```

### Consumer Config

```go
type ConsumerConfig struct {
    ValidateSchema bool  // Validate schema exists on receive
    CacheSchemas   bool  // Cache schemas (always true currently)
}
```

### Registry Config

```go
type Config struct {
    DataDir              string            // Storage directory
    DefaultCompatibility CompatibilityMode // Default compatibility mode
    MaxVersions          int               // Max versions per subject
    CacheSize            int               // Schema cache size
}
```

---

## Error Handling

```go
// Check for specific errors
wireData, err := producer.Send(data)
if err != nil {
    if schemaErr, ok := err.(*schema.SchemaError); ok {
        switch schemaErr.ErrorCode {
        case schema.ErrCodeIncompatibleSchema:
            // Handle incompatibility
        case schema.ErrCodeInvalidSchema:
            // Handle invalid schema
        case schema.ErrCodeSubjectNotFound:
            // Handle missing subject
        }
    }
}
```

### Error Codes

```go
schema.ErrCodeSubjectNotFound      // 40401
schema.ErrCodeVersionNotFound      // 40402
schema.ErrCodeSchemaNotFound       // 40403
schema.ErrCodeIncompatibleSchema   // 409
schema.ErrCodeInvalidSchema        // 42201
schema.ErrCodeInvalidVersion       // 42202
```

---

## Examples

### Example 1: Simple Producer/Consumer

```go
// Producer
producer, _ := schema.NewSchemaAwareProducer(client, schema.ProducerConfig{
    Subject:      "users-value",
    SchemaType:   schema.SchemaTypeJSON,
    Schema:       `{"type":"object","properties":{"name":{"type":"string"}}}`,
    AutoRegister: true,
})

user := map[string]string{"name": "Alice"}
wireData, _ := producer.SendJSON(user)

// Consumer
consumer := schema.NewSchemaAwareConsumer(client, schema.ConsumerConfig{
    ValidateSchema: true,
})

var receivedUser map[string]string
schema, _ := consumer.ReceiveJSON(wireData, &receivedUser)
fmt.Printf("User: %v (schema v%d)\n", receivedUser, schema.Version)
```

### Example 2: Avro Schema Evolution

```go
// Initial schema
schemaV1 := `{
    "type": "record",
    "name": "Event",
    "fields": [
        {"name": "id", "type": "int"},
        {"name": "name", "type": "string"}
    ]
}`

// Evolved schema (backward compatible - added optional field)
schemaV2 := `{
    "type": "record",
    "name": "Event",
    "fields": [
        {"name": "id", "type": "int"},
        {"name": "name", "type": "string"},
        {"name": "timestamp", "type": "long", "default": 0}
    ]
}`

producer, _ := schema.NewSchemaAwareProducer(client, schema.ProducerConfig{
    Subject:      "events-value",
    SchemaType:   schema.SchemaTypeAvro,
    Schema:       schemaV1,
    AutoRegister: true,
})

// Send with v1
producer.SendJSON(map[string]interface{}{"id": 1, "name": "login"})

// Evolve
producer.UpdateSchema(schemaV2)

// Send with v2
producer.SendJSON(map[string]interface{}{
    "id": 2, 
    "name": "logout", 
    "timestamp": 1234567890,
})
```

### Example 3: Multiple Schema Types

```go
// Avro producer
avroProducer, _ := schema.NewSchemaAwareProducer(client, schema.ProducerConfig{
    Subject:    "events-avro-value",
    SchemaType: schema.SchemaTypeAvro,
    Schema:     avroSchema,
})

// JSON producer  
jsonProducer, _ := schema.NewSchemaAwareProducer(client, schema.ProducerConfig{
    Subject:    "events-json-value",
    SchemaType: schema.SchemaTypeJSON,
    Schema:     jsonSchema,
})

// Protobuf producer
protoProducer, _ := schema.NewSchemaAwareProducer(client, schema.ProducerConfig{
    Subject:    "events-proto-value",
    SchemaType: schema.SchemaTypeProtobuf,
    Schema:     protoSchema,
})
```

---

## Performance Tips

1. **Reuse Producers/Consumers**: Create once, use many times
2. **Enable Caching**: Schema lookups are cached automatically
3. **Disable Validation in Dev**: Skip validation for faster iteration
4. **Batch Operations**: Send multiple messages with same schema
5. **Schema Reuse**: Identical schemas reuse same ID (no new versions)

---

## Testing

### Unit Tests

```go
func TestMyProducer(t *testing.T) {
    cfg := &schema.Config{
        DataDir:              t.TempDir(),
        DefaultCompatibility: schema.CompatibilityBackward,
    }
    
    client, _ := schema.NewClient("", cfg)
    defer client.Close()
    
    producer, err := schema.NewSchemaAwareProducer(client, schema.ProducerConfig{
        Subject:      "test-subject",
        SchemaType:   schema.SchemaTypeJSON,
        Schema:       `{"type":"object"}`,
        AutoRegister: true,
    })
    
    require.NoError(t, err)
    
    data := map[string]string{"key": "value"}
    wireData, err := producer.SendJSON(data)
    
    assert.NoError(t, err)
    assert.NotEmpty(t, wireData)
}
```

---

## Troubleshooting

### Issue: Schema Not Found
```
Error: schema error 40403: schema not found
```
**Solution**: Enable `AutoRegister: true` or register schema manually first

### Issue: Incompatible Schema
```
Error: schema error 409: incompatible schema
```
**Solution**: Check schema evolution rules, add default values to new fields

### Issue: Invalid Magic Byte
```
Error: invalid magic byte: expected 0, got 255
```
**Solution**: Data is not schema-encoded, check producer configuration

### Issue: Data Too Short
```
Error: data too short: expected at least 5 bytes, got 3
```
**Solution**: Wire format requires minimum 5 bytes (magic + schema ID)

---

## Integration with Kafka

```go
// In Kafka producer
producer := createKafkaProducer()
schemaProducer, _ := schema.NewSchemaAwareProducer(schemaClient, cfg)

data := MyData{Field: "value"}
wireData, _ := schemaProducer.SendJSON(data)

// Send to Kafka with schema embedded
producer.Send(&kafka.Message{
    Topic: "my-topic",
    Value: wireData,  // Contains schema ID
})

// In Kafka consumer
consumer := createKafkaConsumer()
schemaConsumer := schema.NewSchemaAwareConsumer(schemaClient, cfg)

msg := consumer.ReadMessage()

var data MyData
receivedSchema, _ := schemaConsumer.ReceiveJSON(msg.Value, &data)
fmt.Printf("Received v%d: %+v\n", receivedSchema.Version, data)
```

---

## See Also

- **Full Documentation**: `TASK_6.2_COMPLETION_SUMMARY.md`
- **Working Example**: `backend/examples/schema_integration.go`
- **Schema Registry Core**: `TASK_6.1_COMPLETION_SUMMARY.md`
- **REST API Reference**: `TASK_6.1_QUICK_REFERENCE.md`

---

**Last Updated**: 2026-01-06  
**Version**: 1.0  
**Status**: Production Ready
