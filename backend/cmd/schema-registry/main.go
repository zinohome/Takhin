// Copyright 2025 Takhin Data, Inc.

package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/takhin-data/takhin/pkg/schema"
)

var (
	dataDir              = flag.String("data-dir", "/tmp/takhin-schema-registry", "Data directory for schema storage")
	addr                 = flag.String("addr", ":8081", "HTTP server address")
	defaultCompatibility = flag.String("default-compatibility", "BACKWARD", "Default compatibility mode")
	maxVersions          = flag.Int("max-versions", 100, "Maximum number of schema versions per subject")
	cacheSize            = flag.Int("cache-size", 1000, "Schema cache size")
	logLevel             = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
)

func main() {
	flag.Parse()

	setupLogger(*logLevel)

	cfg := &schema.Config{
		DataDir:              *dataDir,
		DefaultCompatibility: schema.CompatibilityMode(*defaultCompatibility),
		MaxVersions:          *maxVersions,
		CacheSize:            *cacheSize,
	}

	slog.Info("Starting Takhin Schema Registry",
		"addr", *addr,
		"data_dir", cfg.DataDir,
		"default_compatibility", cfg.DefaultCompatibility,
	)

	server, err := schema.NewServer(cfg, *addr)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			errChan <- err
		}
	}()

	slog.Info("Schema Registry started successfully", "addr", *addr)

	select {
	case err := <-errChan:
		slog.Error("Server error", "error", err)
		os.Exit(1)
	case sig := <-sigChan:
		slog.Info("Received signal, shutting down", "signal", sig)
		if err := server.Close(); err != nil {
			slog.Error("Error closing server", "error", err)
		}
	}
}

func setupLogger(level string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nExample:\n")
	fmt.Fprintf(os.Stderr, "  %s -addr :8081 -data-dir /var/lib/schema-registry\n", os.Args[0])
}
