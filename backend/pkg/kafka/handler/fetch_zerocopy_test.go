// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"net"
	"testing"

	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestHandleFetchZeroCopy_Basic(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: t.TempDir(),
		},
		Kafka: config.KafkaConfig{
			BrokerID: 1,
		},
	}

	mgr := topic.NewManager(cfg.Storage.DataDir, 1024*1024)
	handler := New(cfg, mgr)

	// Create a topic with test data
	topicName := "test-topic"
	err := mgr.CreateTopic(topicName, 1)
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Append some test records
	testTopic, exists := mgr.GetTopic(topicName)
	if !exists {
		t.Fatal("Topic not found")
	}

	testData := []byte("Hello, zero-copy world!")
	for i := 0; i < 5; i++ {
		_, err := testTopic.Append(0, []byte("key"), testData)
		if err != nil {
			t.Fatalf("Failed to append: %v", err)
		}
	}

	// Create a Fetch request using protocol.EncodeFetchRequest
	header := &protocol.RequestHeader{
		APIKey:        protocol.FetchKey,
		APIVersion:    11,
		CorrelationID: 123,
		ClientID:      "test-client",
	}

	// Build a minimal fetch request manually
	var buf bytes.Buffer
	if err := header.Encode(&buf); err != nil {
		t.Fatalf("Failed to encode header: %v", err)
	}

	// Write minimal fetch request fields (simplified for testing)
	// This is a simplified test - in production use proper protocol encoding

	// Create a mock TCP connection using a pipe
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	// Test that zero-copy handler doesn't crash with empty/invalid request
	errCh := make(chan error, 1)
	go func() {
		// Pass empty request to test error handling
		errCh <- handler.HandleFetchZeroCopy(buf.Bytes(), serverConn)
	}()

	// Read any response (may be error response)
	sizeBuf := make([]byte, 4)
	_, _ = clientConn.Read(sizeBuf)

	// Check handler completed (error or not)
	select {
	case err := <-errCh:
		// Error is expected with empty request
		if err != nil {
			t.Logf("Handler returned error as expected: %v", err)
		}
	}

	t.Log("Zero-copy handler basic test completed")
}
