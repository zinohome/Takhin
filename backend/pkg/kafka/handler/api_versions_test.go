// Copyright 2025 Takhin Data, Inc.

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

func TestHandleApiVersions_Success(t *testing.T) {
	// Create handler
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 9092,
		},
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			LogSegmentSize: 1024 * 1024,
		},
		Kafka: config.KafkaConfig{
			BrokerID: 1,
		},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	// Create request header
	header := &protocol.RequestHeader{
		APIKey:        protocol.ApiVersionsKey,
		APIVersion:    3,
		CorrelationID: 123,
		ClientID:      "test-client",
	}

	// Create request (version 3 with client info)
	reqBuf := make([]byte, 0, 128)
	reqBuf = append(reqBuf, encodeString("kafka-go")...)
	reqBuf = append(reqBuf, encodeString("1.0.0")...)

	reader := bytes.NewReader(reqBuf)

	// Handle request
	responseBytes, err := h.handleApiVersions(reader, header)
	require.NoError(t, err)
	require.NotNil(t, responseBytes)

	// Decode response
	respReader := bytes.NewReader(responseBytes)

	// Read correlation ID
	var corrID int32
	err = binary.Read(respReader, binary.BigEndian, &corrID)
	require.NoError(t, err)
	assert.Equal(t, header.CorrelationID, corrID)

	// Decode response body
	resp, err := protocol.DecodeApiVersionsResponse(respReader, header.APIVersion)
	require.NoError(t, err)
	assert.Equal(t, protocol.None, resp.ErrorCode)
	assert.NotEmpty(t, resp.APIVersions)

	// Verify some expected APIs are present
	apiMap := make(map[int16]protocol.APIVersion)
	for _, api := range resp.APIVersions {
		apiMap[api.APIKey] = api
	}

	// Check for essential APIs
	assert.Contains(t, apiMap, int16(protocol.ApiVersionsKey))
	assert.Contains(t, apiMap, int16(protocol.ProduceKey))
	assert.Contains(t, apiMap, int16(protocol.FetchKey))
	assert.Contains(t, apiMap, int16(protocol.MetadataKey))

	// Verify API version ranges
	produceAPI := apiMap[int16(protocol.ProduceKey)]
	assert.Equal(t, int16(0), produceAPI.MinVersion)
	assert.GreaterOrEqual(t, produceAPI.MaxVersion, int16(0))
}

func TestHandleApiVersions_Version0(t *testing.T) {
	// Create handler
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 9092,
		},
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			LogSegmentSize: 1024 * 1024,
		},
		Kafka: config.KafkaConfig{
			BrokerID: 1,
		},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	// Create request header (version 0 - no client info)
	header := &protocol.RequestHeader{
		APIKey:        protocol.ApiVersionsKey,
		APIVersion:    0,
		CorrelationID: 456,
		ClientID:      "test-client",
	}

	// Empty request for version 0
	reader := bytes.NewReader([]byte{})

	// Handle request
	responseBytes, err := h.handleApiVersions(reader, header)
	require.NoError(t, err)
	require.NotNil(t, responseBytes)

	// Decode response
	respReader := bytes.NewReader(responseBytes)

	// Read correlation ID
	var corrID int32
	err = binary.Read(respReader, binary.BigEndian, &corrID)
	require.NoError(t, err)
	assert.Equal(t, header.CorrelationID, corrID)

	// Decode response body
	resp, err := protocol.DecodeApiVersionsResponse(respReader, header.APIVersion)
	require.NoError(t, err)
	assert.Equal(t, protocol.None, resp.ErrorCode)
	assert.NotEmpty(t, resp.APIVersions)

	// Version 0 doesn't include ThrottleTimeMs
	assert.Equal(t, int32(0), resp.ThrottleTimeMs)
}

func TestHandleApiVersions_AllExpectedAPIs(t *testing.T) {
	// Create handler
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 9092,
		},
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			LogSegmentSize: 1024 * 1024,
		},
		Kafka: config.KafkaConfig{
			BrokerID: 1,
		},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	// Get supported APIs directly
	apiVersions := h.getSupportedAPIVersions()

	// Verify count matches expected
	assert.Equal(t, 28, len(apiVersions), "Expected 28 supported APIs")

	apiMap := make(map[int16]bool)
	for _, api := range apiVersions {
		apiMap[api.APIKey] = true
	}

	// Verify all critical APIs are supported
	expectedAPIs := []protocol.APIKey{
		protocol.ProduceKey,
		protocol.FetchKey,
		protocol.ListOffsetsKey,
		protocol.MetadataKey,
		protocol.OffsetCommitKey,
		protocol.OffsetFetchKey,
		protocol.FindCoordinatorKey,
		protocol.JoinGroupKey,
		protocol.HeartbeatKey,
		protocol.LeaveGroupKey,
		protocol.SyncGroupKey,
		protocol.DescribeGroupsKey,
		protocol.ListGroupsKey,
		protocol.SaslHandshakeKey,
		protocol.ApiVersionsKey,
		protocol.CreateTopicsKey,
		protocol.DeleteTopicsKey,
		protocol.DeleteRecordsKey,
		protocol.InitProducerIDKey,
		protocol.AddPartitionsToTxnKey,
		protocol.AddOffsetsToTxnKey,
		protocol.EndTxnKey,
		protocol.WriteTxnMarkersKey,
		protocol.TxnOffsetCommitKey,
		protocol.DescribeConfigsKey,
		protocol.AlterConfigsKey,
		protocol.DescribeLogDirsKey,
		protocol.SaslAuthenticateKey,
	}

	for _, expectedAPI := range expectedAPIs {
		assert.True(t, apiMap[int16(expectedAPI)], "API %d should be supported", expectedAPI)
	}
}

// Helper to encode string
func encodeString(s string) []byte {
	length := int16(len(s))
	buf := make([]byte, 2+len(s))
	binary.BigEndian.PutUint16(buf, uint16(length))
	copy(buf[2:], s)
	return buf
}
