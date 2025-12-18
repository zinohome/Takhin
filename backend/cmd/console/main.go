// Copyright 2025 Takhin Data, Inc.

// @title           Takhin Console API
// @version         1.0
// @description     HTTP REST API for Takhin Console - manage topics, messages, and consumer groups
// @termsOfService  https://takhin.io/terms

// @contact.name   Takhin Support
// @contact.url    https://takhin.io/support
// @contact.email  support@takhin.io

// @license.name  Business Source License 1.1
// @license.url   https://github.com/redpanda-data/redpanda/blob/dev/licenses/bsl.md

// @host      localhost:8080
// @BasePath  /api

// @schemes http https

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description API Key authentication. Use 'your-api-key' or 'Bearer your-api-key' format.

// @tag.name Topics
// @tag.description Topic management operations

// @tag.name Messages
// @tag.description Message produce and consume operations

// @tag.name Consumer Groups
// @tag.description Consumer group monitoring operations

// @tag.name Health
// @tag.description Health check endpoints

package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/takhin-data/takhin/pkg/console"
	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/logger"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func main() {
	// Parse command-line flags
	dataDir := flag.String("data-dir", "/tmp/takhin-console-data", "Data directory for topics")
	apiAddr := flag.String("api-addr", ":8080", "Console API server address")
	enableAuth := flag.Bool("enable-auth", false, "Enable API key authentication")
	apiKeys := flag.String("api-keys", "", "Comma-separated list of valid API keys")
	flag.Parse()

	// Initialize logger
	log := logger.Default().WithComponent("console-main")

	log.Info("starting Takhin Console",
		"data_dir", *dataDir,
		"api_addr", *apiAddr,
		"auth_enabled", *enableAuth,
	)

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatal("failed to create data directory", "error", err)
	}

	// Create topic manager
	topicManager := topic.NewManager(*dataDir, 1024*1024*100) // 100MB segments
	defer topicManager.Close()

	// Create and start coordinator
	coord := coordinator.NewCoordinator()
	coord.Start()

	// Configure authentication
	authConfig := console.AuthConfig{
		Enabled: *enableAuth,
		APIKeys: parseAPIKeys(*apiKeys),
	}

	// Create and start Console API server
	server := console.NewServer(*apiAddr, topicManager, coord, authConfig)

	// Handle shutdown gracefully
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Info("shutting down...")
		os.Exit(0)
	}()

	// Start server
	log.Info("console API server ready", "addr", *apiAddr)
	if err := server.Start(); err != nil {
		log.Fatal("server error", "error", err)
	}
}

// parseAPIKeys splits comma-separated API keys and filters empty strings
func parseAPIKeys(keys string) []string {
	if keys == "" {
		return []string{}
	}

	parts := strings.Split(keys, ",")
	result := make([]string, 0, len(parts))
	for _, key := range parts {
		trimmed := strings.TrimSpace(key)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
