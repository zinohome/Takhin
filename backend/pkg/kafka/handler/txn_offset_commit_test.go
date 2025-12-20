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

func TestTxnOffsetCommit_Success(t *testing.T) {
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

	err := topicMgr.CreateTopic("test-topic", 2)
	require.NoError(t, err)

	addPartsReq := &protocol.AddPartitionsToTxnRequest{
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

	metadata := "test-metadata"
	req := &protocol.TxnOffsetCommitRequest{
		TransactionalID: "test-txn",
		GroupID:         "test-group",
		ProducerID:      1001,
		ProducerEpoch:   0,
		Topics: []protocol.TxnOffsetCommitTopic{
			{
				Name: "test-topic",
				Partitions: []protocol.TxnOffsetCommitPartition{
					{
						PartitionIndex: 0,
						Offset:         100,
						Metadata:       &metadata,
					},
					{
						PartitionIndex: 1,
						Offset:         200,
						Metadata:       nil,
					},
				},
			},
		},
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.TxnOffsetCommitKey,
		APIVersion:    0,
		CorrelationID: 2,
		ClientID:      "test-client",
	}

	reqData, err := protocol.EncodeTxnOffsetCommitRequest(req, header.APIVersion)
	require.NoError(t, err)

	reader = bytes.NewReader(reqData)
	respData, err := handler.handleTxnOffsetCommit(reader, header)
	assert.NoError(t, err)
	assert.NotNil(t, respData)

	respReader := bytes.NewReader(respData)
	var correlationID int32
	binary.Read(respReader, binary.BigEndian, &correlationID)

	respBody, err := io.ReadAll(respReader)
	assert.NoError(t, err)

	resp, err := protocol.DecodeTxnOffsetCommitResponse(bytes.NewReader(respBody), header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), resp.ThrottleTimeMs)
	assert.Len(t, resp.Topics, 1)
	assert.Equal(t, "test-topic", resp.Topics[0].Name)
	assert.Len(t, resp.Topics[0].Partitions, 2)

	for _, partition := range resp.Topics[0].Partitions {
		assert.Equal(t, protocol.None, partition.ErrorCode)
	}
}

func TestTxnOffsetCommit_MultipleTopics(t *testing.T) {
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

	err := topicMgr.CreateTopic("topic1", 2)
	require.NoError(t, err)
	err = topicMgr.CreateTopic("topic2", 2)
	require.NoError(t, err)

	addPartsReq := &protocol.AddPartitionsToTxnRequest{
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

	req := &protocol.TxnOffsetCommitRequest{
		TransactionalID: "test-txn",
		GroupID:         "test-group",
		ProducerID:      1001,
		ProducerEpoch:   0,
		Topics: []protocol.TxnOffsetCommitTopic{
			{
				Name: "topic1",
				Partitions: []protocol.TxnOffsetCommitPartition{
					{PartitionIndex: 0, Offset: 100, Metadata: nil},
					{PartitionIndex: 1, Offset: 200, Metadata: nil},
				},
			},
			{
				Name: "topic2",
				Partitions: []protocol.TxnOffsetCommitPartition{
					{PartitionIndex: 0, Offset: 300, Metadata: nil},
					{PartitionIndex: 1, Offset: 400, Metadata: nil},
				},
			},
		},
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.TxnOffsetCommitKey,
		APIVersion:    0,
		CorrelationID: 2,
		ClientID:      "test-client",
	}

	reqData, err := protocol.EncodeTxnOffsetCommitRequest(req, header.APIVersion)
	require.NoError(t, err)

	reader = bytes.NewReader(reqData)
	respData, err := handler.handleTxnOffsetCommit(reader, header)
	assert.NoError(t, err)

	respReader := bytes.NewReader(respData)
	var correlationID int32
	binary.Read(respReader, binary.BigEndian, &correlationID)

	respBody, err := io.ReadAll(respReader)
	assert.NoError(t, err)

	resp, err := protocol.DecodeTxnOffsetCommitResponse(bytes.NewReader(respBody), header.APIVersion)
	assert.NoError(t, err)
	assert.Len(t, resp.Topics, 2)

	for _, topicResult := range resp.Topics {
		assert.Len(t, topicResult.Partitions, 2)
		for _, partition := range topicResult.Partitions {
			assert.Equal(t, protocol.None, partition.ErrorCode)
		}
	}
}

func TestTxnOffsetCommit_ProducerIDMismatch(t *testing.T) {
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

	err := topicMgr.CreateTopic("test-topic", 2)
	require.NoError(t, err)

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

	req := &protocol.TxnOffsetCommitRequest{
		TransactionalID: "test-txn",
		GroupID:         "test-group",
		ProducerID:      1002,
		ProducerEpoch:   0,
		Topics: []protocol.TxnOffsetCommitTopic{
			{
				Name: "test-topic",
				Partitions: []protocol.TxnOffsetCommitPartition{
					{PartitionIndex: 0, Offset: 100, Metadata: nil},
				},
			},
		},
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.TxnOffsetCommitKey,
		APIVersion:    0,
		CorrelationID: 2,
		ClientID:      "test-client",
	}

	reqData, err := protocol.EncodeTxnOffsetCommitRequest(req, header.APIVersion)
	require.NoError(t, err)

	reader = bytes.NewReader(reqData)
	respData, err := handler.handleTxnOffsetCommit(reader, header)
	assert.NoError(t, err)

	respReader := bytes.NewReader(respData)
	var correlationID int32
	binary.Read(respReader, binary.BigEndian, &correlationID)

	respBody, err := io.ReadAll(respReader)
	assert.NoError(t, err)

	resp, err := protocol.DecodeTxnOffsetCommitResponse(bytes.NewReader(respBody), header.APIVersion)
	assert.NoError(t, err)
	assert.Len(t, resp.Topics, 1)
	assert.Len(t, resp.Topics[0].Partitions, 1)
	assert.Equal(t, protocol.InvalidProducerIDMapping, resp.Topics[0].Partitions[0].ErrorCode)
}
