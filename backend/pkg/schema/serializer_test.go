// Copyright 2025 Takhin Data, Inc.

package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializer(t *testing.T) {
	registry, cleanup := setupTestRegistry(t)
	defer cleanup()

	subject := "test-value"
	schema := `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`

	t.Run("serialize with auto-register", func(t *testing.T) {
		serializer := NewSerializer(registry, subject, SchemaTypeAvro, true)

		data := []byte(`{"id":123}`)
		wireData, err := serializer.Serialize(schema, data)
		require.NoError(t, err)

		// Verify wire format
		assert.Equal(t, byte(0x0), wireData[0]) // Magic byte
		assert.Equal(t, 5+len(data), len(wireData))

		// Verify schema was registered
		schemas, err := registry.GetSubjects()
		require.NoError(t, err)
		assert.Contains(t, schemas, subject)
	})

	t.Run("serialize without auto-register", func(t *testing.T) {
		// Register schema first
		_, err := registry.RegisterSchema(subject+"-manual", schema, SchemaTypeAvro, nil)
		require.NoError(t, err)

		serializer := NewSerializer(registry, subject+"-manual", SchemaTypeAvro, false)

		data := []byte(`{"id":456}`)
		wireData, err := serializer.Serialize(schema, data)
		require.NoError(t, err)

		assert.Equal(t, byte(0x0), wireData[0])
	})

	t.Run("serialize JSON", func(t *testing.T) {
		serializer := NewSerializer(registry, subject+"-json", SchemaTypeJSON, true)

		testData := map[string]interface{}{
			"name": "test",
			"age":  30,
		}

		jsonSchema := `{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}}}`
		wireData, err := serializer.SerializeJSON(jsonSchema, testData)
		require.NoError(t, err)

		assert.True(t, len(wireData) > 5)
		assert.Equal(t, byte(0x0), wireData[0])
	})

	t.Run("serialize Avro", func(t *testing.T) {
		serializer := NewSerializer(registry, subject+"-avro", SchemaTypeAvro, true)

		testData := map[string]interface{}{
			"id": 789,
		}

		wireData, err := serializer.SerializeAvro(schema, testData)
		require.NoError(t, err)

		assert.True(t, len(wireData) > 5)
	})
}

func TestDeserializer(t *testing.T) {
	registry, cleanup := setupTestRegistry(t)
	defer cleanup()

	subject := "test-deserialize"
	schema := `{"type":"record","name":"Test","fields":[{"name":"value","type":"string"}]}`

	// Register schema
	registeredSchema, err := registry.RegisterSchema(subject, schema, SchemaTypeAvro, nil)
	require.NoError(t, err)

	t.Run("deserialize valid data", func(t *testing.T) {
		deserializer := NewDeserializer(registry, true)

		// Create wire format data
		data := []byte(`{"value":"test"}`)
		wireData := make([]byte, 5+len(data))
		wireData[0] = 0x0 // magic byte
		wireData[1] = 0x0
		wireData[2] = 0x0
		wireData[3] = 0x0
		wireData[4] = byte(registeredSchema.ID)
		copy(wireData[5:], data)

		schemaID, extractedData, err := deserializer.Deserialize(wireData)
		require.NoError(t, err)

		assert.Equal(t, registeredSchema.ID, schemaID)
		assert.Equal(t, data, extractedData)
	})

	t.Run("deserialize with schema", func(t *testing.T) {
		deserializer := NewDeserializer(registry, true)

		data := []byte(`{"value":"test2"}`)
		wireData := make([]byte, 5+len(data))
		wireData[0] = 0x0
		wireData[4] = byte(registeredSchema.ID)
		copy(wireData[5:], data)

		schema, extractedData, err := deserializer.DeserializeWithSchema(wireData)
		require.NoError(t, err)

		assert.Equal(t, registeredSchema.ID, schema.ID)
		assert.Equal(t, subject, schema.Subject)
		assert.Equal(t, data, extractedData)
	})

	t.Run("deserialize JSON", func(t *testing.T) {
		deserializer := NewDeserializer(registry, false)

		testData := map[string]string{"value": "test3"}
		jsonBytes, _ := json.Marshal(testData)

		wireData := make([]byte, 5+len(jsonBytes))
		wireData[0] = 0x0
		wireData[4] = byte(registeredSchema.ID)
		copy(wireData[5:], jsonBytes)

		var result map[string]string
		schema, err := deserializer.DeserializeJSON(wireData, &result)
		require.NoError(t, err)

		assert.Equal(t, registeredSchema.ID, schema.ID)
		assert.Equal(t, "test3", result["value"])
	})

	t.Run("deserialize invalid magic byte", func(t *testing.T) {
		deserializer := NewDeserializer(registry, true)

		wireData := []byte{0xFF, 0x0, 0x0, 0x0, 0x1}
		_, _, err := deserializer.Deserialize(wireData)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid magic byte")
	})

	t.Run("deserialize too short", func(t *testing.T) {
		deserializer := NewDeserializer(registry, true)

		wireData := []byte{0x0, 0x0}
		_, _, err := deserializer.Deserialize(wireData)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "data too short")
	})

	t.Run("deserialize non-existent schema", func(t *testing.T) {
		deserializer := NewDeserializer(registry, true)

		wireData := []byte{0x0, 0x0, 0x0, 0x03, 0xE8} // schema ID 1000
		_, _, err := deserializer.Deserialize(wireData)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "schema validation failed")
	})
}

func TestProducerInterceptor(t *testing.T) {
	registry, cleanup := setupTestRegistry(t)
	defer cleanup()

	subject := "test-producer"
	schema := `{"type":"string"}`

	t.Run("intercept with auto-register", func(t *testing.T) {
		interceptor := NewProducerInterceptor(registry, subject, SchemaTypeAvro)

		data := []byte(`"hello"`)
		wireData, err := interceptor.Intercept(schema, data)
		require.NoError(t, err)

		// Verify wire format
		assert.Equal(t, byte(0x0), wireData[0])
		assert.True(t, len(wireData) > 5)

		// Verify data is embedded
		assert.Equal(t, data, wireData[5:])

		// Verify schema was registered
		schemas, err := registry.GetSubjects()
		require.NoError(t, err)
		assert.Contains(t, schemas, subject)
	})

	t.Run("intercept disabled", func(t *testing.T) {
		interceptor := NewProducerInterceptor(registry, subject+"-disabled", SchemaTypeJSON)
		interceptor.SetEnabled(false)

		data := []byte(`{"test":"data"}`)
		wireData, err := interceptor.Intercept(schema, data)
		require.NoError(t, err)

		// Should return original data
		assert.Equal(t, data, wireData)
	})

	t.Run("intercept multiple messages", func(t *testing.T) {
		interceptor := NewProducerInterceptor(registry, subject+"-multi", SchemaTypeAvro)

		for i := 0; i < 3; i++ {
			data := []byte(`"message"`)
			wireData, err := interceptor.Intercept(schema, data)
			require.NoError(t, err)
			assert.Equal(t, byte(0x0), wireData[0])
		}

		// Should only register schema once
		versions, err := registry.GetAllVersions(subject + "-multi")
		require.NoError(t, err)
		assert.Equal(t, 1, len(versions))
	})
}

func TestConsumerInterceptor(t *testing.T) {
	registry, cleanup := setupTestRegistry(t)
	defer cleanup()

	subject := "test-consumer"
	schema := `{"type":"int"}`

	// Register schema
	registeredSchema, err := registry.RegisterSchema(subject, schema, SchemaTypeAvro, nil)
	require.NoError(t, err)

	t.Run("intercept with validation", func(t *testing.T) {
		interceptor := NewConsumerInterceptor(registry, true)

		data := []byte(`123`)
		wireData := make([]byte, 5+len(data))
		wireData[0] = 0x0
		wireData[4] = byte(registeredSchema.ID)
		copy(wireData[5:], data)

		schemaID, extractedData, err := interceptor.Intercept(wireData)
		require.NoError(t, err)

		assert.Equal(t, registeredSchema.ID, schemaID)
		assert.Equal(t, data, extractedData)
	})

	t.Run("intercept with schema", func(t *testing.T) {
		interceptor := NewConsumerInterceptor(registry, false)

		data := []byte(`456`)
		wireData := make([]byte, 5+len(data))
		wireData[0] = 0x0
		wireData[4] = byte(registeredSchema.ID)
		copy(wireData[5:], data)

		schema, extractedData, err := interceptor.InterceptWithSchema(wireData)
		require.NoError(t, err)

		assert.Equal(t, registeredSchema.ID, schema.ID)
		assert.Equal(t, subject, schema.Subject)
		assert.Equal(t, data, extractedData)
	})

	t.Run("intercept disabled", func(t *testing.T) {
		interceptor := NewConsumerInterceptor(registry, true)
		interceptor.SetEnabled(false)

		data := []byte(`raw data`)
		schemaID, extractedData, err := interceptor.Intercept(data)
		require.NoError(t, err)

		assert.Equal(t, 0, schemaID)
		assert.Equal(t, data, extractedData)
	})
}

func TestSerializationRoundTrip(t *testing.T) {
	registry, cleanup := setupTestRegistry(t)
	defer cleanup()

	subject := "roundtrip-test"
	schema := `{"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}`

	t.Run("serialize and deserialize", func(t *testing.T) {
		// Serialize
		serializer := NewSerializer(registry, subject, SchemaTypeAvro, true)
		testData := map[string]interface{}{
			"name": "Alice",
			"age":  25,
		}

		wireData, err := serializer.SerializeAvro(schema, testData)
		require.NoError(t, err)

		// Deserialize
		deserializer := NewDeserializer(registry, true)
		var result map[string]interface{}
		resultSchema, err := deserializer.DeserializeJSON(wireData, &result)
		require.NoError(t, err)

		assert.Equal(t, "Alice", result["name"])
		assert.Equal(t, float64(25), result["age"]) // JSON unmarshals numbers as float64
		assert.Equal(t, subject, resultSchema.Subject)
	})
}

// Helper function to setup test registry
func setupTestRegistry(t *testing.T) (*Registry, func()) {
	cfg := &Config{
		DataDir:              t.TempDir(),
		DefaultCompatibility: CompatibilityBackward,
		CacheSize:            100,
	}

	registry, err := NewRegistry(cfg)
	require.NoError(t, err)

	cleanup := func() {
		registry.Close()
	}

	return registry, cleanup
}
