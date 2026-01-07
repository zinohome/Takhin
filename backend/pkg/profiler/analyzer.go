// Copyright 2025 Takhin Data, Inc.

package profiler

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/pprof/profile"
)

type ProfileStats struct {
	TotalSamples   int64
	TopFunctions   []FunctionStat
	MemoryStats    *MemoryStats
	LatencyStats   *LatencyStats
	GoroutineStats *GoroutineStats
}

type FunctionStat struct {
	Name        string
	SelfTime    time.Duration
	CumTime     time.Duration
	SelfPercent float64
	CumPercent  float64
	Samples     int64
}

type MemoryStats struct {
	TotalAlloc      uint64
	Alloc           uint64
	Sys             uint64
	HeapAlloc       uint64
	HeapInuse       uint64
	HeapIdle        uint64
	StackInuse      uint64
	NumGC           uint32
	TopAllocations  []AllocationStat
}

type AllocationStat struct {
	Function    string
	AllocBytes  int64
	AllocCount  int64
	InuseBytes  int64
	InuseCount  int64
}

type LatencyStats struct {
	P50  time.Duration
	P90  time.Duration
	P95  time.Duration
	P99  time.Duration
	Max  time.Duration
	Mean time.Duration
}

type GoroutineStats struct {
	Total       int
	Running     int
	Waiting     int
	TopStacks   []GoroutineStack
}

type GoroutineStack struct {
	State string
	Count int
	Stack string
}

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) Analyze(profilePath string, profileType ProfileType) (*ProfileStats, error) {
	switch profileType {
	case ProfileTypeCPU:
		return a.analyzeCPU(profilePath)
	case ProfileTypeHeap, ProfileTypeMemory:
		return a.analyzeHeap(profilePath)
	case ProfileTypeGoroutine:
		return a.analyzeGoroutine(profilePath)
	default:
		return a.analyzeGeneric(profilePath)
	}
}

func (a *Analyzer) analyzeCPU(profilePath string) (*ProfileStats, error) {
	f, err := os.Open(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open profile: %w", err)
	}
	defer f.Close()

	prof, err := profile.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}

	stats := &ProfileStats{
		TopFunctions: make([]FunctionStat, 0),
	}

	var totalSamples int64
	for _, sample := range prof.Sample {
		totalSamples += sample.Value[0]
	}
	stats.TotalSamples = totalSamples

	funcStats := make(map[string]*FunctionStat)
	for _, sample := range prof.Sample {
		value := sample.Value[0]
		if len(sample.Location) > 0 && len(sample.Location[0].Line) > 0 {
			funcName := sample.Location[0].Line[0].Function.Name
			if stat, ok := funcStats[funcName]; ok {
				stat.Samples += value
			} else {
				funcStats[funcName] = &FunctionStat{
					Name:    funcName,
					Samples: value,
				}
			}
		}
	}

	for _, stat := range funcStats {
		if totalSamples > 0 {
			stat.SelfPercent = float64(stat.Samples) / float64(totalSamples) * 100
		}
		stats.TopFunctions = append(stats.TopFunctions, *stat)
	}

	return stats, nil
}

func (a *Analyzer) analyzeHeap(profilePath string) (*ProfileStats, error) {
	f, err := os.Open(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open profile: %w", err)
	}
	defer f.Close()

	prof, err := profile.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}

	stats := &ProfileStats{
		MemoryStats: &MemoryStats{
			TopAllocations: make([]AllocationStat, 0),
		},
	}

	allocStats := make(map[string]*AllocationStat)
	for _, sample := range prof.Sample {
		if len(sample.Location) > 0 && len(sample.Location[0].Line) > 0 {
			funcName := sample.Location[0].Line[0].Function.Name
			if len(sample.Value) >= 4 {
				allocBytes := sample.Value[0]
				allocCount := sample.Value[1]
				inuseBytes := sample.Value[2]
				inuseCount := sample.Value[3]

				if stat, ok := allocStats[funcName]; ok {
					stat.AllocBytes += allocBytes
					stat.AllocCount += allocCount
					stat.InuseBytes += inuseBytes
					stat.InuseCount += inuseCount
				} else {
					allocStats[funcName] = &AllocationStat{
						Function:   funcName,
						AllocBytes: allocBytes,
						AllocCount: allocCount,
						InuseBytes: inuseBytes,
						InuseCount: inuseCount,
					}
				}
			}
		}
	}

	for _, stat := range allocStats {
		stats.MemoryStats.TopAllocations = append(stats.MemoryStats.TopAllocations, *stat)
	}

	return stats, nil
}

func (a *Analyzer) analyzeGoroutine(profilePath string) (*ProfileStats, error) {
	f, err := os.Open(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open profile: %w", err)
	}
	defer f.Close()

	prof, err := profile.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}

	stats := &ProfileStats{
		GoroutineStats: &GoroutineStats{
			TopStacks: make([]GoroutineStack, 0),
		},
	}

	stackCounts := make(map[string]*GoroutineStack)
	for _, sample := range prof.Sample {
		if len(sample.Value) > 0 {
			count := int(sample.Value[0])
			stats.GoroutineStats.Total += count

			var stackBuilder strings.Builder
			for _, loc := range sample.Location {
				for _, line := range loc.Line {
					stackBuilder.WriteString(fmt.Sprintf("  %s\n", line.Function.Name))
				}
			}
			stackStr := stackBuilder.String()

			if stack, ok := stackCounts[stackStr]; ok {
				stack.Count += count
			} else {
				stackCounts[stackStr] = &GoroutineStack{
					Stack: stackStr,
					Count: count,
					State: "unknown",
				}
			}
		}
	}

	for _, stack := range stackCounts {
		stats.GoroutineStats.TopStacks = append(stats.GoroutineStats.TopStacks, *stack)
	}

	return stats, nil
}

func (a *Analyzer) analyzeGeneric(profilePath string) (*ProfileStats, error) {
	f, err := os.Open(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open profile: %w", err)
	}
	defer f.Close()

	prof, err := profile.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}

	stats := &ProfileStats{
		TopFunctions: make([]FunctionStat, 0),
	}

	var totalSamples int64
	for _, sample := range prof.Sample {
		if len(sample.Value) > 0 {
			totalSamples += sample.Value[0]
		}
	}
	stats.TotalSamples = totalSamples

	return stats, nil
}

func (a *Analyzer) GenerateReport(stats *ProfileStats, profileType ProfileType) string {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "\n=== Profile Report: %s ===\n\n", profileType)

	if profileType == ProfileTypeCPU {
		fmt.Fprintf(w, "Total Samples:\t%d\n\n", stats.TotalSamples)
		fmt.Fprintf(w, "Top Functions by CPU Time:\n")
		fmt.Fprintf(w, "Function\tSamples\tPercent\n")
		fmt.Fprintf(w, "--------\t-------\t-------\n")
		
		count := 0
		for _, fn := range stats.TopFunctions {
			if count >= 10 {
				break
			}
			fmt.Fprintf(w, "%s\t%d\t%.2f%%\n", fn.Name, fn.Samples, fn.SelfPercent)
			count++
		}
	}

	if profileType == ProfileTypeHeap && stats.MemoryStats != nil {
		fmt.Fprintf(w, "Top Memory Allocations:\n")
		fmt.Fprintf(w, "Function\tAlloc Bytes\tAlloc Count\tInuse Bytes\tInuse Count\n")
		fmt.Fprintf(w, "--------\t-----------\t-----------\t-----------\t-----------\n")
		
		count := 0
		for _, alloc := range stats.MemoryStats.TopAllocations {
			if count >= 10 {
				break
			}
			fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\n",
				alloc.Function,
				alloc.AllocBytes,
				alloc.AllocCount,
				alloc.InuseBytes,
				alloc.InuseCount,
			)
			count++
		}
	}

	if profileType == ProfileTypeGoroutine && stats.GoroutineStats != nil {
		fmt.Fprintf(w, "Total Goroutines:\t%d\n\n", stats.GoroutineStats.Total)
		fmt.Fprintf(w, "Top Goroutine Stacks:\n")
		fmt.Fprintf(w, "Count\tStack\n")
		fmt.Fprintf(w, "-----\t-----\n")
		
		count := 0
		for _, stack := range stats.GoroutineStats.TopStacks {
			if count >= 5 {
				break
			}
			fmt.Fprintf(w, "%d\t%s\n", stack.Count, strings.TrimSpace(stack.Stack))
			count++
		}
	}

	w.Flush()
	return buf.String()
}

func (a *Analyzer) GenerateFlameGraph(profilePath string, outputPath string) error {
	goToolPath, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("go tool not found: %w", err)
	}

	cmd := exec.Command(goToolPath, "tool", "pprof", "-http=:0", "-no_browser", profilePath)
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate flame graph: %s: %w", stderr.String(), err)
	}

	return nil
}
