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

func TestHandleSyncGroup_Leader(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	// First join group
	joinReq := &protocol.JoinGroupRequest{
		GroupID:          "test-group",
		SessionTimeout:   10000,
		RebalanceTimeout: 30000,
		MemberID:         "",
		ProtocolType:     "consumer",
		Protocols: []protocol.GroupProtocol{
			{
				Name:     "range",
				Metadata: []byte("metadata"),
			},
		},
	}

	joinHeader := &protocol.RequestHeader{
		APIKey:        protocol.JoinGroupKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	joinRespBytes, err := h.handleJoinGroup(bytes.NewReader(joinReq.Encode()), joinHeader)
	require.NoError(t, err)

	joinResp := &protocol.JoinGroupResponse{}
	err = joinResp.Decode(joinRespBytes[4:])
	require.NoError(t, err)

	// Now sync group (as leader with assignments)
	// Note: Generation increments due to PrepareRebalance() in handleJoinGroup
	req := &protocol.SyncGroupRequest{
		GroupID:      "test-group",
		MemberID:     joinResp.MemberID,
		GenerationID: joinResp.GenerationID + 1, // Generation was incremented by PrepareRebalance
		Assignments: []protocol.SyncGroupAssignment{
			{
				MemberID:   joinResp.MemberID,
				Assignment: []byte("assignment-data"),
			},
		},
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.SyncGroupKey,
		APIVersion:    0,
		CorrelationID: 2,
		ClientID:      "test-client",
	}

	respBytes, err := h.handleSyncGroup(bytes.NewReader(req.Encode()), header)
	require.NoError(t, err)
	require.NotNil(t, respBytes)

	// Decode response (skip correlation ID - first 4 bytes)
	resp := &protocol.SyncGroupResponse{}
	err = resp.Decode(respBytes[4:])
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, int16(protocol.None), resp.ErrorCode)
	assert.Equal(t, []byte("assignment-data"), resp.Assignment)
}

func TestHandleSyncGroup_Follower(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	// First join group
	joinReq := &protocol.JoinGroupRequest{
		GroupID:          "test-group",
		SessionTimeout:   10000,
		RebalanceTimeout: 30000,
		MemberID:         "",
		ProtocolType:     "consumer",
		Protocols: []protocol.GroupProtocol{
			{
				Name:     "range",
				Metadata: []byte("metadata"),
			},
		},
	}

	joinHeader := &protocol.RequestHeader{
		APIKey:        protocol.JoinGroupKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	joinRespBytes, err := h.handleJoinGroup(bytes.NewReader(joinReq.Encode()), joinHeader)
	require.NoError(t, err)

	joinResp := &protocol.JoinGroupResponse{}
	err = joinResp.Decode(joinRespBytes[4:])
	require.NoError(t, err)

	// Sync as follower (empty assignments)
	// Note: Generation increments due to PrepareRebalance() in handleJoinGroup
	req := &protocol.SyncGroupRequest{
		GroupID:      "test-group",
		MemberID:     joinResp.MemberID,
		GenerationID: joinResp.GenerationID + 1,        // Generation was incremented by PrepareRebalance
		Assignments:  []protocol.SyncGroupAssignment{}, // Follower sends empty
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.SyncGroupKey,
		APIVersion:    0,
		CorrelationID: 2,
		ClientID:      "test-client",
	}

	respBytes, err := h.handleSyncGroup(bytes.NewReader(req.Encode()), header)
	require.NoError(t, err)

	// Decode response
	resp := &protocol.SyncGroupResponse{}
	err = resp.Decode(respBytes[4:])
	require.NoError(t, err)

	// Should succeed with empty assignment
	assert.Equal(t, int16(protocol.None), resp.ErrorCode)
}

func TestHandleSyncGroup_UnknownMember(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	// Try to sync without joining
	req := &protocol.SyncGroupRequest{
		GroupID:      "test-group",
		MemberID:     "unknown-member",
		GenerationID: 1,
		Assignments:  []protocol.SyncGroupAssignment{},
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.SyncGroupKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	respBytes, err := h.handleSyncGroup(bytes.NewReader(req.Encode()), header)
	require.NoError(t, err)

	// Decode response
	resp := &protocol.SyncGroupResponse{}
	err = resp.Decode(respBytes[4:])
	require.NoError(t, err)

	// Should return error
	assert.Equal(t, int16(protocol.IllegalGeneration), resp.ErrorCode)
}
