// Copyright 2025 Takhin Data, Inc.

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/takhin-data/takhin/pkg/logger"
	"github.com/takhin-data/takhin/pkg/profiler"
)

var (
	profileType    = flag.String("type", "cpu", "Profile type: cpu, heap, allocs, goroutine, block, mutex, trace, all")
	duration       = flag.Duration("duration", 30*time.Second, "Profile duration (for cpu/trace)")
	output         = flag.String("output", "", "Output path (default: /tmp/takhin_<type>_<timestamp>.prof)")
	outputDir      = flag.String("output-dir", "/tmp", "Output directory for 'all' type")
	sampleRate     = flag.Int("sample-rate", 1, "Sample rate for block/mutex profiles")
	memProfileRate = flag.Int("mem-rate", 0, "Memory profile rate (0 = default)")
	remote         = flag.String("remote", "", "Remote server address (e.g., localhost:6060)")
	analyze        = flag.Bool("analyze", false, "Analyze profile after collection")
	flamegraph     = flag.Bool("flamegraph", false, "Generate flame graph (requires go tool pprof)")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	log := logger.Default().WithComponent("takhin-profiler")
	ctx := context.Background()

	if *remote != "" {
		return collectRemoteProfile(ctx, log)
	}

	return collectLocalProfile(ctx, log)
}

func collectLocalProfile(ctx context.Context, log *logger.Logger) error {
	prof := profiler.New()

	ptype := profiler.ProfileType(*profileType)

	if ptype == "all" {
		log.Info("collecting all profile types", "output_dir", *outputDir)
		results, err := prof.ProfileAll(ctx, *outputDir, *duration)
		if err != nil {
			return fmt.Errorf("failed to collect profiles: %w", err)
		}

		fmt.Println("\nProfile collection completed:")
		for typ, path := range results {
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			fmt.Printf("  %s: %s (%.2f KB)\n", typ, path, float64(info.Size())/1024)

			if *analyze {
				if err := analyzeProfile(path, typ); err != nil {
					log.Error("failed to analyze profile", "type", typ, "error", err)
				}
			}
		}
		return nil
	}

	opts := &profiler.ProfileOptions{
		Type:           ptype,
		Duration:       *duration,
		OutputPath:     *output,
		SampleRate:     *sampleRate,
		MemProfileRate: *memProfileRate,
	}

	log.Info("starting profile collection", "type", ptype, "duration", *duration)

	path, err := prof.Profile(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to collect profile: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat profile file: %w", err)
	}

	fmt.Printf("\nProfile collected successfully!\n")
	fmt.Printf("  Type: %s\n", ptype)
	fmt.Printf("  Path: %s\n", path)
	fmt.Printf("  Size: %.2f KB\n", float64(info.Size())/1024)

	if *analyze {
		if err := analyzeProfile(path, ptype); err != nil {
			return fmt.Errorf("failed to analyze profile: %w", err)
		}
	}

	if *flamegraph {
		fmt.Println("\nGenerating flame graph...")
		fmt.Printf("Run: go tool pprof -http=:8080 %s\n", path)
	}

	return nil
}

func collectRemoteProfile(ctx context.Context, log *logger.Logger) error {
	if !strings.HasPrefix(*remote, "http://") && !strings.HasPrefix(*remote, "https://") {
		*remote = "http://" + *remote
	}

	var endpoint string
	ptype := profiler.ProfileType(*profileType)

	switch ptype {
	case profiler.ProfileTypeCPU:
		endpoint = fmt.Sprintf("%s/debug/pprof/profile?seconds=%d", *remote, int(duration.Seconds()))
	case profiler.ProfileTypeHeap, profiler.ProfileTypeMemory:
		endpoint = fmt.Sprintf("%s/debug/pprof/heap", *remote)
	case profiler.ProfileTypeAllocs:
		endpoint = fmt.Sprintf("%s/debug/pprof/allocs", *remote)
	case profiler.ProfileTypeGoroutine:
		endpoint = fmt.Sprintf("%s/debug/pprof/goroutine", *remote)
	case profiler.ProfileTypeBlock:
		endpoint = fmt.Sprintf("%s/debug/pprof/block", *remote)
	case profiler.ProfileTypeMutex:
		endpoint = fmt.Sprintf("%s/debug/pprof/mutex", *remote)
	case profiler.ProfileTypeTrace:
		endpoint = fmt.Sprintf("%s/debug/pprof/trace?seconds=%d", *remote, int(duration.Seconds()))
	default:
		return fmt.Errorf("unsupported profile type for remote collection: %s", ptype)
	}

	log.Info("collecting remote profile", "endpoint", endpoint)

	resp, err := http.Get(endpoint)
	if err != nil {
		return fmt.Errorf("failed to fetch remote profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("remote server returned status %d", resp.StatusCode)
	}

	outputPath := *output
	if outputPath == "" {
		timestamp := time.Now().Format("20060102_150405")
		outputPath = filepath.Join(*outputDir, fmt.Sprintf("takhin_%s_%s.prof", ptype, timestamp))
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write profile: %w", err)
	}

	fmt.Printf("\nRemote profile collected successfully!\n")
	fmt.Printf("  Type: %s\n", ptype)
	fmt.Printf("  Path: %s\n", outputPath)
	fmt.Printf("  Size: %.2f KB\n", float64(n)/1024)

	if *analyze {
		if err := analyzeProfile(outputPath, ptype); err != nil {
			return fmt.Errorf("failed to analyze profile: %w", err)
		}
	}

	if *flamegraph {
		fmt.Println("\nGenerating flame graph...")
		fmt.Printf("Run: go tool pprof -http=:8080 %s\n", outputPath)
	}

	return nil
}

func analyzeProfile(path string, ptype profiler.ProfileType) error {
	fmt.Println("\nAnalyzing profile...")

	analyzer := profiler.NewAnalyzer()
	stats, err := analyzer.Analyze(path, ptype)
	if err != nil {
		return err
	}

	report := analyzer.GenerateReport(stats, ptype)
	fmt.Println(report)

	return nil
}
