// Copyright 2025 Takhin Data, Inc.

package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

var topicCmd = &cobra.Command{
	Use:   "topic",
	Short: "Manage topics",
	Long:  "Commands for managing Takhin topics",
}

var topicListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all topics",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)

		topics := mgr.ListTopics()

		if len(topics) == 0 {
			fmt.Println("No topics found")
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Topic", "Partitions", "Replication Factor")

		for _, t := range topics {
			topicObj, exists := mgr.GetTopic(t)
			if !exists {
				continue
			}
			_ = table.Append(t,
				strconv.Itoa(len(topicObj.Partitions)),
				strconv.Itoa(int(topicObj.ReplicationFactor)),
			)
		}

		table.Render()
		return nil
	},
}

var topicCreateCmd = &cobra.Command{
	Use:   "create <topic-name>",
	Short: "Create a new topic",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		topicName := args[0]
		partitions, _ := cmd.Flags().GetInt("partitions")
		replicationFactor, _ := cmd.Flags().GetInt("replication-factor")

		mgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)

		err := mgr.CreateTopic(topicName, int32(partitions))
		if err != nil {
			return fmt.Errorf("failed to create topic: %w", err)
		}

		// Set replication factor if specified
		if replicationFactor > 0 {
			t, exists := mgr.GetTopic(topicName)
			if !exists {
				return fmt.Errorf("failed to get topic after creation")
			}
			t.SetReplicationFactor(int16(replicationFactor))
		}

		fmt.Printf("Topic '%s' created successfully with %d partitions\n", topicName, partitions)
		return nil
	},
}

var topicDeleteCmd = &cobra.Command{
	Use:   "delete <topic-name>",
	Short: "Delete a topic",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		topicName := args[0]
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("Are you sure you want to delete topic '%s'? (yes/no): ", topicName)
			var response string
			fmt.Scanln(&response)
			if response != "yes" {
				fmt.Println("Delete cancelled")
				return nil
			}
		}

		mgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)

		err := mgr.DeleteTopic(topicName)
		if err != nil {
			return fmt.Errorf("failed to delete topic: %w", err)
		}

		fmt.Printf("Topic '%s' deleted successfully\n", topicName)
		return nil
	},
}

var topicDescribeCmd = &cobra.Command{
	Use:   "describe <topic-name>",
	Short: "Describe a topic",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		topicName := args[0]

		mgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)

		t, exists := mgr.GetTopic(topicName)
		if !exists {
			return fmt.Errorf("topic not found: %s", topicName)
		}

		fmt.Printf("Topic: %s\n", t.Name)
		fmt.Printf("Partitions: %d\n", len(t.Partitions))
		fmt.Printf("Replication Factor: %d\n", t.ReplicationFactor)
		fmt.Println("\nPartition Details:")

		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Partition", "Size (bytes)", "Replicas", "ISR")

		for partID, partition := range t.Partitions {
			size, _ := partition.Size()
			replicas := t.GetReplicas(partID)
			isr := t.GetISR(partID)

			replicasStr := fmt.Sprintf("%v", replicas)
			if len(replicas) == 0 {
				replicasStr = "N/A"
			}
			isrStr := fmt.Sprintf("%v", isr)
			if len(isr) == 0 {
				isrStr = "N/A"
			}

			_ = table.Append(
				strconv.Itoa(int(partID)),
				strconv.FormatInt(size, 10),
				replicasStr,
				isrStr,
			)
		}

		table.Render()
		return nil
	},
}

var topicConfigCmd = &cobra.Command{
	Use:   "config <topic-name>",
	Short: "Get or set topic configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		topicName := args[0]
		setConfig, _ := cmd.Flags().GetStringToString("set")

		mgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)

		t, exists := mgr.GetTopic(topicName)
		if !exists {
			return fmt.Errorf("topic not found: %s", topicName)
		}

		if len(setConfig) > 0 {
			// Set configurations
			for key, value := range setConfig {
				switch key {
				case "replication.factor":
					rf, err := strconv.Atoi(value)
					if err != nil {
						return fmt.Errorf("invalid replication factor: %w", err)
					}
					t.SetReplicationFactor(int16(rf))
					fmt.Printf("Set replication.factor=%d\n", rf)
				case "replica.lag.max.ms":
					lagMs, err := strconv.ParseInt(value, 10, 64)
					if err != nil {
						return fmt.Errorf("invalid replica lag max ms: %w", err)
					}
					t.ReplicaLagMaxMs = lagMs
					fmt.Printf("Set replica.lag.max.ms=%d\n", lagMs)
				default:
					fmt.Printf("Warning: Unknown config key '%s'\n", key)
				}
			}
		} else {
			// Get configurations
			fmt.Printf("Topic: %s\n", topicName)
			fmt.Printf("replication.factor=%d\n", t.ReplicationFactor)
			fmt.Printf("replica.lag.max.ms=%d\n", t.ReplicaLagMaxMs)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(topicCmd)

	// Add subcommands
	topicCmd.AddCommand(topicListCmd)
	topicCmd.AddCommand(topicCreateCmd)
	topicCmd.AddCommand(topicDeleteCmd)
	topicCmd.AddCommand(topicDescribeCmd)
	topicCmd.AddCommand(topicConfigCmd)

	// Flags for create command
	topicCreateCmd.Flags().IntP("partitions", "p", 1, "Number of partitions")
	topicCreateCmd.Flags().IntP("replication-factor", "r", 1, "Replication factor")

	// Flags for delete command
	topicDeleteCmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")

	// Flags for config command
	topicConfigCmd.Flags().StringToStringP("set", "s", nil, "Set config key=value")
}
