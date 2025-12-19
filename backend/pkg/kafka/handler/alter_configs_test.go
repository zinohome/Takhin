// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestAlterConfigs(t *testing.T) {
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
	handler := New(cfg, topicMgr)

	// Create a test topic first
	topicName := "test-alter-configs"
	err := topicMgr.CreateTopic(topicName, 1)
	assert.NoError(t, err)

	// Prepare request
	newValue := "gzip"
	req := &protocol.AlterConfigsRequest{
		Resources: []protocol.AlterConfigsResource{
			{
				ResourceType: protocol.ResourceTypeTopic,
				ResourceName: topicName,
				Configs: []protocol.AlterableConfig{
					{
						Name:  "compression.type",
						Value: &newValue,
					},
				},
			},
		},
		ValidateOnly: false,
	}

	// Encode request
	header := &protocol.RequestHeader{
		APIKey:        protocol.AlterConfigsKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	reqData := protocol.EncodeAlterConfigsRequest(req, header.APIVersion)

	// Handle request
	reader := bytes.NewReader(reqData)
	respData, err := handler.handleAlterConfigs(reader, header)
	assert.NoError(t, err)
	assert.NotNil(t, respData)

	// Skip response header (correlation ID)
	respReader := bytes.NewReader(respData)
	var correlationID int32
	binary.Read(respReader, binary.BigEndian, &correlationID)

	// Read remaining response body
	respBody, err := io.ReadAll(respReader)
	assert.NoError(t, err)

	// Decode and validate response
	resp, err := protocol.DecodeAlterConfigsResponse(respBody, header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resp.Resources))
	assert.Equal(t, protocol.None, resp.Resources[0].ErrorCode)
	assert.Equal(t, topicName, resp.Resources[0].ResourceName)
}

func TestAlterConfigs_NonexistentTopic(t *testing.T) {
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
	handler := New(cfg, topicMgr)

	// Prepare request for non-existent topic
	newValue := "gzip"
	req := &protocol.AlterConfigsRequest{
		Resources: []protocol.AlterConfigsResource{
			{
				ResourceType: protocol.ResourceTypeTopic,
				ResourceName: "nonexistent-topic",
				Configs: []protocol.AlterableConfig{
					{
						Name:  "compression.type",
						Value: &newValue,
					},
				},
			},
		},
		ValidateOnly: false,
	}

	// Encode request
	header := &protocol.RequestHeader{
		APIKey:        protocol.AlterConfigsKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	reqData := protocol.EncodeAlterConfigsRequest(req, header.APIVersion)

	// Handle request
	reader := bytes.NewReader(reqData)
	respData, err := handler.handleAlterConfigs(reader, header)
	assert.NoError(t, err)
	assert.NotNil(t, respData)

	// Skip response header (correlation ID)
	respReader := bytes.NewReader(respData)
	var correlationID int32
	binary.Read(respReader, binary.BigEndian, &correlationID)

	// Read remaining response body
	respBody, err := io.ReadAll(respReader)
	assert.NoError(t, err)

	// Decode and validate response - should return error for unknown topic
	resp, err := protocol.DecodeAlterConfigsResponse(respBody, header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resp.Resources))
	assert.Equal(t, protocol.UnknownTopicOrPartition, resp.Resources[0].ErrorCode)
	assert.NotNil(t, resp.Resources[0].ErrorMessage)
}

func TestAlterConfigs_ValidateOnly(t *testing.T) {
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
	handler := New(cfg, topicMgr)

	// Create a test topic
	topicName := "test-validate-configs"
	err := topicMgr.CreateTopic(topicName, 1)
	assert.NoError(t, err)

	// Prepare request with validate_only = true
	newValue := "lz4"
	req := &protocol.AlterConfigsRequest{
		Resources: []protocol.AlterConfigsResource{
			{
				ResourceType: protocol.ResourceTypeTopic,
				ResourceName: topicName,
				Configs: []protocol.AlterableConfig{
					{
						Name:  "compression.type",
						Value: &newValue,
					},
				},
			},
		},
		ValidateOnly: true,
	}

	// Encode request
	header := &protocol.RequestHeader{
		APIKey:        protocol.AlterConfigsKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	reqData := protocol.EncodeAlterConfigsRequest(req, header.APIVersion)

	// Handle request
	reader := bytes.NewReader(reqData)
	respData, err := handler.handleAlterConfigs(reader, header)
	assert.NoError(t, err)
	assert.NotNil(t, respData)

	// Skip response header (correlation ID)
	respReader := bytes.NewReader(respData)
	var correlationID int32
	binary.Read(respReader, binary.BigEndian, &correlationID)

	// Read remaining response body
	respBody, err := io.ReadAll(respReader)
	assert.NoError(t, err)

	// Decode and validate response - should succeed without actually changing anything
	resp, err := protocol.DecodeAlterConfigsResponse(respBody, header.APIVersion)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(resp.Resources))
	assert.Equal(t, protocol.None, resp.Resources[0].ErrorCode)
}
