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

func TestAddOffsetsToTxn_Success(t *testing.T) {
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

	addPartsReq := &protocol.AddPartitionsToTxnRequest{
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

	addPartsHeader := &protocol.RequestHeader{
		APIKey:        protocol.AddPartitionsToTxnKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	addPartsReqData, err := protocol.EncodeAddPartitionsToTxnRequest(addPartsReq, addPartsHeader.APIVersion)
	require.NoError(t, err)

	reader := bytes.NewReader(addPartsReqData)
	_, err = handler.handleAddPartitionsToTxn(reader, addPartsHeader)
	assert.NoError(t, err)

	req := &protocol.AddOffsetsToTxnRequest{
		TransactionalID: "test-txn",
		ProducerID:      1001,
		ProducerEpoch:   0,
		GroupID:         "test-group",
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.AddOffsetsToTxnKey,
		APIVersion:    0,
		CorrelationID: 2,
		ClientID:      "test-client",
	}

	reqData, err := protocol.EncodeAddOffsetsToTxnRequest(req, header.APIVersion)
	require.NoError(t, err)

	reader = bytes.NewReader(reqData)
	respData, err := handler.handleAddOffsetsToTxn(reader, header)
	assert.NoError(t, err)
	assert.NotNil(t, respData)

	respReader := bytes.NewReader(respData)
	var correlationID int32
	binary.Read(respReader, binary.BigEndian, &correlationID)

	respBody, err := io.ReadAll(respReader)
	assert.NoError(t, err)

	resp, err := protocol.DecodeAddOffsetsToTxnResponse(bytes.NewReader(respBody), header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), resp.ThrottleTimeMs)
	assert.Equal(t, protocol.None, resp.ErrorCode)
}

func TestAddOffsetsToTxn_ProducerIDMismatch(t *testing.T) {
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

	addPartsReq := &protocol.AddPartitionsToTxnRequest{
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

	addPartsHeader := &protocol.RequestHeader{
		APIKey:        protocol.AddPartitionsToTxnKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	addPartsReqData, err := protocol.EncodeAddPartitionsToTxnRequest(addPartsReq, addPartsHeader.APIVersion)
	require.NoError(t, err)

	reader := bytes.NewReader(addPartsReqData)
	_, err = handler.handleAddPartitionsToTxn(reader, addPartsHeader)
	assert.NoError(t, err)

	req := &protocol.AddOffsetsToTxnRequest{
		TransactionalID: "test-txn",
		ProducerID:      1002,
		ProducerEpoch:   0,
		GroupID:         "test-group",
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.AddOffsetsToTxnKey,
		APIVersion:    0,
		CorrelationID: 2,
		ClientID:      "test-client",
	}

	reqData, err := protocol.EncodeAddOffsetsToTxnRequest(req, header.APIVersion)
	require.NoError(t, err)

	reader = bytes.NewReader(reqData)
	respData, err := handler.handleAddOffsetsToTxn(reader, header)
	assert.NoError(t, err)

	respReader := bytes.NewReader(respData)
	var correlationID int32
	binary.Read(respReader, binary.BigEndian, &correlationID)

	respBody, err := io.ReadAll(respReader)
	assert.NoError(t, err)

	resp, err := protocol.DecodeAddOffsetsToTxnResponse(bytes.NewReader(respBody), header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, protocol.InvalidProducerIDMapping, resp.ErrorCode)
}

func TestAddOffsetsToTxn_NoTransaction(t *testing.T) {
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

	req := &protocol.AddOffsetsToTxnRequest{
		TransactionalID: "non-existent-txn",
		ProducerID:      1001,
		ProducerEpoch:   0,
		GroupID:         "test-group",
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.AddOffsetsToTxnKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	reqData, err := protocol.EncodeAddOffsetsToTxnRequest(req, header.APIVersion)
	require.NoError(t, err)

	reader := bytes.NewReader(reqData)
	respData, err := handler.handleAddOffsetsToTxn(reader, header)
	assert.NoError(t, err)

	respReader := bytes.NewReader(respData)
	var correlationID int32
	binary.Read(respReader, binary.BigEndian, &correlationID)

	respBody, err := io.ReadAll(respReader)
	assert.NoError(t, err)

	resp, err := protocol.DecodeAddOffsetsToTxnResponse(bytes.NewReader(respBody), header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, protocol.InvalidProducerIDMapping, resp.ErrorCode)
}
