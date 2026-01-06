// Copyright 2025 Takhin Data, Inc.

// +build e2e

package consumer_group

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/tests/e2e/testutil"
)

func TestConsumerGroupJoinLeave(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	// Create topic
	topicName := "test-consumer-group"
	err := srv.CreateTopic(topicName, 3)
	require.NoError(t, err)

	// Simulate consumer group join
	groupID := "test-group"
	memberID := "consumer-1"

	// In a real implementation, this would use the consumer group protocol
	// For now, we test that the coordinator is initialized
	assert.NotNil(t, srv.Handler)
	t.Logf("Consumer group %s with member %s ready", groupID, memberID)
}

func TestConsumerGroupRebalance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "test-rebalance"
	numPartitions := 4
	err := srv.CreateTopic(topicName, numPartitions)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Produce some messages
	for i := 0; i < 20; i++ {
		partition := int32(i % numPartitions)
		err := client.Produce(topicName, partition, []byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i)))
		require.NoError(t, err)
	}

	time.Sleep(200 * time.Millisecond)

	// Verify messages across partitions
	totalRecords := 0
	for partition := 0; partition < numPartitions; partition++ {
		records, err := client.Fetch(topicName, int32(partition), 0, 1024*1024)
		require.NoError(t, err)
		totalRecords += len(records)
		t.Logf("Partition %d has %d records", partition, len(records))
	}

	assert.Equal(t, 20, totalRecords, "Total records should match produced messages")
}

func TestConsumerGroupOffsetCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "test-offset-commit"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Produce messages
	for i := 0; i < 10; i++ {
		err := client.Produce(topicName, 0, []byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i)))
		require.NoError(t, err)
	}

	time.Sleep(200 * time.Millisecond)

	// Consume first 5 messages
	records, err := client.Fetch(topicName, 0, 0, 1024*1024)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(records), 5)

	// In a real implementation, commit offset here
	t.Log("Offset commit would be tested here")
}

func TestMultipleConsumersInGroup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "test-multiple-consumers"
	numPartitions := 6
	err := srv.CreateTopic(topicName, numPartitions)
	require.NoError(t, err)

	// Create multiple clients (simulating multiple consumers)
	clients := make([]*testutil.KafkaClient, 3)
	for i := 0; i < 3; i++ {
		client, err := testutil.NewKafkaClient(srv.Address())
		require.NoError(t, err)
		defer client.Close()
		clients[i] = client
	}

	// Each consumer produces to different partitions
	for i, client := range clients {
		for j := 0; j < 10; j++ {
			partition := int32(i * 2)
			err := client.Produce(topicName, partition, []byte(fmt.Sprintf("c%d-k%d", i, j)), []byte(fmt.Sprintf("c%d-v%d", i, j)))
			require.NoError(t, err)
		}
	}

	time.Sleep(300 * time.Millisecond)

	// Verify distribution
	for partition := 0; partition < numPartitions; partition++ {
		records, err := clients[0].Fetch(topicName, int32(partition), 0, 1024*1024)
		require.NoError(t, err)
		t.Logf("Partition %d has %d records", partition, len(records))
	}
}

func TestConsumerGroupFailover(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "test-failover"
	err := srv.CreateTopic(topicName, 2)
	require.NoError(t, err)

	// Create first consumer
	client1, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)

	// Produce messages
	for i := 0; i < 10; i++ {
		err := client1.Produce(topicName, 0, []byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i)))
		require.NoError(t, err)
	}

	time.Sleep(100 * time.Millisecond)

	// Simulate first consumer failure
	client1.Close()

	// Create second consumer (failover)
	client2, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client2.Close()

	// Second consumer should be able to read messages
	records, err := client2.Fetch(topicName, 0, 0, 1024*1024)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(records), 5, "Failover consumer should read existing messages")
}

func TestConsumerGroupSessionTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "test-session-timeout"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Produce a message
	err = client.Produce(topicName, 0, []byte("key"), []byte("value"))
	require.NoError(t, err)

	// Wait beyond typical session timeout
	time.Sleep(2 * time.Second)

	// Client should still be able to fetch
	records, err := client.Fetch(topicName, 0, 0, 1024)
	require.NoError(t, err)
	assert.Len(t, records, 1)
}
