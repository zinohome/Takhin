// Copyright 2025 Takhin Data
// Licensed under the Apache License, Version 2.0

package handler

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestDeleteRecords(t *testing.T) {
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

	// Create topic and add messages
	topicName := "test-delete-records"
	err := topicMgr.CreateTopic(topicName, 1)
	require.NoError(t, err)

	tp, exists := topicMgr.GetTopic(topicName)
	require.True(t, exists)

	// Append 10 messages
	for i := 0; i < 10; i++ {
		_, err := tp.Append(0, []byte("key"), []byte("value"))
		require.NoError(t, err)
	}

	// Test deleting records before offset 5
	req := &protocol.DeleteRecordsRequest{
		Topics: []protocol.DeleteRecordsTopic{
			{
				Name: topicName,
				Partitions: []protocol.DeleteRecordsPartition{
					{
						PartitionIndex: 0,
						Offset:         5,
					},
				},
			},
		},
		TimeoutMs: 5000,
	}

	// Encode request
	reqBody := encodeDeleteRecordsRequest(req)

	header := &protocol.RequestHeader{
		APIKey:        protocol.DeleteRecordsKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	// Handle request
	r := bytes.NewReader(reqBody)
	respData, err := handler.handleDeleteRecords(r, header)
	require.NoError(t, err)

	// Decode response
	resp := decodeDeleteRecordsResponse(t, respData)

	// Verify response
	require.Len(t, resp.Topics, 1)
	assert.Equal(t, topicName, resp.Topics[0].Name)
	require.Len(t, resp.Topics[0].Partitions, 1)

	partResp := resp.Topics[0].Partitions[0]
	assert.Equal(t, int32(0), partResp.PartitionIndex)
	assert.Equal(t, protocol.None, partResp.ErrorCode)
	assert.Equal(t, int64(5), partResp.LowWatermark)
}

func TestDeleteRecordsNonexistentTopic(t *testing.T) {
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

	// Try to delete records from non-existent topic
	req := &protocol.DeleteRecordsRequest{
		Topics: []protocol.DeleteRecordsTopic{
			{
				Name: "nonexistent-topic",
				Partitions: []protocol.DeleteRecordsPartition{
					{
						PartitionIndex: 0,
						Offset:         5,
					},
				},
			},
		},
		TimeoutMs: 5000,
	}

	reqBody := encodeDeleteRecordsRequest(req)

	header := &protocol.RequestHeader{
		APIKey:        protocol.DeleteRecordsKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	r := bytes.NewReader(reqBody)
	respData, err := handler.handleDeleteRecords(r, header)
	require.NoError(t, err)

	resp := decodeDeleteRecordsResponse(t, respData)

	// Verify error response
	require.Len(t, resp.Topics, 1)
	require.Len(t, resp.Topics[0].Partitions, 1)
	assert.Equal(t, protocol.UnknownTopicOrPartition, resp.Topics[0].Partitions[0].ErrorCode)
	assert.Equal(t, int64(-1), resp.Topics[0].Partitions[0].LowWatermark)
}

func TestDeleteRecordsBeyondHWM(t *testing.T) {
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

	// Create topic with 5 messages
	topicName := "test-delete-beyond-hwm"
	err := topicMgr.CreateTopic(topicName, 1)
	require.NoError(t, err)

	tp, exists := topicMgr.GetTopic(topicName)
	require.True(t, exists)

	for i := 0; i < 5; i++ {
		_, err := tp.Append(0, []byte("key"), []byte("value"))
		require.NoError(t, err)
	}

	// Try to delete beyond HWM (offset 10 > HWM 5)
	req := &protocol.DeleteRecordsRequest{
		Topics: []protocol.DeleteRecordsTopic{
			{
				Name: topicName,
				Partitions: []protocol.DeleteRecordsPartition{
					{
						PartitionIndex: 0,
						Offset:         10,
					},
				},
			},
		},
		TimeoutMs: 5000,
	}

	reqBody := encodeDeleteRecordsRequest(req)

	header := &protocol.RequestHeader{
		APIKey:        protocol.DeleteRecordsKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	r := bytes.NewReader(reqBody)
	respData, err := handler.handleDeleteRecords(r, header)
	require.NoError(t, err)

	resp := decodeDeleteRecordsResponse(t, respData)

	// Should return error
	require.Len(t, resp.Topics, 1)
	require.Len(t, resp.Topics[0].Partitions, 1)
	assert.Equal(t, protocol.InvalidRequest, resp.Topics[0].Partitions[0].ErrorCode)
}

// Helper functions

func encodeDeleteRecordsRequest(req *protocol.DeleteRecordsRequest) []byte {
	buf := make([]byte, 0, 1024)

	// Topics array
	topicsLen := make([]byte, 4)
	binary.BigEndian.PutUint32(topicsLen, uint32(len(req.Topics)))
	buf = append(buf, topicsLen...)

	for _, topic := range req.Topics {
		// Topic name
		nameLen := make([]byte, 2)
		binary.BigEndian.PutUint16(nameLen, uint16(len(topic.Name)))
		buf = append(buf, nameLen...)
		buf = append(buf, []byte(topic.Name)...)

		// Partitions array
		partsLen := make([]byte, 4)
		binary.BigEndian.PutUint32(partsLen, uint32(len(topic.Partitions)))
		buf = append(buf, partsLen...)

		for _, part := range topic.Partitions {
			// Partition index
			partIdx := make([]byte, 4)
			binary.BigEndian.PutUint32(partIdx, uint32(part.PartitionIndex))
			buf = append(buf, partIdx...)

			// Offset
			offset := make([]byte, 8)
			binary.BigEndian.PutUint64(offset, uint64(part.Offset))
			buf = append(buf, offset...)
		}
	}

	// TimeoutMs
	timeout := make([]byte, 4)
	binary.BigEndian.PutUint32(timeout, uint32(req.TimeoutMs))
	buf = append(buf, timeout...)

	return buf
}

func decodeDeleteRecordsResponse(t *testing.T, data []byte) *protocol.DeleteRecordsResponse {
	r := bytes.NewReader(data)

	// Skip response header
	var correlationID int32
	err := binary.Read(r, binary.BigEndian, &correlationID)
	require.NoError(t, err)

	// Read response body
	resp := &protocol.DeleteRecordsResponse{}

	// ThrottleTimeMs
	err = binary.Read(r, binary.BigEndian, &resp.ThrottleTimeMs)
	require.NoError(t, err)

	// Topics array length
	var topicsLen int32
	err = binary.Read(r, binary.BigEndian, &topicsLen)
	require.NoError(t, err)

	resp.Topics = make([]protocol.DeleteRecordsTopicResponse, topicsLen)
	for i := 0; i < int(topicsLen); i++ {
		// Topic name
		var nameLen int16
		err = binary.Read(r, binary.BigEndian, &nameLen)
		require.NoError(t, err)

		nameBytes := make([]byte, nameLen)
		_, err = r.Read(nameBytes)
		require.NoError(t, err)
		resp.Topics[i].Name = string(nameBytes)

		// Partitions array length
		var partsLen int32
		err = binary.Read(r, binary.BigEndian, &partsLen)
		require.NoError(t, err)

		resp.Topics[i].Partitions = make([]protocol.DeleteRecordsPartitionResponse, partsLen)
		for j := 0; j < int(partsLen); j++ {
			// Partition index
			err = binary.Read(r, binary.BigEndian, &resp.Topics[i].Partitions[j].PartitionIndex)
			require.NoError(t, err)

			// LowWatermark
			err = binary.Read(r, binary.BigEndian, &resp.Topics[i].Partitions[j].LowWatermark)
			require.NoError(t, err)

			// ErrorCode
			var errCode int16
			err = binary.Read(r, binary.BigEndian, &errCode)
			require.NoError(t, err)
			resp.Topics[i].Partitions[j].ErrorCode = protocol.ErrorCode(errCode)
		}
	}

	return resp
}
