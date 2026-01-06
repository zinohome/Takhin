// Copyright 2025 Takhin Data, Inc.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

var dataCmd = &cobra.Command{
	Use:   "data",
	Short: "Import/Export data",
	Long:  "Commands for importing and exporting data",
}

var dataExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export topic data",
	RunE: func(cmd *cobra.Command, args []string) error {
		topicName, _ := cmd.Flags().GetString("topic")
		partition, _ := cmd.Flags().GetInt32("partition")
		outputFile, _ := cmd.Flags().GetString("output")
		maxMessages, _ := cmd.Flags().GetInt("max-messages")
		fromOffset, _ := cmd.Flags().GetInt64("from-offset")

		if topicName == "" {
			return fmt.Errorf("--topic flag is required")
		}

		mgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)

		t, exists := mgr.GetTopic(topicName)
		if !exists {
			return fmt.Errorf("topic not found: %s", topicName)
		}

		partitionLog, pexists := t.Partitions[partition]
		if !pexists {
			return fmt.Errorf("partition %d not found", partition)
		}

		var writer *bufio.Writer

		if outputFile != "" {
			file, err := os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer file.Close()
			writer = bufio.NewWriter(file)
			defer writer.Flush()
		} else {
			writer = bufio.NewWriter(os.Stdout)
			defer writer.Flush()
		}

		// Export messages
		offset := fromOffset
		exported := 0

		for {
			if maxMessages > 0 && exported >= maxMessages {
				break
			}

			record, err := partitionLog.Read(offset)
			if err != nil {
				// End of log reached
				break
			}

			if record == nil {
				break
			}

			msg := map[string]interface{}{
				"offset":    record.Offset,
				"timestamp": record.Timestamp,
				"key":       string(record.Key),
				"value":     string(record.Value),
			}

			data, err := json.Marshal(msg)
			if err != nil {
				return fmt.Errorf("failed to marshal message: %w", err)
			}

			writer.Write(data)
			writer.WriteString("\n")

			offset = record.Offset + 1
			exported++

			if maxMessages > 0 && exported >= maxMessages {
				break
			}
		}

		if outputFile != "" {
			fmt.Printf("Exported %d messages to %s\n", exported, outputFile)
		} else {
			fmt.Fprintf(os.Stderr, "Exported %d messages\n", exported)
		}

		return nil
	},
}

var dataImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import topic data",
	RunE: func(cmd *cobra.Command, args []string) error {
		topicName, _ := cmd.Flags().GetString("topic")
		partition, _ := cmd.Flags().GetInt32("partition")
		inputFile, _ := cmd.Flags().GetString("input")

		if topicName == "" {
			return fmt.Errorf("--topic flag is required")
		}

		if inputFile == "" {
			return fmt.Errorf("--input flag is required")
		}

		mgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)

		// Create topic if it doesn't exist
		t, exists := mgr.GetTopic(topicName)
		if !exists {
			// Topic doesn't exist, create it
			if err := mgr.CreateTopic(topicName, partition+1); err != nil {
				return fmt.Errorf("failed to create topic: %w", err)
			}
			t, exists = mgr.GetTopic(topicName)
			if !exists {
				return fmt.Errorf("failed to get topic after creation: %s", topicName)
			}
		}

		partitionLog, pexists := t.Partitions[partition]
		if !pexists {
			return fmt.Errorf("partition %d not found", partition)
		}

		// Open input file
		file, err := os.Open(inputFile)
		if err != nil {
			return fmt.Errorf("failed to open input file: %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		imported := 0

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var msg map[string]interface{}
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to parse line: %v\n", err)
				continue
			}

			// Extract fields
			key := ""
			if k, ok := msg["key"].(string); ok {
				key = k
			}

			value := ""
			if v, ok := msg["value"].(string); ok {
				value = v
			}

			// Append to log (timestamp is managed internally)
			if _, err := partitionLog.Append([]byte(key), []byte(value)); err != nil {
				return fmt.Errorf("failed to append record: %w", err)
			}

			imported++
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading file: %w", err)
		}

		fmt.Printf("Imported %d messages to topic '%s' partition %d\n", imported, topicName, partition)
		return nil
	},
}

var dataStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show data statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		topicFilter, _ := cmd.Flags().GetString("topic")

		mgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)

		topics := mgr.ListTopics()

		var totalSize int64
		var totalMessages int64

		for _, topicName := range topics {
			if topicFilter != "" && topicName != topicFilter {
				continue
			}

			t, exists := mgr.GetTopic(topicName)
			if !exists {
				continue
			}

			fmt.Printf("\nTopic: %s\n", topicName)
			fmt.Printf("Partitions: %d\n", len(t.Partitions))

			var topicSize int64
			var topicMessages int64

			for partID, partition := range t.Partitions {
				size, err := partition.Size()
				if err != nil {
					continue
				}
				leo := partition.HighWaterMark()

				fmt.Printf("  Partition %d: Size=%d bytes, Messages=%d\n", partID, size, leo)

				topicSize += size
				topicMessages += leo
			}

			fmt.Printf("Total Size: %d bytes (%.2f MB)\n", topicSize, float64(topicSize)/(1024*1024))
			fmt.Printf("Total Messages: %d\n", topicMessages)

			totalSize += topicSize
			totalMessages += topicMessages
		}

		if topicFilter == "" && len(topics) > 1 {
			fmt.Printf("\n=== Overall Statistics ===\n")
			fmt.Printf("Total Topics: %d\n", len(topics))
			fmt.Printf("Total Size: %d bytes (%.2f MB)\n", totalSize, float64(totalSize)/(1024*1024))
			fmt.Printf("Total Messages: %d\n", totalMessages)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dataCmd)

	// Add subcommands
	dataCmd.AddCommand(dataExportCmd)
	dataCmd.AddCommand(dataImportCmd)
	dataCmd.AddCommand(dataStatsCmd)

	// Flags for export command
	dataExportCmd.Flags().StringP("topic", "t", "", "Topic name (required)")
	dataExportCmd.Flags().Int32P("partition", "p", 0, "Partition ID")
	dataExportCmd.Flags().StringP("output", "o", "", "Output file path (defaults to stdout)")
	dataExportCmd.Flags().IntP("max-messages", "m", 0, "Maximum number of messages to export (0=all)")
	dataExportCmd.Flags().Int64("from-offset", 0, "Starting offset")

	// Flags for import command
	dataImportCmd.Flags().StringP("topic", "t", "", "Topic name (required)")
	dataImportCmd.Flags().Int32P("partition", "p", 0, "Partition ID")
	dataImportCmd.Flags().StringP("input", "i", "", "Input file path (required)")

	// Flags for stats command
	dataStatsCmd.Flags().StringP("topic", "t", "", "Filter by topic name (optional)")
}
