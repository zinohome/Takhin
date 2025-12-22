// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"io"
)

// FetchRequest represents a Fetch request
type FetchRequest struct {
	Header         *RequestHeader
	ReplicaID      int32 // -1 for consumer, broker ID for follower fetch
	MaxWaitMs      int32
	MinBytes       int32
	MaxBytes       int32
	IsolationLevel int8
	Topics         []FetchTopic
}

// FetchTopic represents a topic in a Fetch request
type FetchTopic struct {
	TopicName  string
	Partitions []FetchPartition
}

// FetchPartition represents a partition in a Fetch request
type FetchPartition struct {
	PartitionIndex    int32
	FetchOffset       int64
	PartitionMaxBytes int32
}

// FetchResponse represents a Fetch response
type FetchResponse struct {
	ThrottleTimeMs int32
	ErrorCode      ErrorCode
	SessionID      int32
	Responses      []FetchTopicResponse
}

// FetchTopicResponse represents a topic response in a Fetch response
type FetchTopicResponse struct {
	TopicName          string
	PartitionResponses []FetchPartitionResponse
}

// FetchPartitionResponse represents a partition response in a Fetch response
type FetchPartitionResponse struct {
	PartitionIndex   int32
	ErrorCode        ErrorCode
	HighWatermark    int64
	LastStableOffset int64
	LogStartOffset   int64
	Records          []byte
}

// DecodeFetchRequest decodes a Fetch request
func DecodeFetchRequest(r io.Reader, header *RequestHeader) (*FetchRequest, error) {
	req := &FetchRequest{
		Header: header,
	}

	// Read replica ID
	replicaID, err := ReadInt32(r)
	if err != nil {
		return nil, err
	}
	req.ReplicaID = replicaID

	// Read max wait ms
	maxWaitMs, err := ReadInt32(r)
	if err != nil {
		return nil, err
	}
	req.MaxWaitMs = maxWaitMs

	// Read min bytes
	minBytes, err := ReadInt32(r)
	if err != nil {
		return nil, err
	}
	req.MinBytes = minBytes

	// Read max bytes
	maxBytes, err := ReadInt32(r)
	if err != nil {
		return nil, err
	}
	req.MaxBytes = maxBytes

	// Read isolation level
	isolationLevel, err := ReadInt8(r)
	if err != nil {
		return nil, err
	}
	req.IsolationLevel = isolationLevel

	// Read topics array length
	topicCount, err := ReadArrayLength(r)
	if err != nil {
		return nil, err
	}

	req.Topics = make([]FetchTopic, topicCount)
	for i := int32(0); i < topicCount; i++ {
		// Read topic name
		topicName, err := ReadString(r)
		if err != nil {
			return nil, err
		}

		// Read partitions array length
		partitionCount, err := ReadArrayLength(r)
		if err != nil {
			return nil, err
		}

		partitions := make([]FetchPartition, partitionCount)
		for j := int32(0); j < partitionCount; j++ {
			// Read partition index
			partitionIndex, err := ReadInt32(r)
			if err != nil {
				return nil, err
			}

			// Read fetch offset
			fetchOffset, err := ReadInt64(r)
			if err != nil {
				return nil, err
			}

			// Read partition max bytes
			partitionMaxBytes, err := ReadInt32(r)
			if err != nil {
				return nil, err
			}

			partitions[j] = FetchPartition{
				PartitionIndex:    partitionIndex,
				FetchOffset:       fetchOffset,
				PartitionMaxBytes: partitionMaxBytes,
			}
		}

		req.Topics[i] = FetchTopic{
			TopicName:  topicName,
			Partitions: partitions,
		}
	}

	return req, nil
}

// Encode encodes the Fetch response
func (r *FetchResponse) Encode(w io.Writer) error {
	// Write throttle time
	if err := WriteInt32(w, r.ThrottleTimeMs); err != nil {
		return err
	}

	// Write error code
	if err := WriteInt16(w, int16(r.ErrorCode)); err != nil {
		return err
	}

	// Write session ID
	if err := WriteInt32(w, r.SessionID); err != nil {
		return err
	}

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

			// Write high watermark
			if err := WriteInt64(w, partResp.HighWatermark); err != nil {
				return err
			}

			// Write last stable offset
			if err := WriteInt64(w, partResp.LastStableOffset); err != nil {
				return err
			}

			// Write log start offset
			if err := WriteInt64(w, partResp.LogStartOffset); err != nil {
				return err
			}

			// Write records
			if err := WriteBytes(w, partResp.Records); err != nil {
				return err
			}
		}
	}

	return nil
}
