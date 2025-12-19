// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"encoding/binary"
	"errors"
	"io"
)

// InitProducerID API (Key: 22) - 初始化 Producer ID
// 用于获取 Producer ID 和 Epoch，支持幂等性和事务

// InitProducerIDRequest 请求结构
type InitProducerIDRequest struct {
	Header               *RequestHeader
	TransactionalID      *string // null 表示非事务性 producer
	TransactionTimeoutMs int32
	ProducerID           int64 // -1 表示需要分配新的 ID
	ProducerEpoch        int16 // -1 表示需要分配新的 Epoch
}

// InitProducerIDResponse 响应结构
type InitProducerIDResponse struct {
	ThrottleTimeMs int32
	ErrorCode      ErrorCode
	ProducerID     int64
	ProducerEpoch  int16
}

// DecodeInitProducerIDRequest 解码请求
func DecodeInitProducerIDRequest(r io.Reader, version int16) (*InitProducerIDRequest, error) {
	req := &InitProducerIDRequest{}

	// TransactionalID (nullable string)
	transactionalID, err := ReadNullableString(r)
	if err != nil {
		return nil, err
	}
	req.TransactionalID = transactionalID

	// TransactionTimeoutMs
	timeout, err := ReadInt32(r)
	if err != nil {
		return nil, err
	}
	req.TransactionTimeoutMs = timeout

	// Version 3+ includes ProducerID and ProducerEpoch
	if version >= 3 {
		producerID, err := ReadInt64(r)
		if err != nil {
			return nil, err
		}
		req.ProducerID = producerID

		producerEpoch, err := ReadInt16(r)
		if err != nil {
			return nil, err
		}
		req.ProducerEpoch = producerEpoch
	} else {
		req.ProducerID = -1
		req.ProducerEpoch = -1
	}

	return req, nil
}

// EncodeInitProducerIDResponse 编码响应
func EncodeInitProducerIDResponse(resp *InitProducerIDResponse, version int16) []byte {
	buf := make([]byte, 0, 256)

	// ThrottleTimeMs
	throttle := make([]byte, 4)
	binary.BigEndian.PutUint32(throttle, uint32(resp.ThrottleTimeMs))
	buf = append(buf, throttle...)

	// ErrorCode
	errCode := make([]byte, 2)
	binary.BigEndian.PutUint16(errCode, uint16(resp.ErrorCode))
	buf = append(buf, errCode...)

	// ProducerID
	producerID := make([]byte, 8)
	binary.BigEndian.PutUint64(producerID, uint64(resp.ProducerID))
	buf = append(buf, producerID...)

	// ProducerEpoch
	producerEpoch := make([]byte, 2)
	binary.BigEndian.PutUint16(producerEpoch, uint16(resp.ProducerEpoch))
	buf = append(buf, producerEpoch...)

	return buf
}

// EncodeInitProducerIDRequest 编码请求（用于测试）
func EncodeInitProducerIDRequest(req *InitProducerIDRequest, version int16) ([]byte, error) {
	buf := make([]byte, 0, 256)

	// TransactionalID (nullable string)
	if req.TransactionalID == nil {
		// Null string
		nullLen := make([]byte, 2)
		binary.BigEndian.PutUint16(nullLen, 0xFFFF)
		buf = append(buf, nullLen...)
	} else {
		strLen := make([]byte, 2)
		binary.BigEndian.PutUint16(strLen, uint16(len(*req.TransactionalID)))
		buf = append(buf, strLen...)
		buf = append(buf, []byte(*req.TransactionalID)...)
	}

	// TransactionTimeoutMs
	timeout := make([]byte, 4)
	binary.BigEndian.PutUint32(timeout, uint32(req.TransactionTimeoutMs))
	buf = append(buf, timeout...)

	// Version 3+ includes ProducerID and ProducerEpoch
	if version >= 3 {
		producerID := make([]byte, 8)
		binary.BigEndian.PutUint64(producerID, uint64(req.ProducerID))
		buf = append(buf, producerID...)

		producerEpoch := make([]byte, 2)
		binary.BigEndian.PutUint16(producerEpoch, uint16(req.ProducerEpoch))
		buf = append(buf, producerEpoch...)
	}

	return buf, nil
}

// DecodeInitProducerIDResponse 解码响应（用于测试）
func DecodeInitProducerIDResponse(data []byte, version int16) (*InitProducerIDResponse, error) {
	resp := &InitProducerIDResponse{}
	offset := 0

	// ThrottleTimeMs
	if offset+4 > len(data) {
		return nil, errors.New("insufficient data for throttle time")
	}
	resp.ThrottleTimeMs = int32(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	// ErrorCode
	if offset+2 > len(data) {
		return nil, errors.New("insufficient data for error code")
	}
	resp.ErrorCode = ErrorCode(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	// ProducerID
	if offset+8 > len(data) {
		return nil, errors.New("insufficient data for producer id")
	}
	resp.ProducerID = int64(binary.BigEndian.Uint64(data[offset:]))
	offset += 8

	// ProducerEpoch
	if offset+2 > len(data) {
		return nil, errors.New("insufficient data for producer epoch")
	}
	resp.ProducerEpoch = int16(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	return resp, nil
}
