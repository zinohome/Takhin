// Copyright 2025 Takhin Data, Inc.
// Example gRPC client for Takhin

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to Takhin gRPC server
	conn, err := grpc.NewClient(
		"localhost:9092",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	log.Println("Connected to Takhin gRPC server")
	log.Println("Note: Uncomment proto client code after running 'task backend:grpc:proto'")

	// Example workflow
	fmt.Println("\n=== gRPC Client Example ===")
	fmt.Println("1. Create topic")
	fmt.Println("2. Produce messages")
	fmt.Println("3. Consume messages")
	fmt.Println("4. List topics")
	fmt.Println("5. Health check")
	
	time.Sleep(1 * time.Second)
	log.Println("\nAll examples completed (stubs)")
}
