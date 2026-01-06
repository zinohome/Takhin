// Copyright 2025 Takhin Data, Inc.

package tiered

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/takhin-data/takhin/pkg/logger"
)

// TierType defines data tier classification
type TierType string

const (
	TierHot  TierType = "hot"
	TierWarm TierType = "warm"
	TierCold TierType = "cold"
)

// AccessPattern tracks segment access statistics
type AccessPattern struct {
	SegmentPath   string
	AccessCount   int64
	LastAccessAt  time.Time
	FirstAccessAt time.Time
	ReadBytes     int64
	WriteBytes    int64
	AverageReadHz float64 // Access frequency (reads per hour)
}

// TierPolicy defines rules for tier classification
type TierPolicy struct {
	HotMinAccessHz    float64       // Min accesses per hour to stay hot
	HotMaxAge         time.Duration // Max age before considering for warm
	WarmMinAge        time.Duration // Min age before eligible for warm
	WarmMaxAge        time.Duration // Max age before considering for cold
	ColdMinAge        time.Duration // Min age before eligible for cold
	HotMinAccessCount int64         // Min access count to stay hot
	LocalCacheMaxSize int64         // Max local cache size for warm/cold data
}

// TierManager handles automatic hot-warm-cold data classification
type TierManager struct {
	policy         TierPolicy
	accessPatterns map[string]*AccessPattern
	tierStorage    *TieredStorage
	mu             sync.RWMutex
	stopCh         chan struct{}
	wg             sync.WaitGroup
	logger         *logger.Logger
	
	// Metrics
	promotionCount int64
	demotionCount  int64
	cacheHits      int64
	cacheMisses    int64
}

// TierManagerConfig holds tier manager configuration
type TierManagerConfig struct {
	Policy         TierPolicy
	TieredStorage  *TieredStorage
	CheckInterval  time.Duration
}

// NewTierManager creates a new tier manager
func NewTierManager(config TierManagerConfig) *TierManager {
	tm := &TierManager{
		policy:         config.Policy,
		accessPatterns: make(map[string]*AccessPattern),
		tierStorage:    config.TieredStorage,
		stopCh:         make(chan struct{}),
		logger:         logger.Default().WithComponent("tier-manager"),
	}

	if config.CheckInterval > 0 {
		tm.startTierMonitor(config.CheckInterval)
	}

	return tm
}

// RecordAccess tracks segment access for tier decision making
func (tm *TierManager) RecordAccess(segmentPath string, readBytes int64) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	pattern, exists := tm.accessPatterns[segmentPath]
	if !exists {
		pattern = &AccessPattern{
			SegmentPath:   segmentPath,
			FirstAccessAt: time.Now(),
		}
		tm.accessPatterns[segmentPath] = pattern
	}

	pattern.AccessCount++
	pattern.LastAccessAt = time.Now()
	pattern.ReadBytes += readBytes

	// Calculate access frequency (reads per hour)
	duration := time.Since(pattern.FirstAccessAt)
	if duration > 0 {
		pattern.AverageReadHz = float64(pattern.AccessCount) / duration.Hours()
	}
}

// DetermineTier determines appropriate tier for a segment
func (tm *TierManager) DetermineTier(segmentPath string, segmentAge time.Duration) TierType {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	pattern := tm.accessPatterns[segmentPath]

	// No access pattern - use age-based classification
	if pattern == nil {
		if segmentAge < tm.policy.WarmMinAge {
			return TierHot
		} else if segmentAge < tm.policy.ColdMinAge {
			return TierWarm
		}
		return TierCold
	}

	// Hot tier: recently accessed with high frequency
	if pattern.AverageReadHz >= tm.policy.HotMinAccessHz && 
	   pattern.AccessCount >= tm.policy.HotMinAccessCount {
		return TierHot
	}

	// Check age constraints
	if segmentAge < tm.policy.WarmMinAge {
		return TierHot
	}

	if segmentAge >= tm.policy.ColdMinAge {
		return TierCold
	}

	// Default to warm tier
	return TierWarm
}

// PromoteSegment moves segment to a hotter tier
func (tm *TierManager) PromoteSegment(ctx context.Context, segmentPath string, fromTier, toTier TierType) error {
	if fromTier == TierCold && toTier != TierCold {
		// Restore from S3
		if err := tm.tierStorage.RestoreSegment(ctx, segmentPath); err != nil {
			return fmt.Errorf("restore segment: %w", err)
		}
		tm.logger.Info("segment promoted",
			"segment", segmentPath,
			"from", fromTier,
			"to", toTier)
		tm.promotionCount++
	}
	return nil
}

// DemoteSegment moves segment to a colder tier
func (tm *TierManager) DemoteSegment(ctx context.Context, segmentPath string, fromTier, toTier TierType) error {
	if toTier == TierCold && fromTier != TierCold {
		// Archive to S3
		if err := tm.tierStorage.ArchiveSegment(ctx, segmentPath); err != nil {
			return fmt.Errorf("archive segment: %w", err)
		}
		tm.logger.Info("segment demoted",
			"segment", segmentPath,
			"from", fromTier,
			"to", toTier)
		tm.demotionCount++
	}
	return nil
}

// EvaluateAndApplyTiers evaluates all segments and applies tier policies
func (tm *TierManager) EvaluateAndApplyTiers(ctx context.Context) error {
	tm.tierStorage.mu.RLock()
	segments := make([]*SegmentMetadata, 0, len(tm.tierStorage.metadata))
	for _, meta := range tm.tierStorage.metadata {
		segments = append(segments, meta)
	}
	tm.tierStorage.mu.RUnlock()

	now := time.Now()
	for _, meta := range segments {
		age := now.Sub(meta.LastModified)
		currentTier := tm.convertPolicyToTier(meta.Policy)
		desiredTier := tm.DetermineTier(meta.Path, age)

		if currentTier != desiredTier {
			if tm.shouldPromote(currentTier, desiredTier) {
				if err := tm.PromoteSegment(ctx, meta.Path, currentTier, desiredTier); err != nil {
					tm.logger.Error("failed to promote segment",
						"segment", meta.Path,
						"error", err)
				}
			} else if tm.shouldDemote(currentTier, desiredTier) {
				if err := tm.DemoteSegment(ctx, meta.Path, currentTier, desiredTier); err != nil {
					tm.logger.Error("failed to demote segment",
						"segment", meta.Path,
						"error", err)
				}
			}
		}
	}

	return nil
}

// startTierMonitor starts background tier monitoring
func (tm *TierManager) startTierMonitor(interval time.Duration) {
	tm.wg.Add(1)
	go func() {
		defer tm.wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := tm.EvaluateAndApplyTiers(context.Background()); err != nil {
					tm.logger.Error("tier evaluation failed", "error", err)
				}
			case <-tm.stopCh:
				return
			}
		}
	}()
}

// GetAccessStats returns access statistics for a segment
func (tm *TierManager) GetAccessStats(segmentPath string) *AccessPattern {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	if pattern, exists := tm.accessPatterns[segmentPath]; exists {
		// Return a copy to avoid race conditions
		copy := *pattern
		return &copy
	}
	return nil
}

// GetTierStats returns tier manager statistics
func (tm *TierManager) GetTierStats() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	hotCount := 0
	warmCount := 0
	coldCount := 0

	for _, pattern := range tm.accessPatterns {
		// Use a default age of 0 for current classification
		tier := tm.DetermineTier(pattern.SegmentPath, 0)
		switch tier {
		case TierHot:
			hotCount++
		case TierWarm:
			warmCount++
		case TierCold:
			coldCount++
		}
	}

	return map[string]interface{}{
		"hot_segments":      hotCount,
		"warm_segments":     warmCount,
		"cold_segments":     coldCount,
		"promotion_count":   tm.promotionCount,
		"demotion_count":    tm.demotionCount,
		"cache_hits":        tm.cacheHits,
		"cache_misses":      tm.cacheMisses,
		"tracked_segments":  len(tm.accessPatterns),
	}
}

// GetCostAnalysis provides cost analysis for tiered storage
func (tm *TierManager) GetCostAnalysis() map[string]interface{} {
	stats := tm.tierStorage.GetStats()
	
	// Simplified cost model (adjust based on actual cloud costs)
	const (
		hotStorageCostPerGB  = 0.023  // Local SSD cost per GB/month
		coldStorageCostPerGB = 0.004  // S3 Standard-IA cost per GB/month
		retrievalCostPerGB   = 0.01   // S3 retrieval cost per GB
	)

	hotSizeGB := float64(stats["total_size_bytes"].(int64)) / (1024 * 1024 * 1024)
	archivedCount := float64(stats["archived_segments"].(int))
	
	// Assume average segment size of 100MB for archived segments
	coldSizeGB := archivedCount * 0.1
	
	hotCost := hotSizeGB * hotStorageCostPerGB
	coldCost := coldSizeGB * coldStorageCostPerGB
	
	return map[string]interface{}{
		"hot_storage_gb":           hotSizeGB,
		"cold_storage_gb":          coldSizeGB,
		"hot_storage_cost_monthly": fmt.Sprintf("$%.2f", hotCost),
		"cold_storage_cost_monthly": fmt.Sprintf("$%.2f", coldCost),
		"total_cost_monthly":       fmt.Sprintf("$%.2f", hotCost+coldCost),
		"cost_savings_pct":         fmt.Sprintf("%.1f%%", (coldCost/(hotCost+coldCost))*100),
		"retrieval_cost_per_restore": fmt.Sprintf("$%.4f", 0.1*retrievalCostPerGB),
	}
}

// Close stops the tier manager
func (tm *TierManager) Close() error {
	close(tm.stopCh)
	tm.wg.Wait()
	return nil
}

// Helper methods

func (tm *TierManager) convertPolicyToTier(policy StoragePolicy) TierType {
	switch policy {
	case PolicyHot:
		return TierHot
	case PolicyWarm:
		return TierWarm
	case PolicyCold:
		return TierCold
	default:
		return TierHot
	}
}

func (tm *TierManager) shouldPromote(from, to TierType) bool {
	if from == TierCold && to == TierWarm {
		return true
	}
	if from == TierCold && to == TierHot {
		return true
	}
	if from == TierWarm && to == TierHot {
		return true
	}
	return false
}

func (tm *TierManager) shouldDemote(from, to TierType) bool {
	if from == TierHot && to == TierWarm {
		return true
	}
	if from == TierHot && to == TierCold {
		return true
	}
	if from == TierWarm && to == TierCold {
		return true
	}
	return false
}
