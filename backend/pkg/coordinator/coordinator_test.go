// Copyright 2025 Takhin Data, Inc.

package coordinator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroup(t *testing.T) {
	group := NewGroup("test-group", "consumer")

	assert.Equal(t, "test-group", group.ID)
	assert.Equal(t, "consumer", group.ProtocolType)
	assert.Equal(t, GroupStateEmpty, group.State)
	assert.Equal(t, int32(0), group.Generation)
	assert.True(t, group.IsEmpty())
}

func TestGroupAddMember(t *testing.T) {
	group := NewGroup("test-group", "consumer")

	member := &Member{
		ID:             "member1",
		ClientID:       "client1",
		ProtocolType:   "consumer",
		SessionTimeout: 30000,
		Protocols: []MemberProtocol{
			{Name: "range", Metadata: []byte("metadata")},
		},
	}

	err := group.AddMember(member)
	require.NoError(t, err)

	assert.False(t, group.IsEmpty())
	assert.Equal(t, 1, group.Size())
	assert.True(t, group.HasMember("member1"))

	retrieved, exists := group.GetMember("member1")
	require.True(t, exists)
	assert.Equal(t, "member1", retrieved.ID)
	assert.Equal(t, MemberStateJoining, retrieved.State)
}

func TestGroupAddMemberProtocolMismatch(t *testing.T) {
	group := NewGroup("test-group", "consumer")

	member := &Member{
		ID:           "member1",
		ProtocolType: "connect", // Wrong protocol
	}

	err := group.AddMember(member)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "protocol type mismatch")
}

func TestGroupRemoveMember(t *testing.T) {
	group := NewGroup("test-group", "consumer")

	member := &Member{
		ID:           "member1",
		ProtocolType: "consumer",
	}

	group.AddMember(member)
	assert.Equal(t, 1, group.Size())

	group.RemoveMember("member1")
	assert.Equal(t, 0, group.Size())
	assert.True(t, group.IsEmpty())
	assert.Equal(t, GroupStateEmpty, group.State)
}

func TestGroupSelectProtocol(t *testing.T) {
	group := NewGroup("test-group", "consumer")

	// Add members with different protocol preferences
	member1 := &Member{
		ID:           "member1",
		ProtocolType: "consumer",
		Protocols: []MemberProtocol{
			{Name: "range", Metadata: []byte("m1")},
			{Name: "roundrobin", Metadata: []byte("m1")},
		},
	}
	member2 := &Member{
		ID:           "member2",
		ProtocolType: "consumer",
		Protocols: []MemberProtocol{
			{Name: "range", Metadata: []byte("m2")},
		},
	}

	group.Members["member1"] = member1
	group.Members["member2"] = member2

	// Should select "range" (supported by all)
	protocol, err := group.SelectProtocol()
	require.NoError(t, err)
	assert.Equal(t, "range", protocol)
}

func TestGroupSelectProtocolNoCommon(t *testing.T) {
	group := NewGroup("test-group", "consumer")

	member1 := &Member{
		ID:           "member1",
		ProtocolType: "consumer",
		Protocols: []MemberProtocol{
			{Name: "range", Metadata: []byte("m1")},
		},
	}
	member2 := &Member{
		ID:           "member2",
		ProtocolType: "consumer",
		Protocols: []MemberProtocol{
			{Name: "roundrobin", Metadata: []byte("m2")},
		},
	}

	group.Members["member1"] = member1
	group.Members["member2"] = member2

	_, err := group.SelectProtocol()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no protocol supported by all members")
}

func TestGroupCommitAndFetchOffset(t *testing.T) {
	group := NewGroup("test-group", "consumer")

	// Commit offset
	group.CommitOffset("test-topic", 0, 100, "metadata1")

	// Fetch offset
	offset, exists := group.FetchOffset("test-topic", 0)
	require.True(t, exists)
	assert.Equal(t, int64(100), offset.Offset)
	assert.Equal(t, "metadata1", offset.Metadata)
	assert.False(t, offset.CommitTime.IsZero())

	// Fetch non-existent offset
	_, exists = group.FetchOffset("test-topic", 1)
	assert.False(t, exists)

	_, exists = group.FetchOffset("other-topic", 0)
	assert.False(t, exists)
}

func TestGroupRebalance(t *testing.T) {
	group := NewGroup("test-group", "consumer")

	// Add initial member
	member1 := &Member{
		ID:           "member1",
		ProtocolType: "consumer",
	}
	group.Members["member1"] = member1
	group.State = GroupStateStable
	group.Generation = 5

	// Prepare rebalance
	group.PrepareRebalance()

	assert.Equal(t, GroupStatePreparingRebalance, group.State)
	assert.Equal(t, int32(6), group.Generation)
	assert.Equal(t, 0, len(group.Members))
	assert.Equal(t, 1, len(group.PendingMembers))
	assert.Equal(t, MemberStateJoining, member1.State)

	// Complete rebalance
	group.CompleteRebalance()

	assert.Equal(t, GroupStateStable, group.State)
	assert.Equal(t, 1, len(group.Members))
	assert.Equal(t, 0, len(group.PendingMembers))
	assert.Equal(t, MemberStateStable, member1.State)
}

func TestGroupNeedsRebalance(t *testing.T) {
	group := NewGroup("test-group", "consumer")

	// Empty group doesn't need rebalance
	assert.False(t, group.NeedsRebalance())

	// Group with pending members needs rebalance
	member := &Member{
		ID: "member1",
	}
	group.PendingMembers["member1"] = member
	assert.True(t, group.NeedsRebalance())

	// Group with expired member needs rebalance
	group.PendingMembers = make(map[string]*Member)
	member.SessionTimeout = 1000 // 1 second
	member.LastHeartbeat = time.Now().Add(-2 * time.Second)
	group.Members["member1"] = member
	assert.True(t, group.NeedsRebalance())
}

func TestCoordinator(t *testing.T) {
	coordinator := NewCoordinator()

	// Create group
	group := coordinator.GetOrCreateGroup("test-group", "consumer")
	assert.NotNil(t, group)
	assert.Equal(t, "test-group", group.ID)

	// Get existing group
	group2 := coordinator.GetOrCreateGroup("test-group", "consumer")
	assert.Equal(t, group, group2)

	// List groups
	groups := coordinator.ListGroups()
	assert.Equal(t, 1, len(groups))
	assert.Contains(t, groups, "test-group")
}

func TestCoordinatorJoinGroup(t *testing.T) {
	coordinator := NewCoordinator()

	protocols := []MemberProtocol{
		{Name: "range", Metadata: []byte("metadata")},
	}

	// First member joins
	group1, member1, rebalance1, err := coordinator.JoinGroup(
		"test-group",
		"member1",
		"client1",
		"localhost",
		"consumer",
		protocols,
		30000,
		60000,
	)

	require.NoError(t, err)
	assert.NotNil(t, group1)
	assert.NotNil(t, member1)
	assert.True(t, rebalance1)
	assert.Equal(t, "member1", group1.Leader)

	// Second member joins
	group2, member2, rebalance2, err := coordinator.JoinGroup(
		"test-group",
		"member2",
		"client2",
		"localhost",
		"consumer",
		protocols,
		30000,
		60000,
	)

	require.NoError(t, err)
	assert.Equal(t, group1, group2)
	assert.NotNil(t, member2)
	assert.True(t, rebalance2)
	assert.Equal(t, 2, group2.Size())
}

func TestCoordinatorSyncGroup(t *testing.T) {
	coordinator := NewCoordinator()

	protocols := []MemberProtocol{
		{Name: "range", Metadata: []byte("metadata")},
	}

	// Join members
	group, _, _, _ := coordinator.JoinGroup(
		"test-group", "member1", "client1", "localhost",
		"consumer", protocols, 30000, 60000,
	)

	coordinator.JoinGroup(
		"test-group", "member2", "client2", "localhost",
		"consumer", protocols, 30000, 60000,
	)

	// Leader syncs with assignments
	assignments := map[string][]byte{
		"member1": []byte("assignment1"),
		"member2": []byte("assignment2"),
	}

	assignment1, err := coordinator.SyncGroup("test-group", "member1", group.Generation, assignments)
	require.NoError(t, err)
	assert.Equal(t, []byte("assignment1"), assignment1)

	// Non-leader syncs
	assignment2, err := coordinator.SyncGroup("test-group", "member2", group.Generation, nil)
	require.NoError(t, err)
	assert.Equal(t, []byte("assignment2"), assignment2)
}

func TestCoordinatorHeartbeat(t *testing.T) {
	coordinator := NewCoordinator()

	protocols := []MemberProtocol{
		{Name: "range", Metadata: []byte("metadata")},
	}

	group, _, _, _ := coordinator.JoinGroup(
		"test-group", "member1", "client1", "localhost",
		"consumer", protocols, 30000, 60000,
	)

	// Complete rebalance to move to stable
	group.CompleteRebalance()

	// Send heartbeat
	err := coordinator.Heartbeat("test-group", "member1", group.Generation)
	assert.NoError(t, err)

	// Heartbeat with wrong generation
	err = coordinator.Heartbeat("test-group", "member1", 999)
	assert.Error(t, err)

	// Heartbeat for non-existent member
	err = coordinator.Heartbeat("test-group", "member999", group.Generation)
	assert.Error(t, err)
}

func TestCoordinatorLeaveGroup(t *testing.T) {
	coordinator := NewCoordinator()

	protocols := []MemberProtocol{
		{Name: "range", Metadata: []byte("metadata")},
	}

	group, _, _, _ := coordinator.JoinGroup(
		"test-group", "member1", "client1", "localhost",
		"consumer", protocols, 30000, 60000,
	)

	assert.Equal(t, 1, group.Size())

	// Member leaves
	err := coordinator.LeaveGroup("test-group", "member1")
	require.NoError(t, err)

	// Group should be deleted when empty
	_, exists := coordinator.GetGroup("test-group")
	assert.False(t, exists)
}

func TestCoordinatorOffsetCommitAndFetch(t *testing.T) {
	coordinator := NewCoordinator()

	// Commit offset
	err := coordinator.CommitOffset("test-group", "test-topic", 0, 100, "meta")
	require.NoError(t, err)

	// Fetch offset
	offset, exists := coordinator.FetchOffset("test-group", "test-topic", 0)
	require.True(t, exists)
	assert.Equal(t, int64(100), offset.Offset)
	assert.Equal(t, "meta", offset.Metadata)

	// Fetch non-existent offset
	_, exists = coordinator.FetchOffset("test-group", "test-topic", 1)
	assert.False(t, exists)

	_, exists = coordinator.FetchOffset("other-group", "test-topic", 0)
	assert.False(t, exists)
}
