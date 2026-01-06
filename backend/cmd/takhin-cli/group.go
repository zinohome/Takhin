// Copyright 2025 Takhin Data, Inc.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/takhin-data/takhin/pkg/coordinator"
)

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage consumer groups",
	Long:  "Commands for managing consumer groups",
}

var groupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all consumer groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		coord := coordinator.NewCoordinator()

		groups := coord.ListGroups()

		if len(groups) == 0 {
			fmt.Println("No consumer groups found")
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Group ID", "Protocol Type", "State", "Members")

		for _, groupID := range groups {
			group, exists := coord.GetGroup(groupID)
			if !exists {
				continue
			}

			_ = table.Append(groupID,
				group.ProtocolType,
				string(group.State),
				strconv.Itoa(len(group.Members)),
			)
		}

		table.Render()
		return nil
	},
}

var groupDescribeCmd = &cobra.Command{
	Use:   "describe <group-id>",
	Short: "Describe a consumer group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID := args[0]

		coord := coordinator.NewCoordinator()
		group, exists := coord.GetGroup(groupID)
		if !exists {
			return fmt.Errorf("consumer group '%s' not found", groupID)
		}

		fmt.Printf("Group ID: %s\n", group.ID)
		fmt.Printf("Protocol Type: %s\n", group.ProtocolType)
		fmt.Printf("State: %s\n", string(group.State))
		fmt.Printf("Protocol: %s\n", group.ProtocolName)
		fmt.Printf("Leader: %s\n", group.Leader)
		fmt.Printf("Generation ID: %d\n", group.Generation)
		fmt.Printf("Members: %d\n", len(group.Members))

		if len(group.Members) > 0 {
			fmt.Println("\nMembers:")
			table := tablewriter.NewWriter(os.Stdout)
			table.Header("Member ID", "Client ID", "Host", "Session Timeout", "Last Heartbeat")

			for _, member := range group.Members {
				lastHB := time.Since(member.LastHeartbeat).Round(time.Second).String()
				_ = table.Append(member.ID,
					member.ClientID,
					member.ClientHost,
					fmt.Sprintf("%dms", member.SessionTimeout),
					lastHB + " ago",
				)
			}

			table.Render()
		}

		return nil
	},
}

var groupDeleteCmd = &cobra.Command{
	Use:   "delete <group-id>",
	Short: "Delete a consumer group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID := args[0]
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("Are you sure you want to delete consumer group '%s'? (yes/no): ", groupID)
			var response string
			fmt.Scanln(&response)
			if response != "yes" {
				fmt.Println("Delete cancelled")
				return nil
			}
		}

		coord := coordinator.NewCoordinator()
		coord.DeleteGroup(groupID)

		fmt.Printf("Consumer group '%s' deleted successfully\n", groupID)
		return nil
	},
}

var groupResetCmd = &cobra.Command{
	Use:   "reset <group-id>",
	Short: "Reset consumer group offsets",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupID := args[0]
		topic, _ := cmd.Flags().GetString("topic")
		partition, _ := cmd.Flags().GetInt32("partition")
		offset, _ := cmd.Flags().GetInt64("offset")
		toEarliest, _ := cmd.Flags().GetBool("to-earliest")
		toLatest, _ := cmd.Flags().GetBool("to-latest")

		if topic == "" {
			return fmt.Errorf("--topic flag is required")
		}

		coord := coordinator.NewCoordinator()
		group, exists := coord.GetGroup(groupID)
		if !exists {
			return fmt.Errorf("consumer group '%s' not found", groupID)
		}

		// Determine target offset
		var targetOffset int64
		if toEarliest {
			targetOffset = 0
		} else if toLatest {
			targetOffset = -1 // Special marker for latest
		} else {
			targetOffset = offset
		}

		// Reset offset
		if err := group.ResetOffset(topic, partition, targetOffset); err != nil {
			return fmt.Errorf("failed to reset offset: %w", err)
		}

		fmt.Printf("Reset offset for group '%s', topic '%s', partition %d to %d\n",
			groupID, topic, partition, targetOffset)
		return nil
	},
}

var groupExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export consumer group offsets",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFile, _ := cmd.Flags().GetString("output")
		groupID, _ := cmd.Flags().GetString("group")

		coord := coordinator.NewCoordinator()

		var groups []string
		if groupID != "" {
			groups = []string{groupID}
		} else {
			groups = coord.ListGroups()
		}

		exportData := make(map[string]interface{})

		for _, gid := range groups {
			group, exists := coord.GetGroup(gid)
			if !exists {
				continue
			}

			groupData := map[string]interface{}{
				"protocol_type": group.ProtocolType,
				"state":         string(group.State),
				"generation":    group.Generation,
				"leader":        group.Leader,
				"offsets":       group.OffsetCommits,
			}

			exportData[gid] = groupData
		}

		data, err := json.MarshalIndent(exportData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}

		if outputFile != "" {
			err = os.WriteFile(outputFile, data, 0644)
			if err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
			fmt.Printf("Exported consumer groups to %s\n", outputFile)
		} else {
			fmt.Println(string(data))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(groupCmd)

	// Add subcommands
	groupCmd.AddCommand(groupListCmd)
	groupCmd.AddCommand(groupDescribeCmd)
	groupCmd.AddCommand(groupDeleteCmd)
	groupCmd.AddCommand(groupResetCmd)
	groupCmd.AddCommand(groupExportCmd)

	// Flags for delete command
	groupDeleteCmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")

	// Flags for reset command
	groupResetCmd.Flags().StringP("topic", "t", "", "Topic name (required)")
	groupResetCmd.Flags().Int32P("partition", "p", 0, "Partition ID")
	groupResetCmd.Flags().Int64P("offset", "o", 0, "Target offset")
	groupResetCmd.Flags().Bool("to-earliest", false, "Reset to earliest offset")
	groupResetCmd.Flags().Bool("to-latest", false, "Reset to latest offset")

	// Flags for export command
	groupExportCmd.Flags().StringP("output", "o", "", "Output file path (defaults to stdout)")
	groupExportCmd.Flags().StringP("group", "g", "", "Specific group ID to export (defaults to all)")
}
