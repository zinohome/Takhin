// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// WriteTxnMarkersRequest represents a request to write transaction markers
type WriteTxnMarkersRequest struct {
	Markers []TxnMarkerEntry
}

// TxnMarkerEntry represents a transaction marker entry
type TxnMarkerEntry struct {
	ProducerID        int64
	ProducerEpoch     int16
	TransactionResult bool // true = COMMIT, false = ABORT
	Topics            []TxnMarkerTopic
	CoordinatorEpoch  int32
}

// TxnMarkerTopic represents topics and partitions for transaction markers
type TxnMarkerTopic struct {
	Topic      string
	Partitions []int32
}

// WriteTxnMarkersResponse represents the response for WriteTxnMarkers
type WriteTxnMarkersResponse struct {
	Markers []TxnMarkerResult
}

// TxnMarkerResult represents the result of writing markers
type TxnMarkerResult struct {
	ProducerID int64
	Topics     []TxnMarkerTopicResult
}

// TxnMarkerTopicResult represents per-topic results
type TxnMarkerTopicResult struct {
	Topic      string
	Partitions []TxnMarkerPartitionResult
}

// TxnMarkerPartitionResult represents per-partition results
type TxnMarkerPartitionResult struct {
	PartitionIndex int32
	ErrorCode      ErrorCode
}

// DecodeWriteTxnMarkersRequest decodes a WriteTxnMarkers request
func DecodeWriteTxnMarkersRequest(r io.Reader, version int16) (*WriteTxnMarkersRequest, error) {
	// Read markers array length
	markersLen, err := ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read markers length: %w", err)
	}

	markers := make([]TxnMarkerEntry, markersLen)
	for i := int32(0); i < markersLen; i++ {
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

		// Read TransactionResult (boolean as int8)
		var transactionResult int8
		if err := binary.Read(r, binary.BigEndian, &transactionResult); err != nil {
			return nil, fmt.Errorf("failed to read transaction_result: %w", err)
		}

		// Read Topics array
		topicsLen, err := ReadInt32(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read topics length: %w", err)
		}

		topics := make([]TxnMarkerTopic, topicsLen)
		for j := int32(0); j < topicsLen; j++ {
			// Read Topic name
			topic, err := ReadString(r)
			if err != nil {
				return nil, fmt.Errorf("failed to read topic name: %w", err)
			}

			// Read Partitions array
			partitionsLen, err := ReadInt32(r)
			if err != nil {
				return nil, fmt.Errorf("failed to read partitions length: %w", err)
			}

			partitions := make([]int32, partitionsLen)
			for k := int32(0); k < partitionsLen; k++ {
				partition, err := ReadInt32(r)
				if err != nil {
					return nil, fmt.Errorf("failed to read partition: %w", err)
				}
				partitions[k] = partition
			}

			topics[j] = TxnMarkerTopic{
				Topic:      topic,
				Partitions: partitions,
			}
		}

		// Read CoordinatorEpoch
		coordinatorEpoch, err := ReadInt32(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read coordinator_epoch: %w", err)
		}

		markers[i] = TxnMarkerEntry{
			ProducerID:        producerID,
			ProducerEpoch:     producerEpoch,
			TransactionResult: transactionResult != 0,
			Topics:            topics,
			CoordinatorEpoch:  coordinatorEpoch,
		}
	}

	return &WriteTxnMarkersRequest{
		Markers: markers,
	}, nil
}

// EncodeWriteTxnMarkersResponse encodes a WriteTxnMarkers response
func EncodeWriteTxnMarkersResponse(resp *WriteTxnMarkersResponse, version int16) ([]byte, error) {
	buf := make([]byte, 0, 512)

	// Write markers array length
	markersLen := make([]byte, 4)
	binary.BigEndian.PutUint32(markersLen, uint32(len(resp.Markers)))
	buf = append(buf, markersLen...)

	for _, marker := range resp.Markers {
		// Write ProducerID
		producerID := make([]byte, 8)
		binary.BigEndian.PutUint64(producerID, uint64(marker.ProducerID))
		buf = append(buf, producerID...)

		// Write topics array length
		topicsLen := make([]byte, 4)
		binary.BigEndian.PutUint32(topicsLen, uint32(len(marker.Topics)))
		buf = append(buf, topicsLen...)

		for _, topic := range marker.Topics {
			// Write topic name
			buf = append(buf, encodeString(topic.Topic)...)

			// Write partitions array length
			partitionsLen := make([]byte, 4)
			binary.BigEndian.PutUint32(partitionsLen, uint32(len(topic.Partitions)))
			buf = append(buf, partitionsLen...)

			for _, partition := range topic.Partitions {
				// Write partition index
				partitionIdx := make([]byte, 4)
				binary.BigEndian.PutUint32(partitionIdx, uint32(partition.PartitionIndex))
				buf = append(buf, partitionIdx...)

				// Write error code
				errCode := make([]byte, 2)
				binary.BigEndian.PutUint16(errCode, uint16(partition.ErrorCode))
				buf = append(buf, errCode...)
			}
		}
	}

	return buf, nil
}

// WriteWriteTxnMarkersResponse writes the response to a writer
func WriteWriteTxnMarkersResponse(w io.Writer, header *RequestHeader, resp *WriteTxnMarkersResponse) error {
	// Encode the response
	respData, err := EncodeWriteTxnMarkersResponse(resp, header.APIVersion)
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
