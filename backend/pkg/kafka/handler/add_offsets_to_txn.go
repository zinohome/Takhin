// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"fmt"
	"io"

	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/logger"
)

// AddOffsetsToTransaction adds consumer group offsets to a transaction
func (tc *TransactionCoordinator) AddOffsetsToTransaction(transactionalID string, producerID int64, producerEpoch int16, groupID string) protocol.ErrorCode {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	txn, exists := tc.transactions[transactionalID]
	if !exists {
		return protocol.InvalidProducerIDMapping
	}

	if txn.ProducerID != producerID {
		return protocol.InvalidProducerIDMapping
	}

	if txn.ProducerEpoch != producerEpoch {
		return protocol.InvalidProducerEpoch
	}

	if txn.State != TransactionStatusOngoing && txn.State != TransactionStatusEmpty {
		return protocol.InvalidTxnState
	}

	if txn.State == TransactionStatusEmpty {
		txn.State = TransactionStatusOngoing
	}

	return protocol.None
}

// handleAddOffsetsToTxn handles AddOffsetsToTxn requests
func (h *Handler) handleAddOffsetsToTxn(reader io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeAddOffsetsToTxnRequest(reader, header.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}

	logger.Info("add offsets to txn request",
		"component", "kafka-handler",
		"transactional_id", req.TransactionalID,
		"producer_id", req.ProducerID,
		"producer_epoch", req.ProducerEpoch,
		"group_id", req.GroupID,
	)

	errorCode := h.txnCoordinator.AddOffsetsToTransaction(req.TransactionalID, req.ProducerID, req.ProducerEpoch, req.GroupID)

	if errorCode == protocol.None {
		logger.Info("added offsets to transaction",
			"component", "kafka-handler",
			"transactional_id", req.TransactionalID,
			"group_id", req.GroupID,
		)
	} else {
		logger.Warn("add offsets to txn failed",
			"component", "kafka-handler",
			"transactional_id", req.TransactionalID,
			"group_id", req.GroupID,
			"error_code", errorCode,
		)
	}

	resp := &protocol.AddOffsetsToTxnResponse{
		ThrottleTimeMs: 0,
		ErrorCode:      errorCode,
	}

	var buf bytes.Buffer
	if err := protocol.WriteAddOffsetsToTxnResponse(&buf, header, resp); err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}

	return buf.Bytes(), nil
}
