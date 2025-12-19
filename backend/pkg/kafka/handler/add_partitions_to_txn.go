// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/logger"
)

// TransactionCoordinator manages transaction state
type TransactionCoordinator struct {
	mu           sync.RWMutex
	transactions map[string]*TransactionState
}

// TransactionState represents the state of a transaction
type TransactionState struct {
	TransactionalID string
	ProducerID      int64
	ProducerEpoch   int16
	Partitions      map[string][]int32 // topic -> partitions
	State           TransactionStatus
}

// TransactionStatus represents the status of a transaction
type TransactionStatus int

const (
	TransactionStatusEmpty TransactionStatus = iota
	TransactionStatusOngoing
	TransactionStatusPrepareCommit
	TransactionStatusPrepareAbort
	TransactionStatusCompleteCommit
	TransactionStatusCompleteAbort
	TransactionStatusDead
)

// NewTransactionCoordinator creates a new transaction coordinator
func NewTransactionCoordinator() *TransactionCoordinator {
	return &TransactionCoordinator{
		transactions: make(map[string]*TransactionState),
	}
}

// AddPartitions adds partitions to a transaction
func (tc *TransactionCoordinator) AddPartitions(transactionalID string, producerID int64, producerEpoch int16, topics []protocol.AddPartitionsToTxnTopic) map[string][]AddPartitionError {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	errors := make(map[string][]AddPartitionError)

	// Get or create transaction state
	txn, exists := tc.transactions[transactionalID]
	if !exists {
		txn = &TransactionState{
			TransactionalID: transactionalID,
			ProducerID:      producerID,
			ProducerEpoch:   producerEpoch,
			Partitions:      make(map[string][]int32),
			State:           TransactionStatusOngoing,
		}
		tc.transactions[transactionalID] = txn
	}

	// Verify producer ID and epoch match
	if txn.ProducerID != producerID {
		// Producer ID mismatch
		for _, topic := range topics {
			topicErrors := make([]AddPartitionError, len(topic.Partitions))
			for i, partition := range topic.Partitions {
				topicErrors[i] = AddPartitionError{
					Partition: partition,
					ErrorCode: protocol.InvalidProducerIDMapping,
				}
			}
			errors[topic.Name] = topicErrors
		}
		return errors
	}

	if txn.ProducerEpoch != producerEpoch {
		// Producer epoch mismatch
		for _, topic := range topics {
			topicErrors := make([]AddPartitionError, len(topic.Partitions))
			for i, partition := range topic.Partitions {
				topicErrors[i] = AddPartitionError{
					Partition: partition,
					ErrorCode: protocol.InvalidProducerEpoch,
				}
			}
			errors[topic.Name] = topicErrors
		}
		return errors
	}

	// Add partitions to transaction
	for _, topic := range topics {
		topicErrors := make([]AddPartitionError, 0)

		existingPartitions := txn.Partitions[topic.Name]
		for _, partition := range topic.Partitions {
			// Check if partition already added
			alreadyAdded := false
			for _, existing := range existingPartitions {
				if existing == partition {
					alreadyAdded = true
					break
				}
			}

			if !alreadyAdded {
				existingPartitions = append(existingPartitions, partition)
			}

			// Success for this partition
			topicErrors = append(topicErrors, AddPartitionError{
				Partition: partition,
				ErrorCode: protocol.None,
			})
		}

		txn.Partitions[topic.Name] = existingPartitions
		errors[topic.Name] = topicErrors
	}

	return errors
}

// AddPartitionError represents an error for a specific partition
type AddPartitionError struct {
	Partition int32
	ErrorCode protocol.ErrorCode
}

// handleAddPartitionsToTxn handles AddPartitionsToTxn requests
func (h *Handler) handleAddPartitionsToTxn(reader io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeAddPartitionsToTxnRequest(reader, header.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}

	logger.Info("add partitions to txn request",
		"component", "kafka-handler",
		"transactional_id", req.TransactionalID,
		"producer_id", req.ProducerID,
		"producer_epoch", req.ProducerEpoch,
		"num_topics", len(req.Topics),
	)

	// Add partitions to transaction
	errors := h.txnCoordinator.AddPartitions(req.TransactionalID, req.ProducerID, req.ProducerEpoch, req.Topics)

	// Build response
	results := make([]protocol.AddPartitionsToTxnTopicResult, 0, len(req.Topics))
	for _, topic := range req.Topics {
		topicErrors := errors[topic.Name]
		partitionResults := make([]protocol.AddPartitionsToTxnPartitionResult, len(topicErrors))
		for i, err := range topicErrors {
			partitionResults[i] = protocol.AddPartitionsToTxnPartitionResult{
				PartitionIndex: err.Partition,
				ErrorCode:      err.ErrorCode,
			}
		}

		results = append(results, protocol.AddPartitionsToTxnTopicResult{
			Name:             topic.Name,
			PartitionResults: partitionResults,
		})

		logger.Info("added partitions to transaction",
			"component", "kafka-handler",
			"transactional_id", req.TransactionalID,
			"topic", topic.Name,
			"num_partitions", len(topic.Partitions),
		)
	}

	resp := &protocol.AddPartitionsToTxnResponse{
		ThrottleTimeMs: 0,
		Results:        results,
	}

	// Encode response
	var buf bytes.Buffer
	if err := protocol.WriteAddPartitionsToTxnResponse(&buf, header, resp); err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}

	return buf.Bytes(), nil
}
