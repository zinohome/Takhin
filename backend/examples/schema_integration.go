// Copyright 2025 Takhin Data, Inc.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/takhin-data/takhin/pkg/schema"
)

// User represents a user record
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func main() {
	fmt.Println("=== Takhin Schema Registry Integration Example ===\n")

	// Setup schema registry client
	cfg := &schema.Config{
		DataDir:              "/tmp/takhin-schema-example",
		DefaultCompatibility: schema.CompatibilityBackward,
		CacheSize:            1000,
	}

	// Ensure clean start
	os.RemoveAll(cfg.DataDir)
	defer os.RemoveAll(cfg.DataDir)

	client, err := schema.NewClient("", cfg)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	fmt.Println("✓ Schema Registry client initialized")

	// Example 1: Basic Producer/Consumer with Auto-Registration
	fmt.Println("\n--- Example 1: Auto-Registration ---")
	example1(client)

	// Example 2: Schema Evolution
	fmt.Println("\n--- Example 2: Schema Evolution ---")
	example2(client)

	// Example 3: Compatibility Testing
	fmt.Println("\n--- Example 3: Compatibility Testing ---")
	example3(client)

	fmt.Println("\n=== All examples completed successfully ===")
}

func example1(client *schema.Client) {
	subject := "users-value"

	// Define Avro schema
	userSchema := `{
		"type": "record",
		"name": "User",
		"fields": [
			{"name": "name", "type": "string"},
			{"name": "email", "type": "string"}
		]
	}`

	// Create producer with auto-registration
	producerCfg := schema.ProducerConfig{
		Subject:      subject,
		SchemaType:   schema.SchemaTypeAvro,
		Schema:       userSchema,
		AutoRegister: true,
	}

	producer, err := schema.NewSchemaAwareProducer(client, producerCfg)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}

	// Produce a message
	user := User{
		Name:  "Alice Smith",
		Email: "alice@example.com",
	}

	wireData, err := producer.SendJSON(user)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	fmt.Printf("✓ Produced message (%d bytes)\n", len(wireData))
	fmt.Printf("  Schema ID: %d (embedded in wire format)\n", int(wireData[4]))

	// Create consumer
	consumerCfg := schema.ConsumerConfig{
		ValidateSchema: true,
	}

	consumer := schema.NewSchemaAwareConsumer(client, consumerCfg)

	// Consume the message
	var receivedUser User
	receivedSchema, err := consumer.ReceiveJSON(wireData, &receivedUser)
	if err != nil {
		log.Fatalf("Failed to receive message: %v", err)
	}

	fmt.Printf("✓ Consumed message\n")
	fmt.Printf("  Schema: %s (version %d)\n", receivedSchema.Subject, receivedSchema.Version)
	fmt.Printf("  Data: %+v\n", receivedUser)
}

func example2(client *schema.Client) {
	subject := "events-value"

	// Initial schema (v1)
	schemaV1 := `{
		"type": "record",
		"name": "Event",
		"fields": [
			{"name": "id", "type": "int"},
			{"name": "name", "type": "string"}
		]
	}`

	// Evolved schema (v2) - added optional timestamp field
	schemaV2 := `{
		"type": "record",
		"name": "Event",
		"fields": [
			{"name": "id", "type": "int"},
			{"name": "name", "type": "string"},
			{"name": "timestamp", "type": "long", "default": 0}
		]
	}`

	// Producer with v1 schema
	producerCfg := schema.ProducerConfig{
		Subject:      subject,
		SchemaType:   schema.SchemaTypeAvro,
		Schema:       schemaV1,
		AutoRegister: true,
	}

	producer, err := schema.NewSchemaAwareProducer(client, producerCfg)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}

	// Send message with v1
	eventV1 := map[string]interface{}{
		"id":   1,
		"name": "login",
	}

	wireDataV1, err := producer.SendJSON(eventV1)
	if err != nil {
		log.Fatalf("Failed to send v1 message: %v", err)
	}

	fmt.Printf("✓ Sent event with schema v1\n")

	// Evolve schema to v2
	err = producer.UpdateSchema(schemaV2)
	if err != nil {
		log.Fatalf("Failed to update schema: %v", err)
	}

	fmt.Printf("✓ Schema evolved to v2 (backward compatible)\n")

	// Send message with v2
	eventV2 := map[string]interface{}{
		"id":        2,
		"name":      "logout",
		"timestamp": int64(1704544800),
	}

	wireDataV2, err := producer.SendJSON(eventV2)
	if err != nil {
		log.Fatalf("Failed to send v2 message: %v", err)
	}

	fmt.Printf("✓ Sent event with schema v2\n")

	// Consumer can read both versions
	consumerCfg := schema.ConsumerConfig{
		ValidateSchema: true,
	}
	consumer := schema.NewSchemaAwareConsumer(client, consumerCfg)

	var receivedV1, receivedV2 map[string]interface{}

	receivedSchemaV1, err := consumer.ReceiveJSON(wireDataV1, &receivedV1)
	if err != nil {
		log.Fatalf("Failed to consume v1: %v", err)
	}

	receivedSchemaV2, err := consumer.ReceiveJSON(wireDataV2, &receivedV2)
	if err != nil {
		log.Fatalf("Failed to consume v2: %v", err)
	}

	fmt.Printf("✓ Consumed v1 event: %v (schema version %d)\n", receivedV1, receivedSchemaV1.Version)
	fmt.Printf("✓ Consumed v2 event: %v (schema version %d)\n", receivedV2, receivedSchemaV2.Version)
}

func example3(client *schema.Client) {
	subject := "products-value"

	// Original schema
	originalSchema := `{
		"type": "record",
		"name": "Product",
		"fields": [
			{"name": "id", "type": "int"},
			{"name": "name", "type": "string"},
			{"name": "price", "type": "double"}
		]
	}`

	// Register original schema
	registeredSchema, err := client.RegisterSchema(subject, originalSchema, schema.SchemaTypeAvro)
	if err != nil {
		log.Fatalf("Failed to register schema: %v", err)
	}

	fmt.Printf("✓ Registered schema (ID: %d, Version: %d)\n", registeredSchema.ID, registeredSchema.Version)

	// Compatible schema (adds optional field)
	compatibleSchema := `{
		"type": "record",
		"name": "Product",
		"fields": [
			{"name": "id", "type": "int"},
			{"name": "name", "type": "string"},
			{"name": "price", "type": "double"},
			{"name": "description", "type": "string", "default": ""}
		]
	}`

	// Test compatibility
	compatible, err := client.TestCompatibility(subject, compatibleSchema, schema.SchemaTypeAvro)
	if err != nil {
		log.Fatalf("Compatibility test failed: %v", err)
	}

	fmt.Printf("✓ Compatible schema test: %v\n", compatible)

	// Incompatible schema (removes required field)
	incompatibleSchema := `{
		"type": "record",
		"name": "Product",
		"fields": [
			{"name": "id", "type": "int"},
			{"name": "name", "type": "string"}
		]
	}`

	// This should fail compatibility check
	compatible, err = client.TestCompatibility(subject, incompatibleSchema, schema.SchemaTypeAvro)
	if err != nil {
		log.Fatalf("Compatibility test failed: %v", err)
	}

	fmt.Printf("✓ Incompatible schema test: %v\n", compatible)

	// Set compatibility mode
	err = client.SetCompatibility(subject, schema.CompatibilityFull)
	if err != nil {
		log.Fatalf("Failed to set compatibility: %v", err)
	}

	mode, err := client.GetCompatibility(subject)
	if err != nil {
		log.Fatalf("Failed to get compatibility: %v", err)
	}

	fmt.Printf("✓ Compatibility mode set to: %s\n", mode)
}

// Example helper function to pretty print JSON
func prettyPrint(v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("%v\n", v)
		return
	}
	fmt.Println(string(b))
}
