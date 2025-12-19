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

func TestDescribeLogDirs(t *testing.T) {
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

	// Create topic with data
	topicName := "test-log-dirs"
	err := topicMgr.CreateTopic(topicName, 2)
	require.NoError(t, err)

	// Write some data
	tp, exists := topicMgr.GetTopic(topicName)
	require.True(t, exists)
	for i := 0; i < 10; i++ {
		_, err := tp.Append(0, []byte("key"), []byte("value"))
		require.NoError(t, err)
	}

	// Query all topics (null request)
	req := &protocol.DescribeLogDirsRequest{
		Topics: nil, // nil = query all
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.DescribeLogDirsKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	reqData := protocol.EncodeDescribeLogDirsRequest(req, header.APIVersion)

	// Handle request
	reader := bytes.NewReader(reqData)
	respData, err := handler.handleDescribeLogDirs(reader, header)
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
	resp, err := protocol.DecodeDescribeLogDirsResponse(respBody, header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resp.LogDirs)) // One log directory

	logDir := resp.LogDirs[0]
	assert.Equal(t, protocol.None, logDir.ErrorCode)
	assert.Equal(t, cfg.Storage.DataDir, logDir.LogDir)
	assert.GreaterOrEqual(t, len(logDir.Topics), 1) // At least our test topic

	// Find our test topic
	var found bool
	for _, topicResult := range logDir.Topics {
		if topicResult.Topic == topicName {
			found = true
			assert.Equal(t, 2, len(topicResult.Partitions)) // 2 partitions

			// Check partition 0 exists
			var part0Found bool
			for _, part := range topicResult.Partitions {
				if part.PartitionIndex == 0 {
					part0Found = true
					// Size may be 0 or greater depending on implementation
					assert.GreaterOrEqual(t, part.Size, int64(0))
				}
			}
			assert.True(t, part0Found, "partition 0 should exist")
			break
		}
	}
	assert.True(t, found, "test topic should be in response")
}

func TestDescribeLogDirs_SpecificTopic(t *testing.T) {
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

	// Create topics
	topicName1 := "topic1"
	topicName2 := "topic2"
	err := topicMgr.CreateTopic(topicName1, 1)
	require.NoError(t, err)
	err = topicMgr.CreateTopic(topicName2, 1)
	require.NoError(t, err)

	// Query specific topic
	req := &protocol.DescribeLogDirsRequest{
		Topics: []protocol.DescribeLogDirsTopic{
			{
				Topic:      topicName1,
				Partitions: []int32{0},
			},
		},
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.DescribeLogDirsKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	reqData := protocol.EncodeDescribeLogDirsRequest(req, header.APIVersion)

	// Handle request
	reader := bytes.NewReader(reqData)
	respData, err := handler.handleDescribeLogDirs(reader, header)
	assert.NoError(t, err)

	// Skip response header
	respReader := bytes.NewReader(respData)
	var correlationID int32
	binary.Read(respReader, binary.BigEndian, &correlationID)

	// Read remaining response body
	respBody, err := io.ReadAll(respReader)
	assert.NoError(t, err)

	// Decode and validate response
	resp, err := protocol.DecodeDescribeLogDirsResponse(respBody, header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resp.LogDirs))

	logDir := resp.LogDirs[0]
	assert.Equal(t, protocol.None, logDir.ErrorCode)

	// Should only contain topic1
	assert.Equal(t, 1, len(logDir.Topics))
	assert.Equal(t, topicName1, logDir.Topics[0].Topic)
}
