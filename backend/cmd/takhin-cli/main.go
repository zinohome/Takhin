// Copyright 2025 Takhin Data, Inc.

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/takhin-data/takhin/pkg/config"
)

var (
	dataDir    string
	configFile string
	cfg        *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "takhin-cli",
	Short: "Takhin CLI - Command line tool for managing Takhin",
	Long: `Takhin CLI is a command line management tool for Takhin message broker.
It provides commands for managing topics, consumer groups, configurations, and data.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for commands that don't need it
		if cmd.Name() == "help" || cmd.Name() == "version" {
			return nil
		}

		// Load configuration if config file is provided
		if configFile != "" {
			var err error
			cfg, err = config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			// Override data directory if specified
			if dataDir != "" {
				cfg.Storage.DataDir = dataDir
			}
		} else if dataDir == "" {
			return fmt.Errorf("either --data-dir or --config must be specified")
		} else {
			// Create minimal config with just data directory
			cfg = &config.Config{
				Storage: config.StorageConfig{
					DataDir: dataDir,
				},
			}
			// Set defaults by loading empty config
			tempCfg, err := config.Load("")
			if err == nil {
				cfg.Storage.LogSegmentSize = tempCfg.Storage.LogSegmentSize
			} else {
				cfg.Storage.LogSegmentSize = 1024 * 1024 * 1024 // 1GB default
			}
		}

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&dataDir, "data-dir", "d", "", "Data directory path")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Config file path")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
