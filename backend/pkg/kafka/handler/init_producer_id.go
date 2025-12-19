// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/takhin-data/takhin/pkg/kafka/protocol"
)

// ProducerIDManager 管理 Producer ID 分配
type ProducerIDManager struct {
	nextProducerID int64
	producers      map[string]*ProducerState // transactionalID -> state
	mu             sync.RWMutex
}

// ProducerState 保存 Producer 状态
type ProducerState struct {
	ProducerID    int64
	ProducerEpoch int16
	LastUpdate    int64 // timestamp
}

// NewProducerIDManager 创建 Producer ID 管理器
func NewProducerIDManager() *ProducerIDManager {
	return &ProducerIDManager{
		nextProducerID: 1000, // 从 1000 开始分配
		producers:      make(map[string]*ProducerState),
	}
}

// GetOrCreateProducerID 获取或创建 Producer ID
func (m *ProducerIDManager) GetOrCreateProducerID(transactionalID *string) (int64, int16) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 非事务性 producer - 直接分配新 ID
	if transactionalID == nil {
		producerID := atomic.AddInt64(&m.nextProducerID, 1)
		return producerID, 0
	}

	// 事务性 producer - 检查是否已存在
	if state, exists := m.producers[*transactionalID]; exists {
		// 增加 epoch
		state.ProducerEpoch++
		return state.ProducerID, state.ProducerEpoch
	}

	// 创建新的事务性 producer
	producerID := atomic.AddInt64(&m.nextProducerID, 1)
	m.producers[*transactionalID] = &ProducerState{
		ProducerID:    producerID,
		ProducerEpoch: 0,
	}

	return producerID, 0
}

// handleInitProducerID 处理 InitProducerID 请求
func (h *Handler) handleInitProducerID(r io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	// Decode request
	req, err := protocol.DecodeInitProducerIDRequest(r, header.APIVersion)
	if err != nil {
		h.logger.Error("failed to decode init producer id request", "error", err)
		return nil, err
	}
	req.Header = header

	var txnIDStr string
	if req.TransactionalID != nil {
		txnIDStr = *req.TransactionalID
	} else {
		txnIDStr = "<none>"
	}

	h.logger.Info("init producer id request",
		"transactional_id", txnIDStr,
		"timeout_ms", req.TransactionTimeoutMs,
	)

	// Get or create producer ID
	producerID, producerEpoch := h.producerIDManager.GetOrCreateProducerID(req.TransactionalID)

	// Create response
	resp := &protocol.InitProducerIDResponse{
		ThrottleTimeMs: 0,
		ErrorCode:      protocol.None,
		ProducerID:     producerID,
		ProducerEpoch:  producerEpoch,
	}

	h.logger.Info("allocated producer id",
		"transactional_id", txnIDStr,
		"producer_id", producerID,
		"producer_epoch", producerEpoch,
	)

	// Encode response
	respData := protocol.EncodeInitProducerIDResponse(resp, header.APIVersion)

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
