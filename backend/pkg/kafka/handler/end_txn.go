// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"fmt"
	"io"

	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/logger"
)

// EndTransaction ends a transaction (commit or abort)
func (tc *TransactionCoordinator) EndTransaction(transactionalID string, producerID int64, producerEpoch int16, committed bool) protocol.ErrorCode {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Get transaction state
	txn, exists := tc.transactions[transactionalID]
	if !exists {
		return protocol.InvalidProducerIDMapping
	}

	// Verify producer ID and epoch match
	if txn.ProducerID != producerID {
		return protocol.InvalidProducerIDMapping
	}

	if txn.ProducerEpoch != producerEpoch {
		return protocol.InvalidProducerEpoch
	}

	// Update transaction state
	if committed {
		txn.State = TransactionStatusCompleteCommit
		// In a real implementation, we would:
		// 1. Write commit marker to all partitions
		// 2. Update consumer offsets
		// 3. Clean up transaction state
	} else {
		txn.State = TransactionStatusCompleteAbort
		// In a real implementation, we would:
		// 1. Write abort marker to all partitions
		// 2. Mark records as aborted
		// 3. Clean up transaction state
	}

	// Clear partitions (transaction completed)
	txn.Partitions = make(map[string][]int32)

	return protocol.None
}

// handleEndTxn handles EndTxn requests
func (h *Handler) handleEndTxn(reader io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeEndTxnRequest(reader, header.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}

	action := "abort"
	if req.Committed {
		action = "commit"
	}

	logger.Info("end txn request",
		"component", "kafka-handler",
		"transactional_id", req.TransactionalID,
		"producer_id", req.ProducerID,
		"producer_epoch", req.ProducerEpoch,
		"action", action,
	)

	// End transaction
	errorCode := h.txnCoordinator.EndTransaction(req.TransactionalID, req.ProducerID, req.ProducerEpoch, req.Committed)

	if errorCode == protocol.None {
		logger.Info("transaction completed",
			"component", "kafka-handler",
			"transactional_id", req.TransactionalID,
			"action", action,
		)
	} else {
		logger.Warn("transaction end failed",
			"component", "kafka-handler",
			"transactional_id", req.TransactionalID,
			"action", action,
			"error_code", errorCode,
		)
	}

	resp := &protocol.EndTxnResponse{
		ThrottleTimeMs: 0,
		ErrorCode:      errorCode,
	}

	// Encode response
	var buf bytes.Buffer
	if err := protocol.WriteEndTxnResponse(&buf, header, resp); err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}

	return buf.Bytes(), nil
}
