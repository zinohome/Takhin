// Copyright 2025 Takhin Data
// Licensed under the Apache License, Version 2.0

package protocol

import (
	"encoding/binary"
	"errors"
)

// ErrInsufficientData 数据不足错误
var ErrInsufficientData = errors.New("insufficient data")

// ListOffsets API (Key: 2) - 查询 topic partition 的 offset 信息
// 支持查询最早、最新、特定时间戳的 offset

// ListOffsetsRequest 请求结构
type ListOffsetsRequest struct {
	Header         *RequestHeader
	ReplicaID      int32 // -1 表示普通客户端
	IsolationLevel int8  // 0=READ_UNCOMMITTED, 1=READ_COMMITTED
	Topics         []ListOffsetsTopic
}

// ListOffsetsTopic 主题级别的 offset 请求
type ListOffsetsTopic struct {
	Name       string
	Partitions []ListOffsetsPartition
}

// ListOffsetsPartition 分区级别的 offset 请求
type ListOffsetsPartition struct {
	PartitionIndex     int32
	CurrentLeaderEpoch int32 // -1 表示未知
	Timestamp          int64 // -2=earliest, -1=latest, 其他=具体时间戳
	MaxNumOffsets      int32 // 已废弃，仅用于旧版本
}

// ListOffsetsResponse 响应结构
type ListOffsetsResponse struct {
	ThrottleTimeMs int32
	Topics         []ListOffsetsTopicResponse
}

// ListOffsetsTopicResponse 主题级别的响应
type ListOffsetsTopicResponse struct {
	Name       string
	Partitions []ListOffsetsPartitionResponse
}

// ListOffsetsPartitionResponse 分区级别的响应
type ListOffsetsPartitionResponse struct {
	PartitionIndex int32
	ErrorCode      ErrorCode
	Timestamp      int64 // 实际的时间戳
	Offset         int64 // 对应的 offset
	LeaderEpoch    int32 // Leader epoch
}

// 特殊时间戳常量
const (
	TimestampEarliest = int64(-2) // 查询最早的 offset
	TimestampLatest   = int64(-1) // 查询最新的 offset
)

// DecodeListOffsetsRequest 解码 ListOffsets 请求
func DecodeListOffsetsRequest(data []byte, version int16) (*ListOffsetsRequest, error) {
	req := &ListOffsetsRequest{}
	offset := 0

	// ReplicaID
	if offset+4 > len(data) {
		return nil, ErrInsufficientData
	}
	req.ReplicaID = int32(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	// IsolationLevel (v2+)
	if version >= 2 {
		if offset+1 > len(data) {
			return nil, ErrInsufficientData
		}
		req.IsolationLevel = int8(data[offset])
		offset += 1
	}

	// Topics array
	if offset+4 > len(data) {
		return nil, ErrInsufficientData
	}
	topicsLen := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	req.Topics = make([]ListOffsetsTopic, topicsLen)
	for i := 0; i < topicsLen; i++ {
		// Topic name
		if offset+2 > len(data) {
			return nil, ErrInsufficientData
		}
		nameLen := int(binary.BigEndian.Uint16(data[offset:]))
		offset += 2

		if offset+nameLen > len(data) {
			return nil, ErrInsufficientData
		}
		req.Topics[i].Name = string(data[offset : offset+nameLen])
		offset += nameLen

		// Partitions array
		if offset+4 > len(data) {
			return nil, ErrInsufficientData
		}
		partLen := int(binary.BigEndian.Uint32(data[offset:]))
		offset += 4

		req.Topics[i].Partitions = make([]ListOffsetsPartition, partLen)
		for j := 0; j < partLen; j++ {
			// Partition index
			if offset+4 > len(data) {
				return nil, ErrInsufficientData
			}
			req.Topics[i].Partitions[j].PartitionIndex = int32(binary.BigEndian.Uint32(data[offset:]))
			offset += 4

			// CurrentLeaderEpoch (v4+)
			if version >= 4 {
				if offset+4 > len(data) {
					return nil, ErrInsufficientData
				}
				req.Topics[i].Partitions[j].CurrentLeaderEpoch = int32(binary.BigEndian.Uint32(data[offset:]))
				offset += 4
			}

			// Timestamp
			if offset+8 > len(data) {
				return nil, ErrInsufficientData
			}
			req.Topics[i].Partitions[j].Timestamp = int64(binary.BigEndian.Uint64(data[offset:]))
			offset += 8

			// MaxNumOffsets (v0 only)
			if version == 0 {
				if offset+4 > len(data) {
					return nil, ErrInsufficientData
				}
				req.Topics[i].Partitions[j].MaxNumOffsets = int32(binary.BigEndian.Uint32(data[offset:]))
				offset += 4
			}
		}
	}

	return req, nil
}

// EncodeListOffsetsResponse 编码 ListOffsets 响应
func EncodeListOffsetsResponse(resp *ListOffsetsResponse, version int16) []byte {
	buf := make([]byte, 0, 1024)

	// ThrottleTimeMs
	throttle := make([]byte, 4)
	binary.BigEndian.PutUint32(throttle, uint32(resp.ThrottleTimeMs))
	buf = append(buf, throttle...)

	// Topics array length
	topicsLen := make([]byte, 4)
	binary.BigEndian.PutUint32(topicsLen, uint32(len(resp.Topics)))
	buf = append(buf, topicsLen...)

	for _, topic := range resp.Topics {
		// Topic name
		nameLen := make([]byte, 2)
		binary.BigEndian.PutUint16(nameLen, uint16(len(topic.Name)))
		buf = append(buf, nameLen...)
		buf = append(buf, []byte(topic.Name)...)

		// Partitions array length
		partsLen := make([]byte, 4)
		binary.BigEndian.PutUint32(partsLen, uint32(len(topic.Partitions)))
		buf = append(buf, partsLen...)

		for _, part := range topic.Partitions {
			// Partition index
			partIdx := make([]byte, 4)
			binary.BigEndian.PutUint32(partIdx, uint32(part.PartitionIndex))
			buf = append(buf, partIdx...)

			// Error code
			errCode := make([]byte, 2)
			binary.BigEndian.PutUint16(errCode, uint16(part.ErrorCode))
			buf = append(buf, errCode...)

			// Timestamp (v1+)
			if version >= 1 {
				ts := make([]byte, 8)
				binary.BigEndian.PutUint64(ts, uint64(part.Timestamp))
				buf = append(buf, ts...)
			}

			// Offset
			off := make([]byte, 8)
			binary.BigEndian.PutUint64(off, uint64(part.Offset))
			buf = append(buf, off...)

			// LeaderEpoch (v4+)
			if version >= 4 {
				epoch := make([]byte, 4)
				binary.BigEndian.PutUint32(epoch, uint32(part.LeaderEpoch))
				buf = append(buf, epoch...)
			}
		}
	}

	return buf
}
