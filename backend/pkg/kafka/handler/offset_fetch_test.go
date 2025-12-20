// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestHandleOffsetFetch_Success(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	// First commit some offsets
	commitReq := &protocol.OffsetCommitRequest{
		GroupID: "test-group",
		Topics: []protocol.OffsetCommitRequestTopic{
			{
				Name: "test-topic",
				Partitions: []protocol.OffsetCommitRequestPartition{
					{
						PartitionIndex: 0,
						Offset:         100,
						Metadata:       "test-metadata",
					},
					{
						PartitionIndex: 1,
						Offset:         200,
						Metadata:       "test-metadata-2",
					},
				},
			},
		},
	}

	// Commit offsets first
	commitHeader := &protocol.RequestHeader{
		APIKey:        protocol.OffsetCommitKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}
	_, err := h.handleOffsetCommit(bytes.NewReader(commitReq.Encode()), commitHeader)
	require.NoError(t, err)

	// Now fetch the committed offsets
	fetchReq := &protocol.OffsetFetchRequest{
		GroupID: "test-group",
		Topics: []protocol.OffsetFetchRequestTopic{
			{
				Name:             "test-topic",
				PartitionIndexes: []int32{0, 1},
			},
		},
	}

	fetchHeader := &protocol.RequestHeader{
		APIKey:        protocol.OffsetFetchKey,
		APIVersion:    0,
		CorrelationID: 2,
		ClientID:      "test-client",
	}

	respBytes, err := h.handleOffsetFetch(bytes.NewReader(fetchReq.Encode()), fetchHeader)
	require.NoError(t, err)
	require.NotNil(t, respBytes)

	// Decode response (skip correlation ID - first 4 bytes)
	resp := &protocol.OffsetFetchResponse{}
	err = resp.Decode(respBytes[4:])
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, int16(protocol.None), resp.ErrorCode)
	assert.Len(t, resp.Topics, 1)
	assert.Equal(t, "test-topic", resp.Topics[0].Name)
	assert.Len(t, resp.Topics[0].Partitions, 2)

	// Check partition 0
	assert.Equal(t, int32(0), resp.Topics[0].Partitions[0].PartitionIndex)
	assert.Equal(t, int64(100), resp.Topics[0].Partitions[0].Offset)
	assert.Equal(t, "test-metadata", resp.Topics[0].Partitions[0].Metadata)
	assert.Equal(t, int16(protocol.None), resp.Topics[0].Partitions[0].ErrorCode)

	// Check partition 1
	assert.Equal(t, int32(1), resp.Topics[0].Partitions[1].PartitionIndex)
	assert.Equal(t, int64(200), resp.Topics[0].Partitions[1].Offset)
	assert.Equal(t, "test-metadata-2", resp.Topics[0].Partitions[1].Metadata)
	assert.Equal(t, int16(protocol.None), resp.Topics[0].Partitions[1].ErrorCode)
}

func TestHandleOffsetFetch_NoCommittedOffsets(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	// Fetch offsets without committing first
	req := &protocol.OffsetFetchRequest{
		GroupID: "test-group",
		Topics: []protocol.OffsetFetchRequestTopic{
			{
				Name:             "test-topic",
				PartitionIndexes: []int32{0, 1},
			},
		},
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.OffsetFetchKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	respBytes, err := h.handleOffsetFetch(bytes.NewReader(req.Encode()), header)
	require.NoError(t, err)
	require.NotNil(t, respBytes)

	// Decode response (skip correlation ID - first 4 bytes)
	resp := &protocol.OffsetFetchResponse{}
	err = resp.Decode(respBytes[4:])
	require.NoError(t, err)

	// Verify response - should return -1 for uncommitted offsets
	assert.Equal(t, int16(protocol.None), resp.ErrorCode)
	assert.Len(t, resp.Topics, 1)
	assert.Equal(t, "test-topic", resp.Topics[0].Name)
	assert.Len(t, resp.Topics[0].Partitions, 2)

	// Both partitions should have offset -1 (no committed offset)
	assert.Equal(t, int32(0), resp.Topics[0].Partitions[0].PartitionIndex)
	assert.Equal(t, int64(-1), resp.Topics[0].Partitions[0].Offset)
	assert.Equal(t, int16(protocol.None), resp.Topics[0].Partitions[0].ErrorCode)

	assert.Equal(t, int32(1), resp.Topics[0].Partitions[1].PartitionIndex)
	assert.Equal(t, int64(-1), resp.Topics[0].Partitions[1].Offset)
	assert.Equal(t, int16(protocol.None), resp.Topics[0].Partitions[1].ErrorCode)
}

func TestHandleOffsetFetch_MultipleTopics(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	// Commit offsets for multiple topics
	commitReq := &protocol.OffsetCommitRequest{
		GroupID: "test-group",
		Topics: []protocol.OffsetCommitRequestTopic{
			{
				Name: "topic1",
				Partitions: []protocol.OffsetCommitRequestPartition{
					{
						PartitionIndex: 0,
						Offset:         100,
						Metadata:       "metadata1",
					},
				},
			},
			{
				Name: "topic2",
				Partitions: []protocol.OffsetCommitRequestPartition{
					{
						PartitionIndex: 0,
						Offset:         200,
						Metadata:       "metadata2",
					},
				},
			},
		},
	}

	commitHeader := &protocol.RequestHeader{
		APIKey:        protocol.OffsetCommitKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}
	_, err := h.handleOffsetCommit(bytes.NewReader(commitReq.Encode()), commitHeader)
	require.NoError(t, err)

	// Fetch offsets for both topics
	fetchReq := &protocol.OffsetFetchRequest{
		GroupID: "test-group",
		Topics: []protocol.OffsetFetchRequestTopic{
			{
				Name:             "topic1",
				PartitionIndexes: []int32{0},
			},
			{
				Name:             "topic2",
				PartitionIndexes: []int32{0},
			},
		},
	}

	fetchHeader := &protocol.RequestHeader{
		APIKey:        protocol.OffsetFetchKey,
		APIVersion:    0,
		CorrelationID: 2,
		ClientID:      "test-client",
	}

	respBytes, err := h.handleOffsetFetch(bytes.NewReader(fetchReq.Encode()), fetchHeader)
	require.NoError(t, err)
	require.NotNil(t, respBytes)

	// Decode response (skip correlation ID - first 4 bytes)
	resp := &protocol.OffsetFetchResponse{}
	err = resp.Decode(respBytes[4:])
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, int16(protocol.None), resp.ErrorCode)
	assert.Len(t, resp.Topics, 2)

	// Check topic1
	assert.Equal(t, "topic1", resp.Topics[0].Name)
	assert.Len(t, resp.Topics[0].Partitions, 1)
	assert.Equal(t, int64(100), resp.Topics[0].Partitions[0].Offset)
	assert.Equal(t, "metadata1", resp.Topics[0].Partitions[0].Metadata)

	// Check topic2
	assert.Equal(t, "topic2", resp.Topics[1].Name)
	assert.Len(t, resp.Topics[1].Partitions, 1)
	assert.Equal(t, int64(200), resp.Topics[1].Partitions[0].Offset)
	assert.Equal(t, "metadata2", resp.Topics[1].Partitions[0].Metadata)
}

func TestHandleOffsetFetch_UpdatedOffsets(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	// First commit
	commitReq := &protocol.OffsetCommitRequest{
		GroupID: "test-group",
		Topics: []protocol.OffsetCommitRequestTopic{
			{
				Name: "test-topic",
				Partitions: []protocol.OffsetCommitRequestPartition{
					{
						PartitionIndex: 0,
						Offset:         100,
						Metadata:       "first",
					},
				},
			},
		},
	}

	header1 := &protocol.RequestHeader{
		APIKey:        protocol.OffsetCommitKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}
	_, err := h.handleOffsetCommit(bytes.NewReader(commitReq.Encode()), header1)
	require.NoError(t, err)

	// Update offset
	commitReq.Topics[0].Partitions[0].Offset = 200
	commitReq.Topics[0].Partitions[0].Metadata = "second"
	header2 := &protocol.RequestHeader{
		APIKey:        protocol.OffsetCommitKey,
		APIVersion:    0,
		CorrelationID: 2,
		ClientID:      "test-client",
	}
	_, err = h.handleOffsetCommit(bytes.NewReader(commitReq.Encode()), header2)
	require.NoError(t, err)

	// Fetch updated offset
	fetchReq := &protocol.OffsetFetchRequest{
		GroupID: "test-group",
		Topics: []protocol.OffsetFetchRequestTopic{
			{
				Name:             "test-topic",
				PartitionIndexes: []int32{0},
			},
		},
	}

	fetchHeader := &protocol.RequestHeader{
		APIKey:        protocol.OffsetFetchKey,
		APIVersion:    0,
		CorrelationID: 3,
		ClientID:      "test-client",
	}

	respBytes, err := h.handleOffsetFetch(bytes.NewReader(fetchReq.Encode()), fetchHeader)
	require.NoError(t, err)

	// Decode response (skip correlation ID - first 4 bytes)
	resp := &protocol.OffsetFetchResponse{}
	err = resp.Decode(respBytes[4:])
	require.NoError(t, err)

	// Verify updated offset
	assert.Equal(t, int16(protocol.None), resp.ErrorCode)
	assert.Len(t, resp.Topics, 1)
	assert.Equal(t, int64(200), resp.Topics[0].Partitions[0].Offset)
	assert.Equal(t, "second", resp.Topics[0].Partitions[0].Metadata)
}
