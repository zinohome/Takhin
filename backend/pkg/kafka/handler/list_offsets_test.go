// Copyright 2025 Takhin Data
// Licensed under the Apache License, Version 2.0

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

func TestListOffsets(t *testing.T) {
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

	// Create topic
	topicName := "test-offsets"
	err := topicMgr.CreateTopic(topicName, 1)
	require.NoError(t, err)

	// Append some messages
	tp, exists := topicMgr.GetTopic(topicName)
	require.True(t, exists)

	for i := 0; i < 10; i++ {
		key := []byte("key")
		value := []byte("value")
		_, err := tp.Append(0, key, value)
		require.NoError(t, err)
	}

	tests := []struct {
		name           string
		timestamp      int64
		expectedOffset int64
		expectError    bool
	}{
		{
			name:           "earliest offset",
			timestamp:      protocol.TimestampEarliest,
			expectedOffset: 0,
			expectError:    false,
		},
		{
			name:           "latest offset",
			timestamp:      protocol.TimestampLatest,
			expectedOffset: 10, // HWM after 10 appends
			expectError:    false,
		},
		{
			name:           "specific timestamp",
			timestamp:      1000,
			expectedOffset: 0, // 当前实现简化，timestamp查找失败返回0
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := &protocol.ListOffsetsRequest{
				ReplicaID:      -1,
				IsolationLevel: 0,
				Topics: []protocol.ListOffsetsTopic{
					{
						Name: topicName,
						Partitions: []protocol.ListOffsetsPartition{
							{
								PartitionIndex:     0,
								CurrentLeaderEpoch: -1,
								Timestamp:          tt.timestamp,
								MaxNumOffsets:      1,
							},
						},
					},
				},
			}

			// Encode request body
			reqBody := encodeListOffsetsRequest(req, 1)

			// Create request header
			header := &protocol.RequestHeader{
				APIKey:        protocol.ListOffsetsKey,
				APIVersion:    1,
				CorrelationID: 1,
				ClientID:      "test-client",
			}

			// Handle request
			r := bytes.NewReader(reqBody)
			respData, err := handler.handleListOffsets(r, header)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Decode response
			resp := decodeListOffsetsResponse(t, respData, 1)

			// Verify response
			require.Len(t, resp.Topics, 1)
			assert.Equal(t, topicName, resp.Topics[0].Name)
			require.Len(t, resp.Topics[0].Partitions, 1)

			partResp := resp.Topics[0].Partitions[0]
			assert.Equal(t, protocol.None, partResp.ErrorCode)
			assert.Equal(t, int32(0), partResp.PartitionIndex)
			assert.Equal(t, tt.expectedOffset, partResp.Offset)
		})
	}
}

func TestListOffsetsNonexistentTopic(t *testing.T) {
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

	// Create request for non-existent topic
	req := &protocol.ListOffsetsRequest{
		ReplicaID:      -1,
		IsolationLevel: 0,
		Topics: []protocol.ListOffsetsTopic{
			{
				Name: "nonexistent-topic",
				Partitions: []protocol.ListOffsetsPartition{
					{
						PartitionIndex:     0,
						CurrentLeaderEpoch: -1,
						Timestamp:          protocol.TimestampLatest,
						MaxNumOffsets:      1,
					},
				},
			},
		},
	}

	// Encode request
	reqBody := encodeListOffsetsRequest(req, 1)

	header := &protocol.RequestHeader{
		APIKey:        protocol.ListOffsetsKey,
		APIVersion:    1,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	// Handle request
	r := bytes.NewReader(reqBody)
	respData, err := handler.handleListOffsets(r, header)
	require.NoError(t, err)

	// Decode response
	resp := decodeListOffsetsResponse(t, respData, 1)

	// Verify error response
	require.Len(t, resp.Topics, 1)
	require.Len(t, resp.Topics[0].Partitions, 1)
	assert.Equal(t, protocol.UnknownTopicOrPartition, resp.Topics[0].Partitions[0].ErrorCode)
}

// Helper functions

func encodeListOffsetsRequest(req *protocol.ListOffsetsRequest, version int16) []byte {
	buf := make([]byte, 0, 1024)

	// ReplicaID
	buf = appendInt32(buf, req.ReplicaID)

	// IsolationLevel (v2+)
	if version >= 2 {
		buf = append(buf, byte(req.IsolationLevel))
	}

	// Topics array
	buf = appendInt32(buf, int32(len(req.Topics)))
	for _, topic := range req.Topics {
		buf = appendString(buf, topic.Name)
		buf = appendInt32(buf, int32(len(topic.Partitions)))
		for _, part := range topic.Partitions {
			buf = appendInt32(buf, part.PartitionIndex)
			if version >= 4 {
				buf = appendInt32(buf, part.CurrentLeaderEpoch)
			}
			buf = appendInt64(buf, part.Timestamp)
			if version == 0 {
				buf = appendInt32(buf, part.MaxNumOffsets)
			}
		}
	}

	return buf
}

func decodeListOffsetsResponse(t *testing.T, data []byte, version int16) *protocol.ListOffsetsResponse {
	r := bytes.NewReader(data)

	// Skip response header (just correlation ID)
	var correlationID int32
	err := binary.Read(r, binary.BigEndian, &correlationID)
	require.NoError(t, err)
	assert.Equal(t, int32(1), correlationID)

	// Read remaining data
	respBody, err := io.ReadAll(r)
	require.NoError(t, err)

	resp := &protocol.ListOffsetsResponse{}
	offset := 0

	// ThrottleTimeMs
	resp.ThrottleTimeMs = int32(binary.BigEndian.Uint32(respBody[offset:]))
	offset += 4

	// Topics array
	topicsLen := int(binary.BigEndian.Uint32(respBody[offset:]))
	offset += 4

	resp.Topics = make([]protocol.ListOffsetsTopicResponse, topicsLen)
	for i := 0; i < topicsLen; i++ {
		// Topic name
		nameLen := int(binary.BigEndian.Uint16(respBody[offset:]))
		offset += 2
		resp.Topics[i].Name = string(respBody[offset : offset+nameLen])
		offset += nameLen

		// Partitions array
		partLen := int(binary.BigEndian.Uint32(respBody[offset:]))
		offset += 4

		resp.Topics[i].Partitions = make([]protocol.ListOffsetsPartitionResponse, partLen)
		for j := 0; j < partLen; j++ {
			resp.Topics[i].Partitions[j].PartitionIndex = int32(binary.BigEndian.Uint32(respBody[offset:]))
			offset += 4

			resp.Topics[i].Partitions[j].ErrorCode = protocol.ErrorCode(binary.BigEndian.Uint16(respBody[offset:]))
			offset += 2

			if version >= 1 {
				resp.Topics[i].Partitions[j].Timestamp = int64(binary.BigEndian.Uint64(respBody[offset:]))
				offset += 8
			}

			resp.Topics[i].Partitions[j].Offset = int64(binary.BigEndian.Uint64(respBody[offset:]))
			offset += 8

			if version >= 4 {
				resp.Topics[i].Partitions[j].LeaderEpoch = int32(binary.BigEndian.Uint32(respBody[offset:]))
				offset += 4
			}
		}
	}

	return resp
}

// Helper append functions
func appendInt32(buf []byte, v int32) []byte {
	tmp := make([]byte, 4)
	binary.BigEndian.PutUint32(tmp, uint32(v))
	return append(buf, tmp...)
}

func appendInt64(buf []byte, v int64) []byte {
	tmp := make([]byte, 8)
	binary.BigEndian.PutUint64(tmp, uint64(v))
	return append(buf, tmp...)
}

func appendString(buf []byte, s string) []byte {
	nameLen := make([]byte, 2)
	binary.BigEndian.PutUint16(nameLen, uint16(len(s)))
	buf = append(buf, nameLen...)
	return append(buf, []byte(s)...)
}
