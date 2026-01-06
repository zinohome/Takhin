// Copyright 2025 Takhin Data, Inc.

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "1.0.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Takhin CLI Version: %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
