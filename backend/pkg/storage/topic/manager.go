package topic

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/takhin-data/takhin/pkg/storage/log"
)

// Manager manages topics and their partitions
type Manager struct {
	dataDir   string
	topics    map[string]*Topic
	mu        sync.RWMutex
	logConfig log.LogConfig
	cleaner   *log.Cleaner
}

// Topic represents a topic with its partitions
type Topic struct {
	Name              string
	Partitions        map[int32]*log.Log
	ReplicationFactor int16
	// Replicas maps partitionID -> list of replica broker IDs
	Replicas map[int32][]int32
	// ISR maps partitionID -> list of in-sync replica broker IDs
	ISR map[int32][]int32
	// FollowerLEO tracks Log End Offset for each follower: partitionID -> brokerID -> LEO
	FollowerLEO map[int32]map[int32]int64
	// LastFetchTime tracks last fetch time for each follower: partitionID -> brokerID -> time
	LastFetchTime map[int32]map[int32]time.Time
	// ReplicaLagMaxMs is the max lag time before removing from ISR (default 10000ms)
	ReplicaLagMaxMs int64
	mu              sync.RWMutex
}

// SetReplicationFactor updates the metadata replication factor
func (t *Topic) SetReplicationFactor(rf int16) {
	if rf <= 0 {
		rf = 1
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ReplicationFactor = rf
}

// SetReplicas updates replica assignments for a partition
func (t *Topic) SetReplicas(partitionID int32, replicas []int32) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Replicas == nil {
		t.Replicas = make(map[int32][]int32)
	}
	t.Replicas[partitionID] = replicas
	// Initialize ISR with all replicas (assume all in-sync initially)
	if t.ISR == nil {
		t.ISR = make(map[int32][]int32)
	}
	t.ISR[partitionID] = replicas
}

// GetReplicas returns replica assignment for a partition
func (t *Topic) GetReplicas(partitionID int32) []int32 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.Replicas == nil {
		return nil
	}
	return t.Replicas[partitionID]
}

// GetISR returns in-sync replicas for a partition
func (t *Topic) GetISR(partitionID int32) []int32 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.ISR == nil {
		return nil
	}
	return t.ISR[partitionID]
}

// SetISR sets the in-sync replica set for a partition (for testing)
func (t *Topic) SetISR(partitionID int32, isr []int32) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ISR == nil {
		t.ISR = make(map[int32][]int32)
	}
	t.ISR[partitionID] = isr
}

// UpdateFollowerLEO updates the Log End Offset for a follower replica
func (t *Topic) UpdateFollowerLEO(partitionID int32, followerID int32, leo int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.FollowerLEO == nil {
		t.FollowerLEO = make(map[int32]map[int32]int64)
	}
	if t.FollowerLEO[partitionID] == nil {
		t.FollowerLEO[partitionID] = make(map[int32]int64)
	}
	t.FollowerLEO[partitionID][followerID] = leo

	if t.LastFetchTime == nil {
		t.LastFetchTime = make(map[int32]map[int32]time.Time)
	}
	if t.LastFetchTime[partitionID] == nil {
		t.LastFetchTime[partitionID] = make(map[int32]time.Time)
	}
	t.LastFetchTime[partitionID][followerID] = time.Now()
}

// GetFollowerLEO returns the Log End Offset for a follower replica
func (t *Topic) GetFollowerLEO(partitionID int32, followerID int32) (int64, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.FollowerLEO == nil || t.FollowerLEO[partitionID] == nil {
		return 0, false
	}
	leo, exists := t.FollowerLEO[partitionID][followerID]
	return leo, exists
}

// UpdateISR updates the in-sync replica set for a partition based on lag
func (t *Topic) UpdateISR(partitionID int32, leaderLEO int64) []int32 {
	t.mu.Lock()
	defer t.mu.Unlock()

	replicas := t.Replicas[partitionID]
	if replicas == nil || len(replicas) == 0 {
		return nil
	}

	// Default lag max time: 10 seconds
	lagMaxMs := t.ReplicaLagMaxMs
	if lagMaxMs <= 0 {
		lagMaxMs = 10000
	}

	newISR := make([]int32, 0, len(replicas))
	now := time.Now()

	// Leader is always in ISR
	leader := replicas[0]
	newISR = append(newISR, leader)

	// Check each follower
	for _, replicaID := range replicas[1:] {
		inSync := false

		// Follower must have both: caught up LEO AND recent fetch
		hasLEO := false
		hasFetch := false

		// Check if follower LEO is caught up (within 1 offset)
		if t.FollowerLEO != nil && t.FollowerLEO[partitionID] != nil {
			followerLEO, exists := t.FollowerLEO[partitionID][replicaID]
			if exists && leaderLEO-followerLEO <= 1 {
				hasLEO = true
			}
		}

		// Check if follower fetched recently
		if t.LastFetchTime != nil && t.LastFetchTime[partitionID] != nil {
			lastFetch, exists := t.LastFetchTime[partitionID][replicaID]
			if exists && now.Sub(lastFetch).Milliseconds() <= lagMaxMs {
				hasFetch = true
			}
		}

		// Follower is in-sync if both LEO is caught up AND fetch is recent
		if hasLEO && hasFetch {
			inSync = true
		}

		if inSync {
			newISR = append(newISR, replicaID)
		}
	}

	// Update ISR if changed
	if t.ISR == nil {
		t.ISR = make(map[int32][]int32)
	}
	t.ISR[partitionID] = newISR

	return newISR
}

// GetLeaderForPartition returns the leader broker ID for a partition
func (t *Topic) GetLeaderForPartition(partitionID int32) (int32, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	replicas := t.Replicas[partitionID]
	if replicas == nil || len(replicas) == 0 {
		return -1, false
	}
	return replicas[0], true
}

// NewManager creates a new topic manager
func NewManager(dataDir string, maxSegmentSize int64) *Manager {
	return &Manager{
		dataDir: dataDir,
		topics:  make(map[string]*Topic),
		logConfig: log.LogConfig{
			MaxSegmentSize: maxSegmentSize,
		},
		cleaner: nil, // Will be initialized by SetCleaner if needed
	}
}

// SetCleaner sets the background cleaner for this manager
func (m *Manager) SetCleaner(cleaner *log.Cleaner) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleaner = cleaner
}

// CreateTopic creates a new topic with the specified number of partitions
func (m *Manager) CreateTopic(name string, numPartitions int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.topics[name]; exists {
		return fmt.Errorf("topic already exists: %s", name)
	}

	topic := &Topic{
		Name:              name,
		Partitions:        make(map[int32]*log.Log),
		Replicas:          make(map[int32][]int32),
		ISR:               make(map[int32][]int32),
		FollowerLEO:       make(map[int32]map[int32]int64),
		LastFetchTime:     make(map[int32]map[int32]time.Time),
		ReplicationFactor: 1,
		ReplicaLagMaxMs:   10000, // Default 10 seconds
	}

	// Create partitions
	for i := int32(0); i < numPartitions; i++ {
		partitionDir := filepath.Join(m.dataDir, name, fmt.Sprintf("partition-%d", i))

		logConfig := m.logConfig
		logConfig.Dir = partitionDir

		partition, err := log.NewLog(logConfig)
		if err != nil {
			return fmt.Errorf("create partition %d: %w", i, err)
		}
		topic.Partitions[i] = partition

		// Register with cleaner if available
		if m.cleaner != nil {
			logName := fmt.Sprintf("%s-%d", name, i)
			m.cleaner.RegisterLog(logName, partition)
		}
	}

	m.topics[name] = topic
	return nil
}

// DeleteTopic deletes a topic and all its partitions
func (m *Manager) DeleteTopic(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	topic, exists := m.topics[name]
	if !exists {
		return fmt.Errorf("topic not found: %s", name)
	}

	// Unregister from cleaner and close all partition logs
	for partitionID, partition := range topic.Partitions {
		if m.cleaner != nil {
			logName := fmt.Sprintf("%s-%d", name, partitionID)
			m.cleaner.UnregisterLog(logName)
		}
		if err := partition.Close(); err != nil {
			return fmt.Errorf("close partition: %w", err)
		}
	}

	// Remove topic data directory
	topicDir := filepath.Join(m.dataDir, name)
	if err := os.RemoveAll(topicDir); err != nil {
		return fmt.Errorf("remove topic directory: %w", err)
	}

	delete(m.topics, name)
	return nil
}

// GetTopic returns a topic by name
func (m *Manager) GetTopic(name string) (*Topic, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	topic, exists := m.topics[name]
	return topic, exists
}

// ListTopics returns all topic names
func (m *Manager) ListTopics() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.topics))
	for name := range m.topics {
		names = append(names, name)
	}
	return names
}

// Append appends a message to a topic partition
func (t *Topic) Append(partition int32, key, value []byte) (int64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	log, exists := t.Partitions[partition]
	if !exists {
		return 0, fmt.Errorf("partition not found: %d", partition)
	}
	return log.Append(key, value)
}

// Read reads a message from a topic partition
func (t *Topic) Read(partition int32, offset int64) (*log.Record, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	log, exists := t.Partitions[partition]
	if !exists {
		return nil, fmt.Errorf("partition not found: %d", partition)
	}
	return log.Read(offset)
}

// HighWaterMark returns the high water mark for a partition
func (t *Topic) HighWaterMark(partition int32) (int64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	log, exists := t.Partitions[partition]
	if !exists {
		return 0, fmt.Errorf("partition not found: %d", partition)
	}
	return log.HighWaterMark(), nil
}

// Size returns the total size in bytes of all partitions in this topic
func (t *Topic) Size() (int64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	totalSize := int64(0)
	for _, logInstance := range t.Partitions {
		size, err := logInstance.Size()
		if err != nil {
			return 0, fmt.Errorf("get partition size: %w", err)
		}
		totalSize += size
	}

	return totalSize, nil
}

// PartitionSize returns the size in bytes of a specific partition
func (t *Topic) PartitionSize(partition int32) (int64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	logInstance, exists := t.Partitions[partition]
	if !exists {
		return 0, fmt.Errorf("partition not found: %d", partition)
	}

	return logInstance.Size()
}

// NumPartitions returns the number of partitions in this topic
func (t *Topic) NumPartitions() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.Partitions)
}

// GetEarliestOffset returns the earliest (oldest) available offset for a partition
func (t *Topic) GetEarliestOffset(partition int32) (int64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, exists := t.Partitions[partition]
	if !exists {
		return 0, fmt.Errorf("partition not found: %d", partition)
	}
	// 最早的 offset 通常是 0，除非有 log compaction
	return 0, nil
}

// GetLatestOffset returns the latest (newest) available offset for a partition
// This is the same as HighWaterMark
func (t *Topic) GetLatestOffset(partition int32) (int64, error) {
	return t.HighWaterMark(partition)
}

// GetOffsetByTimestamp returns the offset for a given timestamp
// Returns the earliest offset whose timestamp >= the given timestamp
func (t *Topic) GetOffsetByTimestamp(partition int32, timestamp int64) (int64, int64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	partitionLog, exists := t.Partitions[partition]
	if !exists {
		return 0, 0, fmt.Errorf("partition not found: %d", partition)
	}

	// 使用 TimeIndex 查找
	offset, actualTimestamp, err := partitionLog.SearchByTimestamp(timestamp)
	if err != nil {
		// 如果找不到，返回 HWM
		hwm := partitionLog.HighWaterMark()
		return hwm, timestamp, nil
	}

	return offset, actualTimestamp, nil
}

// DeleteRecordsBeforeOffset deletes records before the specified offset
// Returns the new low watermark after deletion
func (t *Topic) DeleteRecordsBeforeOffset(partition int32, offset int64) (int64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	partitionLog, exists := t.Partitions[partition]
	if !exists {
		return 0, fmt.Errorf("partition not found: %d", partition)
	}

	// 在实际的实现中，这里应该真正删除segment文件
	// 目前简化实现：只返回请求的offset作为新的低水位
	// 真正的实现需要：
	// 1. 找到包含该offset的segment
	// 2. 删除该offset之前的所有segments
	// 3. 如果offset在segment中间，需要截断该segment

	hwm := partitionLog.HighWaterMark()
	if offset > hwm {
		return hwm, fmt.Errorf("offset %d is beyond high watermark %d", offset, hwm)
	}

	// 简化实现：返回请求的offset作为新的低水位
	return offset, nil
}

// Close closes all partitions
func (t *Topic) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	var errs []error
	for _, partition := range t.Partitions {
		if err := partition.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close partitions: %v", errs)
	}
	return nil
}

// Close closes all topics
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var errs []error
	for _, topic := range m.topics {
		if err := topic.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close topics: %v", errs)
	}
	return nil
}
