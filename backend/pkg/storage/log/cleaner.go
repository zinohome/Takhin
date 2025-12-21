// Copyright 2025 Takhin Data, Inc.

package log

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CleanerConfig defines configuration for the background cleaner
type CleanerConfig struct {
	// CleanupIntervalSeconds is how often to run cleanup (default: 300s = 5min)
	CleanupIntervalSeconds int

	// CompactionIntervalSeconds is how often to analyze compaction (default: 600s = 10min)
	CompactionIntervalSeconds int

	// RetentionPolicy is the retention policy to apply
	RetentionPolicy RetentionPolicy

	// CompactionPolicy is the compaction policy to apply
	CompactionPolicy CompactionPolicy

	// Enabled controls whether the cleaner runs
	Enabled bool
}

// DefaultCleanerConfig returns the default cleaner configuration
func DefaultCleanerConfig() CleanerConfig {
	return CleanerConfig{
		CleanupIntervalSeconds:    300, // 5 minutes
		CompactionIntervalSeconds: 600, // 10 minutes
		RetentionPolicy:           DefaultRetentionPolicy(),
		CompactionPolicy:          DefaultCompactionPolicy(),
		Enabled:                   true,
	}
}

// Cleaner manages background cleanup tasks for logs
type Cleaner struct {
	config         CleanerConfig
	logs           map[string]*Log // topic-partition -> Log
	mu             sync.RWMutex
	stopChan       chan struct{}
	wg             sync.WaitGroup
	logger         *zap.Logger
	stats          CleanerStats
	lastCleanup    time.Time
	lastCompaction time.Time
}

// CleanerStats tracks cleaner statistics
type CleanerStats struct {
	mu                     sync.RWMutex
	TotalCleanupRuns       int64
	TotalSegmentsDeleted   int64
	TotalBytesReclaimed    int64
	TotalCompactionRuns    int64
	LastCleanupDuration    time.Duration
	LastCompactionDuration time.Duration
	LastError              error
}

// NewCleaner creates a new background cleaner
func NewCleaner(config CleanerConfig) *Cleaner {
	logger, _ := zap.NewProduction()
	return &Cleaner{
		config:   config,
		logs:     make(map[string]*Log),
		stopChan: make(chan struct{}),
		logger:   logger,
	}
}

// RegisterLog registers a log for background cleanup
func (c *Cleaner) RegisterLog(name string, log *Log) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logs[name] = log
	c.logger.Info("Registered log for cleanup", zap.String("name", name))
}

// UnregisterLog removes a log from background cleanup
func (c *Cleaner) UnregisterLog(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.logs, name)
	c.logger.Info("Unregistered log from cleanup", zap.String("name", name))
}

// Start starts the background cleaner
func (c *Cleaner) Start() error {
	if !c.config.Enabled {
		c.logger.Info("Cleaner is disabled")
		return nil
	}

	c.wg.Add(2)

	// Start cleanup task
	go c.runCleanupLoop()

	// Start compaction analysis task
	go c.runCompactionLoop()

	c.logger.Info("Started background cleaner",
		zap.Int("cleanup_interval_sec", c.config.CleanupIntervalSeconds),
		zap.Int("compaction_interval_sec", c.config.CompactionIntervalSeconds))

	return nil
}

// Stop stops the background cleaner
func (c *Cleaner) Stop() error {
	close(c.stopChan)
	c.wg.Wait()
	c.logger.Info("Stopped background cleaner")
	return nil
}

// runCleanupLoop runs the cleanup task periodically
func (c *Cleaner) runCleanupLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(time.Duration(c.config.CleanupIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.runCleanup()
		}
	}
}

// runCompactionLoop runs the compaction analysis task periodically
func (c *Cleaner) runCompactionLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(time.Duration(c.config.CompactionIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.runCompactionAnalysis()
		}
	}
}

// runCleanup executes cleanup for all registered logs
func (c *Cleaner) runCleanup() {
	startTime := time.Now()
	c.lastCleanup = startTime

	c.mu.RLock()
	logs := make(map[string]*Log, len(c.logs))
	for name, log := range c.logs {
		logs[name] = log
	}
	c.mu.RUnlock()

	totalDeleted := 0
	totalBytes := int64(0)

	for name, log := range logs {
		deleted, bytes, err := log.DeleteSegmentsIfNeeded(c.config.RetentionPolicy)
		if err != nil {
			c.logger.Error("Cleanup failed",
				zap.String("log", name),
				zap.Error(err))
			c.updateStats(func(s *CleanerStats) {
				s.LastError = err
			})
			continue
		}

		if deleted > 0 {
			c.logger.Info("Cleaned up segments",
				zap.String("log", name),
				zap.Int("deleted", deleted),
				zap.Int64("bytes", bytes))
			totalDeleted += deleted
			totalBytes += bytes
		}
	}

	duration := time.Since(startTime)

	c.updateStats(func(s *CleanerStats) {
		s.TotalCleanupRuns++
		s.TotalSegmentsDeleted += int64(totalDeleted)
		s.TotalBytesReclaimed += totalBytes
		s.LastCleanupDuration = duration
	})

	if totalDeleted > 0 {
		c.logger.Info("Cleanup completed",
			zap.Int("total_deleted", totalDeleted),
			zap.Int64("total_bytes", totalBytes),
			zap.Duration("duration", duration))
	}
}

// runCompactionAnalysis analyzes compaction needs for all registered logs
func (c *Cleaner) runCompactionAnalysis() {
	startTime := time.Now()
	c.lastCompaction = startTime

	c.mu.RLock()
	logs := make(map[string]*Log, len(c.logs))
	for name, log := range c.logs {
		logs[name] = log
	}
	c.mu.RUnlock()

	for name, log := range logs {
		if log.NeedsCompaction(c.config.CompactionPolicy) {
			analysis := log.AnalyzeCompaction(c.config.CompactionPolicy)
			c.logger.Info("Compaction recommended",
				zap.String("log", name),
				zap.Int("total_segments", analysis.TotalSegments),
				zap.Int("compactable", analysis.CompactableSegments),
				zap.Int64("estimated_savings", analysis.EstimatedSavings))

			// In a production system, you would trigger actual compaction here
			// For now, we just analyze and log
		}
	}

	duration := time.Since(startTime)

	c.updateStats(func(s *CleanerStats) {
		s.TotalCompactionRuns++
		s.LastCompactionDuration = duration
	})
}

// GetStats returns current cleaner statistics
func (c *Cleaner) GetStats() CleanerStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()
	return c.stats
}

// updateStats safely updates cleaner statistics
func (c *Cleaner) updateStats(fn func(*CleanerStats)) {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()
	fn(&c.stats)
}

// GetStatus returns the current status of the cleaner
func (c *Cleaner) GetStatus() CleanerStatus {
	c.mu.RLock()
	numLogs := len(c.logs)
	c.mu.RUnlock()

	stats := c.GetStats()

	return CleanerStatus{
		Enabled:                c.config.Enabled,
		NumRegisteredLogs:      numLogs,
		TotalCleanupRuns:       stats.TotalCleanupRuns,
		TotalSegmentsDeleted:   stats.TotalSegmentsDeleted,
		TotalBytesReclaimed:    stats.TotalBytesReclaimed,
		TotalCompactionRuns:    stats.TotalCompactionRuns,
		LastCleanup:            c.lastCleanup,
		LastCompaction:         c.lastCompaction,
		LastCleanupDuration:    stats.LastCleanupDuration,
		LastCompactionDuration: stats.LastCompactionDuration,
		LastError:              stats.LastError,
	}
}

// CleanerStatus represents the current status of the cleaner
type CleanerStatus struct {
	Enabled                bool
	NumRegisteredLogs      int
	TotalCleanupRuns       int64
	TotalSegmentsDeleted   int64
	TotalBytesReclaimed    int64
	TotalCompactionRuns    int64
	LastCleanup            time.Time
	LastCompaction         time.Time
	LastCleanupDuration    time.Duration
	LastCompactionDuration time.Duration
	LastError              error
}

// ForceCleanup manually triggers a cleanup run
func (c *Cleaner) ForceCleanup() error {
	c.logger.Info("Manual cleanup triggered")
	c.runCleanup()
	return nil
}

// ForceCompactionAnalysis manually triggers a compaction analysis
func (c *Cleaner) ForceCompactionAnalysis() error {
	c.logger.Info("Manual compaction analysis triggered")
	c.runCompactionAnalysis()
	return nil
}

// UpdateConfig updates the cleaner configuration
func (c *Cleaner) UpdateConfig(config CleanerConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldConfig := c.config
	c.config = config

	c.logger.Info("Updated cleaner configuration",
		zap.Any("old", oldConfig),
		zap.Any("new", config))

	// If enabled state changed, log it
	if oldConfig.Enabled != config.Enabled {
		if config.Enabled {
			c.logger.Info("Cleaner enabled")
		} else {
			c.logger.Info("Cleaner disabled")
		}
	}

	return nil
}

// String returns a string representation of the cleaner status
func (s CleanerStatus) String() string {
	if s.LastError != nil {
		return fmt.Sprintf("Cleaner[enabled=%v, logs=%d, runs=%d/%d, deleted=%d segments (%.2f MB), last_error=%v]",
			s.Enabled, s.NumRegisteredLogs, s.TotalCleanupRuns, s.TotalCompactionRuns,
			s.TotalSegmentsDeleted, float64(s.TotalBytesReclaimed)/(1024*1024), s.LastError)
	}
	return fmt.Sprintf("Cleaner[enabled=%v, logs=%d, runs=%d/%d, deleted=%d segments (%.2f MB)]",
		s.Enabled, s.NumRegisteredLogs, s.TotalCleanupRuns, s.TotalCompactionRuns,
		s.TotalSegmentsDeleted, float64(s.TotalBytesReclaimed)/(1024*1024))
}
