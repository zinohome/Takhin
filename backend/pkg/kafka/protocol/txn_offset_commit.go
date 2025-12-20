// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// TxnOffsetCommitRequest represents a request to commit offsets as part of a transaction
type TxnOffsetCommitRequest struct {
	TransactionalID string
	GroupID         string
	ProducerID      int64
	ProducerEpoch   int16
	Topics          []TxnOffsetCommitTopic
}

// TxnOffsetCommitTopic represents a topic in the TxnOffsetCommit request
type TxnOffsetCommitTopic struct {
	Name       string
	Partitions []TxnOffsetCommitPartition
}

// TxnOffsetCommitPartition represents a partition in the TxnOffsetCommit request
type TxnOffsetCommitPartition struct {
	PartitionIndex int32
	Offset         int64
	Metadata       *string
}

// TxnOffsetCommitResponse represents the response for TxnOffsetCommit
type TxnOffsetCommitResponse struct {
	ThrottleTimeMs int32
	Topics         []TxnOffsetCommitTopicResult
}

// TxnOffsetCommitTopicResult represents the result for a topic
type TxnOffsetCommitTopicResult struct {
	Name       string
	Partitions []TxnOffsetCommitPartitionResult
}

// TxnOffsetCommitPartitionResult represents the result for a partition
type TxnOffsetCommitPartitionResult struct {
	PartitionIndex int32
	ErrorCode      ErrorCode
}

// DecodeTxnOffsetCommitRequest decodes a TxnOffsetCommit request
func DecodeTxnOffsetCommitRequest(r io.Reader, version int16) (*TxnOffsetCommitRequest, error) {
	// Read TransactionalID
	transactionalID, err := ReadString(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read transactional_id: %w", err)
	}

	// Read GroupID
	groupID, err := ReadString(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read group_id: %w", err)
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

	// Read Topics array
	topicsLen, err := ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read topics length: %w", err)
	}

	topics := make([]TxnOffsetCommitTopic, topicsLen)
	for i := int32(0); i < topicsLen; i++ {
		// Read topic name
		name, err := ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read topic name: %w", err)
		}

		// Read partitions array
		partitionsLen, err := ReadInt32(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read partitions length: %w", err)
		}

		partitions := make([]TxnOffsetCommitPartition, partitionsLen)
		for j := int32(0); j < partitionsLen; j++ {
			// Read partition index
			partitionIndex, err := ReadInt32(r)
			if err != nil {
				return nil, fmt.Errorf("failed to read partition index: %w", err)
			}

			// Read offset
			offset, err := ReadInt64(r)
			if err != nil {
				return nil, fmt.Errorf("failed to read offset: %w", err)
			}

			// Read metadata (nullable string)
			metadata, err := ReadNullableString(r)
			if err != nil {
				return nil, fmt.Errorf("failed to read metadata: %w", err)
			}

			partitions[j] = TxnOffsetCommitPartition{
				PartitionIndex: partitionIndex,
				Offset:         offset,
				Metadata:       metadata,
			}
		}

		topics[i] = TxnOffsetCommitTopic{
			Name:       name,
			Partitions: partitions,
		}
	}

	return &TxnOffsetCommitRequest{
		TransactionalID: transactionalID,
		GroupID:         groupID,
		ProducerID:      producerID,
		ProducerEpoch:   producerEpoch,
		Topics:          topics,
	}, nil
}

// EncodeTxnOffsetCommitRequest encodes a TxnOffsetCommit request
func EncodeTxnOffsetCommitRequest(req *TxnOffsetCommitRequest, version int16) ([]byte, error) {
	buf := make([]byte, 0, 256)

	// Write TransactionalID
	strLen := make([]byte, 2)
	binary.BigEndian.PutUint16(strLen, uint16(len(req.TransactionalID)))
	buf = append(buf, strLen...)
	buf = append(buf, []byte(req.TransactionalID)...)

	// Write GroupID
	groupLen := make([]byte, 2)
	binary.BigEndian.PutUint16(groupLen, uint16(len(req.GroupID)))
	buf = append(buf, groupLen...)
	buf = append(buf, []byte(req.GroupID)...)

	// Write ProducerID
	producerID := make([]byte, 8)
	binary.BigEndian.PutUint64(producerID, uint64(req.ProducerID))
	buf = append(buf, producerID...)

	// Write ProducerEpoch
	producerEpoch := make([]byte, 2)
	binary.BigEndian.PutUint16(producerEpoch, uint16(req.ProducerEpoch))
	buf = append(buf, producerEpoch...)

	// Write Topics array
	topicsLen := make([]byte, 4)
	binary.BigEndian.PutUint32(topicsLen, uint32(len(req.Topics)))
	buf = append(buf, topicsLen...)

	for _, topic := range req.Topics {
		// Write topic name
		nameLen := make([]byte, 2)
		binary.BigEndian.PutUint16(nameLen, uint16(len(topic.Name)))
		buf = append(buf, nameLen...)
		buf = append(buf, []byte(topic.Name)...)

		// Write partitions array
		partLen := make([]byte, 4)
		binary.BigEndian.PutUint32(partLen, uint32(len(topic.Partitions)))
		buf = append(buf, partLen...)

		for _, partition := range topic.Partitions {
			// Write partition index
			partIdx := make([]byte, 4)
			binary.BigEndian.PutUint32(partIdx, uint32(partition.PartitionIndex))
			buf = append(buf, partIdx...)

			// Write offset
			offset := make([]byte, 8)
			binary.BigEndian.PutUint64(offset, uint64(partition.Offset))
			buf = append(buf, offset...)

			// Write metadata (nullable string)
			if partition.Metadata == nil {
				nullLen := make([]byte, 2)
				binary.BigEndian.PutUint16(nullLen, 0xFFFF)
				buf = append(buf, nullLen...)
			} else {
				metaLen := make([]byte, 2)
				binary.BigEndian.PutUint16(metaLen, uint16(len(*partition.Metadata)))
				buf = append(buf, metaLen...)
				buf = append(buf, []byte(*partition.Metadata)...)
			}
		}
	}

	return buf, nil
}

// DecodeTxnOffsetCommitResponse decodes a TxnOffsetCommit response
func DecodeTxnOffsetCommitResponse(r io.Reader, version int16) (*TxnOffsetCommitResponse, error) {
	// Read ThrottleTimeMs
	throttleTimeMs, err := ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read throttle_time_ms: %w", err)
	}

	// Read Topics array
	topicsLen, err := ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read topics length: %w", err)
	}

	topics := make([]TxnOffsetCommitTopicResult, topicsLen)
	for i := int32(0); i < topicsLen; i++ {
		// Read topic name
		name, err := ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read topic name: %w", err)
		}

		// Read partitions array
		partitionsLen, err := ReadInt32(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read partitions length: %w", err)
		}

		partitions := make([]TxnOffsetCommitPartitionResult, partitionsLen)
		for j := int32(0); j < partitionsLen; j++ {
			// Read partition index
			partitionIndex, err := ReadInt32(r)
			if err != nil {
				return nil, fmt.Errorf("failed to read partition index: %w", err)
			}

			// Read error code
			errorCode, err := ReadInt16(r)
			if err != nil {
				return nil, fmt.Errorf("failed to read error code: %w", err)
			}

			partitions[j] = TxnOffsetCommitPartitionResult{
				PartitionIndex: partitionIndex,
				ErrorCode:      ErrorCode(errorCode),
			}
		}

		topics[i] = TxnOffsetCommitTopicResult{
			Name:       name,
			Partitions: partitions,
		}
	}

	return &TxnOffsetCommitResponse{
		ThrottleTimeMs: throttleTimeMs,
		Topics:         topics,
	}, nil
}

// EncodeTxnOffsetCommitResponse encodes a TxnOffsetCommit response
func EncodeTxnOffsetCommitResponse(resp *TxnOffsetCommitResponse, version int16) ([]byte, error) {
	buf := make([]byte, 0, 256)

	// Write ThrottleTimeMs
	throttle := make([]byte, 4)
	binary.BigEndian.PutUint32(throttle, uint32(resp.ThrottleTimeMs))
	buf = append(buf, throttle...)

	// Write Topics array
	topicsLen := make([]byte, 4)
	binary.BigEndian.PutUint32(topicsLen, uint32(len(resp.Topics)))
	buf = append(buf, topicsLen...)

	for _, topic := range resp.Topics {
		// Write topic name
		nameLen := make([]byte, 2)
		binary.BigEndian.PutUint16(nameLen, uint16(len(topic.Name)))
		buf = append(buf, nameLen...)
		buf = append(buf, []byte(topic.Name)...)

		// Write partitions array
		partLen := make([]byte, 4)
		binary.BigEndian.PutUint32(partLen, uint32(len(topic.Partitions)))
		buf = append(buf, partLen...)

		for _, partition := range topic.Partitions {
			// Write partition index
			partIdx := make([]byte, 4)
			binary.BigEndian.PutUint32(partIdx, uint32(partition.PartitionIndex))
			buf = append(buf, partIdx...)

			// Write error code
			errCode := make([]byte, 2)
			binary.BigEndian.PutUint16(errCode, uint16(partition.ErrorCode))
			buf = append(buf, errCode...)
		}
	}

	return buf, nil
}

// WriteTxnOffsetCommitResponse writes the response to a writer
func WriteTxnOffsetCommitResponse(w io.Writer, header *RequestHeader, resp *TxnOffsetCommitResponse) error {
	// Encode the response
	respData, err := EncodeTxnOffsetCommitResponse(resp, header.APIVersion)
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
