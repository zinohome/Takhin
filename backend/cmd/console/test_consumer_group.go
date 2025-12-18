// Copyright 2025 Takhin Data, Inc.
//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"net/http"

	"github.com/takhin-data/takhin/pkg/coordinator"
)

// Simple test to create consumer group via HTTP
func main() {
	// Create coordinator
	coord := coordinator.NewCoordinator()

	// Create test group
	group := coord.GetOrCreateGroup("demo-consumer-group", "consumer")

	// Add member
	member := &coordinator.Member{
		ID:               "consumer-1",
		ClientID:         "test-client",
		ClientHost:       "localhost",
		SessionTimeout:   30000,
		RebalanceTimeout: 60000,
		ProtocolType:     "consumer",
	}
	group.AddMember(member)

	// Commit some offsets
	group.CommitOffset("demo-topic", 0, 150, "metadata-1")
	group.CommitOffset("demo-topic", 1, 200, "metadata-2")
	group.CommitOffset("demo-topic", 2, 175, "metadata-3")

	fmt.Println("Consumer group created successfully!")
	fmt.Println("Group ID:", group.ID)
	fmt.Println("State:", group.State)
	fmt.Println("Members:", len(group.Members))

	// Test HTTP endpoints
	resp, err := http.Get("http://localhost:18080/api/consumer-groups")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("\nHTTP Response Status:", resp.Status)
}
