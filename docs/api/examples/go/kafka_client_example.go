// Package main demonstrates how to use the Takhin Kafka API with kafka-go client
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	brokerAddr = "localhost:9092"
	topicName  = "demo-topic"
)

func main() {
	ctx := context.Background()

	// Example 1: Create topic (via admin operations)
	if err := createTopic(ctx); err != nil {
		log.Printf("Warning: failed to create topic: %v", err)
	}

	// Example 2: Produce messages
	if err := produceMessages(ctx); err != nil {
		log.Fatalf("Failed to produce messages: %v", err)
	}

	// Example 3: Consume messages
	if err := consumeMessages(ctx); err != nil {
		log.Fatalf("Failed to consume messages: %v", err)
	}

	// Example 4: Consumer group
	if err := consumerGroupExample(ctx); err != nil {
		log.Fatalf("Failed consumer group example: %v", err)
	}

	fmt.Println("All examples completed successfully!")
}

// createTopic creates a new topic using kafka-go admin client
func createTopic(ctx context.Context) error {
	conn, err := kafka.Dial("tcp", brokerAddr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("get controller: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("dial controller: %w", err)
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topicName,
			NumPartitions:     3,
			ReplicationFactor: 1,
		},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		return fmt.Errorf("create topics: %w", err)
	}

	fmt.Printf("Topic '%s' created successfully\n", topicName)
	return nil
}

// produceMessages demonstrates message production
func produceMessages(ctx context.Context) error {
	// Create a writer
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{brokerAddr},
		Topic:        topicName,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: -1, // Wait for all ISR replicas
		MaxAttempts:  3,
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
	})
	defer writer.Close()

	// Write messages
	messages := []kafka.Message{
		{
			Key:   []byte("user-1"),
			Value: []byte(`{"action":"login","timestamp":"2024-01-01T10:00:00Z"}`),
		},
		{
			Key:   []byte("user-2"),
			Value: []byte(`{"action":"purchase","amount":99.99}`),
		},
		{
			Key:   []byte("user-1"),
			Value: []byte(`{"action":"logout","timestamp":"2024-01-01T11:00:00Z"}`),
		},
	}

	for _, msg := range messages {
		err := writer.WriteMessages(ctx, msg)
		if err != nil {
			return fmt.Errorf("write message: %w", err)
		}
		fmt.Printf("Produced message: key=%s, value=%s\n", msg.Key, msg.Value)
	}

	fmt.Printf("Produced %d messages successfully\n", len(messages))
	return nil
}

// consumeMessages demonstrates message consumption
func consumeMessages(ctx context.Context) error {
	// Create a reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{brokerAddr},
		Topic:     topicName,
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  10e6, // 10MB
		MaxWait:   500 * time.Millisecond,
	})
	defer reader.Close()

	// Read messages
	fmt.Println("Reading messages from partition 0:")
	for i := 0; i < 3; i++ {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			return fmt.Errorf("read message: %w", err)
		}
		fmt.Printf("Consumed message: offset=%d, key=%s, value=%s\n",
			msg.Offset, msg.Key, msg.Value)
	}

	return nil
}

// consumerGroupExample demonstrates consumer group usage
func consumerGroupExample(ctx context.Context) error {
	// Create a consumer group
	groupID := "demo-consumer-group"
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddr},
		GroupID:  groupID,
		Topic:    topicName,
		MinBytes: 1,
		MaxBytes: 10e6,
		MaxWait:  500 * time.Millisecond,
	})
	defer reader.Close()

	fmt.Printf("Consumer group '%s' reading messages:\n", groupID)

	// Read and commit messages
	for i := 0; i < 3; i++ {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			return fmt.Errorf("fetch message: %w", err)
		}

		fmt.Printf("Group message: partition=%d, offset=%d, key=%s\n",
			msg.Partition, msg.Offset, msg.Key)

		// Commit the message
		if err := reader.CommitMessages(ctx, msg); err != nil {
			return fmt.Errorf("commit message: %w", err)
		}
	}

	return nil
}

// Example: List topics
func listTopics(ctx context.Context) error {
	conn, err := kafka.Dial("tcp", brokerAddr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return fmt.Errorf("read partitions: %w", err)
	}

	// Group partitions by topic
	topics := make(map[string][]kafka.Partition)
	for _, p := range partitions {
		topics[p.Topic] = append(topics[p.Topic], p)
	}

	fmt.Println("Available topics:")
	for topic, parts := range topics {
		fmt.Printf("  %s: %d partitions\n", topic, len(parts))
	}

	return nil
}

// Example: Get offset information
func getOffsets(ctx context.Context, topic string, partition int) error {
	conn, err := kafka.DialLeader(ctx, "tcp", brokerAddr, topic, partition)
	if err != nil {
		return fmt.Errorf("dial leader: %w", err)
	}
	defer conn.Close()

	// Get first offset
	firstOffset, err := conn.ReadFirstOffset()
	if err != nil {
		return fmt.Errorf("read first offset: %w", err)
	}

	// Get last offset
	lastOffset, err := conn.ReadLastOffset()
	if err != nil {
		return fmt.Errorf("read last offset: %w", err)
	}

	fmt.Printf("Topic '%s' partition %d:\n", topic, partition)
	fmt.Printf("  First offset: %d\n", firstOffset)
	fmt.Printf("  Last offset: %d\n", lastOffset)
	fmt.Printf("  Messages: %d\n", lastOffset-firstOffset)

	return nil
}

// Example: Transactional producer
func transactionalProduce(ctx context.Context) error {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{brokerAddr},
		Topic:        topicName,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: -1,
		// Enable idempotence and transactions
		Idempotent: true,
	})
	defer writer.Close()

	// Begin transaction
	txn := writer.BeginTransaction()

	messages := []kafka.Message{
		{Key: []byte("txn-1"), Value: []byte("message 1")},
		{Key: []byte("txn-2"), Value: []byte("message 2")},
	}

	// Write messages in transaction
	if err := txn.WriteMessages(ctx, messages...); err != nil {
		txn.Abort()
		return fmt.Errorf("write messages: %w", err)
	}

	// Commit transaction
	if err := txn.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	fmt.Println("Transaction committed successfully")
	return nil
}
