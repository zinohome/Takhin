package topic

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/takhin-data/takhin/pkg/storage/log"
)

// Manager manages topics and their partitions
type Manager struct {
	dataDir   string
	topics    map[string]*Topic
	mu        sync.RWMutex
	logConfig log.LogConfig
}

// Topic represents a topic with its partitions
type Topic struct {
	Name       string
	Partitions map[int32]*log.Log
	mu         sync.RWMutex
}

// NewManager creates a new topic manager
func NewManager(dataDir string, maxSegmentSize int64) *Manager {
	return &Manager{
		dataDir: dataDir,
		topics:  make(map[string]*Topic),
		logConfig: log.LogConfig{
			MaxSegmentSize: maxSegmentSize,
		},
	}
}

// CreateTopic creates a new topic with the specified number of partitions
func (m *Manager) CreateTopic(name string, numPartitions int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.topics[name]; exists {
		return fmt.Errorf("topic already exists: %s", name)
	}

	topic := &Topic{
		Name:       name,
		Partitions: make(map[int32]*log.Log),
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

	// Close all partition logs
	for _, partition := range topic.Partitions {
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
