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
