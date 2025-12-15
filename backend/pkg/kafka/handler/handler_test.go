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

func TestHandleApiVersions(t *testing.T) {
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			AdvertisedHost: "localhost",
			AdvertisedPort: 9092,
		},
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			LogSegmentSize: 1024 * 1024,
		},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	defer topicMgr.Close()

	handler := New(cfg, topicMgr)

	// Create ApiVersions request
	var reqBuf bytes.Buffer
	header := &protocol.RequestHeader{
		APIKey:        protocol.ApiVersionsKey,
		APIVersion:    2,
		CorrelationID: 123,
		ClientID:      "test-client",
	}
	err := header.Encode(&reqBuf)
	require.NoError(t, err)

	// Handle request
	resp, err := handler.HandleRequest(reqBuf.Bytes())
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	// Verify response has correlation ID
	respReader := bytes.NewReader(resp)
	correlationID, err := protocol.ReadInt32(respReader)
	require.NoError(t, err)
	assert.Equal(t, int32(123), correlationID)
}

func TestHandleMetadata(t *testing.T) {
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			AdvertisedHost: "localhost",
			AdvertisedPort: 9092,
		},
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			LogSegmentSize: 1024 * 1024,
		},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	defer topicMgr.Close()

	handler := New(cfg, topicMgr)

	// Create Metadata request
	var reqBuf bytes.Buffer
	header := &protocol.RequestHeader{
		APIKey:        protocol.MetadataKey,
		APIVersion:    5,
		CorrelationID: 456,
		ClientID:      "test-client",
	}
	err := header.Encode(&reqBuf)
	require.NoError(t, err)

	// Write empty topics array (request all topics)
	err = protocol.WriteArray(&reqBuf, 0)
	require.NoError(t, err)

	// Handle request
	resp, err := handler.HandleRequest(reqBuf.Bytes())
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	// Verify response has correlation ID
	respReader := bytes.NewReader(resp)
	correlationID, err := protocol.ReadInt32(respReader)
	require.NoError(t, err)
	assert.Equal(t, int32(456), correlationID)
}
