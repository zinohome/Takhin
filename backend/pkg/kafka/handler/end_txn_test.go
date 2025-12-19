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

func TestEndTxn_Commit(t *testing.T) {
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

	// First add partitions to transaction
	addReq := &protocol.AddPartitionsToTxnRequest{
		TransactionalID: "test-txn",
		ProducerID:      1001,
		ProducerEpoch:   0,
		Topics: []protocol.AddPartitionsToTxnTopic{
			{
				Name:       "test-topic",
				Partitions: []int32{0, 1},
			},
		},
	}

	addHeader := &protocol.RequestHeader{
		APIKey:        protocol.AddPartitionsToTxnKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	addReqData, err := protocol.EncodeAddPartitionsToTxnRequest(addReq, addHeader.APIVersion)
	require.NoError(t, err)

	reader := bytes.NewReader(addReqData)
	_, err = handler.handleAddPartitionsToTxn(reader, addHeader)
	assert.NoError(t, err)

	// Now end transaction with commit
	req := &protocol.EndTxnRequest{
		TransactionalID: "test-txn",
		ProducerID:      1001,
		ProducerEpoch:   0,
		Committed:       true,
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.EndTxnKey,
		APIVersion:    0,
		CorrelationID: 2,
		ClientID:      "test-client",
	}

	reqData, err := protocol.EncodeEndTxnRequest(req, header.APIVersion)
	require.NoError(t, err)

	// Handle request
	reader = bytes.NewReader(reqData)
	respData, err := handler.handleEndTxn(reader, header)
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
	resp, err := protocol.DecodeEndTxnResponse(bytes.NewReader(respBody), header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), resp.ThrottleTimeMs)
	assert.Equal(t, protocol.None, resp.ErrorCode)
}

func TestEndTxn_Abort(t *testing.T) {
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

	// First add partitions to transaction
	addReq := &protocol.AddPartitionsToTxnRequest{
		TransactionalID: "test-txn",
		ProducerID:      1001,
		ProducerEpoch:   0,
		Topics: []protocol.AddPartitionsToTxnTopic{
			{
				Name:       "test-topic",
				Partitions: []int32{0, 1},
			},
		},
	}

	addHeader := &protocol.RequestHeader{
		APIKey:        protocol.AddPartitionsToTxnKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	addReqData, err := protocol.EncodeAddPartitionsToTxnRequest(addReq, addHeader.APIVersion)
	require.NoError(t, err)

	reader := bytes.NewReader(addReqData)
	_, err = handler.handleAddPartitionsToTxn(reader, addHeader)
	assert.NoError(t, err)

	// Now end transaction with abort
	req := &protocol.EndTxnRequest{
		TransactionalID: "test-txn",
		ProducerID:      1001,
		ProducerEpoch:   0,
		Committed:       false,
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.EndTxnKey,
		APIVersion:    0,
		CorrelationID: 2,
		ClientID:      "test-client",
	}

	reqData, err := protocol.EncodeEndTxnRequest(req, header.APIVersion)
	require.NoError(t, err)

	// Handle request
	reader = bytes.NewReader(reqData)
	respData, err := handler.handleEndTxn(reader, header)
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
	resp, err := protocol.DecodeEndTxnResponse(bytes.NewReader(respBody), header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), resp.ThrottleTimeMs)
	assert.Equal(t, protocol.None, resp.ErrorCode)
}

func TestEndTxn_ProducerIDMismatch(t *testing.T) {
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

	// First add partitions to transaction
	addReq := &protocol.AddPartitionsToTxnRequest{
		TransactionalID: "test-txn",
		ProducerID:      1001,
		ProducerEpoch:   0,
		Topics: []protocol.AddPartitionsToTxnTopic{
			{
				Name:       "test-topic",
				Partitions: []int32{0, 1},
			},
		},
	}

	addHeader := &protocol.RequestHeader{
		APIKey:        protocol.AddPartitionsToTxnKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	addReqData, err := protocol.EncodeAddPartitionsToTxnRequest(addReq, addHeader.APIVersion)
	require.NoError(t, err)

	reader := bytes.NewReader(addReqData)
	_, err = handler.handleAddPartitionsToTxn(reader, addHeader)
	assert.NoError(t, err)

	// Now try to end transaction with wrong producer ID
	req := &protocol.EndTxnRequest{
		TransactionalID: "test-txn",
		ProducerID:      1002, // Different producer ID
		ProducerEpoch:   0,
		Committed:       true,
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.EndTxnKey,
		APIVersion:    0,
		CorrelationID: 2,
		ClientID:      "test-client",
	}

	reqData, err := protocol.EncodeEndTxnRequest(req, header.APIVersion)
	require.NoError(t, err)

	// Handle request
	reader = bytes.NewReader(reqData)
	respData, err := handler.handleEndTxn(reader, header)
	assert.NoError(t, err)

	// Skip response header
	respReader := bytes.NewReader(respData)
	var correlationID int32
	binary.Read(respReader, binary.BigEndian, &correlationID)

	// Read remaining response body
	respBody, err := io.ReadAll(respReader)
	assert.NoError(t, err)

	// Decode and validate response
	resp, err := protocol.DecodeEndTxnResponse(bytes.NewReader(respBody), header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, protocol.InvalidProducerIDMapping, resp.ErrorCode)
}
