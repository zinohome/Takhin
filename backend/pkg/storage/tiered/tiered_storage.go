// Copyright 2025 Takhin Data, Inc.

package tiered

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/takhin-data/takhin/pkg/logger"
)

type StoragePolicy string

const (
	PolicyHot  StoragePolicy = "hot"
	PolicyWarm StoragePolicy = "warm"
	PolicyCold StoragePolicy = "cold"
)

type SegmentMetadata struct {
	Path          string
	BaseOffset    int64
	Size          int64
	LastAccessAt  time.Time
	LastModified  time.Time
	Policy        StoragePolicy
	IsArchived    bool
	S3Key         string
}

type TieredStorageConfig struct {
	DataDir            string
	S3Config           S3Config
	ColdAgeThreshold   time.Duration
	WarmAgeThreshold   time.Duration
	ArchiveInterval    time.Duration
	LocalCacheSize     int64
	AutoArchiveEnabled bool
}

type TieredStorage struct {
	config    TieredStorageConfig
	s3Client  *S3Client
	metadata  map[string]*SegmentMetadata
	mu        sync.RWMutex
	stopCh    chan struct{}
	wg        sync.WaitGroup
	logger    *logger.Logger
}

func NewTieredStorage(ctx context.Context, config TieredStorageConfig) (*TieredStorage, error) {
	s3Client, err := NewS3Client(ctx, config.S3Config)
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}

	ts := &TieredStorage{
		config:   config,
		s3Client: s3Client,
		metadata: make(map[string]*SegmentMetadata),
		stopCh:   make(chan struct{}),
		logger:   logger.Default().WithComponent("tiered-storage"),
	}

	if err := ts.loadMetadata(); err != nil {
		return nil, fmt.Errorf("load metadata: %w", err)
	}

	if config.AutoArchiveEnabled {
		ts.startArchiver()
	}

	return ts, nil
}

func (ts *TieredStorage) loadMetadata() error {
	return filepath.Walk(ts.config.DataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".log") {
			return nil
		}

		relPath, err := filepath.Rel(ts.config.DataDir, path)
		if err != nil {
			return err
		}

		meta := &SegmentMetadata{
			Path:         path,
			Size:         info.Size(),
			LastModified: info.ModTime(),
			LastAccessAt: time.Now(),
			Policy:       PolicyHot,
			IsArchived:   false,
		}

		ts.mu.Lock()
		ts.metadata[relPath] = meta
		ts.mu.Unlock()

		return nil
	})
}

func (ts *TieredStorage) ArchiveSegment(ctx context.Context, segmentPath string) error {
	ts.mu.Lock()
	meta, exists := ts.metadata[segmentPath]
	if !exists {
		ts.mu.Unlock()
		return fmt.Errorf("segment not found: %s", segmentPath)
	}

	if meta.IsArchived {
		ts.mu.Unlock()
		return nil
	}
	ts.mu.Unlock()

	absPath := filepath.Join(ts.config.DataDir, segmentPath)
	s3Key := segmentPath

	ts.logger.Info("archiving segment to s3",
		"segment", segmentPath,
		"s3_key", s3Key)

	if err := ts.s3Client.UploadFile(ctx, absPath, s3Key); err != nil {
		return fmt.Errorf("upload segment: %w", err)
	}

	indexPath := strings.TrimSuffix(absPath, ".log") + ".index"
	if _, err := os.Stat(indexPath); err == nil {
		indexKey := strings.TrimSuffix(s3Key, ".log") + ".index"
		if err := ts.s3Client.UploadFile(ctx, indexPath, indexKey); err != nil {
			ts.logger.Warn("failed to upload index", "error", err)
		}
	}

	timeIndexPath := strings.TrimSuffix(absPath, ".log") + ".timeindex"
	if _, err := os.Stat(timeIndexPath); err == nil {
		timeIndexKey := strings.TrimSuffix(s3Key, ".log") + ".timeindex"
		if err := ts.s3Client.UploadFile(ctx, timeIndexPath, timeIndexKey); err != nil {
			ts.logger.Warn("failed to upload timeindex", "error", err)
		}
	}

	ts.mu.Lock()
	meta.IsArchived = true
	meta.S3Key = s3Key
	meta.Policy = PolicyCold
	ts.mu.Unlock()

	if err := os.Remove(absPath); err != nil {
		ts.logger.Warn("failed to remove local segment", "error", err)
	}

	ts.logger.Info("segment archived successfully", "segment", segmentPath)
	return nil
}

func (ts *TieredStorage) RestoreSegment(ctx context.Context, segmentPath string) error {
	ts.mu.Lock()
	meta, exists := ts.metadata[segmentPath]
	if !exists {
		ts.mu.Unlock()
		return fmt.Errorf("segment not found: %s", segmentPath)
	}

	if !meta.IsArchived {
		ts.mu.Unlock()
		return nil
	}

	s3Key := meta.S3Key
	ts.mu.Unlock()

	absPath := filepath.Join(ts.config.DataDir, segmentPath)

	ts.logger.Info("restoring segment from s3",
		"segment", segmentPath,
		"s3_key", s3Key)

	if err := ts.s3Client.DownloadFile(ctx, s3Key, absPath); err != nil {
		return fmt.Errorf("download segment: %w", err)
	}

	indexKey := strings.TrimSuffix(s3Key, ".log") + ".index"
	indexPath := strings.TrimSuffix(absPath, ".log") + ".index"
	if exists, _ := ts.s3Client.FileExists(ctx, indexKey); exists {
		if err := ts.s3Client.DownloadFile(ctx, indexKey, indexPath); err != nil {
			ts.logger.Warn("failed to download index", "error", err)
		}
	}

	timeIndexKey := strings.TrimSuffix(s3Key, ".log") + ".timeindex"
	timeIndexPath := strings.TrimSuffix(absPath, ".log") + ".timeindex"
	if exists, _ := ts.s3Client.FileExists(ctx, timeIndexKey); exists {
		if err := ts.s3Client.DownloadFile(ctx, timeIndexKey, timeIndexPath); err != nil {
			ts.logger.Warn("failed to download timeindex", "error", err)
		}
	}

	ts.mu.Lock()
	meta.IsArchived = false
	meta.LastAccessAt = time.Now()
	meta.Policy = PolicyHot
	ts.mu.Unlock()

	ts.logger.Info("segment restored successfully", "segment", segmentPath)
	return nil
}

func (ts *TieredStorage) GetSegmentPolicy(segmentPath string) StoragePolicy {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if meta, exists := ts.metadata[segmentPath]; exists {
		return meta.Policy
	}

	return PolicyHot
}

func (ts *TieredStorage) IsSegmentArchived(segmentPath string) bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if meta, exists := ts.metadata[segmentPath]; exists {
		return meta.IsArchived
	}

	return false
}

func (ts *TieredStorage) startArchiver() {
	ts.wg.Add(1)
	go func() {
		defer ts.wg.Done()

		ticker := time.NewTicker(ts.config.ArchiveInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := ts.runArchivePolicy(context.Background()); err != nil {
					ts.logger.Error("archive policy run failed", "error", err)
				}
			case <-ts.stopCh:
				return
			}
		}
	}()
}

func (ts *TieredStorage) runArchivePolicy(ctx context.Context) error {
	ts.mu.RLock()
	candidatesForArchive := make([]*SegmentMetadata, 0)

	now := time.Now()
	for _, meta := range ts.metadata {
		if meta.IsArchived {
			continue
		}

		age := now.Sub(meta.LastModified)
		if age > ts.config.ColdAgeThreshold {
			candidatesForArchive = append(candidatesForArchive, meta)
		}
	}
	ts.mu.RUnlock()

	for _, meta := range candidatesForArchive {
		relPath, err := filepath.Rel(ts.config.DataDir, meta.Path)
		if err != nil {
			ts.logger.Warn("failed to get relative path", "path", meta.Path, "error", err)
			continue
		}

		if err := ts.ArchiveSegment(ctx, relPath); err != nil {
			ts.logger.Error("failed to archive segment",
				"segment", relPath,
				"error", err)
		}
	}

	return nil
}

func (ts *TieredStorage) UpdateAccessTime(segmentPath string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if meta, exists := ts.metadata[segmentPath]; exists {
		meta.LastAccessAt = time.Now()
	}
}

func (ts *TieredStorage) GetStats() map[string]interface{} {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	hotCount := 0
	warmCount := 0
	coldCount := 0
	archivedCount := 0
	var totalSize int64

	for _, meta := range ts.metadata {
		totalSize += meta.Size
		switch meta.Policy {
		case PolicyHot:
			hotCount++
		case PolicyWarm:
			warmCount++
		case PolicyCold:
			coldCount++
		}
		if meta.IsArchived {
			archivedCount++
		}
	}

	return map[string]interface{}{
		"total_segments":   len(ts.metadata),
		"hot_segments":     hotCount,
		"warm_segments":    warmCount,
		"cold_segments":    coldCount,
		"archived_segments": archivedCount,
		"total_size_bytes": totalSize,
	}
}

func (ts *TieredStorage) Close() error {
	close(ts.stopCh)
	ts.wg.Wait()
	return nil
}
