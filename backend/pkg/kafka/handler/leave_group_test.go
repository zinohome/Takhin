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

func TestHandleLeaveGroup(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	// First join a group
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

	memberID := joinResp.MemberID

	// Now leave the group
	leaveReq := &protocol.LeaveGroupRequest{
		GroupID:  "test-group",
		MemberID: memberID,
	}

	header := &protocol.RequestHeader{
		APIKey:        protocol.LeaveGroupKey,
		APIVersion:    0,
		CorrelationID: 2,
		ClientID:      "test-client",
	}

	respBytes, err := h.handleLeaveGroup(bytes.NewReader(leaveReq.Encode()), header)
	require.NoError(t, err)
	require.NotNil(t, respBytes)

	// Decode response
	resp := &protocol.LeaveGroupResponse{}
	err = resp.Decode(respBytes[4:])
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, int16(protocol.None), resp.ErrorCode)
}
