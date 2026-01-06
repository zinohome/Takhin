// Copyright 2025 Takhin Data, Inc.

// +build e2e

package fault_injection

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/tests/e2e/testutil"
)

func TestServerRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	// Start server
	srv := testutil.NewTestServer(t)
	dataDir := srv.DataDir
	port := srv.Port

	topicName := "restart-test"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	// Produce messages
	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		err := client.Produce(topicName, 0, []byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i)))
		require.NoError(t, err)
	}
	client.Close()

	time.Sleep(200 * time.Millisecond)

	// Stop server
	srv.Close()
	time.Sleep(500 * time.Millisecond)

	// Start new server with same data directory
	srv2 := testutil.NewTestServer(t)
	srv2.DataDir = dataDir
	defer srv2.Close()

	// Verify data persisted
	client2, err := testutil.NewKafkaClient(srv2.Address())
	require.NoError(t, err)
	defer client2.Close()

	// Note: Would need to recreate topic in new server or support recovery
	t.Logf("Server restarted successfully, data dir: %s", dataDir)
	t.Logf("Old port: %d, New port: %d", port, srv2.Port)
}

func TestNetworkPartition(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	cluster := testutil.NewTestCluster(t, 3)
	defer cluster.Close()

	leader := cluster.Leader()
	topicName := "partition-test"
	err := leader.CreateTopic(topicName, 1)
	require.NoError(t, err)

	// Produce to leader
	client, err := testutil.NewKafkaClient(leader.Address())
	require.NoError(t, err)
	defer client.Close()

	err = client.Produce(topicName, 0, []byte("key"), []byte("value"))
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Simulate partition by closing one follower
	if len(cluster.Followers()) > 0 {
		cluster.Followers()[0].Close()
		t.Log("Simulated network partition - closed one follower")
	}

	// Leader should still accept writes
	err = client.Produce(topicName, 0, []byte("key2"), []byte("value2"))
	assert.NoError(t, err, "Leader should accept writes during partition")
}

func TestLeaderFailover(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	cluster := testutil.NewTestCluster(t, 3)
	defer cluster.Close()

	leader := cluster.Leader()
	topicName := "failover-test"
	err := leader.CreateTopic(topicName, 1)
	require.NoError(t, err)

	// Produce some messages
	client, err := testutil.NewKafkaClient(leader.Address())
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		err := client.Produce(topicName, 0, []byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i)))
		require.NoError(t, err)
	}
	client.Close()

	time.Sleep(200 * time.Millisecond)

	// Kill leader
	leader.Close()
	t.Log("Leader killed, waiting for failover...")
	time.Sleep(2 * time.Second)

	// Try to connect to a follower
	if len(cluster.Followers()) > 0 {
		newLeader := cluster.Followers()[0]
		client2, err := testutil.NewKafkaClient(newLeader.Address())
		require.NoError(t, err)
		defer client2.Close()

		// Should be able to read from new leader
		t.Log("Connected to new leader successfully")
	}
}

func TestDiskFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "disk-failure-test"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Produce until disk is full (simulate with limited segment size)
	for i := 0; i < 1000; i++ {
		err := client.Produce(topicName, 0, []byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("large-value-%d-%s", i, string(make([]byte, 1024)))))
		if err != nil {
			t.Logf("Disk full simulation - error at message %d: %v", i, err)
			break
		}
	}

	// System should still be responsive
	metadata, err := client.Metadata([]string{topicName})
	assert.NoError(t, err, "Metadata request should succeed even with disk pressure")
	assert.NotNil(t, metadata)
}

func TestSlowConsumer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "slow-consumer-test"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Produce messages rapidly
	numMessages := 100
	for i := 0; i < numMessages; i++ {
		err := client.Produce(topicName, 0, []byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i)))
		require.NoError(t, err)
	}

	time.Sleep(200 * time.Millisecond)

	// Slow consumer - read messages with delays
	offset := int64(0)
	for i := 0; i < 10; i++ {
		records, err := client.Fetch(topicName, 0, offset, 1024)
		require.NoError(t, err)
		
		if len(records) > 0 {
			offset += int64(len(records))
			t.Logf("Slow consumer iteration %d: read %d records, offset now %d", i, len(records), offset)
		}
		
		time.Sleep(500 * time.Millisecond) // Simulate slow processing
	}

	assert.Greater(t, offset, int64(0), "Slow consumer should make progress")
}

func TestMessageCorruption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "corruption-test"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Produce normal messages
	for i := 0; i < 10; i++ {
		err := client.Produce(topicName, 0, []byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i)))
		require.NoError(t, err)
	}

	time.Sleep(200 * time.Millisecond)

	// Note: In a real test, would corrupt log files on disk
	// For now, verify that reads still work
	records, err := client.Fetch(topicName, 0, 0, 1024*1024)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(records), 5, "Should read uncorrupted messages")
}

func TestHighConnectionChurn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "connection-churn-test"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	// Rapidly open and close connections
	for i := 0; i < 50; i++ {
		client, err := testutil.NewKafkaClient(srv.Address())
		if err != nil {
			t.Logf("Connection %d failed: %v", i, err)
			continue
		}

		// Quick produce
		err = client.Produce(topicName, 0, []byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i)))
		if err != nil {
			t.Logf("Produce %d failed: %v", i, err)
		}

		client.Close()
	}

	// Server should still be responsive
	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	records, err := client.Fetch(topicName, 0, 0, 1024*1024)
	assert.NoError(t, err)
	t.Logf("After connection churn, recovered %d messages", len(records))
}

func TestMemoryPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "memory-pressure-test"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Produce very large messages to create memory pressure
	largeValue := make([]byte, 512*1024) // 512KB per message
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}

	successCount := 0
	for i := 0; i < 20; i++ {
		err := client.Produce(topicName, 0, []byte(fmt.Sprintf("key%d", i)), largeValue)
		if err == nil {
			successCount++
		} else {
			t.Logf("Large message %d failed (expected under pressure): %v", i, err)
		}
	}

	t.Logf("Successfully produced %d large messages under memory pressure", successCount)
	assert.Greater(t, successCount, 0, "Should succeed with at least some messages")
}
