// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestHandleSaslAuthenticate_Success(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	header := &protocol.RequestHeader{
		APIKey:        protocol.SaslAuthenticateKey,
		APIVersion:    2,
		CorrelationID: 123,
		ClientID:      "test-client",
	}

	credentials := fmt.Sprintf("\x00testuser\x00testpass")
	authBytes := []byte(credentials)

	reqBuf := make([]byte, 4+len(authBytes))
	binary.BigEndian.PutUint32(reqBuf[0:4], uint32(len(authBytes)))
	copy(reqBuf[4:], authBytes)

	reader := bytes.NewReader(reqBuf)
	responseBytes, err := h.handleSaslAuthenticate(reader, header)
	require.NoError(t, err)
	require.NotNil(t, responseBytes)

	respReader := bytes.NewReader(responseBytes)
	var corrID int32
	err = binary.Read(respReader, binary.BigEndian, &corrID)
	require.NoError(t, err)
	assert.Equal(t, header.CorrelationID, corrID)

	resp, err := protocol.DecodeSaslAuthenticateResponse(respReader, header.APIVersion)
	require.NoError(t, err)
	assert.Equal(t, protocol.None, resp.ErrorCode)
	assert.Nil(t, resp.ErrorMessage)
	assert.Greater(t, resp.SessionLifetimeMs, int64(0))
}

func TestHandleSaslAuthenticate_AdminUser(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	header := &protocol.RequestHeader{
		APIKey:        protocol.SaslAuthenticateKey,
		APIVersion:    2,
		CorrelationID: 456,
		ClientID:      "test-client",
	}

	credentials := fmt.Sprintf("\x00admin\x00adminpass")
	authBytes := []byte(credentials)

	reqBuf := make([]byte, 4+len(authBytes))
	binary.BigEndian.PutUint32(reqBuf[0:4], uint32(len(authBytes)))
	copy(reqBuf[4:], authBytes)

	reader := bytes.NewReader(reqBuf)
	responseBytes, err := h.handleSaslAuthenticate(reader, header)
	require.NoError(t, err)

	respReader := bytes.NewReader(responseBytes)
	var corrID int32
	binary.Read(respReader, binary.BigEndian, &corrID)

	resp, err := protocol.DecodeSaslAuthenticateResponse(respReader, header.APIVersion)
	require.NoError(t, err)
	assert.Equal(t, protocol.None, resp.ErrorCode)
	assert.Equal(t, int64(3600000), resp.SessionLifetimeMs)
}

func TestHandleSaslAuthenticate_EmptyCredentials(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	header := &protocol.RequestHeader{
		APIKey:        protocol.SaslAuthenticateKey,
		APIVersion:    2,
		CorrelationID: 789,
		ClientID:      "test-client",
	}

	credentials := fmt.Sprintf("\x00\x00")
	authBytes := []byte(credentials)

	reqBuf := make([]byte, 4+len(authBytes))
	binary.BigEndian.PutUint32(reqBuf[0:4], uint32(len(authBytes)))
	copy(reqBuf[4:], authBytes)

	reader := bytes.NewReader(reqBuf)
	responseBytes, err := h.handleSaslAuthenticate(reader, header)
	require.NoError(t, err)

	respReader := bytes.NewReader(responseBytes)
	var corrID int32
	binary.Read(respReader, binary.BigEndian, &corrID)

	resp, err := protocol.DecodeSaslAuthenticateResponse(respReader, header.APIVersion)
	require.NoError(t, err)
	assert.Equal(t, protocol.SaslAuthenticationFailed, resp.ErrorCode)
	assert.NotNil(t, resp.ErrorMessage)
}

func TestHandleSaslAuthenticate_InvalidFormat(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	header := &protocol.RequestHeader{
		APIKey:        protocol.SaslAuthenticateKey,
		APIVersion:    2,
		CorrelationID: 999,
		ClientID:      "test-client",
	}

	authBytes := []byte("invalidformat")

	reqBuf := make([]byte, 4+len(authBytes))
	binary.BigEndian.PutUint32(reqBuf[0:4], uint32(len(authBytes)))
	copy(reqBuf[4:], authBytes)

	reader := bytes.NewReader(reqBuf)
	responseBytes, err := h.handleSaslAuthenticate(reader, header)
	require.NoError(t, err)

	respReader := bytes.NewReader(responseBytes)
	var corrID int32
	binary.Read(respReader, binary.BigEndian, &corrID)

	resp, err := protocol.DecodeSaslAuthenticateResponse(respReader, header.APIVersion)
	require.NoError(t, err)
	assert.Equal(t, protocol.SaslAuthenticationFailed, resp.ErrorCode)
	assert.NotNil(t, resp.ErrorMessage)
	assert.Contains(t, *resp.ErrorMessage, "invalid")
}
