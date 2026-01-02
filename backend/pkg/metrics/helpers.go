// Copyright 2025 Takhin Data, Inc.

package metrics

import (
	"strconv"
	"time"
)

// RecordKafkaRequest records metrics for a Kafka API request
func RecordKafkaRequest(apiKey int16, version int16, duration time.Duration, errorCode int16) {
	apiKeyStr := strconv.Itoa(int(apiKey))
	versionStr := strconv.Itoa(int(version))

	KafkaRequestsTotal.WithLabelValues(apiKeyStr, versionStr).Inc()
	KafkaRequestDuration.WithLabelValues(apiKeyStr).Observe(duration.Seconds())

	if errorCode != 0 {
		errorCodeStr := strconv.Itoa(int(errorCode))
		KafkaRequestErrors.WithLabelValues(apiKeyStr, errorCodeStr).Inc()
	}
}

// RecordProduceRequest records metrics for a produce request
func RecordProduceRequest(topic string, partition int32, messages int, bytes int64, duration time.Duration) {
	partitionStr := strconv.Itoa(int(partition))

	ProduceRequestsTotal.WithLabelValues(topic).Inc()
	ProduceMessagesTotal.WithLabelValues(topic, partitionStr).Add(float64(messages))
	ProduceBytesTotal.WithLabelValues(topic).Add(float64(bytes))
	ProduceLatency.WithLabelValues(topic).Observe(duration.Seconds())

	StorageIOWrites.WithLabelValues(topic).Inc()
}

// RecordFetchRequest records metrics for a fetch request
func RecordFetchRequest(topic string, partition int32, messages int, bytes int64, duration time.Duration) {
	partitionStr := strconv.Itoa(int(partition))

	FetchRequestsTotal.WithLabelValues(topic).Inc()
	FetchMessagesTotal.WithLabelValues(topic, partitionStr).Add(float64(messages))
	FetchBytesTotal.WithLabelValues(topic).Add(float64(bytes))
	FetchLatency.WithLabelValues(topic).Observe(duration.Seconds())

	StorageIOReads.WithLabelValues(topic).Inc()
}

// UpdateStorageMetrics updates storage-related metrics for a topic partition
func UpdateStorageMetrics(topic string, partition int32, diskUsage int64, segments int, logEndOffset int64, activeSegmentSize int64) {
	partitionStr := strconv.Itoa(int(partition))

	StorageDiskUsageBytes.WithLabelValues(topic, partitionStr).Set(float64(diskUsage))
	StorageLogSegments.WithLabelValues(topic, partitionStr).Set(float64(segments))
	StorageLogEndOffset.WithLabelValues(topic, partitionStr).Set(float64(logEndOffset))
	StorageActiveSegmentBytes.WithLabelValues(topic, partitionStr).Set(float64(activeSegmentSize))
}

// RecordStorageError records a storage I/O error
func RecordStorageError(topic string, operation string) {
	StorageIOErrors.WithLabelValues(topic, operation).Inc()
}

// UpdateReplicationMetrics updates replication metrics for a partition
func UpdateReplicationMetrics(topic string, partition int32, followerID int32, lag int64, isrSize int, replicasTotal int) {
	partitionStr := strconv.Itoa(int(partition))
	followerIDStr := strconv.Itoa(int(followerID))

	if lag >= 0 {
		ReplicationLag.WithLabelValues(topic, partitionStr, followerIDStr).Set(float64(lag))
	}

	ReplicationISRSize.WithLabelValues(topic, partitionStr).Set(float64(isrSize))
	ReplicationReplicasTotal.WithLabelValues(topic, partitionStr).Set(float64(replicasTotal))

	// Set under-replicated status
	if isrSize < replicasTotal {
		ReplicationUnderReplicated.WithLabelValues(topic, partitionStr).Set(1)
	} else {
		ReplicationUnderReplicated.WithLabelValues(topic, partitionStr).Set(0)
	}
}

// UpdateReplicationLagTime updates replication lag time metrics
func UpdateReplicationLagTime(topic string, partition int32, followerID int32, lagMs int64) {
	partitionStr := strconv.Itoa(int(partition))
	followerIDStr := strconv.Itoa(int(followerID))

	ReplicationLagTimeMs.WithLabelValues(topic, partitionStr, followerIDStr).Set(float64(lagMs))
}

// RecordISRShrink records an ISR shrink event
func RecordISRShrink(topic string, partition int32) {
	partitionStr := strconv.Itoa(int(partition))
	ReplicationISRShrinks.WithLabelValues(topic, partitionStr).Inc()
}

// RecordISRExpand records an ISR expand event
func RecordISRExpand(topic string, partition int32) {
	partitionStr := strconv.Itoa(int(partition))
	ReplicationISRExpands.WithLabelValues(topic, partitionStr).Inc()
}

// RecordReplicationBytesIn records bytes received from leader
func RecordReplicationBytesIn(topic string, partition int32, bytes int64) {
	partitionStr := strconv.Itoa(int(partition))
	ReplicationBytesInRate.WithLabelValues(topic, partitionStr).Add(float64(bytes))
}

// RecordReplicationBytesOut records bytes sent to followers
func RecordReplicationBytesOut(topic string, partition int32, bytes int64) {
	partitionStr := strconv.Itoa(int(partition))
	ReplicationBytesOutRate.WithLabelValues(topic, partitionStr).Add(float64(bytes))
}

// RecordReplicationFetch records a replication fetch request
func RecordReplicationFetch(followerID int32, duration time.Duration) {
	followerIDStr := strconv.Itoa(int(followerID))

	ReplicationFetchRequestsTotal.WithLabelValues(followerIDStr).Inc()
	ReplicationFetchLatency.WithLabelValues(followerIDStr).Observe(duration.Seconds())
}

// UpdateConsumerGroupMetrics updates consumer group metrics
func UpdateConsumerGroupMetrics(groupID string, members int, state string) {
	ConsumerGroupMembers.WithLabelValues(groupID).Set(float64(members))

	// Reset all state gauges for this group
	for _, s := range []string{"Dead", "Empty", "PreparingRebalance", "CompletingRebalance", "Stable"} {
		if s == state {
			ConsumerGroupState.WithLabelValues(groupID, s).Set(1)
		} else {
			ConsumerGroupState.WithLabelValues(groupID, s).Set(0)
		}
	}
}

// RecordConsumerGroupRebalance records a consumer group rebalance event
func RecordConsumerGroupRebalance(groupID string) {
	ConsumerGroupRebalances.WithLabelValues(groupID).Inc()
}

// UpdateConsumerGroupLag updates consumer group lag metrics
func UpdateConsumerGroupLag(groupID string, topic string, partition int32, lag int64) {
	partitionStr := strconv.Itoa(int(partition))
	ConsumerGroupLag.WithLabelValues(groupID, topic, partitionStr).Set(float64(lag))
}

// RecordConsumerGroupCommit records an offset commit
func RecordConsumerGroupCommit(groupID string, topic string) {
	ConsumerGroupCommitRate.WithLabelValues(groupID, topic).Inc()
}
