// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"fmt"
	"io"

	"github.com/takhin-data/takhin/pkg/kafka/protocol"
)

// handleDescribeLogDirs 处理 DescribeLogDirs 请求
func (h *Handler) handleDescribeLogDirs(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Read all data from reader
	data, err := io.ReadAll(r)
	if err != nil {
		h.logger.Error("failed to read describe log dirs request", "error", err)
		return nil, err
	}

	// Decode request
	req, err := protocol.DecodeDescribeLogDirsRequest(data, header.APIVersion)
	if err != nil {
		h.logger.Error("failed to decode describe log dirs request", "error", err)
		return nil, err
	}
	req.Header = header

	h.logger.Info("describe log dirs request",
		"topics", req.Topics,
	)

	// Create response
	resp := &protocol.DescribeLogDirsResponse{
		ThrottleTimeMs: 0,
		LogDirs:        make([]protocol.DescribeLogDirsResult, 0, 1),
	}

	// Get data directory from config
	dataDir := h.config.Storage.DataDir

	logDirResult := protocol.DescribeLogDirsResult{
		ErrorCode: protocol.None,
		LogDir:    dataDir,
		Topics:    make([]protocol.DescribeLogDirsTopicResult, 0),
	}

	// Determine which topics to process
	var topicsToProcess []string
	if req.Topics == nil {
		// Query all topics
		topicsToProcess = h.topicManager.ListTopics()
	} else {
		// Query specific topics
		topicsToProcess = make([]string, 0, len(req.Topics))
		for _, topic := range req.Topics {
			topicsToProcess = append(topicsToProcess, topic.Topic)
		}
	}

	// Process each topic
	for _, topicName := range topicsToProcess {
		topic, exists := h.backend.GetTopic(topicName)
		if !exists {
			h.logger.Warn("topic not found", "topic", topicName)
			continue
		}

		topicResult := protocol.DescribeLogDirsTopicResult{
			Topic:      topicName,
			Partitions: make([]protocol.DescribeLogDirsPartitionResult, 0),
		}

		// Get partitions to query
		var partitionsToQuery []int32
		if req.Topics != nil {
			// Specific partitions requested
			for _, t := range req.Topics {
				if t.Topic == topicName {
					partitionsToQuery = t.Partitions
					break
				}
			}
		} else {
			// All partitions
			partitionsToQuery = make([]int32, 0, len(topic.Partitions))
			for partID := range topic.Partitions {
				partitionsToQuery = append(partitionsToQuery, partID)
			}
		}

		// Get info for each partition
		for _, partitionID := range partitionsToQuery {
			_, exists := topic.Partitions[partitionID]
			if !exists {
				continue
			}

			// Get partition size (TODO: implement Size() method)
			// For now, use 0 as placeholder
			size := int64(0)

			partitionResult := protocol.DescribeLogDirsPartitionResult{
				PartitionIndex: partitionID,
				Size:           size,
				OffsetLag:      0, // No replication lag in single-node setup
				IsFuture:       false,
			}

			topicResult.Partitions = append(topicResult.Partitions, partitionResult)

			h.logger.Debug("partition info",
				"topic", topicName,
				"partition", partitionID,
				"size", size,
			)
		}

		if len(topicResult.Partitions) > 0 {
			logDirResult.Topics = append(logDirResult.Topics, topicResult)
		}
	}

	resp.LogDirs = append(resp.LogDirs, logDirResult)

	// Encode response
	respData := protocol.EncodeDescribeLogDirsResponse(resp, header.APIVersion)

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
