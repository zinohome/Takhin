// Copyright 2025 Takhin Data
// Licensed under the Apache License, Version 2.0

package protocol

import (
	"encoding/binary"
	"errors"
)

// DeleteRecords API (Key: 21) - 删除记录到指定 offset
// 这个API用于删除topic partition中到指定offset之前的所有记录

// DeleteRecordsRequest 请求结构
type DeleteRecordsRequest struct {
	Header    *RequestHeader
	Topics    []DeleteRecordsTopic
	TimeoutMs int32
}

// DeleteRecordsTopic 主题级别的删除请求
type DeleteRecordsTopic struct {
	Name       string
	Partitions []DeleteRecordsPartition
}

// DeleteRecordsPartition 分区级别的删除请求
type DeleteRecordsPartition struct {
	PartitionIndex int32
	Offset         int64 // 删除到此offset之前的所有记录(不包括此offset)
}

// DeleteRecordsResponse 响应结构
type DeleteRecordsResponse struct {
	ThrottleTimeMs int32
	Topics         []DeleteRecordsTopicResponse
}

// DeleteRecordsTopicResponse 主题级别的响应
type DeleteRecordsTopicResponse struct {
	Name       string
	Partitions []DeleteRecordsPartitionResponse
}

// DeleteRecordsPartitionResponse 分区级别的响应
type DeleteRecordsPartitionResponse struct {
	PartitionIndex int32
	LowWatermark   int64 // 删除后的低水位(新的起始offset)
	ErrorCode      ErrorCode
}

// DecodeDeleteRecordsRequest 解码 DeleteRecords 请求
func DecodeDeleteRecordsRequest(data []byte, version int16) (*DeleteRecordsRequest, error) {
	req := &DeleteRecordsRequest{}
	offset := 0

	// Topics array
	if offset+4 > len(data) {
		return nil, errors.New("insufficient data for topics array length")
	}
	topicsLen := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	req.Topics = make([]DeleteRecordsTopic, topicsLen)
	for i := 0; i < topicsLen; i++ {
		// Topic name
		if offset+2 > len(data) {
			return nil, errors.New("insufficient data for topic name length")
		}
		nameLen := int(binary.BigEndian.Uint16(data[offset:]))
		offset += 2

		if offset+nameLen > len(data) {
			return nil, errors.New("insufficient data for topic name")
		}
		req.Topics[i].Name = string(data[offset : offset+nameLen])
		offset += nameLen

		// Partitions array
		if offset+4 > len(data) {
			return nil, errors.New("insufficient data for partitions array length")
		}
		partLen := int(binary.BigEndian.Uint32(data[offset:]))
		offset += 4

		req.Topics[i].Partitions = make([]DeleteRecordsPartition, partLen)
		for j := 0; j < partLen; j++ {
			// Partition index
			if offset+4 > len(data) {
				return nil, errors.New("insufficient data for partition index")
			}
			req.Topics[i].Partitions[j].PartitionIndex = int32(binary.BigEndian.Uint32(data[offset:]))
			offset += 4

			// Offset
			if offset+8 > len(data) {
				return nil, errors.New("insufficient data for offset")
			}
			req.Topics[i].Partitions[j].Offset = int64(binary.BigEndian.Uint64(data[offset:]))
			offset += 8
		}
	}

	// TimeoutMs
	if offset+4 > len(data) {
		return nil, errors.New("insufficient data for timeout")
	}
	req.TimeoutMs = int32(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	return req, nil
}

// EncodeDeleteRecordsResponse 编码 DeleteRecords 响应
func EncodeDeleteRecordsResponse(resp *DeleteRecordsResponse, version int16) []byte {
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

			// LowWatermark
			lwm := make([]byte, 8)
			binary.BigEndian.PutUint64(lwm, uint64(part.LowWatermark))
			buf = append(buf, lwm...)

			// Error code
			errCode := make([]byte, 2)
			binary.BigEndian.PutUint16(errCode, uint16(part.ErrorCode))
			buf = append(buf, errCode...)
		}
	}

	return buf
}
