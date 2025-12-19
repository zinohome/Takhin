// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"encoding/binary"
	"errors"
)

// DescribeLogDirs API (Key: 35) - 查询日志目录信息
// 用于获取 broker 的日志目录、磁盘使用情况和 topic/partition 分布

// DescribeLogDirsRequest 请求结构
type DescribeLogDirsRequest struct {
	Header *RequestHeader
	Topics []DescribeLogDirsTopic // null 表示查询所有 topic
}

// DescribeLogDirsTopic 要查询的 topic
type DescribeLogDirsTopic struct {
	Topic      string
	Partitions []int32
}

// DescribeLogDirsResponse 响应结构
type DescribeLogDirsResponse struct {
	ThrottleTimeMs int32
	LogDirs        []DescribeLogDirsResult
}

// DescribeLogDirsResult 日志目录结果
type DescribeLogDirsResult struct {
	ErrorCode ErrorCode
	LogDir    string
	Topics    []DescribeLogDirsTopicResult
}

// DescribeLogDirsTopicResult topic 的日志信息
type DescribeLogDirsTopicResult struct {
	Topic      string
	Partitions []DescribeLogDirsPartitionResult
}

// DescribeLogDirsPartitionResult partition 的日志信息
type DescribeLogDirsPartitionResult struct {
	PartitionIndex int32
	Size           int64 // partition 在该目录中的大小（字节）
	OffsetLag      int64 // 与 HWM 的 offset 差距
	IsFuture       bool  // 是否是 future replica
}

// DecodeDescribeLogDirsRequest 解码请求
func DecodeDescribeLogDirsRequest(data []byte, version int16) (*DescribeLogDirsRequest, error) {
	req := &DescribeLogDirsRequest{}
	offset := 0

	// Topics array (nullable)
	if offset+4 > len(data) {
		return nil, errors.New("insufficient data for topics array length")
	}
	topicsLen := int(int32(binary.BigEndian.Uint32(data[offset:])))
	offset += 4

	if topicsLen == -1 {
		// null - 查询所有 topic
		req.Topics = nil
	} else if topicsLen > 0 {
		req.Topics = make([]DescribeLogDirsTopic, topicsLen)
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
			req.Topics[i].Topic = string(data[offset : offset+nameLen])
			offset += nameLen

			// Partitions array
			if offset+4 > len(data) {
				return nil, errors.New("insufficient data for partitions array length")
			}
			partitionsLen := int(binary.BigEndian.Uint32(data[offset:]))
			offset += 4

			req.Topics[i].Partitions = make([]int32, partitionsLen)
			for j := 0; j < partitionsLen; j++ {
				if offset+4 > len(data) {
					return nil, errors.New("insufficient data for partition index")
				}
				req.Topics[i].Partitions[j] = int32(binary.BigEndian.Uint32(data[offset:]))
				offset += 4
			}
		}
	}

	return req, nil
}

// EncodeDescribeLogDirsResponse 编码响应
func EncodeDescribeLogDirsResponse(resp *DescribeLogDirsResponse, version int16) []byte {
	buf := make([]byte, 0, 2048)

	// ThrottleTimeMs
	throttle := make([]byte, 4)
	binary.BigEndian.PutUint32(throttle, uint32(resp.ThrottleTimeMs))
	buf = append(buf, throttle...)

	// LogDirs array length
	logDirsLen := make([]byte, 4)
	binary.BigEndian.PutUint32(logDirsLen, uint32(len(resp.LogDirs)))
	buf = append(buf, logDirsLen...)

	for _, logDir := range resp.LogDirs {
		// ErrorCode
		errCode := make([]byte, 2)
		binary.BigEndian.PutUint16(errCode, uint16(logDir.ErrorCode))
		buf = append(buf, errCode...)

		// LogDir path
		pathLen := make([]byte, 2)
		binary.BigEndian.PutUint16(pathLen, uint16(len(logDir.LogDir)))
		buf = append(buf, pathLen...)
		buf = append(buf, []byte(logDir.LogDir)...)

		// Topics array length
		topicsLen := make([]byte, 4)
		binary.BigEndian.PutUint32(topicsLen, uint32(len(logDir.Topics)))
		buf = append(buf, topicsLen...)

		for _, topic := range logDir.Topics {
			// Topic name
			topicNameLen := make([]byte, 2)
			binary.BigEndian.PutUint16(topicNameLen, uint16(len(topic.Topic)))
			buf = append(buf, topicNameLen...)
			buf = append(buf, []byte(topic.Topic)...)

			// Partitions array length
			partitionsLen := make([]byte, 4)
			binary.BigEndian.PutUint32(partitionsLen, uint32(len(topic.Partitions)))
			buf = append(buf, partitionsLen...)

			for _, partition := range topic.Partitions {
				// PartitionIndex
				partIdx := make([]byte, 4)
				binary.BigEndian.PutUint32(partIdx, uint32(partition.PartitionIndex))
				buf = append(buf, partIdx...)

				// Size
				size := make([]byte, 8)
				binary.BigEndian.PutUint64(size, uint64(partition.Size))
				buf = append(buf, size...)

				// OffsetLag
				lag := make([]byte, 8)
				binary.BigEndian.PutUint64(lag, uint64(partition.OffsetLag))
				buf = append(buf, lag...)

				// IsFuture
				if partition.IsFuture {
					buf = append(buf, 1)
				} else {
					buf = append(buf, 0)
				}
			}
		}
	}

	return buf
}

// EncodeDescribeLogDirsRequest 编码请求（用于测试）
func EncodeDescribeLogDirsRequest(req *DescribeLogDirsRequest, version int16) []byte {
	buf := make([]byte, 0, 512)

	// Topics array (nullable)
	if req.Topics == nil {
		// null array
		nullLen := make([]byte, 4)
		binary.BigEndian.PutUint32(nullLen, 0xFFFFFFFF)
		buf = append(buf, nullLen...)
	} else {
		topicsLen := make([]byte, 4)
		binary.BigEndian.PutUint32(topicsLen, uint32(len(req.Topics)))
		buf = append(buf, topicsLen...)

		for _, topic := range req.Topics {
			// Topic name
			nameLen := make([]byte, 2)
			binary.BigEndian.PutUint16(nameLen, uint16(len(topic.Topic)))
			buf = append(buf, nameLen...)
			buf = append(buf, []byte(topic.Topic)...)

			// Partitions array
			partitionsLen := make([]byte, 4)
			binary.BigEndian.PutUint32(partitionsLen, uint32(len(topic.Partitions)))
			buf = append(buf, partitionsLen...)

			for _, partition := range topic.Partitions {
				partIdx := make([]byte, 4)
				binary.BigEndian.PutUint32(partIdx, uint32(partition))
				buf = append(buf, partIdx...)
			}
		}
	}

	return buf
}

// DecodeDescribeLogDirsResponse 解码响应（用于测试）
func DecodeDescribeLogDirsResponse(data []byte, version int16) (*DescribeLogDirsResponse, error) {
	resp := &DescribeLogDirsResponse{}
	offset := 0

	// ThrottleTimeMs
	if offset+4 > len(data) {
		return nil, errors.New("insufficient data for throttle time")
	}
	resp.ThrottleTimeMs = int32(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	// LogDirs array length
	if offset+4 > len(data) {
		return nil, errors.New("insufficient data for log dirs array length")
	}
	logDirsLen := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	resp.LogDirs = make([]DescribeLogDirsResult, logDirsLen)
	for i := 0; i < logDirsLen; i++ {
		// ErrorCode
		if offset+2 > len(data) {
			return nil, errors.New("insufficient data for error code")
		}
		resp.LogDirs[i].ErrorCode = ErrorCode(binary.BigEndian.Uint16(data[offset:]))
		offset += 2

		// LogDir path
		if offset+2 > len(data) {
			return nil, errors.New("insufficient data for log dir path length")
		}
		pathLen := int(binary.BigEndian.Uint16(data[offset:]))
		offset += 2

		if offset+pathLen > len(data) {
			return nil, errors.New("insufficient data for log dir path")
		}
		resp.LogDirs[i].LogDir = string(data[offset : offset+pathLen])
		offset += pathLen

		// Topics array length
		if offset+4 > len(data) {
			return nil, errors.New("insufficient data for topics array length")
		}
		topicsLen := int(binary.BigEndian.Uint32(data[offset:]))
		offset += 4

		resp.LogDirs[i].Topics = make([]DescribeLogDirsTopicResult, topicsLen)
		for j := 0; j < topicsLen; j++ {
			// Topic name
			if offset+2 > len(data) {
				return nil, errors.New("insufficient data for topic name length")
			}
			topicNameLen := int(binary.BigEndian.Uint16(data[offset:]))
			offset += 2

			if offset+topicNameLen > len(data) {
				return nil, errors.New("insufficient data for topic name")
			}
			resp.LogDirs[i].Topics[j].Topic = string(data[offset : offset+topicNameLen])
			offset += topicNameLen

			// Partitions array length
			if offset+4 > len(data) {
				return nil, errors.New("insufficient data for partitions array length")
			}
			partitionsLen := int(binary.BigEndian.Uint32(data[offset:]))
			offset += 4

			resp.LogDirs[i].Topics[j].Partitions = make([]DescribeLogDirsPartitionResult, partitionsLen)
			for k := 0; k < partitionsLen; k++ {
				// PartitionIndex
				if offset+4 > len(data) {
					return nil, errors.New("insufficient data for partition index")
				}
				resp.LogDirs[i].Topics[j].Partitions[k].PartitionIndex = int32(binary.BigEndian.Uint32(data[offset:]))
				offset += 4

				// Size
				if offset+8 > len(data) {
					return nil, errors.New("insufficient data for size")
				}
				resp.LogDirs[i].Topics[j].Partitions[k].Size = int64(binary.BigEndian.Uint64(data[offset:]))
				offset += 8

				// OffsetLag
				if offset+8 > len(data) {
					return nil, errors.New("insufficient data for offset lag")
				}
				resp.LogDirs[i].Topics[j].Partitions[k].OffsetLag = int64(binary.BigEndian.Uint64(data[offset:]))
				offset += 8

				// IsFuture
				if offset+1 > len(data) {
					return nil, errors.New("insufficient data for is future")
				}
				resp.LogDirs[i].Topics[j].Partitions[k].IsFuture = data[offset] != 0
				offset += 1
			}
		}
	}

	return resp, nil
}
