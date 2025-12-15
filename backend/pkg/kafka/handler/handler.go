// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/takhin-data/takhin/pkg/config"
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
}

// New creates a new request handler with direct backend
func New(cfg *config.Config, topicMgr *topic.Manager) *Handler {
	return &Handler{
		config:       cfg,
		logger:       logger.Default().WithComponent("kafka-handler"),
		topicManager: topicMgr,
		backend:      NewDirectBackend(topicMgr),
	}
}

// NewWithBackend creates a new request handler with custom backend
func NewWithBackend(cfg *config.Config, topicMgr *topic.Manager, backend Backend) *Handler {
	return &Handler{
		config:       cfg,
		logger:       logger.Default().WithComponent("kafka-handler"),
		topicManager: topicMgr,
		backend:      backend,
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
