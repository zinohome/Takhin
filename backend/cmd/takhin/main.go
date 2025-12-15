// Copyright 2025 Takhin Data, Inc.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/kafka/server"
	"github.com/takhin-data/takhin/pkg/logger"
	"github.com/takhin-data/takhin/pkg/metrics"
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

	// Start metrics server
	metricsServer := metrics.New(cfg)
	if err := metricsServer.Start(); err != nil {
		log.Fatal("failed to start metrics server", "error", err)
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

	if err := topicManager.Close(); err != nil {
		log.Error("failed to close topic manager", "error", err)
	}

	if err := metricsServer.Stop(); err != nil {
		log.Error("failed to stop metrics server", "error", err)
	}

	log.Info("Takhin stopped")
}
