// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestAddPartitionsToTxn_Success(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 9092,
		},
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			LogSegmentSize: 1024 * 1024,
		},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	handler := New(cfg, topicMgr)
	defer topicMgr.Close()

	// Create a test topic
	err := topicMgr.CreateTopic("test-topic", 3)
	require.NoError(t, err)

	// Request to add partitions to transaction
	req := &protocol.AddPartitionsToTxnRequest{
		TransactionalID: "test-txn",
		ProducerID:      1001,
		ProducerEpoch:   0,
		Topics: []protocol.AddPartitionsToTxnTopic{
			{
				Name:       "test-topic",
				Partitions: []int32{0, 1, 2},
			},
		},
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.AddPartitionsToTxnKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	reqData, err := protocol.EncodeAddPartitionsToTxnRequest(req, header.APIVersion)
	require.NoError(t, err)

	// Handle request
	reader := bytes.NewReader(reqData)
	respData, err := handler.handleAddPartitionsToTxn(reader, header)
	assert.NoError(t, err)
	assert.NotNil(t, respData)

	// Skip response header
	respReader := bytes.NewReader(respData)
	var correlationID int32
	binary.Read(respReader, binary.BigEndian, &correlationID)

	// Read remaining response body
	respBody, err := io.ReadAll(respReader)
	assert.NoError(t, err)

	// Decode and validate response
	resp, err := protocol.DecodeAddPartitionsToTxnResponse(bytes.NewReader(respBody), header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), resp.ThrottleTimeMs)
	assert.Len(t, resp.Results, 1)
	assert.Equal(t, "test-topic", resp.Results[0].Name)
	assert.Len(t, resp.Results[0].PartitionResults, 3)

	// Verify all partitions succeeded
	for i, result := range resp.Results[0].PartitionResults {
		assert.Equal(t, int32(i), result.PartitionIndex)
		assert.Equal(t, protocol.None, result.ErrorCode)
	}
}

func TestAddPartitionsToTxn_MultipleTopics(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 9092,
		},
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			LogSegmentSize: 1024 * 1024,
		},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	handler := New(cfg, topicMgr)
	defer topicMgr.Close()

	// Create test topics
	err := topicMgr.CreateTopic("topic1", 2)
	require.NoError(t, err)
	err = topicMgr.CreateTopic("topic2", 2)
	require.NoError(t, err)

	// Request to add partitions from multiple topics
	req := &protocol.AddPartitionsToTxnRequest{
		TransactionalID: "test-txn",
		ProducerID:      1001,
		ProducerEpoch:   0,
		Topics: []protocol.AddPartitionsToTxnTopic{
			{
				Name:       "topic1",
				Partitions: []int32{0, 1},
			},
			{
				Name:       "topic2",
				Partitions: []int32{0, 1},
			},
		},
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.AddPartitionsToTxnKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	reqData, err := protocol.EncodeAddPartitionsToTxnRequest(req, header.APIVersion)
	require.NoError(t, err)

	// Handle request
	reader := bytes.NewReader(reqData)
	respData, err := handler.handleAddPartitionsToTxn(reader, header)
	assert.NoError(t, err)

	// Skip response header
	respReader := bytes.NewReader(respData)
	var correlationID int32
	binary.Read(respReader, binary.BigEndian, &correlationID)

	// Read remaining response body
	respBody, err := io.ReadAll(respReader)
	assert.NoError(t, err)

	// Decode and validate response
	resp, err := protocol.DecodeAddPartitionsToTxnResponse(bytes.NewReader(respBody), header.APIVersion)
	assert.NoError(t, err)
	assert.Len(t, resp.Results, 2)

	// Verify both topics succeeded
	for _, result := range resp.Results {
		assert.Len(t, result.PartitionResults, 2)
		for _, partResult := range result.PartitionResults {
			assert.Equal(t, protocol.None, partResult.ErrorCode)
		}
	}
}

func TestAddPartitionsToTxn_ProducerIDMismatch(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 9092,
		},
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			LogSegmentSize: 1024 * 1024,
		},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	handler := New(cfg, topicMgr)
	defer topicMgr.Close()

	// Create a test topic
	err := topicMgr.CreateTopic("test-topic", 2)
	require.NoError(t, err)

	// First request to establish producer ID
	req1 := &protocol.AddPartitionsToTxnRequest{
		TransactionalID: "test-txn",
		ProducerID:      1001,
		ProducerEpoch:   0,
		Topics: []protocol.AddPartitionsToTxnTopic{
			{
				Name:       "test-topic",
				Partitions: []int32{0},
			},
		},
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.AddPartitionsToTxnKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	reqData1, err := protocol.EncodeAddPartitionsToTxnRequest(req1, header.APIVersion)
	require.NoError(t, err)

	reader1 := bytes.NewReader(reqData1)
	_, err = handler.handleAddPartitionsToTxn(reader1, header)
	assert.NoError(t, err)

	// Second request with different producer ID (should fail)
	req2 := &protocol.AddPartitionsToTxnRequest{
		TransactionalID: "test-txn",
		ProducerID:      1002, // Different producer ID
		ProducerEpoch:   0,
		Topics: []protocol.AddPartitionsToTxnTopic{
			{
				Name:       "test-topic",
				Partitions: []int32{1},
			},
		},
	}

	reqData2, err := protocol.EncodeAddPartitionsToTxnRequest(req2, header.APIVersion)
	require.NoError(t, err)

	reader2 := bytes.NewReader(reqData2)
	respData2, err := handler.handleAddPartitionsToTxn(reader2, header)
	assert.NoError(t, err)

	// Skip response header
	respReader := bytes.NewReader(respData2)
	var correlationID int32
	binary.Read(respReader, binary.BigEndian, &correlationID)

	// Read remaining response body
	respBody, err := io.ReadAll(respReader)
	assert.NoError(t, err)

	// Decode and validate response
	resp, err := protocol.DecodeAddPartitionsToTxnResponse(bytes.NewReader(respBody), header.APIVersion)
	assert.NoError(t, err)
	assert.Len(t, resp.Results, 1)
	assert.Len(t, resp.Results[0].PartitionResults, 1)

	// Verify error code is InvalidProducerIDMapping
	assert.Equal(t, protocol.InvalidProducerIDMapping, resp.Results[0].PartitionResults[0].ErrorCode)
}
