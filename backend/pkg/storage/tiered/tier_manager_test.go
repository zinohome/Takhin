// Copyright 2025 Takhin Data, Inc.

package tiered

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTierDetermination(t *testing.T) {
	policy := TierPolicy{
		HotMinAccessHz:    10.0,
		HotMaxAge:         24 * time.Hour,
		WarmMinAge:        24 * time.Hour,
		WarmMaxAge:        7 * 24 * time.Hour,
		ColdMinAge:        7 * 24 * time.Hour,
		HotMinAccessCount: 100,
		LocalCacheMaxSize: 10 * 1024 * 1024 * 1024,
	}

	config := TierManagerConfig{
		Policy:        policy,
		CheckInterval: 0, // Disable background monitoring for tests
	}

	tm := NewTierManager(config)
	defer tm.Close()

	tests := []struct {
		name        string
		segmentPath string
		age         time.Duration
		setup       func()
		expected    TierType
	}{
		{
			name:        "new segment without access",
			segmentPath: "topic-0/00000000000000000000.log",
			age:         1 * time.Hour,
			setup:       func() {},
			expected:    TierHot,
		},
		{
			name:        "old segment without access",
			segmentPath: "topic-0/00000000000000000001.log",
			age:         10 * 24 * time.Hour,
			setup:       func() {},
			expected:    TierCold,
		},
		{
			name:        "hot segment with high access",
			segmentPath: "topic-0/00000000000000000002.log",
			age:         2 * time.Hour,
			setup: func() {
				// Simulate high access frequency
				for i := 0; i < 150; i++ {
					tm.RecordAccess("topic-0/00000000000000000002.log", 1024)
				}
			},
			expected: TierHot,
		},
		{
			name:        "warm segment with moderate age",
			segmentPath: "topic-0/00000000000000000003.log",
			age:         3 * 24 * time.Hour,
			setup:       func() {},
			expected:    TierWarm,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			tier := tm.DetermineTier(tt.segmentPath, tt.age)
			assert.Equal(t, tt.expected, tier)
		})
	}
}

func TestAccessPatternTracking(t *testing.T) {
	policy := TierPolicy{
		HotMinAccessHz:    10.0,
		HotMaxAge:         24 * time.Hour,
		WarmMinAge:        24 * time.Hour,
		WarmMaxAge:        7 * 24 * time.Hour,
		ColdMinAge:        7 * 24 * time.Hour,
		HotMinAccessCount: 10,
	}

	config := TierManagerConfig{
		Policy:        policy,
		CheckInterval: 0,
	}

	tm := NewTierManager(config)
	defer tm.Close()

	segmentPath := "topic-0/00000000000000000000.log"

	// Record multiple accesses
	for i := 0; i < 50; i++ {
		tm.RecordAccess(segmentPath, 1024)
		time.Sleep(1 * time.Millisecond)
	}

	stats := tm.GetAccessStats(segmentPath)
	require.NotNil(t, stats)
	assert.Equal(t, int64(50), stats.AccessCount)
	assert.Equal(t, int64(50*1024), stats.ReadBytes)
	assert.Greater(t, stats.AverageReadHz, 0.0)
}

func TestTierPromotion(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	s3Config := S3Config{
		Region: "us-east-1",
		Bucket: "test-bucket",
		Prefix: "test",
	}

	tsConfig := TieredStorageConfig{
		DataDir:            tmpDir,
		S3Config:           s3Config,
		ColdAgeThreshold:   7 * 24 * time.Hour,
		WarmAgeThreshold:   3 * 24 * time.Hour,
		AutoArchiveEnabled: false,
	}

	// Note: This will fail without actual S3 credentials, but tests the logic
	ts, err := NewTieredStorage(ctx, tsConfig)
	if err != nil {
		t.Skip("Skipping test requiring S3 credentials")
	}
	defer ts.Close()

	policy := TierPolicy{
		HotMinAccessHz:    5.0,
		HotMaxAge:         24 * time.Hour,
		WarmMinAge:        24 * time.Hour,
		WarmMaxAge:        7 * 24 * time.Hour,
		ColdMinAge:        7 * 24 * time.Hour,
		HotMinAccessCount: 10,
	}

	config := TierManagerConfig{
		Policy:         policy,
		TieredStorage:  ts,
		CheckInterval:  0,
	}

	tm := NewTierManager(config)
	defer tm.Close()

	// Test promotion logic (without actual S3 operations)
	segmentPath := "topic-0/00000000000000000000.log"

	// Record accesses to make it hot
	for i := 0; i < 20; i++ {
		tm.RecordAccess(segmentPath, 1024)
	}

	tier := tm.DetermineTier(segmentPath, 1*time.Hour)
	assert.Equal(t, TierHot, tier)
}

func TestTierDemotion(t *testing.T) {
	policy := TierPolicy{
		HotMinAccessHz:    10.0,
		HotMaxAge:         24 * time.Hour,
		WarmMinAge:        24 * time.Hour,
		WarmMaxAge:        7 * 24 * time.Hour,
		ColdMinAge:        7 * 24 * time.Hour,
		HotMinAccessCount: 100,
	}

	config := TierManagerConfig{
		Policy:        policy,
		CheckInterval: 0,
	}

	tm := NewTierManager(config)
	defer tm.Close()

	segmentPath := "topic-0/00000000000000000000.log"

	// Old segment with no access should be cold
	tier := tm.DetermineTier(segmentPath, 10*24*time.Hour)
	assert.Equal(t, TierCold, tier)
}

func TestCostAnalysis(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	s3Config := S3Config{
		Region: "us-east-1",
		Bucket: "test-bucket",
		Prefix: "test",
	}

	tsConfig := TieredStorageConfig{
		DataDir:            tmpDir,
		S3Config:           s3Config,
		ColdAgeThreshold:   7 * 24 * time.Hour,
		WarmAgeThreshold:   3 * 24 * time.Hour,
		AutoArchiveEnabled: false,
	}

	ts, err := NewTieredStorage(ctx, tsConfig)
	if err != nil {
		t.Skip("Skipping test requiring S3 credentials")
	}
	defer ts.Close()

	policy := TierPolicy{
		HotMinAccessHz:    10.0,
		HotMaxAge:         24 * time.Hour,
		WarmMinAge:        24 * time.Hour,
		WarmMaxAge:        7 * 24 * time.Hour,
		ColdMinAge:        7 * 24 * time.Hour,
		HotMinAccessCount: 100,
	}

	config := TierManagerConfig{
		Policy:         policy,
		TieredStorage:  ts,
		CheckInterval:  0,
	}

	tm := NewTierManager(config)
	defer tm.Close()

	analysis := tm.GetCostAnalysis()
	assert.NotNil(t, analysis)
	assert.Contains(t, analysis, "hot_storage_gb")
	assert.Contains(t, analysis, "cold_storage_gb")
	assert.Contains(t, analysis, "total_cost_monthly")
}

func TestTierStats(t *testing.T) {
	policy := TierPolicy{
		HotMinAccessHz:    10.0,
		HotMaxAge:         24 * time.Hour,
		WarmMinAge:        24 * time.Hour,
		WarmMaxAge:        7 * 24 * time.Hour,
		ColdMinAge:        7 * 24 * time.Hour,
		HotMinAccessCount: 100,
	}

	config := TierManagerConfig{
		Policy:        policy,
		CheckInterval: 0,
	}

	tm := NewTierManager(config)
	defer tm.Close()

	// Record accesses for multiple segments
	segments := []string{
		"topic-0/00000000000000000000.log",
		"topic-0/00000000000000000001.log",
		"topic-0/00000000000000000002.log",
	}

	for _, seg := range segments {
		tm.RecordAccess(seg, 1024)
	}

	stats := tm.GetTierStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 3, stats["tracked_segments"])
}

func TestConcurrentAccessTracking(t *testing.T) {
	policy := TierPolicy{
		HotMinAccessHz:    10.0,
		HotMaxAge:         24 * time.Hour,
		WarmMinAge:        24 * time.Hour,
		WarmMaxAge:        7 * 24 * time.Hour,
		ColdMinAge:        7 * 24 * time.Hour,
		HotMinAccessCount: 100,
	}

	config := TierManagerConfig{
		Policy:        policy,
		CheckInterval: 0,
	}

	tm := NewTierManager(config)
	defer tm.Close()

	segmentPath := "topic-0/00000000000000000000.log"

	// Concurrent access tracking
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				tm.RecordAccess(segmentPath, 1024)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	stats := tm.GetAccessStats(segmentPath)
	require.NotNil(t, stats)
	assert.Equal(t, int64(1000), stats.AccessCount)
}
