// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"io"
)

// ProduceRequest represents a Produce request
type ProduceRequest struct {
	Header          *RequestHeader
	TransactionalID *string
	Acks            int16
	TimeoutMs       int32
	TopicData       []ProduceTopicData
}

// ProduceTopicData represents topic data in a Produce request
type ProduceTopicData struct {
	TopicName     string
	PartitionData []ProducePartitionData
}

// ProducePartitionData represents partition data in a Produce request
type ProducePartitionData struct {
	PartitionIndex int32
	Records        []byte
}

// ProduceResponse represents a Produce response
type ProduceResponse struct {
	Responses      []ProduceTopicResponse
	ThrottleTimeMs int32
}

// ProduceTopicResponse represents topic response in a Produce response
type ProduceTopicResponse struct {
	TopicName          string
	PartitionResponses []ProducePartitionResponse
}

// ProducePartitionResponse represents partition response in a Produce response
type ProducePartitionResponse struct {
	PartitionIndex int32
	ErrorCode      ErrorCode
	BaseOffset     int64
	LogAppendTime  int64
	LogStartOffset int64
}

// DecodeProduceRequest decodes a Produce request
func DecodeProduceRequest(r io.Reader, header *RequestHeader) (*ProduceRequest, error) {
	req := &ProduceRequest{
		Header: header,
	}

	// Read transactional ID
	transactionalID, err := ReadNullableString(r)
	if err != nil {
		return nil, err
	}
	req.TransactionalID = transactionalID

	// Read acks
	acks, err := ReadInt16(r)
	if err != nil {
		return nil, err
	}
	req.Acks = acks

	// Read timeout
	timeoutMs, err := ReadInt32(r)
	if err != nil {
		return nil, err
	}
	req.TimeoutMs = timeoutMs

	// Read topic data array length
	topicCount, err := ReadArrayLength(r)
	if err != nil {
		return nil, err
	}

	req.TopicData = make([]ProduceTopicData, topicCount)
	for i := int32(0); i < topicCount; i++ {
		// Read topic name
		topicName, err := ReadString(r)
		if err != nil {
			return nil, err
		}

		// Read partition data array length
		partitionCount, err := ReadArrayLength(r)
		if err != nil {
			return nil, err
		}

		partitionData := make([]ProducePartitionData, partitionCount)
		for j := int32(0); j < partitionCount; j++ {
			// Read partition index
			partitionIndex, err := ReadInt32(r)
			if err != nil {
				return nil, err
			}

			// Read records
			records, err := ReadBytes(r)
			if err != nil {
				return nil, err
			}

			partitionData[j] = ProducePartitionData{
				PartitionIndex: partitionIndex,
				Records:        records,
			}
		}

		req.TopicData[i] = ProduceTopicData{
			TopicName:     topicName,
			PartitionData: partitionData,
		}
	}

	return req, nil
}

// Encode encodes the Produce response
func (r *ProduceResponse) Encode(w io.Writer) error {
	// Write responses array
	if err := WriteArray(w, len(r.Responses)); err != nil {
		return err
	}

	for _, topicResp := range r.Responses {
		// Write topic name
		if err := WriteString(w, topicResp.TopicName); err != nil {
			return err
		}

		// Write partition responses array
		if err := WriteArray(w, len(topicResp.PartitionResponses)); err != nil {
			return err
		}

		for _, partResp := range topicResp.PartitionResponses {
			// Write partition index
			if err := WriteInt32(w, partResp.PartitionIndex); err != nil {
				return err
			}

			// Write error code
			if err := WriteInt16(w, int16(partResp.ErrorCode)); err != nil {
				return err
			}

			// Write base offset
			if err := WriteInt64(w, partResp.BaseOffset); err != nil {
				return err
			}

			// Write log append time
			if err := WriteInt64(w, partResp.LogAppendTime); err != nil {
				return err
			}

			// Write log start offset
			if err := WriteInt64(w, partResp.LogStartOffset); err != nil {
				return err
			}
		}
	}

	// Write throttle time
	if err := WriteInt32(w, r.ThrottleTimeMs); err != nil {
		return err
	}

	return nil
}
