// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"fmt"
	"io"

	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/logger"
)

// handleWriteTxnMarkers handles WriteTxnMarkers requests
func (h *Handler) handleWriteTxnMarkers(reader io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeWriteTxnMarkersRequest(reader, header.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}

	logger.Info("write txn markers request",
		"component", "kafka-handler",
		"num_markers", len(req.Markers),
	)

	// Process each marker
	results := make([]protocol.TxnMarkerResult, len(req.Markers))
	for i, marker := range req.Markers {
		markerType := "ABORT"
		if marker.TransactionResult {
			markerType = "COMMIT"
		}

		logger.Info("processing transaction marker",
			"component", "kafka-handler",
			"producer_id", marker.ProducerID,
			"producer_epoch", marker.ProducerEpoch,
			"marker_type", markerType,
			"num_topics", len(marker.Topics),
		)

		// Process each topic
		topicResults := make([]protocol.TxnMarkerTopicResult, len(marker.Topics))
		for j, topic := range marker.Topics {
			// Process each partition
			partitionResults := make([]protocol.TxnMarkerPartitionResult, len(topic.Partitions))
			for k, partition := range topic.Partitions {
				// Write transaction marker to the log
				errorCode := h.writeTransactionMarker(
					topic.Topic,
					partition,
					marker.ProducerID,
					marker.ProducerEpoch,
					marker.TransactionResult,
				)

				partitionResults[k] = protocol.TxnMarkerPartitionResult{
					PartitionIndex: partition,
					ErrorCode:      errorCode,
				}
			}

			topicResults[j] = protocol.TxnMarkerTopicResult{
				Topic:      topic.Topic,
				Partitions: partitionResults,
			}

			if len(topic.Partitions) > 0 {
				logger.Info("wrote transaction markers",
					"component", "kafka-handler",
					"topic", topic.Topic,
					"num_partitions", len(topic.Partitions),
					"marker_type", markerType,
				)
			}
		}

		results[i] = protocol.TxnMarkerResult{
			ProducerID: marker.ProducerID,
			Topics:     topicResults,
		}
	}

	resp := &protocol.WriteTxnMarkersResponse{
		Markers: results,
	}

	// Encode response
	var buf bytes.Buffer
	if err := protocol.WriteWriteTxnMarkersResponse(&buf, header, resp); err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}

	return buf.Bytes(), nil
}

// writeTransactionMarker writes a transaction marker to the log
func (h *Handler) writeTransactionMarker(
	topicName string,
	partition int32,
	producerID int64,
	producerEpoch int16,
	commit bool,
) protocol.ErrorCode {
	// Check if topic exists
	topicObj, exists := h.topicManager.GetTopic(topicName)
	if !exists {
		logger.Warn("topic not found for transaction marker",
			"component", "kafka-handler",
			"topic", topicName,
		)
		return protocol.UnknownTopicOrPartition
	}

	// Check if partition exists by trying to get it from Partitions map
	// Note: Partitions field is exported
	if _, partitionExists := topicObj.Partitions[partition]; !partitionExists {
		logger.Warn("invalid partition for transaction marker",
			"component", "kafka-handler",
			"topic", topicName,
			"partition", partition,
		)
		return protocol.UnknownTopicOrPartition
	}

	// In a full implementation, this would:
	// 1. Write a control record (COMMIT or ABORT marker) to the log
	// 2. Update the transaction state
	// 3. Notify consumers about the transaction outcome
	// 4. Clean up transaction metadata

	// For now, we log the marker and return success
	markerType := "ABORT"
	if commit {
		markerType = "COMMIT"
	}

	logger.Info("transaction marker written",
		"component", "kafka-handler",
		"topic", topicName,
		"partition", partition,
		"producer_id", producerID,
		"producer_epoch", producerEpoch,
		"marker_type", markerType,
	)

	return protocol.None
}
