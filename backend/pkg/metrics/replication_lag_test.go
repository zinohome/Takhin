// Copyright 2025 Takhin Data, Inc.

package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestReplicationLagMetrics(t *testing.T) {
	tests := []struct {
		name          string
		topic         string
		partition     int32
		followerID    int32
		lag           int64
		expectedValue float64
	}{
		{
			name:          "zero lag",
			topic:         "test-topic",
			partition:     0,
			followerID:    2,
			lag:           0,
			expectedValue: 0,
		},
		{
			name:          "small lag",
			topic:         "test-topic",
			partition:     1,
			followerID:    3,
			lag:           100,
			expectedValue: 100,
		},
		{
			name:          "large lag",
			topic:         "test-topic-2",
			partition:     0,
			followerID:    2,
			lag:           10000,
			expectedValue: 10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset metric
			ReplicationLag.Reset()

			// Update metrics
			UpdateReplicationMetrics(tt.topic, tt.partition, tt.followerID, tt.lag, 2, 3)

			// Verify lag metric
			labels := prometheus.Labels{
				"topic":       tt.topic,
				"partition":   string(rune('0' + tt.partition)),
				"follower_id": string(rune('0' + tt.followerID)),
			}
			gauge := ReplicationLag.With(labels)
			value := testutil.ToFloat64(gauge)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}

func TestReplicationLagTimeMetrics(t *testing.T) {
	tests := []struct {
		name          string
		topic         string
		partition     int32
		followerID    int32
		lagMs         int64
		expectedValue float64
	}{
		{
			name:          "recent fetch",
			topic:         "test-topic",
			partition:     0,
			followerID:    2,
			lagMs:         100,
			expectedValue: 100,
		},
		{
			name:          "stale fetch",
			topic:         "test-topic",
			partition:     1,
			followerID:    3,
			lagMs:         15000,
			expectedValue: 15000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset metric
			ReplicationLagTimeMs.Reset()

			// Update metrics
			UpdateReplicationLagTime(tt.topic, tt.partition, tt.followerID, tt.lagMs)

			// Verify lag time metric
			labels := prometheus.Labels{
				"topic":       tt.topic,
				"partition":   string(rune('0' + tt.partition)),
				"follower_id": string(rune('0' + tt.followerID)),
			}
			gauge := ReplicationLagTimeMs.With(labels)
			value := testutil.ToFloat64(gauge)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}

func TestISRMetrics(t *testing.T) {
	tests := []struct {
		name              string
		topic             string
		partition         int32
		isrSize           int
		replicasTotal     int
		expectedISRSize   float64
		expectedReplicas  float64
		expectedUnderRepl float64
	}{
		{
			name:              "fully replicated",
			topic:             "test-topic",
			partition:         0,
			isrSize:           3,
			replicasTotal:     3,
			expectedISRSize:   3,
			expectedReplicas:  3,
			expectedUnderRepl: 0,
		},
		{
			name:              "under replicated",
			topic:             "test-topic",
			partition:         1,
			isrSize:           2,
			replicasTotal:     3,
			expectedISRSize:   2,
			expectedReplicas:  3,
			expectedUnderRepl: 1,
		},
		{
			name:              "single replica",
			topic:             "test-topic-2",
			partition:         0,
			isrSize:           1,
			replicasTotal:     1,
			expectedISRSize:   1,
			expectedReplicas:  1,
			expectedUnderRepl: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset metrics
			ReplicationISRSize.Reset()
			ReplicationReplicasTotal.Reset()
			ReplicationUnderReplicated.Reset()

			// Update metrics
			UpdateReplicationMetrics(tt.topic, tt.partition, 0, -1, tt.isrSize, tt.replicasTotal)

			// Verify ISR size
			labels := prometheus.Labels{
				"topic":     tt.topic,
				"partition": string(rune('0' + tt.partition)),
			}
			isrGauge := ReplicationISRSize.With(labels)
			assert.Equal(t, tt.expectedISRSize, testutil.ToFloat64(isrGauge))

			// Verify replicas total
			replicasGauge := ReplicationReplicasTotal.With(labels)
			assert.Equal(t, tt.expectedReplicas, testutil.ToFloat64(replicasGauge))

			// Verify under-replicated status
			underReplGauge := ReplicationUnderReplicated.With(labels)
			assert.Equal(t, tt.expectedUnderRepl, testutil.ToFloat64(underReplGauge))
		})
	}
}

func TestISRChangeMetrics(t *testing.T) {
	// Reset metrics
	ReplicationISRShrinks.Reset()
	ReplicationISRExpands.Reset()

	topic := "test-topic"
	partition := int32(0)

	// Record shrink
	RecordISRShrink(topic, partition)
	labels := prometheus.Labels{
		"topic":     topic,
		"partition": "0",
	}
	shrinkCounter := ReplicationISRShrinks.With(labels)
	assert.Equal(t, float64(1), testutil.ToFloat64(shrinkCounter))

	// Record another shrink
	RecordISRShrink(topic, partition)
	assert.Equal(t, float64(2), testutil.ToFloat64(shrinkCounter))

	// Record expand
	RecordISRExpand(topic, partition)
	expandCounter := ReplicationISRExpands.With(labels)
	assert.Equal(t, float64(1), testutil.ToFloat64(expandCounter))
}

func TestReplicationBytesMetrics(t *testing.T) {
	// Reset metrics
	ReplicationBytesInRate.Reset()
	ReplicationBytesOutRate.Reset()

	topic := "test-topic"
	partition := int32(0)

	// Record bytes in
	RecordReplicationBytesIn(topic, partition, 1024)
	RecordReplicationBytesIn(topic, partition, 2048)

	labels := prometheus.Labels{
		"topic":     topic,
		"partition": "0",
	}
	bytesInCounter := ReplicationBytesInRate.With(labels)
	assert.Equal(t, float64(3072), testutil.ToFloat64(bytesInCounter))

	// Record bytes out
	RecordReplicationBytesOut(topic, partition, 512)
	bytesOutCounter := ReplicationBytesOutRate.With(labels)
	assert.Equal(t, float64(512), testutil.ToFloat64(bytesOutCounter))
}

func TestCollectorReplicationMetrics(t *testing.T) {
	// Create temp directory for test
	dataDir := t.TempDir()

	// Create topic manager
	mgr := topic.NewManager(dataDir, 1024*1024)

	// Create topic with replicas
	err := mgr.CreateTopic("test-topic", 2)
	assert.NoError(t, err)

	// Get topic and set up replication
	testTopic, exists := mgr.GetTopic("test-topic")
	assert.True(t, exists)

	// Set replicas for partition 0
	testTopic.SetReplicas(0, []int32{1, 2, 3})
	testTopic.SetISR(0, []int32{1, 2, 3})

	// Update follower LEO
	testTopic.UpdateFollowerLEO(0, 2, 100)
	testTopic.UpdateFollowerLEO(0, 3, 95)

	// Create collector
	collector := NewCollector(mgr, nil, 30*time.Second)

	// Reset metrics
	ReplicationLag.Reset()
	ReplicationISRSize.Reset()
	ReplicationReplicasTotal.Reset()

	// Collect metrics
	collector.collectStorageMetrics()

	// Verify ISR size was recorded
	labels := prometheus.Labels{
		"topic":     "test-topic",
		"partition": "0",
	}
	isrGauge := ReplicationISRSize.With(labels)
	assert.Equal(t, float64(3), testutil.ToFloat64(isrGauge))

	// Verify replica count
	replicasGauge := ReplicationReplicasTotal.With(labels)
	assert.Equal(t, float64(3), testutil.ToFloat64(replicasGauge))
}

func TestCollectorISRChangeDetection(t *testing.T) {
	// Create temp directory for test
	dataDir := t.TempDir()

	// Create topic manager
	mgr := topic.NewManager(dataDir, 1024*1024)

	// Create topic with replicas
	err := mgr.CreateTopic("test-topic", 1)
	assert.NoError(t, err)

	// Get topic and set up replication
	testTopic, exists := mgr.GetTopic("test-topic")
	assert.True(t, exists)

	// Set replicas for partition 0
	testTopic.SetReplicas(0, []int32{1, 2, 3})
	testTopic.SetISR(0, []int32{1, 2, 3})

	// Create collector
	collector := NewCollector(mgr, nil, 30*time.Second)

	// Reset metrics
	ReplicationISRShrinks.Reset()
	ReplicationISRExpands.Reset()

	// First collection - establish baseline
	collector.collectStorageMetrics()

	// Shrink ISR
	testTopic.SetISR(0, []int32{1, 2})

	// Second collection - should detect shrink
	collector.collectStorageMetrics()

	// Verify shrink was recorded
	labels := prometheus.Labels{
		"topic":     "test-topic",
		"partition": "0",
	}
	shrinkCounter := ReplicationISRShrinks.With(labels)
	assert.Equal(t, float64(1), testutil.ToFloat64(shrinkCounter))

	// Expand ISR back
	testTopic.SetISR(0, []int32{1, 2, 3})

	// Third collection - should detect expand
	collector.collectStorageMetrics()

	// Verify expand was recorded
	expandCounter := ReplicationISRExpands.With(labels)
	assert.Equal(t, float64(1), testutil.ToFloat64(expandCounter))
}

func TestReplicationFetchMetrics(t *testing.T) {
	// Reset metrics
	ReplicationFetchRequestsTotal.Reset()
	ReplicationFetchLatency.Reset()

	followerID := int32(2)
	duration := 50 * time.Millisecond

	// Record fetch
	RecordReplicationFetch(followerID, duration)

	// Verify request count
	labels := prometheus.Labels{
		"follower_id": "2",
	}
	counter := ReplicationFetchRequestsTotal.With(labels)
	assert.Equal(t, float64(1), testutil.ToFloat64(counter))

	// Record another fetch
	RecordReplicationFetch(followerID, 100*time.Millisecond)
	assert.Equal(t, float64(2), testutil.ToFloat64(counter))
}
