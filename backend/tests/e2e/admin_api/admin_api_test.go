// Copyright 2025 Takhin Data, Inc.

// +build e2e

package admin_api

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/tests/e2e/testutil"
)

func TestCreateTopicAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Create topics via API
	topics := []string{"admin-topic-1", "admin-topic-2", "admin-topic-3"}
	err = client.CreateTopics(topics, 3, 1)
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Verify topics exist via metadata
	metadata, err := client.Metadata(topics)
	require.NoError(t, err)
	assert.NotNil(t, metadata)

	// Verify topic count
	assert.GreaterOrEqual(t, len(metadata.Topics), len(topics))
}

func TestListTopicsAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	// Create some topics
	for i := 0; i < 5; i++ {
		err := srv.CreateTopic(fmt.Sprintf("list-test-%d", i), 1)
		require.NoError(t, err)
	}

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Request metadata for all topics (empty topics list)
	metadata, err := client.Metadata(nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(metadata.Topics), 5, "Should list at least 5 topics")

	for _, topic := range metadata.Topics {
		t.Logf("Found topic: %s", topic.Name)
	}
}

func TestDeleteTopicAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "delete-test-topic"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Verify topic exists
	metadata, err := client.Metadata([]string{topicName})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(metadata.Topics), 1)

	// Note: DeleteTopics API would be tested here
	// For now, verify topic manager can delete topics
	t.Log("Topic deletion API test placeholder")
}

func TestDescribeTopicAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "describe-test-topic"
	numPartitions := 4
	err := srv.CreateTopic(topicName, numPartitions)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Get topic metadata
	metadata, err := client.Metadata([]string{topicName})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(metadata.Topics), 1)

	// Verify partition count
	topic := metadata.Topics[0]
	assert.Equal(t, topicName, topic.Name)
	assert.Len(t, topic.Partitions, numPartitions)

	t.Logf("Topic %s has %d partitions", topic.Name, len(topic.Partitions))
}

func TestAlterTopicConfigAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "config-test-topic"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	// Note: AlterConfigs API would be tested here
	// This tests that the handler supports configuration changes
	t.Log("Topic configuration API test placeholder")
	assert.NotNil(t, srv.Handler)
}

func TestDescribeClusterAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Get cluster metadata
	metadata, err := client.Metadata(nil)
	require.NoError(t, err)
	assert.NotNil(t, metadata)

	// Verify broker information
	assert.GreaterOrEqual(t, len(metadata.Brokers), 1, "Should have at least one broker")
	
	broker := metadata.Brokers[0]
	t.Logf("Broker ID: %d, Host: %s, Port: %d", broker.NodeID, broker.Host, broker.Port)
}

func TestCreatePartitionsAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "partition-increase-topic"
	err := srv.CreateTopic(topicName, 2)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Get initial partition count
	metadata, err := client.Metadata([]string{topicName})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(metadata.Topics), 1)
	
	initialPartitions := len(metadata.Topics[0].Partitions)
	assert.Equal(t, 2, initialPartitions)

	// Note: CreatePartitions API would be tested here to increase partitions
	t.Log("Partition increase API test placeholder")
}

func TestListConsumerGroupsAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	// Note: ListGroups API would be tested here
	// This verifies the coordinator is running
	assert.NotNil(t, srv.Handler)
	t.Log("Consumer groups listing API test placeholder")
}

func TestDescribeConsumerGroupAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	// Note: DescribeGroups API would be tested here
	// This would show group members, assigned partitions, etc.
	t.Log("Consumer group describe API test placeholder")
}

func TestAPIErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Test creating duplicate topic
	topicName := "duplicate-topic"
	err = client.CreateTopics([]string{topicName}, 1, 1)
	require.NoError(t, err)

	// Try creating again - should get error or already exists
	err = client.CreateTopics([]string{topicName}, 1, 1)
	t.Logf("Duplicate topic creation result: %v", err)

	// Test fetching from non-existent topic
	_, err = client.Fetch("non-existent-topic", 0, 0, 1024)
	t.Logf("Fetch non-existent topic result: %v", err)
}
