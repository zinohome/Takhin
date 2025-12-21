// Copyright 2025 Takhin Data, Inc.

package coordinator

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Coordinator manages all consumer groups
type Coordinator struct {
	groups map[string]*Group
	mu     sync.RWMutex
	logger *zap.Logger
}

// NewCoordinator creates a new coordinator
func NewCoordinator() *Coordinator {
	logger, _ := zap.NewProduction()
	return &Coordinator{
		groups: make(map[string]*Group),
		logger: logger,
	}
}

// GetOrCreateGroup retrieves or creates a group
func (c *Coordinator) GetOrCreateGroup(groupID string, protocolType string) *Group {
	c.mu.Lock()
	defer c.mu.Unlock()

	if group, exists := c.groups[groupID]; exists {
		return group
	}

	group := NewGroup(groupID, protocolType)
	c.groups[groupID] = group
	c.logger.Info("Created new group", zap.String("group", groupID))

	return group
}

// GetGroup retrieves an existing group
func (c *Coordinator) GetGroup(groupID string) (*Group, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	group, exists := c.groups[groupID]
	return group, exists
}

// DeleteGroup removes a group
func (c *Coordinator) DeleteGroup(groupID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.groups, groupID)
	c.logger.Info("Deleted group", zap.String("group", groupID))
}

// ListGroups returns all group IDs
func (c *Coordinator) ListGroups() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	groups := make([]string, 0, len(c.groups))
	for id := range c.groups {
		groups = append(groups, id)
	}

	return groups
}

// JoinGroup handles a member joining a group
func (c *Coordinator) JoinGroup(
	groupID string,
	memberID string,
	clientID string,
	clientHost string,
	protocolType string,
	protocols []MemberProtocol,
	sessionTimeout int32,
	rebalanceTimeout int32,
) (*Group, *Member, bool, error) {
	group := c.GetOrCreateGroup(groupID, protocolType)

	member := &Member{
		ID:               memberID,
		ClientID:         clientID,
		ClientHost:       clientHost,
		SessionTimeout:   sessionTimeout,
		RebalanceTimeout: rebalanceTimeout,
		ProtocolType:     protocolType,
		Protocols:        protocols,
		LastHeartbeat:    time.Now(),
	}

	if err := group.AddMember(member); err != nil {
		return nil, nil, false, err
	}

	// Set first member as leader
	if group.Leader == "" {
		group.Leader = memberID
	}

	// Check if rebalance needed
	needsRebalance := group.NeedsRebalance()

	c.logger.Info("Member joined group",
		zap.String("group", groupID),
		zap.String("member", memberID),
		zap.Bool("rebalance", needsRebalance))

	return group, member, needsRebalance, nil
}

// SyncGroup handles member synchronization after rebalance
func (c *Coordinator) SyncGroup(
	groupID string,
	memberID string,
	generation int32,
	assignments map[string][]byte,
) ([]byte, error) {
	group, exists := c.GetGroup(groupID)
	if !exists {
		return nil, fmt.Errorf("group not found: %s", groupID)
	}

	member, exists := group.GetMember(memberID)
	if !exists {
		return nil, fmt.Errorf("member not found: %s", memberID)
	}

	// Verify generation
	if generation != group.Generation {
		return nil, fmt.Errorf("illegal generation: expected %d, got %d", group.Generation, generation)
	}

	// Leader provides assignments
	if memberID == group.Leader && assignments != nil {
		for mid, assignment := range assignments {
			if m, ok := group.GetMember(mid); ok {
				m.Assignment = assignment
			}
		}

		// Complete rebalance
		group.CompleteRebalance()
	}

	return member.Assignment, nil
}

// Heartbeat handles member heartbeat
func (c *Coordinator) Heartbeat(groupID string, memberID string, generation int32) error {
	group, exists := c.GetGroup(groupID)
	if !exists {
		return fmt.Errorf("group not found: %s", groupID)
	}

	member, exists := group.GetMember(memberID)
	if !exists {
		return fmt.Errorf("member not found: %s", memberID)
	}

	// Verify generation
	if generation != group.Generation {
		return fmt.Errorf("illegal generation: expected %d, got %d", group.Generation, generation)
	}

	// Update heartbeat
	member.LastHeartbeat = time.Now()

	return nil
}

// LeaveGroup handles a member leaving a group
func (c *Coordinator) LeaveGroup(groupID string, memberID string) error {
	group, exists := c.GetGroup(groupID)
	if !exists {
		return fmt.Errorf("group not found: %s", groupID)
	}

	group.RemoveMember(memberID)

	c.logger.Info("Member left group",
		zap.String("group", groupID),
		zap.String("member", memberID))

	// Delete empty groups
	if group.IsEmpty() {
		c.DeleteGroup(groupID)
	}

	return nil
}

// CommitOffset commits an offset for a group
func (c *Coordinator) CommitOffset(groupID string, topic string, partition int32, offset int64, metadata string) error {
	group := c.GetOrCreateGroup(groupID, "consumer")
	group.CommitOffset(topic, partition, offset, metadata)

	return nil
}

// FetchOffset retrieves the committed offset for a group
func (c *Coordinator) FetchOffset(groupID string, topic string, partition int32) (*OffsetAndMetadata, bool) {
	group, exists := c.GetGroup(groupID)
	if !exists {
		return nil, false
	}

	return group.FetchOffset(topic, partition)
}

// CheckRebalances periodically checks for dead members and triggers rebalances
func (c *Coordinator) CheckRebalances() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for groupID, group := range c.groups {
		if group.NeedsRebalance() {
			c.logger.Info("Group needs rebalance", zap.String("group", groupID))
			group.PrepareRebalance()
		}
	}
}

// Start starts background tasks
func (c *Coordinator) Start() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			c.CheckRebalances()
		}
	}()
}

// GetGroupTopics returns all topics with committed offsets for a group
func (c *Coordinator) GetGroupTopics(groupID string) map[string][]int32 {
	group, exists := c.GetGroup(groupID)
	if !exists {
		return make(map[string][]int32)
	}

	group.mu.RLock()
	defer group.mu.RUnlock()

	result := make(map[string][]int32)
	for topic, partitions := range group.OffsetCommits {
		for partition := range partitions {
			result[topic] = append(result[topic], partition)
		}
	}

	return result
}

// GetTopicPartitions returns all partitions with committed offsets for a topic in a group
func (c *Coordinator) GetTopicPartitions(groupID string, topic string) []int32 {
	group, exists := c.GetGroup(groupID)
	if !exists {
		return []int32{}
	}

	group.mu.RLock()
	defer group.mu.RUnlock()

	partitions := make([]int32, 0)
	if topicOffsets, exists := group.OffsetCommits[topic]; exists {
		for partition := range topicOffsets {
			partitions = append(partitions, partition)
		}
	}

	return partitions
}

// GetAllGroups returns all groups
func (c *Coordinator) GetAllGroups() map[string]*GroupInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*GroupInfo, len(c.groups))
	for groupID, group := range c.groups {
		result[groupID] = &GroupInfo{
			GroupID:      groupID,
			ProtocolType: group.ProtocolType,
			State:        string(group.State),
		}
	}

	return result
}

// GroupInfo contains basic group information
type GroupInfo struct {
	GroupID      string
	ProtocolType string
	State        string
}

// ResetOffsets resets offsets for a group to specified values
// Returns error if group is not in Empty or Dead state
func (c *Coordinator) ResetOffsets(groupID string, offsets map[string]map[int32]int64) error {
	group, exists := c.GetGroup(groupID)
	if !exists {
		return fmt.Errorf("group not found: %s", groupID)
	}

	group.mu.Lock()
	defer group.mu.Unlock()

	// Can only reset offsets for Empty or Dead groups
	if group.State != GroupStateEmpty && group.State != GroupStateDead {
		return fmt.Errorf("cannot reset offsets for group in state: %s", group.State)
	}

	// Reset the offsets
	for topic, partitions := range offsets {
		if _, exists := group.OffsetCommits[topic]; !exists {
			group.OffsetCommits[topic] = make(map[int32]*OffsetAndMetadata)
		}

		for partition, offset := range partitions {
			group.OffsetCommits[topic][partition] = &OffsetAndMetadata{
				Offset:     offset,
				CommitTime: time.Now(),
			}
		}
	}

	c.logger.Info("Reset offsets for group",
		zap.String("group", groupID),
		zap.Int("topics", len(offsets)))

	return nil
}

// DeleteGroupOffsets deletes all offset commits for a group
// This is different from DeleteGroup which removes the entire group
func (c *Coordinator) DeleteGroupOffsets(groupID string) error {
	group, exists := c.GetGroup(groupID)
	if !exists {
		return fmt.Errorf("group not found: %s", groupID)
	}

	group.mu.Lock()
	defer group.mu.Unlock()

	// Can only delete offsets for Empty or Dead groups
	if group.State != GroupStateEmpty && group.State != GroupStateDead {
		return fmt.Errorf("cannot delete offsets for group in state: %s", group.State)
	}

	// Clear all offset commits
	group.OffsetCommits = make(map[string]map[int32]*OffsetAndMetadata)

	c.logger.Info("Deleted offsets for group", zap.String("group", groupID))

	return nil
}

// ForceDeleteGroup forcefully removes a group regardless of state
// WARNING: This should only be used by admin operations
func (c *Coordinator) ForceDeleteGroup(groupID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	group, exists := c.groups[groupID]
	if !exists {
		return fmt.Errorf("group not found: %s", groupID)
	}

	// Mark all members as dead
	group.mu.Lock()
	for _, member := range group.Members {
		member.State = MemberStateDead
	}
	for _, member := range group.PendingMembers {
		member.State = MemberStateDead
	}
	group.State = GroupStateDead
	group.mu.Unlock()

	// Remove the group
	delete(c.groups, groupID)

	c.logger.Info("Force deleted group", zap.String("group", groupID))

	return nil
}

// CanDeleteGroup checks if a group can be safely deleted
func (c *Coordinator) CanDeleteGroup(groupID string) (bool, string) {
	group, exists := c.GetGroup(groupID)
	if !exists {
		return false, "group not found"
	}

	group.mu.RLock()
	defer group.mu.RUnlock()

	// Can only delete Empty or Dead groups
	if group.State != GroupStateEmpty && group.State != GroupStateDead {
		return false, fmt.Sprintf("group is in %s state", group.State)
	}

	// Check if there are any active members
	if len(group.Members) > 0 || len(group.PendingMembers) > 0 {
		return false, "group has active members"
	}

	return true, ""
}
