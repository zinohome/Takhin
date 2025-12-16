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
