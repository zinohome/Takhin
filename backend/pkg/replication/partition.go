// Copyright 2025 Takhin Data, Inc.

package replication

import (
	"fmt"
	"sync"
	"time"

	"github.com/takhin-data/takhin/pkg/storage/log"
)

// Partition represents a partition with replication support
type Partition struct {
	TopicName   string
	PartitionID int32
	Leader      int32   // Leader broker ID
	Replicas    []int32 // All replica broker IDs (including leader)
	ISR         []int32 // In-Sync Replicas
	Log         *log.Log

	// Replication state
	hwm int64 // High Water Mark
	leo int64 // Log End Offset (for leader)

	followerLEOs  map[int32]int64     // Follower broker ID -> LEO
	lastFetchTime map[int32]time.Time // Last fetch time from each follower

	// Configuration
	replicaLagTimeMaxMs int64 // Max lag time for ISR membership

	mu sync.RWMutex
}

// PartitionConfig defines configuration for a partition
type PartitionConfig struct {
	TopicName           string
	PartitionID         int32
	Leader              int32
	Replicas            []int32
	LogConfig           log.LogConfig
	ReplicaLagTimeMaxMs int64 // Default: 10000ms
}

// NewPartition creates a new partition with replication support
func NewPartition(config PartitionConfig) (*Partition, error) {
	// Default configuration
	if config.ReplicaLagTimeMaxMs <= 0 {
		config.ReplicaLagTimeMaxMs = 10000 // Default 10 seconds
	}

	// Create the log
	partitionLog, err := log.NewLog(config.LogConfig)
	if err != nil {
		return nil, fmt.Errorf("create log: %w", err)
	}

	p := &Partition{
		TopicName:           config.TopicName,
		PartitionID:         config.PartitionID,
		Leader:              config.Leader,
		Replicas:            config.Replicas,
		ISR:                 make([]int32, len(config.Replicas)), // Initially all replicas are in ISR
		Log:                 partitionLog,
		hwm:                 0,
		leo:                 partitionLog.HighWaterMark(),
		followerLEOs:        make(map[int32]int64),
		lastFetchTime:       make(map[int32]time.Time),
		replicaLagTimeMaxMs: config.ReplicaLagTimeMaxMs,
	}

	// Initialize ISR with all replicas
	copy(p.ISR, config.Replicas)

	// Initialize follower LEOs
	for _, replicaID := range config.Replicas {
		if replicaID != config.Leader {
			p.followerLEOs[replicaID] = 0
			p.lastFetchTime[replicaID] = time.Now()
		}
	}

	return p, nil
}

// IsLeader returns true if this broker is the leader for this partition
func (p *Partition) IsLeader(brokerID int32) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.Leader == brokerID
}

// IsFollower returns true if this broker is a follower for this partition
func (p *Partition) IsFollower(brokerID int32) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.Leader == brokerID {
		return false
	}

	for _, replicaID := range p.Replicas {
		if replicaID == brokerID {
			return true
		}
	}

	return false
}

// GetISR returns the current ISR (In-Sync Replicas)
func (p *Partition) GetISR() []int32 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	isr := make([]int32, len(p.ISR))
	copy(isr, p.ISR)
	return isr
}

// GetReplicas returns all replicas
func (p *Partition) GetReplicas() []int32 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	replicas := make([]int32, len(p.Replicas))
	copy(replicas, p.Replicas)
	return replicas
}

// GetLeader returns the leader broker ID
func (p *Partition) GetLeader() int32 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.Leader
}

// HighWaterMark returns the current high water mark
func (p *Partition) HighWaterMark() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.hwm
}

// LogEndOffset returns the current log end offset (LEO)
func (p *Partition) LogEndOffset() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.leo
}

// UpdateFollowerLEO updates the LEO for a follower replica
// This is called when the leader receives a fetch request from a follower
func (p *Partition) UpdateFollowerLEO(followerID int32, leo int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.followerLEOs[followerID] = leo
	p.lastFetchTime[followerID] = time.Now()

	// Update ISR based on follower LEO
	p.updateISR()

	// Recalculate HWM
	p.updateHWM()
}

// updateISR updates the ISR based on follower LEOs and lag time
// Must be called with write lock held
func (p *Partition) updateISR() {
	now := time.Now()
	newISR := []int32{p.Leader} // Leader is always in ISR

	for _, replicaID := range p.Replicas {
		if replicaID == p.Leader {
			continue
		}

		// Check if follower is in sync
		followerLEO := p.followerLEOs[replicaID]
		lastFetch := p.lastFetchTime[replicaID]

		// Follower is in sync if:
		// 1. LEO >= HWM (caught up with committed data)
		// 2. Last fetch was within replica.lag.time.max.ms
		lagTimeMs := now.Sub(lastFetch).Milliseconds()
		if followerLEO >= p.hwm && lagTimeMs < p.replicaLagTimeMaxMs {
			newISR = append(newISR, replicaID)
		}
	}

	p.ISR = newISR
}

// updateHWM recalculates the high water mark
// HWM is the minimum LEO among all ISR replicas
// Must be called with write lock held
func (p *Partition) updateHWM() {
	// Leader's LEO
	minLEO := p.leo

	// Check follower LEOs that are in ISR
	for _, replicaID := range p.ISR {
		if replicaID == p.Leader {
			continue
		}

		followerLEO := p.followerLEOs[replicaID]
		if followerLEO < minLEO {
			minLEO = followerLEO
		}
	}

	p.hwm = minLEO
}

// Append appends a record to the partition (leader only)
func (p *Partition) Append(key, value []byte) (int64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	offset, err := p.Log.Append(key, value)
	if err != nil {
		return 0, fmt.Errorf("append to log: %w", err)
	}

	// Update LEO
	p.leo = p.Log.HighWaterMark()

	// Recalculate HWM (might not change if followers haven't caught up)
	p.updateHWM()

	return offset, nil
}

// Read reads a record from the partition
func (p *Partition) Read(offset int64) (*log.Record, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.Log.Read(offset)
}

// Close closes the partition
func (p *Partition) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Log != nil {
		return p.Log.Close()
	}

	return nil
}
