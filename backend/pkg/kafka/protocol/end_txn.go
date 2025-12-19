// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// EndTxnRequest represents a request to end a transaction
type EndTxnRequest struct {
	TransactionalID string
	ProducerID      int64
	ProducerEpoch   int16
	Committed       bool // true to commit, false to abort
}

// EndTxnResponse represents the response for EndTxn
type EndTxnResponse struct {
	ThrottleTimeMs int32
	ErrorCode      ErrorCode
}

// DecodeEndTxnRequest decodes an EndTxn request
func DecodeEndTxnRequest(r io.Reader, version int16) (*EndTxnRequest, error) {
	// Read TransactionalID
	transactionalID, err := ReadString(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read transactional_id: %w", err)
	}

	// Read ProducerID
	producerID, err := ReadInt64(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read producer_id: %w", err)
	}

	// Read ProducerEpoch
	producerEpoch, err := ReadInt16(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read producer_epoch: %w", err)
	}

	// Read Committed flag
	committed, err := ReadInt8(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read committed: %w", err)
	}

	return &EndTxnRequest{
		TransactionalID: transactionalID,
		ProducerID:      producerID,
		ProducerEpoch:   producerEpoch,
		Committed:       committed != 0,
	}, nil
}

// EncodeEndTxnRequest encodes an EndTxn request
func EncodeEndTxnRequest(req *EndTxnRequest, version int16) ([]byte, error) {
	buf := make([]byte, 0, 64)

	// Write TransactionalID
	strLen := make([]byte, 2)
	binary.BigEndian.PutUint16(strLen, uint16(len(req.TransactionalID)))
	buf = append(buf, strLen...)
	buf = append(buf, []byte(req.TransactionalID)...)

	// Write ProducerID
	producerID := make([]byte, 8)
	binary.BigEndian.PutUint64(producerID, uint64(req.ProducerID))
	buf = append(buf, producerID...)

	// Write ProducerEpoch
	producerEpoch := make([]byte, 2)
	binary.BigEndian.PutUint16(producerEpoch, uint16(req.ProducerEpoch))
	buf = append(buf, producerEpoch...)

	// Write Committed flag
	var committedByte byte
	if req.Committed {
		committedByte = 1
	}
	buf = append(buf, committedByte)

	return buf, nil
}

// DecodeEndTxnResponse decodes an EndTxn response
func DecodeEndTxnResponse(r io.Reader, version int16) (*EndTxnResponse, error) {
	// Read ThrottleTimeMs
	throttleTimeMs, err := ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read throttle_time_ms: %w", err)
	}

	// Read ErrorCode
	errorCode, err := ReadInt16(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read error_code: %w", err)
	}

	return &EndTxnResponse{
		ThrottleTimeMs: throttleTimeMs,
		ErrorCode:      ErrorCode(errorCode),
	}, nil
}

// EncodeEndTxnResponse encodes an EndTxn response
func EncodeEndTxnResponse(resp *EndTxnResponse, version int16) ([]byte, error) {
	buf := make([]byte, 0, 8)

	// Write ThrottleTimeMs
	throttle := make([]byte, 4)
	binary.BigEndian.PutUint32(throttle, uint32(resp.ThrottleTimeMs))
	buf = append(buf, throttle...)

	// Write ErrorCode
	errCode := make([]byte, 2)
	binary.BigEndian.PutUint16(errCode, uint16(resp.ErrorCode))
	buf = append(buf, errCode...)

	return buf, nil
}

// WriteEndTxnResponse writes the response to a writer
func WriteEndTxnResponse(w io.Writer, header *RequestHeader, resp *EndTxnResponse) error {
	// Encode the response
	respData, err := EncodeEndTxnResponse(resp, header.APIVersion)
	if err != nil {
		return err
	}

	// Write correlation ID
	if err := binary.Write(w, binary.BigEndian, header.CorrelationID); err != nil {
		return err
	}

	// Write response data
	if _, err := w.Write(respData); err != nil {
		return err
	}

	return nil
}
