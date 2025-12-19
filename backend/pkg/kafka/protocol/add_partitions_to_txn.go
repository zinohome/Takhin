// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// AddPartitionsToTxnRequest represents a request to add partitions to a transaction
type AddPartitionsToTxnRequest struct {
	TransactionalID string
	ProducerID      int64
	ProducerEpoch   int16
	Topics          []AddPartitionsToTxnTopic
}

// AddPartitionsToTxnTopic represents a topic in the AddPartitionsToTxn request
type AddPartitionsToTxnTopic struct {
	Name       string
	Partitions []int32
}

// AddPartitionsToTxnResponse represents the response for AddPartitionsToTxn
type AddPartitionsToTxnResponse struct {
	ThrottleTimeMs int32
	Results        []AddPartitionsToTxnTopicResult
}

// AddPartitionsToTxnTopicResult represents the result for a topic
type AddPartitionsToTxnTopicResult struct {
	Name             string
	PartitionResults []AddPartitionsToTxnPartitionResult
}

// AddPartitionsToTxnPartitionResult represents the result for a partition
type AddPartitionsToTxnPartitionResult struct {
	PartitionIndex int32
	ErrorCode      ErrorCode
}

// DecodeAddPartitionsToTxnRequest decodes an AddPartitionsToTxn request
func DecodeAddPartitionsToTxnRequest(r io.Reader, version int16) (*AddPartitionsToTxnRequest, error) {
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

	// Read Topics array
	topicsLen, err := ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read topics length: %w", err)
	}

	topics := make([]AddPartitionsToTxnTopic, topicsLen)
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

		partitions := make([]int32, partitionsLen)
		for j := int32(0); j < partitionsLen; j++ {
			partition, err := ReadInt32(r)
			if err != nil {
				return nil, fmt.Errorf("failed to read partition: %w", err)
			}
			partitions[j] = partition
		}

		topics[i] = AddPartitionsToTxnTopic{
			Name:       name,
			Partitions: partitions,
		}
	}

	return &AddPartitionsToTxnRequest{
		TransactionalID: transactionalID,
		ProducerID:      producerID,
		ProducerEpoch:   producerEpoch,
		Topics:          topics,
	}, nil
}

// EncodeAddPartitionsToTxnRequest encodes an AddPartitionsToTxn request
func EncodeAddPartitionsToTxnRequest(req *AddPartitionsToTxnRequest, version int16) ([]byte, error) {
	buf := make([]byte, 0, 256)

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

	// Write Topics array length
	topicsLen := make([]byte, 4)
	binary.BigEndian.PutUint32(topicsLen, uint32(len(req.Topics)))
	buf = append(buf, topicsLen...)

	for _, topic := range req.Topics {
		// Write topic name
		nameLen := make([]byte, 2)
		binary.BigEndian.PutUint16(nameLen, uint16(len(topic.Name)))
		buf = append(buf, nameLen...)
		buf = append(buf, []byte(topic.Name)...)

		// Write partitions array length
		partLen := make([]byte, 4)
		binary.BigEndian.PutUint32(partLen, uint32(len(topic.Partitions)))
		buf = append(buf, partLen...)

		for _, partition := range topic.Partitions {
			partBytes := make([]byte, 4)
			binary.BigEndian.PutUint32(partBytes, uint32(partition))
			buf = append(buf, partBytes...)
		}
	}

	return buf, nil
}

// DecodeAddPartitionsToTxnResponse decodes an AddPartitionsToTxn response
func DecodeAddPartitionsToTxnResponse(r io.Reader, version int16) (*AddPartitionsToTxnResponse, error) {
	// Read ThrottleTimeMs
	throttleTimeMs, err := ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read throttle_time_ms: %w", err)
	}

	// Read Results array
	resultsLen, err := ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read results length: %w", err)
	}

	results := make([]AddPartitionsToTxnTopicResult, resultsLen)
	for i := int32(0); i < resultsLen; i++ {
		// Read topic name
		name, err := ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read topic name: %w", err)
		}

		// Read partition results array
		partitionResultsLen, err := ReadInt32(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read partition results length: %w", err)
		}

		partitionResults := make([]AddPartitionsToTxnPartitionResult, partitionResultsLen)
		for j := int32(0); j < partitionResultsLen; j++ {
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

			partitionResults[j] = AddPartitionsToTxnPartitionResult{
				PartitionIndex: partitionIndex,
				ErrorCode:      ErrorCode(errorCode),
			}
		}

		results[i] = AddPartitionsToTxnTopicResult{
			Name:             name,
			PartitionResults: partitionResults,
		}
	}

	return &AddPartitionsToTxnResponse{
		ThrottleTimeMs: throttleTimeMs,
		Results:        results,
	}, nil
}

// EncodeAddPartitionsToTxnResponse encodes an AddPartitionsToTxn response
func EncodeAddPartitionsToTxnResponse(resp *AddPartitionsToTxnResponse, version int16) ([]byte, error) {
	buf := make([]byte, 0, 256)

	// Write ThrottleTimeMs
	throttle := make([]byte, 4)
	binary.BigEndian.PutUint32(throttle, uint32(resp.ThrottleTimeMs))
	buf = append(buf, throttle...)

	// Write Results array length
	resultsLen := make([]byte, 4)
	binary.BigEndian.PutUint32(resultsLen, uint32(len(resp.Results)))
	buf = append(buf, resultsLen...)

	for _, result := range resp.Results {
		// Write topic name
		nameLen := make([]byte, 2)
		binary.BigEndian.PutUint16(nameLen, uint16(len(result.Name)))
		buf = append(buf, nameLen...)
		buf = append(buf, []byte(result.Name)...)

		// Write partition results array length
		partLen := make([]byte, 4)
		binary.BigEndian.PutUint32(partLen, uint32(len(result.PartitionResults)))
		buf = append(buf, partLen...)

		for _, partitionResult := range result.PartitionResults {
			// Write partition index
			partIdx := make([]byte, 4)
			binary.BigEndian.PutUint32(partIdx, uint32(partitionResult.PartitionIndex))
			buf = append(buf, partIdx...)

			// Write error code
			errCode := make([]byte, 2)
			binary.BigEndian.PutUint16(errCode, uint16(partitionResult.ErrorCode))
			buf = append(buf, errCode...)
		}
	}

	return buf, nil
}

// WriteAddPartitionsToTxnResponse writes the response to a writer
func WriteAddPartitionsToTxnResponse(w io.Writer, header *RequestHeader, resp *AddPartitionsToTxnResponse) error {
	// Encode the response
	respData, err := EncodeAddPartitionsToTxnResponse(resp, header.APIVersion)
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
