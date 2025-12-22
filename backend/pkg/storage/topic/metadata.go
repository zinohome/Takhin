package topic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PartitionMetadata stores replication metadata for a single partition
type PartitionMetadata struct {
	PartitionID int32   `json:"partition_id"`
	Replicas    []int32 `json:"replicas"`
	ISR         []int32 `json:"isr"`
	Leader      int32   `json:"leader"`
	LeaderEpoch int32   `json:"leader_epoch"`
}

// TopicMetadata stores all metadata for a topic
type TopicMetadata struct {
	Name              string              `json:"name"`
	ReplicationFactor int16               `json:"replication_factor"`
	Partitions        []PartitionMetadata `json:"partitions"`
	Version           int32               `json:"version"` // Metadata version for compatibility
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

const (
	metadataFileName   = "topic-metadata.json"
	metadataVersion    = 1
	metadataTempSuffix = ".tmp"
)

// SaveMetadata persists topic metadata to disk
func (t *Topic) SaveMetadata(baseDir string) error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.saveMetadataLocked(baseDir)
}

// saveMetadataLocked saves metadata without locking (caller must hold lock)
func (t *Topic) saveMetadataLocked(baseDir string) error {
	// Build metadata structure
	metadata := TopicMetadata{
		Name:              t.Name,
		ReplicationFactor: t.ReplicationFactor,
		Version:           metadataVersion,
		UpdatedAt:         time.Now(),
	}

	// If CreatedAt is not set, set it now
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = time.Now()
	}

	// Collect partition metadata
	partitions := make([]PartitionMetadata, 0, len(t.Partitions))
	for partID := range t.Partitions {
		pm := PartitionMetadata{
			PartitionID: partID,
			Replicas:    t.Replicas[partID],
			ISR:         t.ISR[partID],
		}

		// Determine leader (first replica by default)
		if len(pm.Replicas) > 0 {
			pm.Leader = pm.Replicas[0]
		}
		pm.LeaderEpoch = 0 // TODO: Implement leader epoch tracking

		partitions = append(partitions, pm)
	}
	metadata.Partitions = partitions

	// Ensure topic directory exists
	topicDir := filepath.Join(baseDir, t.Name)
	if err := os.MkdirAll(topicDir, 0755); err != nil {
		return fmt.Errorf("create topic directory: %w", err)
	}

	// Write to temporary file first (atomic write)
	metadataPath := filepath.Join(topicDir, metadataFileName)
	tempPath := metadataPath + metadataTempSuffix

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("write metadata file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, metadataPath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("rename metadata file: %w", err)
	}

	return nil
}

// LoadMetadata loads topic metadata from disk
func LoadMetadata(baseDir, topicName string) (*TopicMetadata, error) {
	metadataPath := filepath.Join(baseDir, topicName, metadataFileName)

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Metadata file doesn't exist (not an error for new topics)
		}
		return nil, fmt.Errorf("read metadata file: %w", err)
	}

	var metadata TopicMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	// Validate version
	if metadata.Version > metadataVersion {
		return nil, fmt.Errorf("unsupported metadata version: %d (current: %d)",
			metadata.Version, metadataVersion)
	}

	return &metadata, nil
}

// ApplyMetadata applies loaded metadata to the topic
func (t *Topic) ApplyMetadata(metadata *TopicMetadata) error {
	if metadata == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Apply replication factor
	t.ReplicationFactor = metadata.ReplicationFactor

	// Apply partition metadata
	for _, pm := range metadata.Partitions {
		// Set replicas
		if t.Replicas == nil {
			t.Replicas = make(map[int32][]int32)
		}
		t.Replicas[pm.PartitionID] = pm.Replicas

		// Set ISR
		if t.ISR == nil {
			t.ISR = make(map[int32][]int32)
		}
		t.ISR[pm.PartitionID] = pm.ISR

		// Initialize follower tracking maps if needed
		if t.FollowerLEO == nil {
			t.FollowerLEO = make(map[int32]map[int32]int64)
		}
		if t.LastFetchTime == nil {
			t.LastFetchTime = make(map[int32]map[int32]time.Time)
		}
	}

	return nil
}

// DeleteMetadata removes metadata file for a topic
func DeleteMetadata(baseDir, topicName string) error {
	metadataPath := filepath.Join(baseDir, topicName, metadataFileName)

	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete metadata file: %w", err)
	}

	return nil
}
