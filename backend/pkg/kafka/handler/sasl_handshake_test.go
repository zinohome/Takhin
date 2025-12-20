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

func TestHandleSaslHandshake_SupportedMechanism(t *testing.T) {
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
		APIKey:        protocol.SaslHandshakeKey,
		APIVersion:    1,
		CorrelationID: 123,
		ClientID:      "test-client",
	}

	// Create request for PLAIN mechanism
	reqBuf := make([]byte, 0, 32)
	reqBuf = append(reqBuf, encodeSaslString("PLAIN")...)

	reader := bytes.NewReader(reqBuf)

	// Handle request
	responseBytes, err := h.handleSaslHandshake(reader, header)
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
	resp, err := protocol.DecodeSaslHandshakeResponse(respReader, header.APIVersion)
	require.NoError(t, err)
	assert.Equal(t, protocol.None, resp.ErrorCode)
	assert.NotEmpty(t, resp.EnabledMechanisms)
	assert.Contains(t, resp.EnabledMechanisms, "PLAIN")
}

func TestHandleSaslHandshake_UnsupportedMechanism(t *testing.T) {
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
		APIKey:        protocol.SaslHandshakeKey,
		APIVersion:    1,
		CorrelationID: 456,
		ClientID:      "test-client",
	}

	// Create request for unsupported mechanism
	reqBuf := make([]byte, 0, 32)
	reqBuf = append(reqBuf, encodeSaslString("UNSUPPORTED")...)

	reader := bytes.NewReader(reqBuf)

	// Handle request
	responseBytes, err := h.handleSaslHandshake(reader, header)
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
	resp, err := protocol.DecodeSaslHandshakeResponse(respReader, header.APIVersion)
	require.NoError(t, err)
	assert.Equal(t, protocol.UnsupportedSaslMechanism, resp.ErrorCode)
	assert.NotEmpty(t, resp.EnabledMechanisms)
}

func TestHandleSaslHandshake_AllMechanisms(t *testing.T) {
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

	// Get supported mechanisms
	mechanisms := h.getSupportedSaslMechanisms()

	// Verify we have expected mechanisms
	assert.Contains(t, mechanisms, "PLAIN")
	assert.Contains(t, mechanisms, "SCRAM-SHA-256")
	assert.Contains(t, mechanisms, "SCRAM-SHA-512")
	assert.GreaterOrEqual(t, len(mechanisms), 3)
}

func TestHandleSaslHandshake_ScramMechanism(t *testing.T) {
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
		APIKey:        protocol.SaslHandshakeKey,
		APIVersion:    1,
		CorrelationID: 789,
		ClientID:      "test-client",
	}

	// Create request for SCRAM-SHA-256 mechanism
	reqBuf := make([]byte, 0, 64)
	reqBuf = append(reqBuf, encodeSaslString("SCRAM-SHA-256")...)

	reader := bytes.NewReader(reqBuf)

	// Handle request
	responseBytes, err := h.handleSaslHandshake(reader, header)
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
	resp, err := protocol.DecodeSaslHandshakeResponse(respReader, header.APIVersion)
	require.NoError(t, err)
	assert.Equal(t, protocol.None, resp.ErrorCode)
	assert.Contains(t, resp.EnabledMechanisms, "SCRAM-SHA-256")
}

// Helper to encode string for SASL
func encodeSaslString(s string) []byte {
	length := int16(len(s))
	buf := make([]byte, 2+len(s))
	binary.BigEndian.PutUint16(buf, uint16(length))
	copy(buf[2:], s)
	return buf
}
