// Copyright 2025 Takhin Data, Inc.

package tiered

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func BenchmarkArchiveSegment(b *testing.B) {
	ctx := context.Background()
	tmpDir := b.TempDir()

	segmentPath := filepath.Join(tmpDir, "00000000000000000000.log")
	segmentData := make([]byte, 10*1024*1024)
	if err := os.WriteFile(segmentPath, segmentData, 0644); err != nil {
		b.Fatal(err)
	}

	ts := &TieredStorage{
		config: TieredStorageConfig{
			DataDir: tmpDir,
		},
		metadata: make(map[string]*SegmentMetadata),
	}

	ts.metadata["00000000000000000000.log"] = &SegmentMetadata{
		Path:         segmentPath,
		Size:         int64(len(segmentData)),
		LastModified: time.Now(),
		Policy:       PolicyHot,
		IsArchived:   false,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ctx
	}
}

func BenchmarkMetadataLookup(b *testing.B) {
	ts := &TieredStorage{
		metadata: make(map[string]*SegmentMetadata),
	}

	for i := 0; i < 1000; i++ {
		key := filepath.Join("topic", "partition", "segment.log")
		ts.metadata[key] = &SegmentMetadata{
			Path:   key,
			Policy: PolicyHot,
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ts.GetSegmentPolicy("topic/partition/segment.log")
	}
}

func BenchmarkPolicyCheck(b *testing.B) {
	now := time.Now()
	lastModified := now.Add(-50 * time.Hour)
	coldThreshold := 48 * time.Hour

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		age := now.Sub(lastModified)
		_ = age > coldThreshold
	}
}

func BenchmarkConcurrentAccess(b *testing.B) {
	ts := &TieredStorage{
		metadata: make(map[string]*SegmentMetadata),
	}

	for i := 0; i < 100; i++ {
		key := filepath.Join("topic", "partition", "segment.log")
		ts.metadata[key] = &SegmentMetadata{
			Path:         key,
			Policy:       PolicyHot,
			LastAccessAt: time.Now(),
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ts.UpdateAccessTime("topic/partition/segment.log")
		}
	})
}

func BenchmarkStatsCollection(b *testing.B) {
	ts := &TieredStorage{
		metadata: make(map[string]*SegmentMetadata),
	}

	for i := 0; i < 1000; i++ {
		key := filepath.Join("topic", "partition", "segment.log")
		ts.metadata[key] = &SegmentMetadata{
			Path:       key,
			Size:       1024 * 1024,
			Policy:     PolicyHot,
			IsArchived: i%10 == 0,
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ts.GetStats()
	}
}
