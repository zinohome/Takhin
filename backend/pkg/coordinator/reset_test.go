// Copyright 2025 Takhin Data, Inc.

package coordinator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResetOffsets(t *testing.T) {
	coord := NewCoordinator()

	groupID := "test-group"

	// Create a group and commit some offsets
	_ = coord.CommitOffset(groupID, "topic1", 0, 100, "")
	_ = coord.CommitOffset(groupID, "topic1", 1, 200, "")

	// Verify initial offsets
	offset, exists := coord.FetchOffset(groupID, "topic1", 0)
	require.True(t, exists)
	assert.Equal(t, int64(100), offset.Offset)

	// Set group to Empty state so we can reset
	group, _ := coord.GetGroup(groupID)
	group.State = GroupStateEmpty

	// Reset offsets
	newOffsets := map[string]map[int32]int64{
		"topic1": {
			0: 50,
			1: 150,
		},
		"topic2": {
			0: 0,
		},
	}

	err := coord.ResetOffsets(groupID, newOffsets)
	require.NoError(t, err)

	// Verify offsets were reset
	offset, exists = coord.FetchOffset(groupID, "topic1", 0)
	require.True(t, exists)
	assert.Equal(t, int64(50), offset.Offset)

	offset, exists = coord.FetchOffset(groupID, "topic1", 1)
	require.True(t, exists)
	assert.Equal(t, int64(150), offset.Offset)

	offset, exists = coord.FetchOffset(groupID, "topic2", 0)
	require.True(t, exists)
	assert.Equal(t, int64(0), offset.Offset)
}

func TestResetOffsetsInvalidState(t *testing.T) {
	coord := NewCoordinator()

	groupID := "test-group"

	// Create a group in Stable state
	group := coord.GetOrCreateGroup(groupID, "consumer")
	group.State = GroupStateStable

	// Try to reset offsets - should fail
	newOffsets := map[string]map[int32]int64{
		"topic1": {0: 100},
	}

	err := coord.ResetOffsets(groupID, newOffsets)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot reset offsets")
}

func TestDeleteGroupOffsets(t *testing.T) {
	coord := NewCoordinator()

	groupID := "test-group"

	// Create a group and commit some offsets
	_ = coord.CommitOffset(groupID, "topic1", 0, 100, "")
	_ = coord.CommitOffset(groupID, "topic1", 1, 200, "")
	_ = coord.CommitOffset(groupID, "topic2", 0, 50, "")

	// Verify offsets exist
	offset, exists := coord.FetchOffset(groupID, "topic1", 0)
	require.True(t, exists)
	assert.Equal(t, int64(100), offset.Offset)

	// Set group to Empty state
	group, _ := coord.GetGroup(groupID)
	group.State = GroupStateEmpty

	// Delete all offsets
	err := coord.DeleteGroupOffsets(groupID)
	require.NoError(t, err)

	// Verify all offsets are gone
	_, exists = coord.FetchOffset(groupID, "topic1", 0)
	assert.False(t, exists)

	_, exists = coord.FetchOffset(groupID, "topic1", 1)
	assert.False(t, exists)

	_, exists = coord.FetchOffset(groupID, "topic2", 0)
	assert.False(t, exists)
}

func TestForceDeleteGroup(t *testing.T) {
	coord := NewCoordinator()

	groupID := "test-group"

	// Create a group with members and offsets
	group := coord.GetOrCreateGroup(groupID, "consumer")
	member := &Member{
		ID:           "member1",
		ClientID:     "client1",
		ClientHost:   "localhost",
		ProtocolType: "consumer",
	}
	_ = group.AddMember(member)
	_ = coord.CommitOffset(groupID, "topic1", 0, 100, "")

	// Verify group exists
	_, exists := coord.GetGroup(groupID)
	require.True(t, exists)

	// Force delete the group
	err := coord.ForceDeleteGroup(groupID)
	require.NoError(t, err)

	// Verify group is gone
	_, exists = coord.GetGroup(groupID)
	assert.False(t, exists)

	// Verify it's also removed from list
	groups := coord.ListGroups()
	assert.NotContains(t, groups, groupID)
}

func TestForceDeleteGroupNotFound(t *testing.T) {
	coord := NewCoordinator()

	err := coord.ForceDeleteGroup("nonexistent-group")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCanDeleteGroup(t *testing.T) {
	tests := []struct {
		name       string
		state      GroupState
		hasMembers bool
		canDelete  bool
		reason     string
	}{
		{
			name:       "empty group can be deleted",
			state:      GroupStateEmpty,
			hasMembers: false,
			canDelete:  true,
			reason:     "",
		},
		{
			name:       "dead group can be deleted",
			state:      GroupStateDead,
			hasMembers: false,
			canDelete:  true,
			reason:     "",
		},
		{
			name:       "stable group cannot be deleted",
			state:      GroupStateStable,
			hasMembers: false,
			canDelete:  false,
			reason:     "group is in Stable state",
		},
		{
			name:       "empty group with members cannot be deleted",
			state:      GroupStateEmpty,
			hasMembers: true,
			canDelete:  false,
			reason:     "group has active members",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := NewCoordinator()
			groupID := "test-group"

			group := coord.GetOrCreateGroup(groupID, "consumer")
			group.State = tt.state

			if tt.hasMembers {
				member := &Member{
					ID:           "member1",
					ClientID:     "client1",
					ProtocolType: "consumer",
				}
				group.Members["member1"] = member
			}

			canDelete, reason := coord.CanDeleteGroup(groupID)
			assert.Equal(t, tt.canDelete, canDelete)
			if !tt.canDelete {
				assert.Contains(t, reason, tt.reason)
			}
		})
	}
}

func TestCanDeleteGroupNotFound(t *testing.T) {
	coord := NewCoordinator()

	canDelete, reason := coord.CanDeleteGroup("nonexistent-group")
	assert.False(t, canDelete)
	assert.Contains(t, reason, "not found")
}

func TestResetOffsetsForMultipleTopics(t *testing.T) {
	coord := NewCoordinator()

	groupID := "test-group"

	// Create group in Empty state
	group := coord.GetOrCreateGroup(groupID, "consumer")
	group.State = GroupStateEmpty

	// Reset offsets for multiple topics with multiple partitions
	newOffsets := map[string]map[int32]int64{
		"topic1": {
			0: 100,
			1: 200,
			2: 300,
		},
		"topic2": {
			0: 50,
			1: 150,
		},
		"topic3": {
			0: 0,
		},
	}

	err := coord.ResetOffsets(groupID, newOffsets)
	require.NoError(t, err)

	// Verify all offsets were set correctly
	for topic, partitions := range newOffsets {
		for partition, expectedOffset := range partitions {
			offset, exists := coord.FetchOffset(groupID, topic, partition)
			require.True(t, exists, "offset should exist for %s:%d", topic, partition)
			assert.Equal(t, expectedOffset, offset.Offset,
				"offset mismatch for %s:%d", topic, partition)
		}
	}

	// Verify GetGroupTopics returns all topics
	topics := coord.GetGroupTopics(groupID)
	assert.Len(t, topics, 3)
	assert.Contains(t, topics, "topic1")
	assert.Contains(t, topics, "topic2")
	assert.Contains(t, topics, "topic3")
}

func TestDeleteGroupOffsetsThenRecommit(t *testing.T) {
	coord := NewCoordinator()

	groupID := "test-group"

	// Commit initial offsets
	_ = coord.CommitOffset(groupID, "topic1", 0, 100, "")

	// Set group to Empty state
	group, _ := coord.GetGroup(groupID)
	group.State = GroupStateEmpty

	// Delete offsets
	err := coord.DeleteGroupOffsets(groupID)
	require.NoError(t, err)

	// Verify offset is gone
	_, exists := coord.FetchOffset(groupID, "topic1", 0)
	assert.False(t, exists)

	// Commit new offset
	_ = coord.CommitOffset(groupID, "topic1", 0, 50, "new commit")

	// Verify new offset is there
	offset, exists := coord.FetchOffset(groupID, "topic1", 0)
	require.True(t, exists)
	assert.Equal(t, int64(50), offset.Offset)
}
