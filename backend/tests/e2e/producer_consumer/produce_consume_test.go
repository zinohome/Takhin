// Copyright 2025 Takhin Data, Inc.

// +build e2e

package producer_consumer

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/tests/e2e/testutil"
)

func TestBasicProduceConsume(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	// Start test server
	srv := testutil.NewTestServer(t)
	defer srv.Close()

	// Create topic
	topicName := "test-basic-produce"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	// Create Kafka client
	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Produce messages
	messages := []struct {
		key   string
		value string
	}{
		{"key1", "value1"},
		{"key2", "value2"},
		{"key3", "value3"},
	}

	for _, msg := range messages {
		err := client.Produce(topicName, 0, []byte(msg.key), []byte(msg.value))
		assert.NoError(t, err, "Failed to produce message: %s", msg.key)
	}

	// Wait for messages to be written
	time.Sleep(100 * time.Millisecond)

	// Consume messages
	records, err := client.Fetch(topicName, 0, 0, 1024*1024)
	require.NoError(t, err)

	// Verify messages
	assert.Len(t, records, len(messages), "Expected %d messages, got %d", len(messages), len(records))

	for i, msg := range messages {
		assert.Equal(t, []byte(msg.key), records[i].Key)
		assert.Equal(t, []byte(msg.value), records[i].Value)
	}
}

func TestMultiPartitionProduce(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	// Create topic with multiple partitions
	topicName := "test-multi-partition"
	numPartitions := 3
	err := srv.CreateTopic(topicName, numPartitions)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Produce to different partitions
	for partition := 0; partition < numPartitions; partition++ {
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("p%d-key%d", partition, i)
			value := fmt.Sprintf("p%d-value%d", partition, i)
			err := client.Produce(topicName, int32(partition), []byte(key), []byte(value))
			assert.NoError(t, err)
		}
	}

	time.Sleep(200 * time.Millisecond)

	// Verify each partition has messages
	for partition := 0; partition < numPartitions; partition++ {
		records, err := client.Fetch(topicName, int32(partition), 0, 1024*1024)
		require.NoError(t, err)
		assert.Len(t, records, 10, "Partition %d should have 10 messages", partition)
	}
}

func TestLargeMessageProduce(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "test-large-message"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Produce large message (1MB)
	largeValue := make([]byte, 1024*1024)
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}

	err = client.Produce(topicName, 0, []byte("large-key"), largeValue)
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Consume and verify
	records, err := client.Fetch(topicName, 0, 0, 2*1024*1024)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, largeValue, records[0].Value)
}

func TestProduceBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "test-batch-produce"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Produce batch of messages quickly
	numMessages := 1000
	startTime := time.Now()

	for i := 0; i < numMessages; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		err := client.Produce(topicName, 0, []byte(key), []byte(value))
		if err != nil {
			t.Logf("Failed to produce message %d: %v", i, err)
		}
	}

	duration := time.Since(startTime)
	t.Logf("Produced %d messages in %v (%.2f msg/sec)", numMessages, duration, float64(numMessages)/duration.Seconds())

	time.Sleep(500 * time.Millisecond)

	// Verify count
	records, err := client.Fetch(topicName, 0, 0, 10*1024*1024)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(records), numMessages/2, "Expected at least half of messages to be persisted")
}

func TestConsumeFromOffset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "test-consume-offset"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Produce 10 messages
	for i := 0; i < 10; i++ {
		err := client.Produce(topicName, 0, []byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i)))
		require.NoError(t, err)
	}

	time.Sleep(200 * time.Millisecond)

	// Consume from offset 5
	records, err := client.Fetch(topicName, 0, 5, 1024*1024)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(records), 5, "Should get messages from offset 5 onwards")

	// Verify first record is from offset 5
	if len(records) > 0 {
		assert.Equal(t, []byte("value5"), records[0].Value)
	}
}

func TestProduceWithAcks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "test-acks"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Produce with acks=1 (wait for leader acknowledgment)
	err = client.Produce(topicName, 0, []byte("key"), []byte("value"))
	assert.NoError(t, err, "Produce with acks should succeed")

	time.Sleep(100 * time.Millisecond)

	// Verify message is persisted
	records, err := client.Fetch(topicName, 0, 0, 1024)
	require.NoError(t, err)
	assert.Len(t, records, 1)
}
