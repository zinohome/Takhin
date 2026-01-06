// Copyright 2025 Takhin Data, Inc.

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/takhin-data/takhin/pkg/config"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  "Commands for managing Takhin configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")

		var data []byte
		var err error

		switch format {
		case "json":
			data, err = json.MarshalIndent(cfg, "", "  ")
		case "yaml":
			data, err = yaml.Marshal(cfg)
		default:
			return fmt.Errorf("unsupported format: %s (use json or yaml)", format)
		}

		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		fmt.Println(string(data))
		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")

		if file == "" {
			return fmt.Errorf("--file flag is required")
		}

		_, err := config.Load(file)
		if err != nil {
			fmt.Printf("❌ Configuration validation failed: %v\n", err)
			return err
		}

		fmt.Printf("✅ Configuration file '%s' is valid\n", file)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		// Convert config to map for easy access
		data, err := json.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		var configMap map[string]interface{}
		if err := json.Unmarshal(data, &configMap); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}

		// Navigate nested keys (e.g., "server.port")
		value, err := getNestedValue(configMap, key)
		if err != nil {
			return err
		}

		// Pretty print the value
		output, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format value: %w", err)
		}

		fmt.Println(string(output))
		return nil
	},
}

var configExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export configuration to file",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFile, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")

		if outputFile == "" {
			return fmt.Errorf("--output flag is required")
		}

		var data []byte
		var err error

		switch format {
		case "json":
			data, err = json.MarshalIndent(cfg, "", "  ")
		case "yaml":
			data, err = yaml.Marshal(cfg)
		default:
			return fmt.Errorf("unsupported format: %s (use json or yaml)", format)
		}

		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		err = os.WriteFile(outputFile, data, 0644)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		fmt.Printf("Configuration exported to %s\n", outputFile)
		return nil
	},
}

func getNestedValue(m map[string]interface{}, key string) (interface{}, error) {
	// Split key by dots for nested access
	keys := splitKey(key)

	var current interface{} = m
	for _, k := range keys {
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("key '%s' not found or not a map", k)
		}

		current, ok = currentMap[k]
		if !ok {
			return nil, fmt.Errorf("key '%s' not found", key)
		}
	}

	return current, nil
}

func splitKey(key string) []string {
	var keys []string
	var current string

	for _, char := range key {
		if char == '.' {
			if current != "" {
				keys = append(keys, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		keys = append(keys, current)
	}

	return keys
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configExportCmd)

	// Flags for show command
	configShowCmd.Flags().StringP("format", "f", "json", "Output format (json or yaml)")

	// Flags for validate command
	configValidateCmd.Flags().String("file", "", "Configuration file to validate")

	// Flags for export command
	configExportCmd.Flags().StringP("output", "o", "", "Output file path")
	configExportCmd.Flags().StringP("format", "f", "yaml", "Output format (json or yaml)")
}
