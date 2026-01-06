// Copyright 2025 Takhin Data, Inc.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/debug"
	"github.com/takhin-data/takhin/pkg/logger"
)

var (
	configFile     = flag.String("config", "configs/takhin.yaml", "Path to configuration file")
	outputPath     = flag.String("output", "", "Output path for debug bundle (default: auto-generated)")
	includeLogs    = flag.Bool("logs", true, "Include log files")
	includeConfig  = flag.Bool("config-data", true, "Include configuration")
	includeMetrics = flag.Bool("metrics", true, "Include metrics")
	includeSystem  = flag.Bool("system", true, "Include system information")
	includeStorage = flag.Bool("storage", false, "Include storage information")
	logsMaxSize    = flag.Int64("logs-max-size", 100, "Maximum size of logs to collect (MB)")
	logsSinceHours = flag.Int("logs-since", 24, "Collect logs from last N hours")
	storageMaxSize = flag.Int64("storage-max-size", 50, "Maximum size of storage info to collect (MB)")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	log := logger.Default().WithComponent("takhin-debug")
	log.Info("starting debug bundle generation")

	cfg, err := config.Load(*configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	opts := &debug.BundleOptions{
		IncludeLogs:      *includeLogs,
		IncludeConfig:    *includeConfig,
		IncludeMetrics:   *includeMetrics,
		IncludeSystem:    *includeSystem,
		IncludeStorage:   *includeStorage,
		LogsMaxSizeMB:    *logsMaxSize,
		LogsSince:        time.Duration(*logsSinceHours) * time.Hour,
		StorageMaxSizeMB: *storageMaxSize,
		OutputPath:       *outputPath,
	}

	bundle := debug.NewBundle(cfg, log)

	ctx := context.Background()
	path, err := bundle.Generate(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to generate debug bundle: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat bundle file: %w", err)
	}

	fmt.Printf("Debug bundle generated successfully!\n")
	fmt.Printf("  Path: %s\n", path)
	fmt.Printf("  Size: %.2f MB\n", float64(info.Size())/(1024*1024))
	fmt.Printf("\nBundle contents:\n")
	if opts.IncludeSystem {
		fmt.Printf("  ✓ System information\n")
	}
	if opts.IncludeConfig {
		fmt.Printf("  ✓ Configuration (sanitized)\n")
	}
	if opts.IncludeLogs {
		fmt.Printf("  ✓ Log files (last %d hours, max %d MB)\n", *logsSinceHours, *logsMaxSize)
	}
	if opts.IncludeMetrics {
		fmt.Printf("  ✓ Metrics\n")
	}
	if opts.IncludeStorage {
		fmt.Printf("  ✓ Storage information (max %d MB)\n", *storageMaxSize)
	}

	return nil
}
