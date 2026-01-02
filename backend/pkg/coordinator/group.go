// Copyright 2025 Takhin Data, Inc.

package coordinator

import (
	"fmt"
	"sync"
	"time"
)

// MemberState represents the state of a consumer group member
type MemberState string

const (
	MemberStateNew     MemberState = "new"
	MemberStateJoining MemberState = "joining"
	MemberStateSync    MemberState = "sync"
	MemberStateStable  MemberState = "stable"
	MemberStateLeaving MemberState = "leaving"
	MemberStateDead    MemberState = "dead"
)

// GroupState represents the state of a consumer group
type GroupState string

const (
	GroupStateEmpty               GroupState = "Empty"
	GroupStatePreparingRebalance  GroupState = "PreparingRebalance"
	GroupStateCompletingRebalance GroupState = "CompletingRebalance"
	GroupStateStable              GroupState = "Stable"
	GroupStateDead                GroupState = "Dead"
)

// Member represents a consumer group member
type Member struct {
	ID               string           // Member ID (client-generated or assigned)
	ClientID         string           // Client ID
	ClientHost       string           // Client host
	SessionTimeout   int32            // Session timeout in ms
	RebalanceTimeout int32            // Rebalance timeout in ms
	ProtocolType     string           // Protocol type (consumer)
	Protocols        []MemberProtocol // Supported protocols
	Assignment       []byte           // Current assignment
	Metadata         []byte           // Member metadata
	State            MemberState      // Current state
	LastHeartbeat    time.Time        // Last heartbeat time
}

// MemberProtocol represents a protocol supported by a member
type MemberProtocol struct {
	Name     string // Protocol name
	Metadata []byte // Protocol metadata
}

// Group represents a consumer group
type Group struct {
	ID             string                                  // Group ID
	State          GroupState                              // Current state
	ProtocolType   string                                  // Protocol type
	ProtocolName   string                                  // Selected protocol
	Generation     int32                                   // Current generation
	Leader         string                                  // Leader member ID
	Members        map[string]*Member                      // Group members
	PendingMembers map[string]*Member                      // Members joining
	OffsetCommits  map[string]map[int32]*OffsetAndMetadata // topic -> partition -> offset
	CreatedAt      time.Time                               // Creation time
	LastRebalance  time.Time                               // Last rebalance time
	mu             sync.RWMutex                            // Protects group state
}

// OffsetAndMetadata represents committed offset with metadata
type OffsetAndMetadata struct {
	Offset      int64     // Committed offset
	LeaderEpoch int32     // Leader epoch
	Metadata    string    // Custom metadata
	CommitTime  time.Time // Commit timestamp
	ExpireTime  time.Time // Expiration time
}

// NewGroup creates a new consumer group
func NewGroup(groupID string, protocolType string) *Group {
	return &Group{
		ID:             groupID,
		State:          GroupStateEmpty,
		ProtocolType:   protocolType,
		Generation:     0,
		Members:        make(map[string]*Member),
		PendingMembers: make(map[string]*Member),
		OffsetCommits:  make(map[string]map[int32]*OffsetAndMetadata),
		CreatedAt:      time.Now(),
	}
}

// AddMember adds a member to the group
func (g *Group) AddMember(member *Member) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Validate protocol type
	if member.ProtocolType != g.ProtocolType {
		return fmt.Errorf("protocol type mismatch: expected %s, got %s", g.ProtocolType, member.ProtocolType)
	}

	// Add to pending members - triggers rebalance
	member.State = MemberStateJoining
	g.PendingMembers[member.ID] = member

	return nil
}

// RemoveMember removes a member from the group
func (g *Group) RemoveMember(memberID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.Members, memberID)
	delete(g.PendingMembers, memberID)

	// Transition to empty if no members left
	if len(g.Members) == 0 && len(g.PendingMembers) == 0 {
		g.State = GroupStateEmpty
		g.Generation = 0
		g.Leader = ""
		g.ProtocolName = ""
	}
}

// GetMember retrieves a member by ID
func (g *Group) GetMember(memberID string) (*Member, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	member, ok := g.Members[memberID]
	if !ok {
		member, ok = g.PendingMembers[memberID]
	}

	return member, ok
}

// AllMembers returns all members (stable + pending)
func (g *Group) AllMembers() []*Member {
	g.mu.RLock()
	defer g.mu.RUnlock()

	members := make([]*Member, 0, len(g.Members)+len(g.PendingMembers))
	for _, m := range g.Members {
		members = append(members, m)
	}
	for _, m := range g.PendingMembers {
		members = append(members, m)
	}

	return members
}

// HasMember checks if a member exists
func (g *Group) HasMember(memberID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	_, ok := g.Members[memberID]
	if !ok {
		_, ok = g.PendingMembers[memberID]
	}

	return ok
}

// Size returns the number of members
func (g *Group) Size() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return len(g.Members) + len(g.PendingMembers)
}

// IsEmpty returns true if group has no members
func (g *Group) IsEmpty() bool {
	return g.Size() == 0
}

// SelectProtocol selects the protocol to use for the group
func (g *Group) SelectProtocol() (string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.Members) == 0 {
		return "", fmt.Errorf("no members in group")
	}

	// Count protocol votes
	votes := make(map[string]int)
	for _, member := range g.Members {
		for _, protocol := range member.Protocols {
			votes[protocol.Name]++
		}
	}

	// Find protocol supported by all members
	memberCount := len(g.Members)
	for protocol, count := range votes {
		if count == memberCount {
			return protocol, nil
		}
	}

	return "", fmt.Errorf("no protocol supported by all members")
}

// CommitOffset commits an offset for a topic partition
func (g *Group) CommitOffset(topic string, partition int32, offset int64, metadata string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.OffsetCommits[topic] == nil {
		g.OffsetCommits[topic] = make(map[int32]*OffsetAndMetadata)
	}

	g.OffsetCommits[topic][partition] = &OffsetAndMetadata{
		Offset:     offset,
		Metadata:   metadata,
		CommitTime: time.Now(),
		ExpireTime: time.Now().Add(7 * 24 * time.Hour), // 7 days retention
	}
}

// FetchOffset retrieves the committed offset for a topic partition
func (g *Group) FetchOffset(topic string, partition int32) (*OffsetAndMetadata, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.OffsetCommits[topic] == nil {
		return nil, false
	}

	offset, ok := g.OffsetCommits[topic][partition]
	return offset, ok
}

// PrepareRebalance transitions the group to rebalancing state
func (g *Group) PrepareRebalance() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.State = GroupStatePreparingRebalance
	g.Generation++
	g.LastRebalance = time.Now()

	// Move all members to pending
	for id, member := range g.Members {
		member.State = MemberStateJoining
		g.PendingMembers[id] = member
	}
	g.Members = make(map[string]*Member)
}

// CompleteRebalance completes the rebalance
func (g *Group) CompleteRebalance() {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Move pending members to stable
	for id, member := range g.PendingMembers {
		member.State = MemberStateStable
		g.Members[id] = member
	}
	g.PendingMembers = make(map[string]*Member)

	g.State = GroupStateStable
}

// NeedsRebalance checks if the group needs rebalancing
func (g *Group) NeedsRebalance() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Check for dead members
	now := time.Now()
	for _, member := range g.Members {
		timeout := time.Duration(member.SessionTimeout) * time.Millisecond
		if now.Sub(member.LastHeartbeat) > timeout {
			return true
		}
	}

	// Check if we have pending members
	return len(g.PendingMembers) > 0
}

// GetState returns the current group state (thread-safe)
func (g *Group) GetState() GroupState {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.State
}

// GetMemberCount returns the number of members in the group (thread-safe)
func (g *Group) GetMemberCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.Members)
}
