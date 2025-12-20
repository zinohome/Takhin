// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// AddOffsetsToTxnRequest represents a request to add consumer group offsets to a transaction
type AddOffsetsToTxnRequest struct {
	TransactionalID string
	ProducerID      int64
	ProducerEpoch   int16
	GroupID         string
}

// AddOffsetsToTxnResponse represents the response for AddOffsetsToTxn
type AddOffsetsToTxnResponse struct {
	ThrottleTimeMs int32
	ErrorCode      ErrorCode
}

// DecodeAddOffsetsToTxnRequest decodes an AddOffsetsToTxn request
func DecodeAddOffsetsToTxnRequest(r io.Reader, version int16) (*AddOffsetsToTxnRequest, error) {
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

	// Read GroupID
	groupID, err := ReadString(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read group_id: %w", err)
	}

	return &AddOffsetsToTxnRequest{
		TransactionalID: transactionalID,
		ProducerID:      producerID,
		ProducerEpoch:   producerEpoch,
		GroupID:         groupID,
	}, nil
}

// EncodeAddOffsetsToTxnRequest encodes an AddOffsetsToTxn request
func EncodeAddOffsetsToTxnRequest(req *AddOffsetsToTxnRequest, version int16) ([]byte, error) {
	buf := make([]byte, 0, 128)

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

	// Write GroupID
	groupLen := make([]byte, 2)
	binary.BigEndian.PutUint16(groupLen, uint16(len(req.GroupID)))
	buf = append(buf, groupLen...)
	buf = append(buf, []byte(req.GroupID)...)

	return buf, nil
}

// DecodeAddOffsetsToTxnResponse decodes an AddOffsetsToTxn response
func DecodeAddOffsetsToTxnResponse(r io.Reader, version int16) (*AddOffsetsToTxnResponse, error) {
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

	return &AddOffsetsToTxnResponse{
		ThrottleTimeMs: throttleTimeMs,
		ErrorCode:      ErrorCode(errorCode),
	}, nil
}

// EncodeAddOffsetsToTxnResponse encodes an AddOffsetsToTxn response
func EncodeAddOffsetsToTxnResponse(resp *AddOffsetsToTxnResponse, version int16) ([]byte, error) {
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

// WriteAddOffsetsToTxnResponse writes the response to a writer
func WriteAddOffsetsToTxnResponse(w io.Writer, header *RequestHeader, resp *AddOffsetsToTxnResponse) error {
	// Encode the response
	respData, err := EncodeAddOffsetsToTxnResponse(resp, header.APIVersion)
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
