// Copyright 2025 Takhin Data, Inc.

package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRecordKafkaRequest(t *testing.T) {
	// Test recording a successful request
	RecordKafkaRequest(1, 5, 100*time.Millisecond, 0)

	// Test recording a failed request
	RecordKafkaRequest(1, 5, 200*time.Millisecond, 1)
}

func TestRecordProduceRequest(t *testing.T) {
	RecordProduceRequest("test-topic", 0, 10, 1024, 50*time.Millisecond)

	// Verify metrics were recorded (basic smoke test)
	// In real tests, you would use prometheus testutil to validate actual values
}

func TestRecordFetchRequest(t *testing.T) {
	RecordFetchRequest("test-topic", 0, 5, 512, 25*time.Millisecond)
}

func TestUpdateStorageMetrics(t *testing.T) {
	UpdateStorageMetrics("test-topic", 0, 10485760, 5, 1000, 2097152)
}

func TestUpdateReplicationMetrics(t *testing.T) {
	UpdateReplicationMetrics("test-topic", 0, 2, 10, 3, 3)
	UpdateReplicationMetrics("test-topic", 0, 3, 5, 3, 3)
}

func TestRecordReplicationFetch(t *testing.T) {
	RecordReplicationFetch(2, 30*time.Millisecond)
}

func TestUpdateConsumerGroupMetrics(t *testing.T) {
	UpdateConsumerGroupMetrics("test-group", 5, "Stable")
	UpdateConsumerGroupMetrics("test-group", 0, "Empty")
}

func TestRecordConsumerGroupRebalance(t *testing.T) {
	RecordConsumerGroupRebalance("test-group")
}

func TestUpdateConsumerGroupLag(t *testing.T) {
	UpdateConsumerGroupLag("test-group", "test-topic", 0, 100)
}

func TestRecordConsumerGroupCommit(t *testing.T) {
	RecordConsumerGroupCommit("test-group", "test-topic")
}

func TestRecordStorageError(t *testing.T) {
	RecordStorageError("test-topic", "read")
	RecordStorageError("test-topic", "write")
}

func TestMetricsServer(t *testing.T) {
	// Test with disabled metrics
	server := &Server{
		stopChan: make(chan struct{}),
	}

	err := server.Stop()
	assert.NoError(t, err)
}
