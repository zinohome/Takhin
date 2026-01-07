// Copyright 2025 Takhin Data, Inc.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/health"
	"github.com/takhin-data/takhin/pkg/kafka/server"
	"github.com/takhin-data/takhin/pkg/logger"
	"github.com/takhin-data/takhin/pkg/metrics"
	"github.com/takhin-data/takhin/pkg/profiler"
	storagelog "github.com/takhin-data/takhin/pkg/storage/log"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	configPath := flag.String("config", "configs/takhin.yaml", "path to configuration file")
	showVersion := flag.Bool("version", false, "show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Takhin version %s (commit: %s, built: %s)\n", version, commit, buildTime)
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})
	logger.SetDefault(log)

	log.Info("starting Takhin",
		"version", version,
		"commit", commit,
		"build_time", buildTime,
	)

	log.Info("loaded configuration",
		"broker_id", cfg.Kafka.BrokerID,
		"data_dir", cfg.Storage.DataDir,
		"log_level", cfg.Logging.Level,
	)

	// Create topic manager
	topicManager := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	log.Info("initialized topic manager")

	// Initialize and start background cleaner if enabled
	var cleaner *storagelog.Cleaner
	if cfg.Storage.CleanerEnabled {
		cleanerConfig := storagelog.CleanerConfig{
			CleanupIntervalSeconds:    cfg.Storage.LogCleanupInterval / 1000, // Convert ms to seconds
			CompactionIntervalSeconds: cfg.Storage.CompactionInterval / 1000,
			RetentionPolicy: storagelog.RetentionPolicy{
				RetentionBytes: cfg.Storage.LogRetentionBytes,
				RetentionMs:    int64(cfg.Storage.LogRetentionHours) * 3600 * 1000,
			},
			CompactionPolicy: storagelog.CompactionPolicy{
				MinCleanableRatio:  cfg.Storage.MinCleanableRatio,
				MinCompactionLagMs: 0,
				DeleteRetentionMs:  24 * 60 * 60 * 1000, // 24 hours
			},
			Enabled: true,
		}
		cleaner = storagelog.NewCleaner(cleanerConfig)
		topicManager.SetCleaner(cleaner)

		if err := cleaner.Start(); err != nil {
			log.Fatal("failed to start background cleaner", "error", err)
		}
		log.Info("started background cleaner",
			"cleanup_interval_sec", cleanerConfig.CleanupIntervalSeconds,
			"compaction_interval_sec", cleanerConfig.CompactionIntervalSeconds)
	} else {
		log.Info("background cleaner is disabled")
	}

	// Start metrics server
	metricsServer := metrics.New(cfg)
	if err := metricsServer.Start(); err != nil {
		log.Fatal("failed to start metrics server", "error", err)
	}

	// Start profiler server
	profilerServer := profiler.NewServer(cfg)
	if err := profilerServer.Start(); err != nil {
		log.Fatal("failed to start profiler server", "error", err)
	}

	// Start health check server
	var healthServer *health.Server
	if cfg.Health.Enabled {
		healthChecker := health.NewChecker(version, topicManager)
		healthAddr := fmt.Sprintf("%s:%d", cfg.Health.Host, cfg.Health.Port)
		healthServer = health.NewServer(healthAddr, healthChecker)
		if err := healthServer.Start(); err != nil {
			log.Fatal("failed to start health check server", "error", err)
		}
		log.Info("started health check server", "port", cfg.Health.Port)
	}

	// Start Kafka server
	kafkaServer := server.New(cfg, topicManager)
	if err := kafkaServer.Start(); err != nil {
		log.Fatal("failed to start kafka server", "error", err)
	}

	log.Info("Takhin started successfully",
		"port", cfg.Server.Port,
		"metrics_port", cfg.Metrics.Port,
	)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Info("shutting down Takhin")

	// Graceful shutdown
	kafkaServer.Stop()

	// Stop health check server
	if healthServer != nil {
		if err := healthServer.Stop(); err != nil {
			log.Error("failed to stop health check server", "error", err)
		}
	}

	// Stop cleaner if running
	if cleaner != nil {
		if err := cleaner.Stop(); err != nil {
			log.Error("failed to stop cleaner", "error", err)
		}
	}

	if err := topicManager.Close(); err != nil {
		log.Error("failed to close topic manager", "error", err)
	}

	if err := metricsServer.Stop(); err != nil {
		log.Error("failed to stop metrics server", "error", err)
	}

	if err := profilerServer.Stop(); err != nil {
		log.Error("failed to stop profiler server", "error", err)
	}

	log.Info("Takhin stopped")
}
