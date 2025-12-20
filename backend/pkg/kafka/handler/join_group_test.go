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

func TestHandleJoinGroup_NewMember(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	req := &protocol.JoinGroupRequest{
		GroupID:          "test-group",
		SessionTimeout:   10000,
		RebalanceTimeout: 30000,
		MemberID:         "", // New member
		ProtocolType:     "consumer",
		Protocols: []protocol.GroupProtocol{
			{
				Name:     "range",
				Metadata: []byte("metadata"),
			},
		},
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.JoinGroupKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	respBytes, err := h.handleJoinGroup(bytes.NewReader(req.Encode()), header)
	require.NoError(t, err)
	require.NotNil(t, respBytes)

	// Decode response (skip correlation ID - first 4 bytes)
	resp := &protocol.JoinGroupResponse{}
	err = resp.Decode(respBytes[4:])
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, int16(protocol.None), resp.ErrorCode)
	assert.NotEmpty(t, resp.MemberID)
	assert.Equal(t, int32(1), resp.GenerationID)
	assert.NotEmpty(t, resp.LeaderID)
	assert.Equal(t, "range", resp.ProtocolName)
}

func TestHandleJoinGroup_ExistingMember(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	// First join
	req := &protocol.JoinGroupRequest{
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

	header := &protocol.RequestHeader{
		APIKey:        protocol.JoinGroupKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}

	respBytes, err := h.handleJoinGroup(bytes.NewReader(req.Encode()), header)
	require.NoError(t, err)

	resp := &protocol.JoinGroupResponse{}
	err = resp.Decode(respBytes[4:])
	require.NoError(t, err)

	firstMemberID := resp.MemberID

	// Second join with same member ID
	req.MemberID = firstMemberID
	header.CorrelationID = 2

	respBytes2, err := h.handleJoinGroup(bytes.NewReader(req.Encode()), header)
	require.NoError(t, err)

	resp2 := &protocol.JoinGroupResponse{}
	err = resp2.Decode(respBytes2[4:])
	require.NoError(t, err)

	// Should return same member ID
	assert.Equal(t, int16(protocol.None), resp2.ErrorCode)
	assert.Equal(t, firstMemberID, resp2.MemberID)
}

func TestHandleJoinGroup_MultipleMembers(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	// First member joins
	req1 := &protocol.JoinGroupRequest{
		GroupID:          "test-group",
		SessionTimeout:   10000,
		RebalanceTimeout: 30000,
		MemberID:         "",
		ProtocolType:     "consumer",
		Protocols: []protocol.GroupProtocol{
			{
				Name:     "range",
				Metadata: []byte("metadata1"),
			},
		},
	}

	header1 := &protocol.RequestHeader{
		APIKey:        protocol.JoinGroupKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client-1",
	}

	respBytes1, err := h.handleJoinGroup(bytes.NewReader(req1.Encode()), header1)
	require.NoError(t, err)

	resp1 := &protocol.JoinGroupResponse{}
	err = resp1.Decode(respBytes1[4:])
	require.NoError(t, err)

	leaderID := resp1.MemberID

	// Second member joins
	req2 := &protocol.JoinGroupRequest{
		GroupID:          "test-group",
		SessionTimeout:   10000,
		RebalanceTimeout: 30000,
		MemberID:         "",
		ProtocolType:     "consumer",
		Protocols: []protocol.GroupProtocol{
			{
				Name:     "range",
				Metadata: []byte("metadata2"),
			},
		},
	}

	header2 := &protocol.RequestHeader{
		APIKey:        protocol.JoinGroupKey,
		APIVersion:    0,
		CorrelationID: 2,
		ClientID:      "test-client-2",
	}

	respBytes2, err := h.handleJoinGroup(bytes.NewReader(req2.Encode()), header2)
	require.NoError(t, err)

	resp2 := &protocol.JoinGroupResponse{}
	err = resp2.Decode(respBytes2[4:])
	require.NoError(t, err)

	// Both should be in same group with same leader
	assert.Equal(t, int16(protocol.None), resp1.ErrorCode)
	assert.Equal(t, int16(protocol.None), resp2.ErrorCode)
	assert.Equal(t, leaderID, resp1.LeaderID)
	assert.Equal(t, leaderID, resp2.LeaderID)
	assert.NotEqual(t, resp1.MemberID, resp2.MemberID)
}
