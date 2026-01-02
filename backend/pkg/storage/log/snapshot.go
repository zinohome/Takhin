// Copyright 2025 Takhin Data, Inc.

package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Snapshot represents a point-in-time snapshot of a log
type Snapshot struct {
	ID             string    `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	BaseOffset     int64     `json:"base_offset"`
	HighWaterMark  int64     `json:"high_water_mark"`
	NumSegments    int       `json:"num_segments"`
	TotalSize      int64     `json:"total_size"`
	SegmentOffsets []int64   `json:"segment_offsets"`
}

// SnapshotMetadata contains metadata for all snapshots
type SnapshotMetadata struct {
	Snapshots []*Snapshot `json:"snapshots"`
	mu        sync.RWMutex
}

// SnapshotManager manages snapshots for a log
type SnapshotManager struct {
	logDir      string
	snapshotDir string
	metadata    *SnapshotMetadata
	mu          sync.RWMutex
}

// SnapshotConfig configures snapshot behavior
type SnapshotConfig struct {
	MaxSnapshots  int           // Maximum number of snapshots to keep
	RetentionTime time.Duration // How long to keep snapshots
	MinInterval   time.Duration // Minimum time between snapshots
}

// DefaultSnapshotConfig returns default snapshot configuration
func DefaultSnapshotConfig() SnapshotConfig {
	return SnapshotConfig{
		MaxSnapshots:  5,
		RetentionTime: 24 * time.Hour,
		MinInterval:   1 * time.Hour,
	}
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(logDir string) (*SnapshotManager, error) {
	snapshotDir := filepath.Join(logDir, ".snapshots")
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return nil, fmt.Errorf("create snapshot directory: %w", err)
	}

	sm := &SnapshotManager{
		logDir:      logDir,
		snapshotDir: snapshotDir,
		metadata: &SnapshotMetadata{
			Snapshots: make([]*Snapshot, 0),
		},
	}

	// Load existing snapshots metadata
	if err := sm.loadMetadata(); err != nil {
		// If metadata file doesn't exist, that's OK
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("load metadata: %w", err)
		}
	}

	return sm, nil
}

// CreateSnapshot creates a new snapshot of the log
func (sm *SnapshotManager) CreateSnapshot(log *Log) (*Snapshot, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	log.mu.RLock()
	defer log.mu.RUnlock()

	// Generate snapshot ID
	snapshotID := fmt.Sprintf("snapshot-%d", time.Now().UnixNano())
	snapshotPath := filepath.Join(sm.snapshotDir, snapshotID)

	// Create snapshot directory
	if err := os.MkdirAll(snapshotPath, 0755); err != nil {
		return nil, fmt.Errorf("create snapshot path: %w", err)
	}

	// Collect segment information
	segmentOffsets := make([]int64, len(log.segments))
	for i, seg := range log.segments {
		segmentOffsets[i] = seg.BaseOffset()
	}

	// Get log metrics
	totalSize, err := log.Size()
	if err != nil {
		return nil, fmt.Errorf("get log size: %w", err)
	}

	// Create snapshot metadata
	snapshot := &Snapshot{
		ID:             snapshotID,
		Timestamp:      time.Now(),
		BaseOffset:     log.segments[0].BaseOffset(),
		HighWaterMark:  log.HighWaterMark(),
		NumSegments:    len(log.segments),
		TotalSize:      totalSize,
		SegmentOffsets: segmentOffsets,
	}

	// Copy segment files
	for _, seg := range log.segments {
		if err := sm.copySegmentFiles(seg, snapshotPath); err != nil {
			// Clean up on failure
			os.RemoveAll(snapshotPath)
			return nil, fmt.Errorf("copy segment files: %w", err)
		}
	}

	// Add to metadata and save
	sm.metadata.mu.Lock()
	sm.metadata.Snapshots = append(sm.metadata.Snapshots, snapshot)
	sm.metadata.mu.Unlock()

	if err := sm.saveMetadata(); err != nil {
		return nil, fmt.Errorf("save metadata: %w", err)
	}

	return snapshot, nil
}

// RestoreSnapshot restores a log from a snapshot
func (sm *SnapshotManager) RestoreSnapshot(snapshotID string, targetLogDir string) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Find snapshot
	snapshot := sm.findSnapshot(snapshotID)
	if snapshot == nil {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	snapshotPath := filepath.Join(sm.snapshotDir, snapshotID)

	// Verify snapshot directory exists
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot directory not found: %s", snapshotPath)
	}

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetLogDir, 0755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	// List all files in snapshot
	entries, err := os.ReadDir(snapshotPath)
	if err != nil {
		return fmt.Errorf("read snapshot directory: %w", err)
	}

	// Copy all files from snapshot to target
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		srcPath := filepath.Join(snapshotPath, entry.Name())
		dstPath := filepath.Join(targetLogDir, entry.Name())

		if err := sm.copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("copy file %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// ListSnapshots returns all available snapshots
func (sm *SnapshotManager) ListSnapshots() []*Snapshot {
	sm.metadata.mu.RLock()
	defer sm.metadata.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	snapshots := make([]*Snapshot, len(sm.metadata.Snapshots))
	copy(snapshots, sm.metadata.Snapshots)

	// Sort by timestamp, newest first
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.After(snapshots[j].Timestamp)
	})

	return snapshots
}

// GetSnapshot returns a specific snapshot by ID
func (sm *SnapshotManager) GetSnapshot(snapshotID string) *Snapshot {
	sm.metadata.mu.RLock()
	defer sm.metadata.mu.RUnlock()
	return sm.findSnapshot(snapshotID)
}

// DeleteSnapshot deletes a specific snapshot
func (sm *SnapshotManager) DeleteSnapshot(snapshotID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Find snapshot index
	sm.metadata.mu.Lock()
	idx := -1
	for i, s := range sm.metadata.Snapshots {
		if s.ID == snapshotID {
			idx = i
			break
		}
	}

	if idx == -1 {
		sm.metadata.mu.Unlock()
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	// Remove from metadata
	sm.metadata.Snapshots = append(sm.metadata.Snapshots[:idx], sm.metadata.Snapshots[idx+1:]...)
	sm.metadata.mu.Unlock()

	// Delete snapshot directory
	snapshotPath := filepath.Join(sm.snapshotDir, snapshotID)
	if err := os.RemoveAll(snapshotPath); err != nil {
		return fmt.Errorf("remove snapshot directory: %w", err)
	}

	// Save updated metadata
	if err := sm.saveMetadata(); err != nil {
		return fmt.Errorf("save metadata: %w", err)
	}

	return nil
}

// CleanupSnapshots removes old snapshots based on the provided policy
func (sm *SnapshotManager) CleanupSnapshots(config SnapshotConfig) (int, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.metadata.mu.Lock()
	defer sm.metadata.mu.Unlock()

	if len(sm.metadata.Snapshots) == 0 {
		return 0, nil
	}

	// Sort snapshots by timestamp (newest first)
	sort.Slice(sm.metadata.Snapshots, func(i, j int) bool {
		return sm.metadata.Snapshots[i].Timestamp.After(sm.metadata.Snapshots[j].Timestamp)
	})

	toDelete := make([]string, 0)
	now := time.Now()

	// Apply retention policies
	for i, snapshot := range sm.metadata.Snapshots {
		shouldDelete := false

		// Policy 1: Keep only MaxSnapshots most recent
		if i >= config.MaxSnapshots {
			shouldDelete = true
		}

		// Policy 2: Delete snapshots older than RetentionTime
		if now.Sub(snapshot.Timestamp) > config.RetentionTime {
			shouldDelete = true
		}

		if shouldDelete {
			toDelete = append(toDelete, snapshot.ID)
		}
	}

	// Delete marked snapshots
	deleted := 0
	for _, snapshotID := range toDelete {
		// Remove from metadata
		for i, s := range sm.metadata.Snapshots {
			if s.ID == snapshotID {
				sm.metadata.Snapshots = append(sm.metadata.Snapshots[:i], sm.metadata.Snapshots[i+1:]...)
				break
			}
		}

		// Delete snapshot directory
		snapshotPath := filepath.Join(sm.snapshotDir, snapshotID)
		if err := os.RemoveAll(snapshotPath); err != nil {
			// Log error but continue with other snapshots
			continue
		}
		deleted++
	}

	// Save updated metadata
	if err := sm.saveMetadata(); err != nil {
		return deleted, fmt.Errorf("save metadata: %w", err)
	}

	return deleted, nil
}

// copySegmentFiles copies all files for a segment to the snapshot directory
func (sm *SnapshotManager) copySegmentFiles(seg *Segment, snapshotPath string) error {
	baseOffset := seg.BaseOffset()

	// Flush segment to ensure all data is written
	if err := seg.Flush(); err != nil {
		return fmt.Errorf("flush segment: %w", err)
	}

	// Copy data file
	dataFile := fmt.Sprintf("%020d.log", baseOffset)
	srcData := filepath.Join(sm.logDir, dataFile)
	dstData := filepath.Join(snapshotPath, dataFile)
	if err := sm.copyFile(srcData, dstData); err != nil {
		return fmt.Errorf("copy data file: %w", err)
	}

	// Copy index file
	indexFile := fmt.Sprintf("%020d.index", baseOffset)
	srcIndex := filepath.Join(sm.logDir, indexFile)
	dstIndex := filepath.Join(snapshotPath, indexFile)
	if err := sm.copyFile(srcIndex, dstIndex); err != nil {
		return fmt.Errorf("copy index file: %w", err)
	}

	// Copy time index file
	timeIndexFile := fmt.Sprintf("%020d.timeindex", baseOffset)
	srcTimeIndex := filepath.Join(sm.logDir, timeIndexFile)
	dstTimeIndex := filepath.Join(snapshotPath, timeIndexFile)
	if err := sm.copyFile(srcTimeIndex, dstTimeIndex); err != nil {
		return fmt.Errorf("copy time index file: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst
func (sm *SnapshotManager) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("sync destination: %w", err)
	}

	return nil
}

// findSnapshot finds a snapshot by ID (caller must hold read lock on metadata)
func (sm *SnapshotManager) findSnapshot(snapshotID string) *Snapshot {
	for _, s := range sm.metadata.Snapshots {
		if s.ID == snapshotID {
			return s
		}
	}
	return nil
}

// loadMetadata loads snapshot metadata from disk
func (sm *SnapshotManager) loadMetadata() error {
	metadataPath := filepath.Join(sm.snapshotDir, "snapshots.json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return err
	}

	sm.metadata.mu.Lock()
	defer sm.metadata.mu.Unlock()

	if err := json.Unmarshal(data, sm.metadata); err != nil {
		return fmt.Errorf("unmarshal metadata: %w", err)
	}

	return nil
}

// saveMetadata saves snapshot metadata to disk
func (sm *SnapshotManager) saveMetadata() error {
	metadataPath := filepath.Join(sm.snapshotDir, "snapshots.json")

	sm.metadata.mu.RLock()
	data, err := json.MarshalIndent(sm.metadata, "", "  ")
	sm.metadata.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	return nil
}

// Size returns the total size of all snapshots
func (sm *SnapshotManager) Size() (int64, error) {
	sm.metadata.mu.RLock()
	defer sm.metadata.mu.RUnlock()

	totalSize := int64(0)
	for _, snapshot := range sm.metadata.Snapshots {
		snapshotPath := filepath.Join(sm.snapshotDir, snapshot.ID)

		entries, err := os.ReadDir(snapshotPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			totalSize += info.Size()
		}
	}

	return totalSize, nil
}
