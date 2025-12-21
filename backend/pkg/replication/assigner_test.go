// Copyright 2025 Takhin Data, Inc.

package replication

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignReplicasSimple(t *testing.T) {
	brokers := []int32{1, 2, 3}
	assigner := NewReplicaAssigner(brokers)

	assignments, err := assigner.AssignReplicas(3, 3)
	require.NoError(t, err)
	assert.Equal(t, 3, len(assignments))

	// Partition 0: [1, 2, 3]
	assert.Equal(t, []int32{1, 2, 3}, assignments[0])
	// Partition 1: [2, 3, 1]
	assert.Equal(t, []int32{2, 3, 1}, assignments[1])
	// Partition 2: [3, 1, 2]
	assert.Equal(t, []int32{3, 1, 2}, assignments[2])
}

func TestGetLeader(t *testing.T) {
	assert.Equal(t, int32(1), GetLeader([]int32{1, 2, 3}))
	assert.Equal(t, int32(5), GetLeader([]int32{5}))
	assert.Equal(t, int32(-1), GetLeader([]int32{}))
}
