// Copyright 2025 Takhin Data, Inc.

package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	cfg := &Config{
		DataDir:              t.TempDir(),
		DefaultCompatibility: CompatibilityBackward,
		CacheSize:            100,
	}

	client, err := NewClient("", cfg)
	require.NoError(t, err)
	defer client.Close()

	t.Run("register and get schema", func(t *testing.T) {
		schema := `{"type":"string"}`
		registeredSchema, err := client.RegisterSchema("test-subject", schema, SchemaTypeAvro)
		require.NoError(t, err)

		assert.NotZero(t, registeredSchema.ID)
		assert.Equal(t, "test-subject", registeredSchema.Subject)
		assert.Equal(t, 1, registeredSchema.Version)

		// Get by ID
		retrieved, err := client.GetSchemaByID(registeredSchema.ID)
		require.NoError(t, err)
		assert.Equal(t, schema, retrieved.Schema)
	})

	t.Run("get serializer", func(t *testing.T) {
		serializer1 := client.GetSerializer("subject1", SchemaTypeJSON, true)
		serializer2 := client.GetSerializer("subject1", SchemaTypeJSON, true)

		// Should return same instance
		assert.Equal(t, serializer1, serializer2)
	})

	t.Run("test compatibility", func(t *testing.T) {
		subject := "compat-test"
		schema1 := `{"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}`
		schema2 := `{"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string","default":""}]}`

		_, err := client.RegisterSchema(subject, schema1, SchemaTypeAvro)
		require.NoError(t, err)

		compatible, err := client.TestCompatibility(subject, schema2, SchemaTypeAvro)
		require.NoError(t, err)
		assert.True(t, compatible)
	})

	t.Run("set and get compatibility", func(t *testing.T) {
		subject := "compat-config-test"

		err := client.SetCompatibility(subject, CompatibilityFull)
		require.NoError(t, err)

		mode, err := client.GetCompatibility(subject)
		require.NoError(t, err)
		assert.Equal(t, CompatibilityFull, mode)
	})
}

func TestSchemaAwareProducer(t *testing.T) {
	cfg := &Config{
		DataDir:              t.TempDir(),
		DefaultCompatibility: CompatibilityBackward,
		CacheSize:            100,
	}

	client, err := NewClient("", cfg)
	require.NoError(t, err)
	defer client.Close()

	t.Run("create and send", func(t *testing.T) {
		producerCfg := ProducerConfig{
			Subject:      "producer-test",
			SchemaType:   SchemaTypeJSON,
			Schema:       `{"type":"object","properties":{"message":{"type":"string"}}}`,
			AutoRegister: true,
		}

		producer, err := NewSchemaAwareProducer(client, producerCfg)
		require.NoError(t, err)

		data := []byte(`{"message":"hello"}`)
		wireData, err := producer.Send(data)
		require.NoError(t, err)

		// Verify wire format
		assert.Equal(t, byte(0x0), wireData[0])
		assert.True(t, len(wireData) > 5)
	})

	t.Run("send JSON", func(t *testing.T) {
		producerCfg := ProducerConfig{
			Subject:      "producer-json-test",
			SchemaType:   SchemaTypeJSON,
			Schema:       `{"type":"object"}`,
			AutoRegister: true,
		}

		producer, err := NewSchemaAwareProducer(client, producerCfg)
		require.NoError(t, err)

		testData := map[string]interface{}{
			"name": "Alice",
			"age":  30,
		}

		wireData, err := producer.SendJSON(testData)
		require.NoError(t, err)

		assert.Equal(t, byte(0x0), wireData[0])
	})

	t.Run("update schema", func(t *testing.T) {
		subject := "schema-evolution-test"
		initialSchema := `{"type":"record","name":"Event","fields":[{"name":"id","type":"int"}]}`
		newSchema := `{"type":"record","name":"Event","fields":[{"name":"id","type":"int"},{"name":"timestamp","type":"long","default":0}]}`

		producerCfg := ProducerConfig{
			Subject:      subject,
			SchemaType:   SchemaTypeAvro,
			Schema:       initialSchema,
			AutoRegister: true,
		}

		producer, err := NewSchemaAwareProducer(client, producerCfg)
		require.NoError(t, err)

		// Send with initial schema
		data1 := []byte(`{"id":1}`)
		wireData1, err := producer.Send(data1)
		require.NoError(t, err)
		assert.NotNil(t, wireData1)

		// Update to new compatible schema
		err = producer.UpdateSchema(newSchema)
		require.NoError(t, err)

		// Send with new schema
		data2 := []byte(`{"id":2,"timestamp":1234567890}`)
		wireData2, err := producer.Send(data2)
		require.NoError(t, err)
		assert.NotNil(t, wireData2)

		// Verify schema was updated
		versions, err := client.registry.GetAllVersions(subject)
		require.NoError(t, err)
		assert.Equal(t, 2, len(versions))
	})

	t.Run("invalid schema", func(t *testing.T) {
		producerCfg := ProducerConfig{
			Subject:        "invalid-schema-test",
			SchemaType:     SchemaTypeAvro,
			Schema:         `{invalid json}`,
			AutoRegister:   true,
			ValidateSchema: true,
		}

		_, err := NewSchemaAwareProducer(client, producerCfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid schema")
	})
}

func TestSchemaAwareConsumer(t *testing.T) {
	cfg := &Config{
		DataDir:              t.TempDir(),
		DefaultCompatibility: CompatibilityBackward,
		CacheSize:            100,
	}

	client, err := NewClient("", cfg)
	require.NoError(t, err)
	defer client.Close()

	t.Run("receive data", func(t *testing.T) {
		// Register schema
		schema := `{"type":"string"}`
		registeredSchema, err := client.RegisterSchema("consumer-test", schema, SchemaTypeAvro)
		require.NoError(t, err)

		// Create wire format data
		data := []byte(`"test message"`)
		wireData := make([]byte, 5+len(data))
		wireData[0] = 0x0
		wireData[4] = byte(registeredSchema.ID)
		copy(wireData[5:], data)

		// Consume
		consumerCfg := ConsumerConfig{
			ValidateSchema: true,
		}
		consumer := NewSchemaAwareConsumer(client, consumerCfg)

		schemaID, extractedData, err := consumer.Receive(wireData)
		require.NoError(t, err)

		assert.Equal(t, registeredSchema.ID, schemaID)
		assert.Equal(t, data, extractedData)
	})

	t.Run("receive with schema", func(t *testing.T) {
		// Register schema
		schema := `{"type":"int"}`
		registeredSchema, err := client.RegisterSchema("consumer-schema-test", schema, SchemaTypeAvro)
		require.NoError(t, err)

		// Create wire format data
		data := []byte(`42`)
		wireData := make([]byte, 5+len(data))
		wireData[0] = 0x0
		wireData[4] = byte(registeredSchema.ID)
		copy(wireData[5:], data)

		// Consume
		consumerCfg := ConsumerConfig{
			ValidateSchema: true,
		}
		consumer := NewSchemaAwareConsumer(client, consumerCfg)

		receivedSchema, extractedData, err := consumer.ReceiveWithSchema(wireData)
		require.NoError(t, err)

		assert.Equal(t, registeredSchema.ID, receivedSchema.ID)
		assert.Equal(t, "consumer-schema-test", receivedSchema.Subject)
		assert.Equal(t, data, extractedData)
	})

	t.Run("receive JSON", func(t *testing.T) {
		// Register schema
		schema := `{"type":"object"}`
		registeredSchema, err := client.RegisterSchema("consumer-json-test", schema, SchemaTypeJSON)
		require.NoError(t, err)

		// Create test data
		testData := map[string]string{"key": "value"}
		serializer := NewSerializer(client.registry, "consumer-json-test", SchemaTypeJSON, false)
		wireData, err := serializer.SerializeJSON(schema, testData)
		require.NoError(t, err)

		// Consume
		consumerCfg := ConsumerConfig{
			ValidateSchema: true,
		}
		consumer := NewSchemaAwareConsumer(client, consumerCfg)

		var result map[string]string
		receivedSchema, err := consumer.ReceiveJSON(wireData, &result)
		require.NoError(t, err)

		assert.Equal(t, registeredSchema.ID, receivedSchema.ID)
		assert.Equal(t, "value", result["key"])
	})
}

func TestProducerConsumerIntegration(t *testing.T) {
	cfg := &Config{
		DataDir:              t.TempDir(),
		DefaultCompatibility: CompatibilityBackward,
		CacheSize:            100,
	}

	client, err := NewClient("", cfg)
	require.NoError(t, err)
	defer client.Close()

	subject := "integration-test"
	schema := `{"type":"record","name":"Message","fields":[{"name":"text","type":"string"},{"name":"timestamp","type":"long"}]}`

	t.Run("producer to consumer", func(t *testing.T) {
		// Setup producer
		producerCfg := ProducerConfig{
			Subject:      subject,
			SchemaType:   SchemaTypeAvro,
			Schema:       schema,
			AutoRegister: true,
		}

		producer, err := NewSchemaAwareProducer(client, producerCfg)
		require.NoError(t, err)

		// Produce message
		testData := map[string]interface{}{
			"text":      "Hello World",
			"timestamp": int64(1234567890),
		}

		wireData, err := producer.SendJSON(testData)
		require.NoError(t, err)

		// Setup consumer
		consumerCfg := ConsumerConfig{
			ValidateSchema: true,
		}
		consumer := NewSchemaAwareConsumer(client, consumerCfg)

		// Consume message
		var receivedData map[string]interface{}
		receivedSchema, err := consumer.ReceiveJSON(wireData, &receivedData)
		require.NoError(t, err)

		assert.Equal(t, subject, receivedSchema.Subject)
		assert.Equal(t, "Hello World", receivedData["text"])
		assert.Equal(t, float64(1234567890), receivedData["timestamp"])
	})

	t.Run("multiple messages", func(t *testing.T) {
		producerCfg := ProducerConfig{
			Subject:      subject + "-multi",
			SchemaType:   SchemaTypeJSON,
			Schema:       `{"type":"object"}`,
			AutoRegister: true,
		}

		producer, err := NewSchemaAwareProducer(client, producerCfg)
		require.NoError(t, err)

		consumerCfg := ConsumerConfig{
			ValidateSchema: true,
		}
		consumer := NewSchemaAwareConsumer(client, consumerCfg)

		// Send and receive multiple messages
		for i := 0; i < 5; i++ {
			data := map[string]interface{}{"index": i}
			wireData, err := producer.SendJSON(data)
			require.NoError(t, err)

			var received map[string]interface{}
			_, err = consumer.ReceiveJSON(wireData, &received)
			require.NoError(t, err)

			assert.Equal(t, float64(i), received["index"])
		}
	})

	t.Run("schema evolution", func(t *testing.T) {
		subject := "evolution-test"
		schemaV1 := `{"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}`
		schemaV2 := `{"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"age","type":"int","default":0}]}`

		// Producer with v1 schema
		producerCfg := ProducerConfig{
			Subject:      subject,
			SchemaType:   SchemaTypeAvro,
			Schema:       schemaV1,
			AutoRegister: true,
		}

		producer, err := NewSchemaAwareProducer(client, producerCfg)
		require.NoError(t, err)

		// Send message with v1
		dataV1 := map[string]interface{}{"name": "Alice"}
		wireDataV1, err := producer.SendJSON(dataV1)
		require.NoError(t, err)

		// Evolve schema to v2
		err = producer.UpdateSchema(schemaV2)
		require.NoError(t, err)

		// Send message with v2
		dataV2 := map[string]interface{}{"name": "Bob", "age": 30}
		wireDataV2, err := producer.SendJSON(dataV2)
		require.NoError(t, err)

		// Consumer can read both versions
		consumerCfg := ConsumerConfig{
			ValidateSchema: true,
		}
		consumer := NewSchemaAwareConsumer(client, consumerCfg)

		var receivedV1, receivedV2 map[string]interface{}

		_, err = consumer.ReceiveJSON(wireDataV1, &receivedV1)
		require.NoError(t, err)
		assert.Equal(t, "Alice", receivedV1["name"])

		_, err = consumer.ReceiveJSON(wireDataV2, &receivedV2)
		require.NoError(t, err)
		assert.Equal(t, "Bob", receivedV2["name"])
		assert.Equal(t, float64(30), receivedV2["age"])
	})
}
