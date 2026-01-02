// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/logger"
	"github.com/takhin-data/takhin/pkg/replication"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

// Handler handles Kafka protocol requests
type Handler struct {
	config            *config.Config
	logger            *logger.Logger
	topicManager      *topic.Manager
	backend           Backend
	coordinator       *coordinator.Coordinator
	producerIDManager *ProducerIDManager
	txnCoordinator    *TransactionCoordinator
	replicaAssigner   *replication.ReplicaAssigner // Replica assignment strategy
	produceWaiter     *ProduceWaiter               // Waits for ISR acknowledgment
}

// New creates a new request handler with direct backend
func New(cfg *config.Config, topicMgr *topic.Manager) *Handler {
	coord := coordinator.NewCoordinator()
	coord.Start()

	// Build broker list for replica assignment
	brokers := buildBrokerList(cfg)

	return &Handler{
		config:            cfg,
		logger:            logger.Default().WithComponent("kafka-handler"),
		topicManager:      topicMgr,
		backend:           NewDirectBackend(topicMgr),
		coordinator:       coord,
		producerIDManager: NewProducerIDManager(),
		txnCoordinator:    NewTransactionCoordinator(),
		replicaAssigner:   replication.NewReplicaAssigner(brokers),
		produceWaiter:     NewProduceWaiter(),
	}
}

// NewWithBackend creates a new request handler with custom backend
func NewWithBackend(cfg *config.Config, topicMgr *topic.Manager, backend Backend) *Handler {
	coord := coordinator.NewCoordinator()
	coord.Start()

	// Build broker list for replica assignment
	brokers := buildBrokerList(cfg)

	return &Handler{
		config:            cfg,
		logger:            logger.Default().WithComponent("kafka-handler"),
		topicManager:      topicMgr,
		backend:           backend,
		coordinator:       coord,
		producerIDManager: NewProducerIDManager(),
		txnCoordinator:    NewTransactionCoordinator(),
		replicaAssigner:   replication.NewReplicaAssigner(brokers),
		produceWaiter:     NewProduceWaiter(),
	}
}

// Close cleans up resources held by the handler
func (h *Handler) Close() error {
	if h.produceWaiter != nil {
		h.produceWaiter.Close()
	}
	return nil
}

// buildBrokerList constructs the broker list for replica assignment
// Priority: 1) ClusterBrokers config, 2) current broker only
func buildBrokerList(cfg *config.Config) []int32 {
	// If cluster brokers are configured, use them
	if len(cfg.Kafka.ClusterBrokers) > 0 {
		brokers := make([]int32, len(cfg.Kafka.ClusterBrokers))
		for i, brokerID := range cfg.Kafka.ClusterBrokers {
			brokers[i] = int32(brokerID)
		}
		return brokers
	}

	// Default: single broker (current broker)
	return []int32{int32(cfg.Kafka.BrokerID)}
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
	case protocol.SaslHandshakeKey:
		response, err = h.handleSaslHandshake(r, header)
	case protocol.SaslAuthenticateKey:
		response, err = h.handleSaslAuthenticate(r, header)
	case protocol.ProduceKey:
		response, err = h.handleProduce(r, header)
	case protocol.FetchKey:
		response, err = h.handleFetch(r, header)
	case protocol.ListOffsetsKey:
		response, err = h.handleListOffsets(r, header)
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
	case protocol.CreateTopicsKey:
		response, err = h.handleCreateTopics(r, header)
	case protocol.DeleteTopicsKey:
		response, err = h.handleDeleteTopics(r, header)
	case protocol.DeleteRecordsKey:
		response, err = h.handleDeleteRecords(r, header)
	case protocol.DescribeLogDirsKey:
		response, err = h.handleDescribeLogDirs(r, header)
	case protocol.DescribeConfigsKey:
		response, err = h.handleDescribeConfigs(r, header)
	case protocol.DescribeGroupsKey:
		response, err = h.handleDescribeGroups(r, header)
	case protocol.ListGroupsKey:
		response, err = h.handleListGroups(r, header)
	case protocol.InitProducerIDKey:
		response, err = h.handleInitProducerID(r, header)
	case protocol.AddPartitionsToTxnKey:
		response, err = h.handleAddPartitionsToTxn(r, header)
	case protocol.AddOffsetsToTxnKey:
		response, err = h.handleAddOffsetsToTxn(r, header)
	case protocol.EndTxnKey:
		response, err = h.handleEndTxn(r, header)
	case protocol.WriteTxnMarkersKey:
		response, err = h.handleWriteTxnMarkers(r, header)
	case protocol.TxnOffsetCommitKey:
		response, err = h.handleTxnOffsetCommit(r, header)
	case protocol.AlterConfigsKey:
		response, err = h.handleAlterConfigs(r, header)
	default:
		return nil, fmt.Errorf("unsupported API key: %d", header.APIKey)
	}

	if err != nil {
		return nil, fmt.Errorf("handle request: %w", err)
	}

	return response, nil
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

	// Add actual topic metadata from storage
	if req.Topics == nil || len(req.Topics) == 0 {
		// Return all topics
		allTopics := h.topicManager.ListTopics()
		for _, topicName := range allTopics {
			topic, exists := h.topicManager.GetTopic(topicName)
			if !exists {
				continue
			}
			topicMeta := protocol.TopicMetadata{
				ErrorCode:         protocol.None,
				TopicName:         topicName,
				IsInternal:        false,
				PartitionMetadata: make([]protocol.PartitionMetadata, 0),
			}
			// Add partition metadata
			for partitionID := range topic.Partitions {
				// Get replica assignment for this partition
				replicas := topic.GetReplicas(partitionID)
				isr := topic.GetISR(partitionID)

				// Default to current broker if no assignment
				if replicas == nil || len(replicas) == 0 {
					replicas = []int32{int32(h.config.Kafka.BrokerID)}
				}
				if isr == nil || len(isr) == 0 {
					isr = []int32{int32(h.config.Kafka.BrokerID)}
				}

				partMeta := protocol.PartitionMetadata{
					ErrorCode:       protocol.None,
					PartitionID:     partitionID,
					Leader:          replicas[0], // First replica is leader
					Replicas:        replicas,
					ISR:             isr,
					OfflineReplicas: []int32{},
				}
				topicMeta.PartitionMetadata = append(topicMeta.PartitionMetadata, partMeta)
			}
			resp.TopicMetadata = append(resp.TopicMetadata, topicMeta)
		}
	} else {
		// Return only requested topics
		for _, topicName := range req.Topics {
			topic, exists := h.topicManager.GetTopic(topicName)
			if !exists {
				topicMeta := protocol.TopicMetadata{
					ErrorCode:         protocol.UnknownTopicOrPartition,
					TopicName:         topicName,
					IsInternal:        false,
					PartitionMetadata: []protocol.PartitionMetadata{},
				}
				resp.TopicMetadata = append(resp.TopicMetadata, topicMeta)
				continue
			}
			topicMeta := protocol.TopicMetadata{
				ErrorCode:         protocol.None,
				TopicName:         topicName,
				IsInternal:        false,
				PartitionMetadata: make([]protocol.PartitionMetadata, 0),
			}
			// Add partition metadata
			for partitionID := range topic.Partitions {
				// Get replica assignment for this partition
				replicas := topic.GetReplicas(partitionID)
				isr := topic.GetISR(partitionID)

				// Default to current broker if no assignment
				if replicas == nil || len(replicas) == 0 {
					replicas = []int32{int32(h.config.Kafka.BrokerID)}
				}
				if isr == nil || len(isr) == 0 {
					isr = []int32{int32(h.config.Kafka.BrokerID)}
				}

				partMeta := protocol.PartitionMetadata{
					ErrorCode:       protocol.None,
					PartitionID:     partitionID,
					Leader:          replicas[0], // First replica is leader
					Replicas:        replicas,
					ISR:             isr,
					OfflineReplicas: []int32{},
				}
				topicMeta.PartitionMetadata = append(topicMeta.PartitionMetadata, partMeta)
			}
			resp.TopicMetadata = append(resp.TopicMetadata, topicMeta)
		}
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
		topic, exists := h.backend.GetTopic(topicData.TopicName)
		if !exists {
			// Auto-create topic with 1 partition
			if err := h.backend.CreateTopic(topicData.TopicName, 1); err != nil {
				h.logger.Error("create topic", "error", err, "topic", topicData.TopicName)
				continue
			}
			topic, _ = h.backend.GetTopic(topicData.TopicName)
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

				// Handle acks=-1: wait for all ISR replicas
				if req.Acks == -1 && topic != nil {
					isr := topic.GetISR(partData.PartitionIndex)
					isrSize := len(isr)

					// Log ISR status
					h.logger.Debug("acks=-1 produce",
						"topic", topicData.TopicName,
						"partition", partData.PartitionIndex,
						"offset", offset,
						"isr_size", isrSize,
						"isr", isr,
					)

					// Check min ISR requirement (default to 1)
					minISR := 1
					if isrSize < minISR {
						partResp.ErrorCode = protocol.NotEnoughReplicas
						h.logger.Warn("not enough replicas in ISR",
							"topic", topicData.TopicName,
							"partition", partData.PartitionIndex,
							"isr_size", isrSize,
							"min_isr", minISR,
						)
					} else {
						// Wait for ISR acknowledgment
						timeout := time.Duration(req.TimeoutMs) * time.Millisecond
						if timeout == 0 {
							timeout = 30 * time.Second // default timeout
						}

						ctx := context.Background()
						err := h.produceWaiter.WaitForAck(ctx, topicData.TopicName, partData.PartitionIndex, offset, req.Acks, timeout)
						if err != nil {
							// Timeout or error
							partResp.ErrorCode = protocol.RequestTimedOut
							h.logger.Error("wait for ISR ack failed",
								"error", err,
								"topic", topicData.TopicName,
								"partition", partData.PartitionIndex,
								"offset", offset,
							)
						} else {
							h.logger.Debug("ISR acknowledgment received",
								"topic", topicData.TopicName,
								"partition", partData.PartitionIndex,
								"offset", offset,
							)
						}
					}
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

	// Check if this is a replica fetch (has ReplicaID >= 0)
	isReplicaFetch := req.ReplicaID >= 0

	h.logger.Debug("fetch request",
		"correlation_id", header.CorrelationID,
		"topics", len(req.Topics),
		"max_wait_ms", req.MaxWaitMs,
		"replica_id", req.ReplicaID,
		"is_replica_fetch", isReplicaFetch,
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

			// If this is a replica fetch, update follower LEO
			if isReplicaFetch && req.ReplicaID != int32(h.config.Kafka.BrokerID) {
				// Follower's LEO is their fetch offset (where they want to read next)
				followerLEO := partReq.FetchOffset
				topic.UpdateFollowerLEO(partReq.PartitionIndex, req.ReplicaID, followerLEO)

				// Update ISR based on current follower state
				leaderLEO := hwm
				newISR := topic.UpdateISR(partReq.PartitionIndex, leaderLEO)

				h.logger.Debug("updated follower state",
					"topic", topicReq.TopicName,
					"partition", partReq.PartitionIndex,
					"follower_id", req.ReplicaID,
					"follower_leo", followerLEO,
					"leader_leo", leaderLEO,
					"isr", newISR,
				)

				// Notify any waiting produce requests that HWM may have advanced
				// HWM is the minimum LEO among all ISR members
				currentHWM, _ := topic.HighWaterMark(partReq.PartitionIndex)
				h.produceWaiter.NotifyHWMAdvanced(topicReq.TopicName, partReq.PartitionIndex, currentHWM)

				h.logger.Debug("notified HWM advancement",
					"topic", topicReq.TopicName,
					"partition", partReq.PartitionIndex,
					"hwm", currentHWM,
				)
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

	// Assign a member ID if client didn't provide one
	if req.MemberID == "" {
		req.MemberID = fmt.Sprintf("%s-%d", header.ClientID, time.Now().UnixNano())
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

	// Trigger rebalance if needed: move members to pending, bump generation, then stabilize
	if needsRebalance {
		group.PrepareRebalance()
		group.CompleteRebalance()
	}

	// Select protocol (after stabilization)
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
		// Fetch all topics for this group
		allTopics := h.coordinator.GetGroupTopics(req.GroupID)
		resp.Topics = make([]protocol.OffsetFetchResponseTopic, 0, len(allTopics))
		for topicName, partitions := range allTopics {
			topicResp := protocol.OffsetFetchResponseTopic{
				Name:       topicName,
				Partitions: make([]protocol.OffsetFetchResponsePartition, 0, len(partitions)),
			}
			for _, partitionIdx := range partitions {
				offset, exists := h.coordinator.FetchOffset(req.GroupID, topicName, partitionIdx)
				partResp := protocol.OffsetFetchResponsePartition{
					PartitionIndex: partitionIdx,
					ErrorCode:      int16(protocol.None),
				}
				if exists {
					partResp.Offset = offset.Offset
					partResp.Metadata = offset.Metadata
				} else {
					partResp.Offset = -1
				}
				topicResp.Partitions = append(topicResp.Partitions, partResp)
			}
			resp.Topics = append(resp.Topics, topicResp)
		}
	} else {
		resp.Topics = make([]protocol.OffsetFetchResponseTopic, len(req.Topics))

		for i, topic := range req.Topics {
			resp.Topics[i].Name = topic.Name

			if topic.PartitionIndexes == nil {
				// Fetch all partitions for this topic
				allPartitions := h.coordinator.GetTopicPartitions(req.GroupID, topic.Name)
				resp.Topics[i].Partitions = make([]protocol.OffsetFetchResponsePartition, 0, len(allPartitions))
				for _, partitionIdx := range allPartitions {
					offset, exists := h.coordinator.FetchOffset(req.GroupID, topic.Name, partitionIdx)
					partResp := protocol.OffsetFetchResponsePartition{
						PartitionIndex: partitionIdx,
						ErrorCode:      int16(protocol.None),
					}
					if exists {
						partResp.Offset = offset.Offset
						partResp.Metadata = offset.Metadata
					} else {
						partResp.Offset = -1
					}
					resp.Topics[i].Partitions = append(resp.Topics[i].Partitions, partResp)
				}
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

// handleCreateTopics handles CreateTopics requests
func (h *Handler) handleCreateTopics(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeCreateTopicsRequest(r, header)
	if err != nil {
		return nil, fmt.Errorf("decode create topics request: %w", err)
	}

	h.logger.Debug("create topics request",
		"correlation_id", header.CorrelationID,
		"topics", len(req.Topics),
		"timeout_ms", req.TimeoutMs,
		"validate_only", req.ValidateOnly,
	)

	resp := &protocol.CreateTopicsResponse{
		ThrottleTimeMs: 0,
		Topics:         make([]protocol.CreatableTopicResult, 0, len(req.Topics)),
	}

	for _, topic := range req.Topics {
		result := protocol.CreatableTopicResult{
			Name:              topic.Name,
			NumPartitions:     topic.NumPartitions,
			ReplicationFactor: topic.ReplicationFactor,
			Configs:           topic.Configs,
		}

		// Validate parameters
		if topic.NumPartitions <= 0 {
			result.ErrorCode = protocol.InvalidRequest
			errMsg := "num_partitions must be greater than 0"
			result.ErrorMessage = &errMsg
			resp.Topics = append(resp.Topics, result)
			continue
		}

		// Check if topic already exists
		if _, exists := h.backend.GetTopic(topic.Name); exists {
			result.ErrorCode = protocol.TopicAlreadyExists
			errMsg := "topic already exists"
			result.ErrorMessage = &errMsg
			resp.Topics = append(resp.Topics, result)
			continue
		}

		// Validate only mode - don't actually create
		if req.ValidateOnly {
			result.ErrorCode = protocol.None
			resp.Topics = append(resp.Topics, result)
			continue
		}

		// Create topic
		rf := topic.ReplicationFactor
		if rf <= 0 {
			rf = h.config.Replication.DefaultReplicationFactor
		}

		err := h.backend.CreateTopic(topic.Name, topic.NumPartitions)
		if err != nil {
			result.ErrorCode = protocol.InvalidRequest
			errMsg := err.Error()
			result.ErrorMessage = &errMsg
			h.logger.Error("create topic", "error", err, "topic", topic.Name)
		} else {
			if createdTopic, ok := h.backend.GetTopic(topic.Name); ok {
				createdTopic.SetReplicationFactor(rf)

				// Assign replicas using ReplicaAssigner
				assignments, err := h.replicaAssigner.AssignReplicas(topic.NumPartitions, rf)
				if err != nil {
					h.logger.Error("assign replicas", "error", err, "topic", topic.Name)
				} else {
					// Store replica assignments in topic metadata
					for partitionID, replicas := range assignments {
						createdTopic.SetReplicas(partitionID, replicas)
					}
				}
			}
			result.ErrorCode = protocol.None
			h.logger.Info("created topic",
				"topic", topic.Name,
				"partitions", topic.NumPartitions,
				"replication_factor", rf,
			)
		}

		resp.Topics = append(resp.Topics, result)
	}

	var buf bytes.Buffer
	respHeader := &protocol.ResponseHeader{
		CorrelationID: header.CorrelationID,
	}
	if err := respHeader.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode response header: %w", err)
	}

	if err := resp.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode create topics response: %w", err)
	}

	return buf.Bytes(), nil
}

// handleDeleteTopics handles DeleteTopics requests
func (h *Handler) handleDeleteTopics(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeDeleteTopicsRequest(r, header)
	if err != nil {
		return nil, fmt.Errorf("decode delete topics request: %w", err)
	}

	h.logger.Debug("delete topics request",
		"correlation_id", header.CorrelationID,
		"topics", len(req.TopicNames),
		"timeout_ms", req.TimeoutMs,
	)

	resp := &protocol.DeleteTopicsResponse{
		ThrottleTimeMs: 0,
		Responses:      make([]protocol.DeletableTopicResult, 0, len(req.TopicNames)),
	}

	for _, topicName := range req.TopicNames {
		result := protocol.DeletableTopicResult{
			Name: topicName,
		}

		// Check if topic exists
		if _, exists := h.backend.GetTopic(topicName); !exists {
			result.ErrorCode = protocol.UnknownTopicOrPartition
			errMsg := "unknown topic"
			result.ErrorMessage = &errMsg
			resp.Responses = append(resp.Responses, result)
			continue
		}

		// Delete topic
		err := h.backend.DeleteTopic(topicName)
		if err != nil {
			result.ErrorCode = protocol.InvalidRequest
			errMsg := err.Error()
			result.ErrorMessage = &errMsg
			h.logger.Error("delete topic", "error", err, "topic", topicName)
		} else {
			result.ErrorCode = protocol.None
			h.logger.Info("deleted topic", "topic", topicName)
		}

		resp.Responses = append(resp.Responses, result)
	}

	var buf bytes.Buffer
	respHeader := &protocol.ResponseHeader{
		CorrelationID: header.CorrelationID,
	}
	if err := respHeader.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode response header: %w", err)
	}

	if err := resp.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode delete topics response: %w", err)
	}

	return buf.Bytes(), nil
}

// handleDescribeConfigs handles DescribeConfigs requests
func (h *Handler) handleDescribeConfigs(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeDescribeConfigsRequest(r, header)
	if err != nil {
		return nil, fmt.Errorf("decode describe configs request: %w", err)
	}

	h.logger.Debug("describe configs request",
		"correlation_id", header.CorrelationID,
		"resources", len(req.Resources),
	)

	resp := &protocol.DescribeConfigsResponse{
		ThrottleTimeMs: 0,
		Results:        make([]protocol.DescribeConfigsResult, 0, len(req.Resources)),
	}

	for _, resource := range req.Resources {
		result := protocol.DescribeConfigsResult{
			ResourceType: resource.ResourceType,
			ResourceName: resource.ResourceName,
			Configs:      make([]protocol.DescribeConfigsEntry, 0),
		}

		// Only support topic configs for now
		if resource.ResourceType != protocol.ResourceTypeTopic {
			result.ErrorCode = protocol.InvalidRequest
			errMsg := "only topic configs are supported"
			result.ErrorMessage = &errMsg
			resp.Results = append(resp.Results, result)
			continue
		}

		// Check if topic exists
		if _, exists := h.backend.GetTopic(resource.ResourceName); !exists {
			result.ErrorCode = protocol.UnknownTopicOrPartition
			errMsg := "unknown topic"
			result.ErrorMessage = &errMsg
			resp.Results = append(resp.Results, result)
			continue
		}

		// Return default topic configurations
		defaultConfigs := []protocol.DescribeConfigsEntry{
			{
				Name:        "compression.type",
				Value:       stringPtr("producer"),
				ReadOnly:    false,
				IsDefault:   true,
				IsSensitive: false,
			},
			{
				Name:        "cleanup.policy",
				Value:       stringPtr("delete"),
				ReadOnly:    false,
				IsDefault:   true,
				IsSensitive: false,
			},
			{
				Name:        "retention.ms",
				Value:       stringPtr("604800000"), // 7 days
				ReadOnly:    false,
				IsDefault:   true,
				IsSensitive: false,
			},
			{
				Name:        "segment.ms",
				Value:       stringPtr("604800000"), // 7 days
				ReadOnly:    false,
				IsDefault:   true,
				IsSensitive: false,
			},
		}

		// Filter by requested config names if specified
		if len(resource.ConfigNames) > 0 {
			filteredConfigs := make([]protocol.DescribeConfigsEntry, 0)
			for _, requestedName := range resource.ConfigNames {
				for _, config := range defaultConfigs {
					if config.Name == requestedName {
						filteredConfigs = append(filteredConfigs, config)
						break
					}
				}
			}
			result.Configs = filteredConfigs
		} else {
			result.Configs = defaultConfigs
		}

		result.ErrorCode = protocol.None
		resp.Results = append(resp.Results, result)
	}

	var buf bytes.Buffer
	respHeader := &protocol.ResponseHeader{
		CorrelationID: header.CorrelationID,
	}
	if err := respHeader.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode response header: %w", err)
	}

	if err := resp.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode describe configs response: %w", err)
	}

	return buf.Bytes(), nil
}

// handleListOffsets handles ListOffsets requests
func (h *Handler) handleListOffsets(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Decode request
	reqData, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}

	req, err := protocol.DecodeListOffsetsRequest(reqData, header.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("decode list offsets request: %w", err)
	}

	h.logger.Debug("handling list offsets request",
		"replica_id", req.ReplicaID,
		"isolation_level", req.IsolationLevel,
		"topics_count", len(req.Topics),
	)

	// Build response
	resp := &protocol.ListOffsetsResponse{
		ThrottleTimeMs: 0,
		Topics:         make([]protocol.ListOffsetsTopicResponse, 0, len(req.Topics)),
	}

	for _, topic := range req.Topics {
		topicResp := protocol.ListOffsetsTopicResponse{
			Name:       topic.Name,
			Partitions: make([]protocol.ListOffsetsPartitionResponse, 0, len(topic.Partitions)),
		}

		// Get topic from backend
		t, exists := h.backend.GetTopic(topic.Name)
		if !exists {
			// Topic不存在，返回错误
			for _, part := range topic.Partitions {
				topicResp.Partitions = append(topicResp.Partitions, protocol.ListOffsetsPartitionResponse{
					PartitionIndex: part.PartitionIndex,
					ErrorCode:      protocol.UnknownTopicOrPartition,
					Timestamp:      -1,
					Offset:         -1,
					LeaderEpoch:    -1,
				})
			}
			resp.Topics = append(resp.Topics, topicResp)
			continue
		}

		// Process each partition
		for _, part := range topic.Partitions {
			partResp := protocol.ListOffsetsPartitionResponse{
				PartitionIndex: part.PartitionIndex,
				LeaderEpoch:    0, // 简化实现，暂不支持 leader epoch
			}

			var offset int64
			var timestamp int64

			switch part.Timestamp {
			case protocol.TimestampEarliest:
				// 查询最早的 offset
				offset, err = t.GetEarliestOffset(part.PartitionIndex)
				if err != nil {
					partResp.ErrorCode = protocol.UnknownTopicOrPartition
					partResp.Offset = -1
					partResp.Timestamp = -1
				} else {
					partResp.ErrorCode = protocol.None
					partResp.Offset = offset
					partResp.Timestamp = 0 // earliest 没有具体时间戳
				}

			case protocol.TimestampLatest:
				// 查询最新的 offset (HWM)
				offset, err = t.GetLatestOffset(part.PartitionIndex)
				if err != nil {
					partResp.ErrorCode = protocol.UnknownTopicOrPartition
					partResp.Offset = -1
					partResp.Timestamp = -1
				} else {
					partResp.ErrorCode = protocol.None
					partResp.Offset = offset
					partResp.Timestamp = -1 // latest 使用 -1
				}

			default:
				// 查询特定时间戳的 offset
				offset, timestamp, err = t.GetOffsetByTimestamp(part.PartitionIndex, part.Timestamp)
				if err != nil {
					partResp.ErrorCode = protocol.UnknownTopicOrPartition
					partResp.Offset = -1
					partResp.Timestamp = -1
				} else {
					partResp.ErrorCode = protocol.None
					partResp.Offset = offset
					partResp.Timestamp = timestamp
				}
			}

			h.logger.Debug("list offsets for partition",
				"topic", topic.Name,
				"partition", part.PartitionIndex,
				"request_timestamp", part.Timestamp,
				"response_offset", partResp.Offset,
				"response_timestamp", partResp.Timestamp,
			)

			topicResp.Partitions = append(topicResp.Partitions, partResp)
		}

		resp.Topics = append(resp.Topics, topicResp)
	}

	// Encode response
	respData := protocol.EncodeListOffsetsResponse(resp, header.APIVersion)

	// Add response header
	var buf bytes.Buffer
	respHeader := &protocol.ResponseHeader{
		CorrelationID: header.CorrelationID,
	}
	if err := respHeader.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode response header: %w", err)
	}

	buf.Write(respData)
	return buf.Bytes(), nil
}

// handleDescribeGroups handles DescribeGroups requests
func (h *Handler) handleDescribeGroups(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Decode request
	reqData, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}

	req, err := protocol.DecodeDescribeGroupsRequest(reqData, header.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("decode describe groups request: %w", err)
	}

	h.logger.Debug("handling describe groups request",
		"groups_count", len(req.Groups),
	)

	// Build response
	resp := &protocol.DescribeGroupsResponse{
		ThrottleTimeMs: 0,
		Groups:         make([]protocol.DescribedGroup, 0, len(req.Groups)),
	}

	for _, groupID := range req.Groups {
		group, exists := h.coordinator.GetGroup(groupID)
		if !exists {
			// 组不存在
			resp.Groups = append(resp.Groups, protocol.DescribedGroup{
				ErrorCode:    protocol.GroupIDNotFound,
				GroupID:      groupID,
				GroupState:   "Dead",
				ProtocolType: "",
				ProtocolData: "",
				Members:      []protocol.DescribedGroupMember{},
			})
			continue
		}

		// 构建成员列表
		members := make([]protocol.DescribedGroupMember, 0)
		for memberID, member := range group.Members {
			members = append(members, protocol.DescribedGroupMember{
				MemberID:         memberID,
				GroupInstanceID:  nil,
				ClientID:         member.ClientID,
				ClientHost:       member.ClientHost,
				MemberMetadata:   member.Metadata,
				MemberAssignment: member.Assignment,
			})
		}

		resp.Groups = append(resp.Groups, protocol.DescribedGroup{
			ErrorCode:            protocol.None,
			GroupID:              groupID,
			GroupState:           string(group.State),
			ProtocolType:         "consumer",
			ProtocolData:         group.ProtocolName,
			Members:              members,
			AuthorizedOperations: -2147483648, // 所有操作
		})
	}

	// Encode response
	respData := protocol.EncodeDescribeGroupsResponse(resp, header.APIVersion)

	// Add response header
	var buf bytes.Buffer
	respHeader := &protocol.ResponseHeader{
		CorrelationID: header.CorrelationID,
	}
	if err := respHeader.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode response header: %w", err)
	}

	buf.Write(respData)
	return buf.Bytes(), nil
}

// handleListGroups handles ListGroups requests
func (h *Handler) handleListGroups(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Decode request (empty body)
	_, err := protocol.DecodeListGroupsRequest(nil, header.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("decode list groups request: %w", err)
	}

	h.logger.Debug("handling list groups request")

	// Build response
	resp := &protocol.ListGroupsResponse{
		ThrottleTimeMs: 0,
		ErrorCode:      protocol.None,
		Groups:         make([]protocol.ListedGroup, 0),
	}

	// 获取所有组
	allGroups := h.coordinator.GetAllGroups()
	for groupID, group := range allGroups {
		resp.Groups = append(resp.Groups, protocol.ListedGroup{
			GroupID:      groupID,
			ProtocolType: group.ProtocolType,
		})
	}

	// Encode response
	respData := protocol.EncodeListGroupsResponse(resp, header.APIVersion)

	// Add response header
	var buf bytes.Buffer
	respHeader := &protocol.ResponseHeader{
		CorrelationID: header.CorrelationID,
	}
	if err := respHeader.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode response header: %w", err)
	}

	buf.Write(respData)
	return buf.Bytes(), nil
}

// handleDeleteRecords handles DeleteRecords requests
func (h *Handler) handleDeleteRecords(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Decode request
	reqData, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}

	req, err := protocol.DecodeDeleteRecordsRequest(reqData, header.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("decode delete records request: %w", err)
	}

	h.logger.Debug("handling delete records request",
		"topics_count", len(req.Topics),
		"timeout_ms", req.TimeoutMs,
	)

	// Build response
	resp := &protocol.DeleteRecordsResponse{
		ThrottleTimeMs: 0,
		Topics:         make([]protocol.DeleteRecordsTopicResponse, 0, len(req.Topics)),
	}

	for _, topic := range req.Topics {
		topicResp := protocol.DeleteRecordsTopicResponse{
			Name:       topic.Name,
			Partitions: make([]protocol.DeleteRecordsPartitionResponse, 0, len(topic.Partitions)),
		}

		// Get topic
		t, exists := h.backend.GetTopic(topic.Name)
		if !exists {
			// Topic不存在
			for _, part := range topic.Partitions {
				topicResp.Partitions = append(topicResp.Partitions, protocol.DeleteRecordsPartitionResponse{
					PartitionIndex: part.PartitionIndex,
					LowWatermark:   -1,
					ErrorCode:      protocol.UnknownTopicOrPartition,
				})
			}
			resp.Topics = append(resp.Topics, topicResp)
			continue
		}

		// Process each partition
		for _, part := range topic.Partitions {
			lowWatermark, err := t.DeleteRecordsBeforeOffset(part.PartitionIndex, part.Offset)

			partResp := protocol.DeleteRecordsPartitionResponse{
				PartitionIndex: part.PartitionIndex,
			}

			if err != nil {
				h.logger.Error("failed to delete records",
					"topic", topic.Name,
					"partition", part.PartitionIndex,
					"offset", part.Offset,
					"error", err,
				)
				partResp.LowWatermark = -1
				partResp.ErrorCode = protocol.InvalidRequest
			} else {
				partResp.LowWatermark = lowWatermark
				partResp.ErrorCode = protocol.None

				h.logger.Info("deleted records",
					"topic", topic.Name,
					"partition", part.PartitionIndex,
					"before_offset", part.Offset,
					"new_low_watermark", lowWatermark,
				)
			}

			topicResp.Partitions = append(topicResp.Partitions, partResp)
		}

		resp.Topics = append(resp.Topics, topicResp)
	}

	// Encode response
	respData := protocol.EncodeDeleteRecordsResponse(resp, header.APIVersion)

	// Add response header
	var buf bytes.Buffer
	respHeader := &protocol.ResponseHeader{
		CorrelationID: header.CorrelationID,
	}
	if err := respHeader.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode response header: %w", err)
	}

	buf.Write(respData)
	return buf.Bytes(), nil
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// handleAlterConfigs 处理 AlterConfigs 请求
func (h *Handler) handleAlterConfigs(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Read all data from reader
	data, err := io.ReadAll(r)
	if err != nil {
		h.logger.Error("failed to read alter configs request", "error", err)
		return nil, err
	}

	// Decode request
	req, err := protocol.DecodeAlterConfigsRequest(data, header.APIVersion)
	if err != nil {
		h.logger.Error("failed to decode alter configs request", "error", err)
		return nil, err
	}
	req.Header = header

	h.logger.Info("alter configs request",
		"resources", len(req.Resources),
		"validate_only", req.ValidateOnly,
	)

	// Create response
	resp := &protocol.AlterConfigsResponse{
		ThrottleTimeMs: 0,
		Resources:      make([]protocol.AlterConfigsResourceResponse, 0, len(req.Resources)),
	}

	// Process each resource
	for _, resource := range req.Resources {
		resourceResp := protocol.AlterConfigsResourceResponse{
			ResourceType: resource.ResourceType,
			ResourceName: resource.ResourceName,
			ErrorCode:    protocol.None,
			ErrorMessage: nil,
		}

		// Validate resource type
		if resource.ResourceType != protocol.ResourceTypeTopic && resource.ResourceType != protocol.ResourceTypeBroker {
			errMsg := "unsupported resource type"
			resourceResp.ErrorCode = protocol.InvalidRequest
			resourceResp.ErrorMessage = &errMsg
			h.logger.Error("invalid resource type", "type", resource.ResourceType)
			resp.Resources = append(resp.Resources, resourceResp)
			continue
		}

		// Process configurations
		if resource.ResourceType == protocol.ResourceTypeTopic {
			// Get topic
			_, exists := h.backend.GetTopic(resource.ResourceName)
			if !exists {
				errMsg := "unknown topic"
				resourceResp.ErrorCode = protocol.UnknownTopicOrPartition
				resourceResp.ErrorMessage = &errMsg
				h.logger.Error("topic not found", "topic", resource.ResourceName)
				resp.Resources = append(resp.Resources, resourceResp)
				continue
			}

			// Validate only mode - don't actually change anything
			if req.ValidateOnly {
				h.logger.Info("validate only mode - no changes made",
					"topic", resource.ResourceName,
					"configs", len(resource.Configs),
				)
				resp.Resources = append(resp.Resources, resourceResp)
				continue
			}

			// Apply configuration changes
			for _, config := range resource.Configs {
				// For now, we'll just log the changes
				// In a full implementation, this would update the topic's configuration
				if config.Value != nil {
					h.logger.Info("alter config",
						"topic", resource.ResourceName,
						"config", config.Name,
						"value", *config.Value,
					)
				} else {
					h.logger.Info("delete config",
						"topic", resource.ResourceName,
						"config", config.Name,
					)
				}
			}

			// Success
			h.logger.Info("altered topic configs",
				"topic", resource.ResourceName,
				"num_configs", len(resource.Configs),
			)
		} else if resource.ResourceType == protocol.ResourceTypeBroker {
			// Broker configuration changes
			// For now, just validate and log
			if !req.ValidateOnly {
				for _, config := range resource.Configs {
					if config.Value != nil {
						h.logger.Info("alter broker config",
							"broker", resource.ResourceName,
							"config", config.Name,
							"value", *config.Value,
						)
					} else {
						h.logger.Info("delete broker config",
							"broker", resource.ResourceName,
							"config", config.Name,
						)
					}
				}
			}

			h.logger.Info("altered broker configs",
				"broker", resource.ResourceName,
				"num_configs", len(resource.Configs),
			)
		}

		resp.Resources = append(resp.Resources, resourceResp)
	}

	// Encode response
	respData := protocol.EncodeAlterConfigsResponse(resp, header.APIVersion)

	// Add response header
	var buf bytes.Buffer
	respHeader := &protocol.ResponseHeader{
		CorrelationID: header.CorrelationID,
	}
	if err := respHeader.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode response header: %w", err)
	}

	buf.Write(respData)
	return buf.Bytes(), nil
}
