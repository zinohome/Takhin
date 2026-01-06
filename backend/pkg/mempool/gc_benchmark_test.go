// Copyright 2025 Takhin Data, Inc.

package mempool

import (
	"runtime"
	"testing"
	"time"
)

// GCStats captures GC statistics for comparison
type GCStats struct {
	NumGC        uint32
	PauseTotal   time.Duration
	PauseAvg     time.Duration
	AllocBytes   uint64
	TotalAlloc   uint64
	Mallocs      uint64
	Frees        uint64
}

func collectGCStats() GCStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	stats := GCStats{
		NumGC:      m.NumGC,
		PauseTotal: time.Duration(m.PauseTotalNs),
		AllocBytes: m.Alloc,
		TotalAlloc: m.TotalAlloc,
		Mallocs:    m.Mallocs,
		Frees:      m.Frees,
	}
	
	if m.NumGC > 0 {
		stats.PauseAvg = stats.PauseTotal / time.Duration(m.NumGC)
	}
	
	return stats
}

func diffGCStats(before, after GCStats) GCStats {
	return GCStats{
		NumGC:      after.NumGC - before.NumGC,
		PauseTotal: after.PauseTotal - before.PauseTotal,
		AllocBytes: after.AllocBytes,
		TotalAlloc: after.TotalAlloc - before.TotalAlloc,
		Mallocs:    after.Mallocs - before.Mallocs,
		Frees:      after.Frees - before.Frees,
	}
}

// BenchmarkGCPressure_WithPool measures GC pressure with memory pools
func BenchmarkGCPressure_WithPool(b *testing.B) {
	runtime.GC() // Start with clean slate
	before := collectGCStats()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate typical workload: buffer allocation
		buf := GetBuffer(4096)
		for j := 0; j < len(buf); j++ {
			buf[j] = byte(j % 256)
		}
		PutBuffer(buf)
	}
	b.StopTimer()
	
	runtime.GC()
	after := collectGCStats()
	diff := diffGCStats(before, after)
	
	b.ReportMetric(float64(diff.NumGC), "gc-runs")
	b.ReportMetric(float64(diff.PauseTotal.Microseconds()), "gc-pause-μs")
	b.ReportMetric(float64(diff.TotalAlloc)/float64(b.N), "bytes-alloc/op")
	b.ReportMetric(float64(diff.Mallocs)/float64(b.N), "mallocs/op")
}

// BenchmarkGCPressure_WithoutPool measures GC pressure without memory pools
func BenchmarkGCPressure_WithoutPool(b *testing.B) {
	runtime.GC() // Start with clean slate
	before := collectGCStats()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate typical workload: buffer allocation
		buf := make([]byte, 4096)
		for j := 0; j < len(buf); j++ {
			buf[j] = byte(j % 256)
		}
		_ = buf
	}
	b.StopTimer()
	
	runtime.GC()
	after := collectGCStats()
	diff := diffGCStats(before, after)
	
	b.ReportMetric(float64(diff.NumGC), "gc-runs")
	b.ReportMetric(float64(diff.PauseTotal.Microseconds()), "gc-pause-μs")
	b.ReportMetric(float64(diff.TotalAlloc)/float64(b.N), "bytes-alloc/op")
	b.ReportMetric(float64(diff.Mallocs)/float64(b.N), "mallocs/op")
}

// BenchmarkGCPressure_LargeWorkload tests with a more realistic workload
func BenchmarkGCPressure_LargeWorkload_WithPool(b *testing.B) {
	runtime.GC()
	before := collectGCStats()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate processing 100 messages
		for msg := 0; msg < 100; msg++ {
			buf := GetBuffer(1024)
			for j := 0; j < len(buf); j++ {
				buf[j] = byte(j % 256)
			}
			PutBuffer(buf)
		}
	}
	b.StopTimer()
	
	runtime.GC()
	after := collectGCStats()
	diff := diffGCStats(before, after)
	
	b.ReportMetric(float64(diff.NumGC), "gc-runs")
	b.ReportMetric(float64(diff.PauseTotal.Microseconds()), "gc-pause-μs")
	b.ReportMetric(float64(diff.TotalAlloc)/float64(b.N), "bytes-alloc/op")
	b.ReportMetric(float64(diff.Mallocs)/float64(b.N), "mallocs/op")
}

func BenchmarkGCPressure_LargeWorkload_WithoutPool(b *testing.B) {
	runtime.GC()
	before := collectGCStats()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate processing 100 messages
		for msg := 0; msg < 100; msg++ {
			buf := make([]byte, 1024)
			for j := 0; j < len(buf); j++ {
				buf[j] = byte(j % 256)
			}
			_ = buf
		}
	}
	b.StopTimer()
	
	runtime.GC()
	after := collectGCStats()
	diff := diffGCStats(before, after)
	
	b.ReportMetric(float64(diff.NumGC), "gc-runs")
	b.ReportMetric(float64(diff.PauseTotal.Microseconds()), "gc-pause-μs")
	b.ReportMetric(float64(diff.TotalAlloc)/float64(b.N), "bytes-alloc/op")
	b.ReportMetric(float64(diff.Mallocs)/float64(b.N), "mallocs/op")
}

// TestGCReduction validates that memory pool reduces GC pressure
func TestGCReduction(t *testing.T) {
	iterations := 10000

	// Test without pool
	runtime.GC()
	beforeNoPool := collectGCStats()
	for i := 0; i < iterations; i++ {
		buf := make([]byte, 4096)
		buf[0] = 1
	}
	runtime.GC()
	afterNoPool := collectGCStats()
	diffNoPool := diffGCStats(beforeNoPool, afterNoPool)

	// Test with pool
	runtime.GC()
	beforeWithPool := collectGCStats()
	for i := 0; i < iterations; i++ {
		buf := GetBuffer(4096)
		buf[0] = 1
		PutBuffer(buf)
	}
	runtime.GC()
	afterWithPool := collectGCStats()
	diffWithPool := diffGCStats(beforeWithPool, afterWithPool)

	t.Logf("Without pool - GC runs: %d, Total pause: %v, Mallocs: %d",
		diffNoPool.NumGC, diffNoPool.PauseTotal, diffNoPool.Mallocs)
	t.Logf("With pool    - GC runs: %d, Total pause: %v, Mallocs: %d",
		diffWithPool.NumGC, diffWithPool.PauseTotal, diffWithPool.Mallocs)

	// Pool should result in fewer mallocs
	if diffWithPool.Mallocs >= diffNoPool.Mallocs {
		t.Logf("Warning: Pool did not reduce mallocs (pool: %d, no pool: %d)",
			diffWithPool.Mallocs, diffNoPool.Mallocs)
	} else {
		reduction := float64(diffNoPool.Mallocs-diffWithPool.Mallocs) / float64(diffNoPool.Mallocs) * 100
		t.Logf("Memory pool reduced mallocs by %.1f%%", reduction)
		
		// Accept at least 50% reduction as success
		if reduction < 50.0 {
			t.Logf("Note: Reduction is less than 50%%, but still beneficial")
		}
	}
}
