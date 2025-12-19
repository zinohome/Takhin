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

func TestInitProducerID_NonTransactional(t *testing.T) {
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

	// Request without transactional ID (non-transactional producer)
	req := &protocol.InitProducerIDRequest{
		TransactionalID:      nil,
		TransactionTimeoutMs: 60000,
		ProducerID:           -1,
		ProducerEpoch:        -1,
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.InitProducerIDKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	reqData, err := protocol.EncodeInitProducerIDRequest(req, header.APIVersion)
	require.NoError(t, err)

	// Handle request
	reader := bytes.NewReader(reqData)
	respData, err := handler.handleInitProducerID(reader, header)
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
	resp, err := protocol.DecodeInitProducerIDResponse(respBody, header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, protocol.None, resp.ErrorCode)
	assert.Greater(t, resp.ProducerID, int64(0))
	assert.Equal(t, int16(0), resp.ProducerEpoch)
}

func TestInitProducerID_Transactional(t *testing.T) {
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

	// Request with transactional ID
	txnID := "test-txn-id"
	req := &protocol.InitProducerIDRequest{
		TransactionalID:      &txnID,
		TransactionTimeoutMs: 60000,
		ProducerID:           -1,
		ProducerEpoch:        -1,
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.InitProducerIDKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	reqData, err := protocol.EncodeInitProducerIDRequest(req, header.APIVersion)
	require.NoError(t, err)

	// First request - should allocate new ID
	reader := bytes.NewReader(reqData)
	respData, err := handler.handleInitProducerID(reader, header)
	assert.NoError(t, err)

	// Skip response header
	respReader := bytes.NewReader(respData)
	var correlationID int32
	binary.Read(respReader, binary.BigEndian, &correlationID)

	// Read remaining response body
	respBody, err := io.ReadAll(respReader)
	assert.NoError(t, err)

	// Decode response
	resp, err := protocol.DecodeInitProducerIDResponse(respBody, header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, protocol.None, resp.ErrorCode)
	assert.Greater(t, resp.ProducerID, int64(0))
	assert.Equal(t, int16(0), resp.ProducerEpoch)

	firstProducerID := resp.ProducerID
	firstEpoch := resp.ProducerEpoch

	// Second request with same transactional ID - should return same producer ID but incremented epoch
	reqData2, err := protocol.EncodeInitProducerIDRequest(req, header.APIVersion)
	require.NoError(t, err)

	reader2 := bytes.NewReader(reqData2)
	respData2, err := handler.handleInitProducerID(reader2, header)
	assert.NoError(t, err)

	// Skip response header
	respReader2 := bytes.NewReader(respData2)
	binary.Read(respReader2, binary.BigEndian, &correlationID)

	// Read remaining response body
	respBody2, err := io.ReadAll(respReader2)
	assert.NoError(t, err)

	// Decode response
	resp2, err := protocol.DecodeInitProducerIDResponse(respBody2, header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, protocol.None, resp2.ErrorCode)
	assert.Equal(t, firstProducerID, resp2.ProducerID) // Same producer ID
	assert.Equal(t, firstEpoch+1, resp2.ProducerEpoch) // Incremented epoch
}

func TestInitProducerID_MultipleProducers(t *testing.T) {
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

	// Create multiple producers with different transactional IDs
	txnIDs := []string{"producer-1", "producer-2", "producer-3"}
	producerIDs := make([]int64, len(txnIDs))

	for i, txnID := range txnIDs {
		txn := txnID
		req := &protocol.InitProducerIDRequest{
			TransactionalID:      &txn,
			TransactionTimeoutMs: 60000,
			ProducerID:           -1,
			ProducerEpoch:        -1,
		}

		header := &protocol.RequestHeader{
			APIKey:        protocol.InitProducerIDKey,
			APIVersion:    0,
			CorrelationID: int32(i + 1),
			ClientID:      "test-client",
		}

		reqData, err := protocol.EncodeInitProducerIDRequest(req, header.APIVersion)
		require.NoError(t, err)

		reader := bytes.NewReader(reqData)
		respData, err := handler.handleInitProducerID(reader, header)
		assert.NoError(t, err)

		// Skip response header
		respReader := bytes.NewReader(respData)
		var correlationID int32
		binary.Read(respReader, binary.BigEndian, &correlationID)

		// Read remaining response body
		respBody, err := io.ReadAll(respReader)
		assert.NoError(t, err)

		// Decode response
		resp, err := protocol.DecodeInitProducerIDResponse(respBody, header.APIVersion)
		assert.NoError(t, err)
		assert.Equal(t, protocol.None, resp.ErrorCode)
		assert.Greater(t, resp.ProducerID, int64(0))

		producerIDs[i] = resp.ProducerID
	}

	// Verify all producer IDs are different
	assert.NotEqual(t, producerIDs[0], producerIDs[1])
	assert.NotEqual(t, producerIDs[1], producerIDs[2])
	assert.NotEqual(t, producerIDs[0], producerIDs[2])
}
