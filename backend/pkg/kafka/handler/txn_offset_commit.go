// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"fmt"
	"io"

	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/logger"
)

// OffsetCommitError represents an error for a specific partition offset commit
type OffsetCommitError struct {
	Partition int32
	ErrorCode protocol.ErrorCode
}

// CommitOffsetsInTransaction commits consumer offsets as part of a transaction
func (tc *TransactionCoordinator) CommitOffsetsInTransaction(transactionalID string, groupID string, producerID int64, producerEpoch int16, topics []protocol.TxnOffsetCommitTopic) map[string][]OffsetCommitError {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	errors := make(map[string][]OffsetCommitError)

	txn, exists := tc.transactions[transactionalID]
	if !exists {
		for _, topic := range topics {
			topicErrors := make([]OffsetCommitError, len(topic.Partitions))
			for i, partition := range topic.Partitions {
				topicErrors[i] = OffsetCommitError{
					Partition: partition.PartitionIndex,
					ErrorCode: protocol.InvalidProducerIDMapping,
				}
			}
			errors[topic.Name] = topicErrors
		}
		return errors
	}

	if txn.ProducerID != producerID {
		for _, topic := range topics {
			topicErrors := make([]OffsetCommitError, len(topic.Partitions))
			for i, partition := range topic.Partitions {
				topicErrors[i] = OffsetCommitError{
					Partition: partition.PartitionIndex,
					ErrorCode: protocol.InvalidProducerIDMapping,
				}
			}
			errors[topic.Name] = topicErrors
		}
		return errors
	}

	if txn.ProducerEpoch != producerEpoch {
		for _, topic := range topics {
			topicErrors := make([]OffsetCommitError, len(topic.Partitions))
			for i, partition := range topic.Partitions {
				topicErrors[i] = OffsetCommitError{
					Partition: partition.PartitionIndex,
					ErrorCode: protocol.InvalidProducerEpoch,
				}
			}
			errors[topic.Name] = topicErrors
		}
		return errors
	}

	if txn.State != TransactionStatusOngoing {
		for _, topic := range topics {
			topicErrors := make([]OffsetCommitError, len(topic.Partitions))
			for i, partition := range topic.Partitions {
				topicErrors[i] = OffsetCommitError{
					Partition: partition.PartitionIndex,
					ErrorCode: protocol.InvalidTxnState,
				}
			}
			errors[topic.Name] = topicErrors
		}
		return errors
	}

	for _, topic := range topics {
		topicErrors := make([]OffsetCommitError, 0)
		for _, partition := range topic.Partitions {
			topicErrors = append(topicErrors, OffsetCommitError{
				Partition: partition.PartitionIndex,
				ErrorCode: protocol.None,
			})
		}
		errors[topic.Name] = topicErrors
	}

	return errors
}

// handleTxnOffsetCommit handles TxnOffsetCommit requests
func (h *Handler) handleTxnOffsetCommit(reader io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeTxnOffsetCommitRequest(reader, header.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}

	logger.Info("txn offset commit request",
		"component", "kafka-handler",
		"transactional_id", req.TransactionalID,
		"group_id", req.GroupID,
		"producer_id", req.ProducerID,
		"producer_epoch", req.ProducerEpoch,
		"num_topics", len(req.Topics),
	)

	errors := h.txnCoordinator.CommitOffsetsInTransaction(req.TransactionalID, req.GroupID, req.ProducerID, req.ProducerEpoch, req.Topics)

	results := make([]protocol.TxnOffsetCommitTopicResult, 0, len(req.Topics))
	for _, topic := range req.Topics {
		topicErrors := errors[topic.Name]
		partitionResults := make([]protocol.TxnOffsetCommitPartitionResult, len(topicErrors))
		for i, err := range topicErrors {
			partitionResults[i] = protocol.TxnOffsetCommitPartitionResult{
				PartitionIndex: err.Partition,
				ErrorCode:      err.ErrorCode,
			}
		}

		results = append(results, protocol.TxnOffsetCommitTopicResult{
			Name:       topic.Name,
			Partitions: partitionResults,
		})

		successCount := 0
		for _, err := range topicErrors {
			if err.ErrorCode == protocol.None {
				successCount++
			}
		}

		logger.Info("committed offsets in transaction",
			"component", "kafka-handler",
			"transactional_id", req.TransactionalID,
			"group_id", req.GroupID,
			"topic", topic.Name,
			"success_count", successCount,
			"total_count", len(topicErrors),
		)
	}

	resp := &protocol.TxnOffsetCommitResponse{
		ThrottleTimeMs: 0,
		Topics:         results,
	}

	var buf bytes.Buffer
	if err := protocol.WriteTxnOffsetCommitResponse(&buf, header, resp); err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}

	return buf.Bytes(), nil
}
