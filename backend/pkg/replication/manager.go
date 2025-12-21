// Copyright 2025 Takhin Data, Inc.

package replication

import (
	"fmt"
	"sync"
)

// ReplicaManager manages all partitions for a broker
type ReplicaManager struct {
	brokerID   int32
	partitions map[string]*Partition // "topic-partition" -> Partition
	mu         sync.RWMutex
}

// NewReplicaManager creates a new replica manager
func NewReplicaManager(brokerID int32) *ReplicaManager {
	return &ReplicaManager{
		brokerID:   brokerID,
		partitions: make(map[string]*Partition),
	}
}

// AddPartition adds a partition to the replica manager
func (rm *ReplicaManager) AddPartition(partition *Partition) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	key := partitionKey(partition.TopicName, partition.PartitionID)
	if _, exists := rm.partitions[key]; exists {
		return fmt.Errorf("partition already exists: %s", key)
	}

	rm.partitions[key] = partition
	return nil
}

// GetPartition returns a partition by topic and partition ID
func (rm *ReplicaManager) GetPartition(topic string, partitionID int32) (*Partition, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	key := partitionKey(topic, partitionID)
	partition, exists := rm.partitions[key]
	return partition, exists
}

// RemovePartition removes a partition from the replica manager
func (rm *ReplicaManager) RemovePartition(topic string, partitionID int32) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	key := partitionKey(topic, partitionID)
	partition, exists := rm.partitions[key]
	if !exists {
		return fmt.Errorf("partition not found: %s", key)
	}

	if err := partition.Close(); err != nil {
		return fmt.Errorf("close partition: %w", err)
	}

	delete(rm.partitions, key)
	return nil
}

// ListPartitions returns all partition keys
func (rm *ReplicaManager) ListPartitions() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	keys := make([]string, 0, len(rm.partitions))
	for key := range rm.partitions {
		keys = append(keys, key)
	}
	return keys
}

// GetLeaderPartitions returns all partitions where this broker is the leader
func (rm *ReplicaManager) GetLeaderPartitions() []*Partition {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	leaders := make([]*Partition, 0)
	for _, partition := range rm.partitions {
		if partition.IsLeader(rm.brokerID) {
			leaders = append(leaders, partition)
		}
	}
	return leaders
}

// GetFollowerPartitions returns all partitions where this broker is a follower
func (rm *ReplicaManager) GetFollowerPartitions() []*Partition {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	followers := make([]*Partition, 0)
	for _, partition := range rm.partitions {
		if partition.IsFollower(rm.brokerID) {
			followers = append(followers, partition)
		}
	}
	return followers
}

// Close closes all partitions
func (rm *ReplicaManager) Close() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	var errs []error
	for _, partition := range rm.partitions {
		if err := partition.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close partitions: %v", errs)
	}
	return nil
}

// partitionKey generates a unique key for a partition
func partitionKey(topic string, partitionID int32) string {
	return fmt.Sprintf("%s-%d", topic, partitionID)
}
