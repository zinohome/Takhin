// Copyright 2025 Takhin Data, Inc.

package replication

import (
	"fmt"
)

// ReplicaAssigner assigns replicas to brokers for topic partitions
type ReplicaAssigner struct {
	brokers []int32
}

// NewReplicaAssigner creates a new replica assigner
func NewReplicaAssigner(brokers []int32) *ReplicaAssigner {
	return &ReplicaAssigner{
		brokers: brokers,
	}
}

// AssignReplicas assigns replicas for a topic using round-robin algorithm
// Returns a map: partitionID -> []replicaIDs
func (ra *ReplicaAssigner) AssignReplicas(numPartitions int32, replicationFactor int16) (map[int32][]int32, error) {
	if len(ra.brokers) == 0 {
		return nil, fmt.Errorf("no brokers available")
	}

	if int(replicationFactor) > len(ra.brokers) {
		return nil, fmt.Errorf("replication factor %d exceeds number of brokers %d", replicationFactor, len(ra.brokers))
	}

	if replicationFactor <= 0 {
		return nil, fmt.Errorf("replication factor must be positive")
	}

	// Round-robin assignment
	// Example with 3 brokers [1,2,3], 3 partitions, RF=3:
	// Partition 0: [1, 2, 3] (leader=1)
	// Partition 1: [2, 3, 1] (leader=2)
	// Partition 2: [3, 1, 2] (leader=3)
	assignments := make(map[int32][]int32)

	for partitionID := int32(0); partitionID < numPartitions; partitionID++ {
		// Start from different broker for each partition to distribute leaders
		startIndex := int(partitionID) % len(ra.brokers)

		replicas := make([]int32, 0, replicationFactor)
		for i := 0; i < int(replicationFactor); i++ {
			brokerIndex := (startIndex + i) % len(ra.brokers)
			replicas = append(replicas, ra.brokers[brokerIndex])
		}

		assignments[partitionID] = replicas
	}

	return assignments, nil
}

// GetLeader returns the leader for a partition (first replica)
func GetLeader(replicas []int32) int32 {
	if len(replicas) == 0 {
		return -1
	}
	return replicas[0]
}

// ValidateAssignment validates a replica assignment
func ValidateAssignment(assignments map[int32][]int32, numPartitions int32, replicationFactor int16) error {
	if len(assignments) != int(numPartitions) {
		return fmt.Errorf("assignment has %d partitions, expected %d", len(assignments), numPartitions)
	}

	for partitionID := int32(0); partitionID < numPartitions; partitionID++ {
		replicas, exists := assignments[partitionID]
		if !exists {
			return fmt.Errorf("missing assignment for partition %d", partitionID)
		}

		if len(replicas) != int(replicationFactor) {
			return fmt.Errorf("partition %d has %d replicas, expected %d", partitionID, len(replicas), replicationFactor)
		}

		// Check for duplicate replicas
		seen := make(map[int32]bool)
		for _, replicaID := range replicas {
			if seen[replicaID] {
				return fmt.Errorf("partition %d has duplicate replica %d", partitionID, replicaID)
			}
			seen[replicaID] = true
		}
	}

	return nil
}
