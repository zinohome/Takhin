// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/logger"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

// Handler handles Kafka protocol requests
type Handler struct {
	config       *config.Config
	logger       *logger.Logger
	topicManager *topic.Manager
	backend      Backend
	coordinator  *coordinator.Coordinator
}

// New creates a new request handler with direct backend
func New(cfg *config.Config, topicMgr *topic.Manager) *Handler {
	coord := coordinator.NewCoordinator()
	coord.Start()
	
	return &Handler{
		config:       cfg,
		logger:       logger.Default().WithComponent("kafka-handler"),
		topicManager: topicMgr,
		backend:      NewDirectBackend(topicMgr),
		coordinator:  coord,
	}
}

// NewWithBackend creates a new request handler with custom backend
func NewWithBackend(cfg *config.Config, topicMgr *topic.Manager, backend Backend) *Handler {
	coord := coordinator.NewCoordinator()
	coord.Start()
	
	return &Handler{
		config:       cfg,
		logger:       logger.Default().WithComponent("kafka-handler"),
		topicManager: topicMgr,
		backend:      backend,
		coordinator:  coord,
	}
}

// HandleRequest processes a Kafka request and returns a response
func (h *Handler) HandleRequest(reqData []byte) ([]byte, error) {
	r := bytes.NewReader(reqData)

	// Decode request header
	header, err := protocol.DecodeRequestHeader(r)
	if err != nil {
		return nil, fmt.Errorf("decode request header: %w", err)
	}

	h.logger.Debug("received request",
		"api_key", header.APIKey,
		"api_version", header.APIVersion,
		"correlation_id", header.CorrelationID,
		"client_id", header.ClientID,
	)

	// Route to appropriate handler
	var response []byte
	switch header.APIKey {
	case protocol.ApiVersionsKey:
		response, err = h.handleApiVersions(r, header)
	case protocol.ProduceKey:
		response, err = h.handleProduce(r, header)
	case protocol.FetchKey:
		response, err = h.handleFetch(r, header)
	case protocol.MetadataKey:
		response, err = h.handleMetadata(r, header)
	case protocol.FindCoordinatorKey:
		response, err = h.handleFindCoordinator(r, header)
	case protocol.JoinGroupKey:
		response, err = h.handleJoinGroup(r, header)
	case protocol.SyncGroupKey:
		response, err = h.handleSyncGroup(r, header)
	case protocol.HeartbeatKey:
		response, err = h.handleHeartbeat(r, header)
	case protocol.OffsetCommitKey:
		response, err = h.handleOffsetCommit(r, header)
	case protocol.OffsetFetchKey:
		response, err = h.handleOffsetFetch(r, header)
	case protocol.LeaveGroupKey:
		response, err = h.handleLeaveGroup(r, header)
	default:
		return nil, fmt.Errorf("unsupported API key: %d", header.APIKey)
	}

	if err != nil {
		return nil, fmt.Errorf("handle request: %w", err)
	}

	return response, nil
}

// handleApiVersions handles ApiVersions requests
func (h *Handler) handleApiVersions(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Decode request
	_, err := protocol.DecodeApiVersionsRequest(r, header)
	if err != nil {
		return nil, fmt.Errorf("decode api versions request: %w", err)
	}

	// Create response
	resp := &protocol.ApiVersionsResponse{
		ErrorCode:   protocol.None,
		APIVersions: protocol.GetSupportedAPIVersions(),
	}

	// Encode response
	var buf bytes.Buffer

	// Write response header
	respHeader := &protocol.ResponseHeader{
		CorrelationID: header.CorrelationID,
	}
	if err := respHeader.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode response header: %w", err)
	}

	// Write response body
	if err := resp.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode api versions response: %w", err)
	}

	h.logger.Debug("api versions response",
		"correlation_id", header.CorrelationID,
		"versions_count", len(resp.APIVersions),
	)

	return buf.Bytes(), nil
}

// handleMetadata handles Metadata requests
func (h *Handler) handleMetadata(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Decode request
	req, err := protocol.DecodeMetadataRequest(r, header)
	if err != nil {
		return nil, fmt.Errorf("decode metadata request: %w", err)
	}

	h.logger.Debug("metadata request",
		"correlation_id", header.CorrelationID,
		"topics", req.Topics,
	)

	// Create response
	clusterID := "takhin-cluster"
	resp := &protocol.MetadataResponse{
		Brokers: []protocol.Broker{
			{
				NodeID: int32(h.config.Kafka.BrokerID),
				Host:   h.config.Kafka.AdvertisedHost,
				Port:   int32(h.config.Kafka.AdvertisedPort),
				Rack:   nil,
			},
		},
		ClusterID:     &clusterID,
		ControllerID:  int32(h.config.Kafka.BrokerID),
		TopicMetadata: []protocol.TopicMetadata{},
	}

	// TODO: Add actual topic metadata from storage
	// For now, return empty topic list

	// Encode response
	var buf bytes.Buffer

	// Write response header
	respHeader := &protocol.ResponseHeader{
		CorrelationID: header.CorrelationID,
	}
	if err := respHeader.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode response header: %w", err)
	}

	// Write response body
	if err := resp.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode metadata response: %w", err)
	}

	h.logger.Debug("metadata response",
		"correlation_id", header.CorrelationID,
		"brokers_count", len(resp.Brokers),
		"topics_count", len(resp.TopicMetadata),
	)

	return buf.Bytes(), nil
}

// handleProduce handles Produce requests
func (h *Handler) handleProduce(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeProduceRequest(r, header)
	if err != nil {
		return nil, fmt.Errorf("decode produce request: %w", err)
	}

	h.logger.Debug("produce request",
		"correlation_id", header.CorrelationID,
		"topics", len(req.TopicData),
		"acks", req.Acks,
	)

	resp := &protocol.ProduceResponse{
		Responses:      make([]protocol.ProduceTopicResponse, 0),
		ThrottleTimeMs: 0,
	}

	for _, topicData := range req.TopicData {
		_, exists := h.backend.GetTopic(topicData.TopicName)
		if !exists {
			// Auto-create topic with 1 partition
			if err := h.backend.CreateTopic(topicData.TopicName, 1); err != nil {
				h.logger.Error("create topic", "error", err, "topic", topicData.TopicName)
				continue
			}
		}

		topicResp := protocol.ProduceTopicResponse{
			TopicName:          topicData.TopicName,
			PartitionResponses: make([]protocol.ProducePartitionResponse, 0),
		}

		for _, partData := range topicData.PartitionData {
			offset, err := h.backend.Append(topicData.TopicName, partData.PartitionIndex, nil, partData.Records)

			partResp := protocol.ProducePartitionResponse{
				PartitionIndex: partData.PartitionIndex,
				LogAppendTime:  time.Now().UnixMilli(),
				LogStartOffset: 0,
			}

			if err != nil {
				partResp.ErrorCode = protocol.UnknownTopicOrPartition
				h.logger.Error("append to partition", "error", err)
			} else {
				partResp.ErrorCode = protocol.None
				partResp.BaseOffset = offset
			}

			topicResp.PartitionResponses = append(topicResp.PartitionResponses, partResp)
		}

		resp.Responses = append(resp.Responses, topicResp)
	}

	var buf bytes.Buffer
	respHeader := &protocol.ResponseHeader{
		CorrelationID: header.CorrelationID,
	}
	if err := respHeader.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode response header: %w", err)
	}

	if err := resp.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode produce response: %w", err)
	}

	h.logger.Debug("produce response", "correlation_id", header.CorrelationID)
	return buf.Bytes(), nil
}

// handleFetch handles Fetch requests
func (h *Handler) handleFetch(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeFetchRequest(r, header)
	if err != nil {
		return nil, fmt.Errorf("decode fetch request: %w", err)
	}

	h.logger.Debug("fetch request",
		"correlation_id", header.CorrelationID,
		"topics", len(req.Topics),
		"max_wait_ms", req.MaxWaitMs,
	)

	resp := &protocol.FetchResponse{
		ThrottleTimeMs: 0,
		ErrorCode:      protocol.None,
		SessionID:      0,
		Responses:      make([]protocol.FetchTopicResponse, 0),
	}

	for _, topicReq := range req.Topics {
		topic, exists := h.backend.GetTopic(topicReq.TopicName)
		if !exists {
			continue
		}

		topicResp := protocol.FetchTopicResponse{
			TopicName:          topicReq.TopicName,
			PartitionResponses: make([]protocol.FetchPartitionResponse, 0),
		}

		for _, partReq := range topicReq.Partitions {
			hwm, _ := topic.HighWaterMark(partReq.PartitionIndex)

			partResp := protocol.FetchPartitionResponse{
				PartitionIndex:   partReq.PartitionIndex,
				ErrorCode:        protocol.None,
				HighWatermark:    hwm,
				LastStableOffset: hwm,
				LogStartOffset:   0,
				Records:          []byte{},
			}

			// Read records if offset is valid
			if partReq.FetchOffset < hwm {
				record, err := topic.Read(partReq.PartitionIndex, partReq.FetchOffset)
				if err == nil && record != nil {
					partResp.Records = record.Value
				}
			}

			topicResp.PartitionResponses = append(topicResp.PartitionResponses, partResp)
		}

		resp.Responses = append(resp.Responses, topicResp)
	}

	var buf bytes.Buffer
	respHeader := &protocol.ResponseHeader{
		CorrelationID: header.CorrelationID,
	}
	if err := respHeader.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode response header: %w", err)
	}

	if err := resp.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode fetch response: %w", err)
	}

	h.logger.Debug("fetch response", "correlation_id", header.CorrelationID)

	return buf.Bytes(), nil
}

// handleFindCoordinator handles FindCoordinator requests
func (h *Handler) handleFindCoordinator(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Decode request
	req := &protocol.FindCoordinatorRequest{}
	if err := req.Decode(readAll(r)); err != nil {
		return nil, fmt.Errorf("decode find coordinator request: %w", err)
	}

	// Create response (always return this broker as coordinator)
	resp := &protocol.FindCoordinatorResponse{
		ErrorCode: int16(protocol.None),
		NodeID:    0, // Broker ID
		Host:      "localhost",
		Port:      9092,
	}

	return h.encodeResponse(header, resp)
}

// handleJoinGroup handles JoinGroup requests
func (h *Handler) handleJoinGroup(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Decode request
	req := &protocol.JoinGroupRequest{}
	if err := req.Decode(readAll(r)); err != nil {
		return nil, fmt.Errorf("decode join group request: %w", err)
	}

	// Convert protocols
	protocols := make([]coordinator.MemberProtocol, len(req.Protocols))
	for i, p := range req.Protocols {
		protocols[i] = coordinator.MemberProtocol{
			Name:     p.Name,
			Metadata: p.Metadata,
		}
	}

	// Join group
	group, member, needsRebalance, err := h.coordinator.JoinGroup(
		req.GroupID,
		req.MemberID,
		header.ClientID,
		"localhost",
		req.ProtocolType,
		protocols,
		req.SessionTimeout,
		req.RebalanceTimeout,
	)

	if err != nil {
		resp := &protocol.JoinGroupResponse{
			ErrorCode: int16(protocol.UnknownMemberID),
		}
		return h.encodeResponse(header, resp)
	}

	// Select protocol
	protocolName, _ := group.SelectProtocol()

	// Build response
	resp := &protocol.JoinGroupResponse{
		ErrorCode:    int16(protocol.None),
		GenerationID: group.Generation,
		ProtocolName: protocolName,
		LeaderID:     group.Leader,
		MemberID:     member.ID,
	}

	// If leader, include all members
	if member.ID == group.Leader {
		members := group.AllMembers()
		resp.Members = make([]protocol.JoinGroupMember, len(members))
		for i, m := range members {
			resp.Members[i] = protocol.JoinGroupMember{
				MemberID: m.ID,
				Metadata: m.Metadata,
			}
		}
	}

	// Trigger rebalance if needed
	if needsRebalance {
		group.PrepareRebalance()
	}

	return h.encodeResponse(header, resp)
}

// handleSyncGroup handles SyncGroup requests
func (h *Handler) handleSyncGroup(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Decode request
	req := &protocol.SyncGroupRequest{}
	if err := req.Decode(readAll(r)); err != nil {
		return nil, fmt.Errorf("decode sync group request: %w", err)
	}

	// Convert assignments
	assignments := make(map[string][]byte)
	for _, a := range req.Assignments {
		assignments[a.MemberID] = a.Assignment
	}

	// Sync group
	assignment, err := h.coordinator.SyncGroup(
		req.GroupID,
		req.MemberID,
		req.GenerationID,
		assignments,
	)

	resp := &protocol.SyncGroupResponse{
		ErrorCode:  int16(protocol.None),
		Assignment: assignment,
	}

	if err != nil {
		resp.ErrorCode = int16(protocol.IllegalGeneration)
	}

	return h.encodeResponse(header, resp)
}

// handleHeartbeat handles Heartbeat requests
func (h *Handler) handleHeartbeat(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Decode request
	req := &protocol.HeartbeatRequest{}
	if err := req.Decode(readAll(r)); err != nil {
		return nil, fmt.Errorf("decode heartbeat request: %w", err)
	}

	// Send heartbeat
	err := h.coordinator.Heartbeat(req.GroupID, req.MemberID, req.GenerationID)

	resp := &protocol.HeartbeatResponse{
		ErrorCode: int16(protocol.None),
	}

	if err != nil {
		resp.ErrorCode = int16(protocol.UnknownMemberID)
	}

	return h.encodeResponse(header, resp)
}

// handleOffsetCommit handles OffsetCommit requests
func (h *Handler) handleOffsetCommit(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Decode request
	req := &protocol.OffsetCommitRequest{}
	if err := req.Decode(readAll(r)); err != nil {
		return nil, fmt.Errorf("decode offset commit request: %w", err)
	}

	// Commit offsets
	resp := &protocol.OffsetCommitResponse{
		Topics: make([]protocol.OffsetCommitResponseTopic, len(req.Topics)),
	}

	for i, topic := range req.Topics {
		resp.Topics[i].Name = topic.Name
		resp.Topics[i].Partitions = make([]protocol.OffsetCommitResponsePartition, len(topic.Partitions))

		for j, partition := range topic.Partitions {
			err := h.coordinator.CommitOffset(
				req.GroupID,
				topic.Name,
				partition.PartitionIndex,
				partition.Offset,
				partition.Metadata,
			)

			resp.Topics[i].Partitions[j].PartitionIndex = partition.PartitionIndex
			if err != nil {
				resp.Topics[i].Partitions[j].ErrorCode = int16(protocol.UnknownTopicOrPartition)
			} else {
				resp.Topics[i].Partitions[j].ErrorCode = int16(protocol.None)
			}
		}
	}

	return h.encodeResponse(header, resp)
}

// handleOffsetFetch handles OffsetFetch requests
func (h *Handler) handleOffsetFetch(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Decode request
	req := &protocol.OffsetFetchRequest{}
	if err := req.Decode(readAll(r)); err != nil {
		return nil, fmt.Errorf("decode offset fetch request: %w", err)
	}

	// Fetch offsets
	resp := &protocol.OffsetFetchResponse{
		ErrorCode: int16(protocol.None),
	}

	if req.Topics == nil {
		// Fetch all topics - not implemented yet
		resp.Topics = []protocol.OffsetFetchResponseTopic{}
	} else {
		resp.Topics = make([]protocol.OffsetFetchResponseTopic, len(req.Topics))

		for i, topic := range req.Topics {
			resp.Topics[i].Name = topic.Name
			
			if topic.PartitionIndexes == nil {
				// Fetch all partitions - not implemented yet
				resp.Topics[i].Partitions = []protocol.OffsetFetchResponsePartition{}
			} else {
				resp.Topics[i].Partitions = make([]protocol.OffsetFetchResponsePartition, len(topic.PartitionIndexes))

				for j, partitionIdx := range topic.PartitionIndexes {
					offset, exists := h.coordinator.FetchOffset(req.GroupID, topic.Name, partitionIdx)

					resp.Topics[i].Partitions[j].PartitionIndex = partitionIdx
					if exists {
						resp.Topics[i].Partitions[j].Offset = offset.Offset
						resp.Topics[i].Partitions[j].Metadata = offset.Metadata
						resp.Topics[i].Partitions[j].ErrorCode = int16(protocol.None)
					} else {
						resp.Topics[i].Partitions[j].Offset = -1
						resp.Topics[i].Partitions[j].ErrorCode = int16(protocol.None)
					}
				}
			}
		}
	}

	return h.encodeResponse(header, resp)
}

// handleLeaveGroup handles LeaveGroup requests
func (h *Handler) handleLeaveGroup(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Decode request
	req := &protocol.LeaveGroupRequest{}
	if err := req.Decode(readAll(r)); err != nil {
		return nil, fmt.Errorf("decode leave group request: %w", err)
	}

	// Leave group
	var errorCode int16
	if req.MemberID != "" {
		// v0-v2: single member
		err := h.coordinator.LeaveGroup(req.GroupID, req.MemberID)
		if err != nil {
			errorCode = int16(protocol.UnknownMemberID)
		} else {
			errorCode = int16(protocol.None)
		}
	}

	resp := &protocol.LeaveGroupResponse{
		ErrorCode: errorCode,
	}

	// v3+: multiple members
	if len(req.Members) > 0 {
		resp.Members = make([]protocol.LeaveGroupMemberResponse, len(req.Members))
		for i, member := range req.Members {
			err := h.coordinator.LeaveGroup(req.GroupID, member.MemberID)
			resp.Members[i].MemberID = member.MemberID
			if err != nil {
				resp.Members[i].ErrorCode = int16(protocol.UnknownMemberID)
			} else {
				resp.Members[i].ErrorCode = int16(protocol.None)
			}
		}
	}

	return h.encodeResponse(header, resp)
}

// Helper functions

// readAll reads all remaining data from reader
func readAll(r io.Reader) []byte {
	data, _ := io.ReadAll(r)
	return data
}

// encodeResponse encodes a response with header
func (h *Handler) encodeResponse(header *protocol.RequestHeader, resp interface{}) ([]byte, error) {
	var buf bytes.Buffer

	// Encode response header
	respHeader := &protocol.ResponseHeader{
		CorrelationID: header.CorrelationID,
	}
	if err := respHeader.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode response header: %w", err)
	}

	// Encode response body
	var respBytes []byte
	switch r := resp.(type) {
	case *protocol.FindCoordinatorResponse:
		respBytes = r.Encode()
	case *protocol.JoinGroupResponse:
		respBytes = r.Encode()
	case *protocol.SyncGroupResponse:
		respBytes = r.Encode()
	case *protocol.HeartbeatResponse:
		respBytes = r.Encode()
	case *protocol.OffsetCommitResponse:
		respBytes = r.Encode()
	case *protocol.OffsetFetchResponse:
		respBytes = r.Encode()
	case *protocol.LeaveGroupResponse:
		respBytes = r.Encode()
	default:
		return nil, fmt.Errorf("unsupported response type")
	}

	buf.Write(respBytes)
	return buf.Bytes(), nil
}
