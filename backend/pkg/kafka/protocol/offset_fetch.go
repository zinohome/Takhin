// Copyright 2025 Takhin Data, Inc.

package protocol

// OffsetFetchRequest fetches committed offsets for a consumer group
type OffsetFetchRequest struct {
	GroupID         string                    // Consumer group ID
	GroupInstanceID string                    // Static member ID (v7+)
	RequireStable   bool                      // Require stable offsets (v7+)
	Topics          []OffsetFetchRequestTopic // Topics and partitions (nil = all)
}

// OffsetFetchRequestTopic represents a topic in the fetch request
type OffsetFetchRequestTopic struct {
	Name             string  // Topic name
	PartitionIndexes []int32 // Partition indexes (nil = all)
}

// OffsetFetchResponse contains committed offsets
type OffsetFetchResponse struct {
	ErrorCode int16                      // Error code (v2+)
	Topics    []OffsetFetchResponseTopic // Topics and offsets
}

// OffsetFetchResponseTopic represents a topic in the fetch response
type OffsetFetchResponseTopic struct {
	Name       string                         // Topic name
	Partitions []OffsetFetchResponsePartition // Partitions
}

// OffsetFetchResponsePartition represents a partition offset
type OffsetFetchResponsePartition struct {
	PartitionIndex int32  // Partition index
	Offset         int64  // Committed offset
	LeaderEpoch    int32  // Leader epoch (v5+)
	Metadata       string // Custom metadata
	ErrorCode      int16  // Error code
}

// Encode encodes the OffsetFetchRequest
func (r *OffsetFetchRequest) Encode() []byte {
	buf := make([]byte, 0, 256)

	// Group ID
	buf = append(buf, encodeString(r.GroupID)...)

	// Group instance ID (v7+)
	buf = append(buf, encodeNullableString(r.GroupInstanceID)...)

	// Require stable (v7+)
	if r.RequireStable {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}

	// Topics array (nil = all topics)
	if r.Topics == nil {
		buf = append(buf, encodeInt32(-1)...)
	} else {
		buf = append(buf, encodeInt32(int32(len(r.Topics)))...)
		for _, topic := range r.Topics {
			buf = append(buf, encodeString(topic.Name)...)

			// Partition indexes (nil = all partitions)
			if topic.PartitionIndexes == nil {
				buf = append(buf, encodeInt32(-1)...)
			} else {
				buf = append(buf, encodeInt32(int32(len(topic.PartitionIndexes)))...)
				for _, partition := range topic.PartitionIndexes {
					buf = append(buf, encodeInt32(partition)...)
				}
			}
		}
	}

	return buf
}

// Decode decodes the OffsetFetchRequest
func (r *OffsetFetchRequest) Decode(data []byte) error {
	offset := 0

	// Group ID
	groupID, n := decodeString(data[offset:])
	r.GroupID = groupID
	offset += n

	// Group instance ID (v7+)
	if offset < len(data) {
		instanceID, n := decodeNullableString(data[offset:])
		r.GroupInstanceID = instanceID
		offset += n
	}

	// Require stable (v7+)
	if offset < len(data) {
		r.RequireStable = data[offset] != 0
		offset++
	}

	// Topics array
	if offset < len(data) {
		numTopics := decodeInt32(data[offset:])
		offset += 4

		if numTopics >= 0 {
			r.Topics = make([]OffsetFetchRequestTopic, numTopics)
			for i := int32(0); i < numTopics; i++ {
				name, n := decodeString(data[offset:])
				r.Topics[i].Name = name
				offset += n

				// Partition indexes
				numPartitions := decodeInt32(data[offset:])
				offset += 4

				if numPartitions >= 0 {
					r.Topics[i].PartitionIndexes = make([]int32, numPartitions)
					for j := int32(0); j < numPartitions; j++ {
						r.Topics[i].PartitionIndexes[j] = decodeInt32(data[offset:])
						offset += 4
					}
				}
			}
		}
	}

	return nil
}

// Encode encodes the OffsetFetchResponse
func (r *OffsetFetchResponse) Encode() []byte {
	buf := make([]byte, 0, 512)

	// Error code (v2+)
	buf = append(buf, encodeInt16(r.ErrorCode)...)

	// Topics array
	buf = append(buf, encodeInt32(int32(len(r.Topics)))...)
	for _, topic := range r.Topics {
		buf = append(buf, encodeString(topic.Name)...)

		// Partitions array
		buf = append(buf, encodeInt32(int32(len(topic.Partitions)))...)
		for _, partition := range topic.Partitions {
			buf = append(buf, encodeInt32(partition.PartitionIndex)...)
			buf = append(buf, encodeInt64(partition.Offset)...)
			buf = append(buf, encodeInt32(partition.LeaderEpoch)...)
			buf = append(buf, encodeNullableString(partition.Metadata)...)
			buf = append(buf, encodeInt16(partition.ErrorCode)...)
		}
	}

	return buf
}

// Decode decodes the OffsetFetchResponse
func (r *OffsetFetchResponse) Decode(data []byte) error {
	offset := 0

	// Error code (v2+)
	r.ErrorCode = decodeInt16(data[offset:])
	offset += 2

	// Topics array
	numTopics := decodeInt32(data[offset:])
	offset += 4

	r.Topics = make([]OffsetFetchResponseTopic, numTopics)
	for i := int32(0); i < numTopics; i++ {
		name, n := decodeString(data[offset:])
		r.Topics[i].Name = name
		offset += n

		// Partitions array
		numPartitions := decodeInt32(data[offset:])
		offset += 4

		r.Topics[i].Partitions = make([]OffsetFetchResponsePartition, numPartitions)
		for j := int32(0); j < numPartitions; j++ {
			r.Topics[i].Partitions[j].PartitionIndex = decodeInt32(data[offset:])
			offset += 4

			r.Topics[i].Partitions[j].Offset = decodeInt64(data[offset:])
			offset += 8

			r.Topics[i].Partitions[j].LeaderEpoch = decodeInt32(data[offset:])
			offset += 4

			metadata, n := decodeNullableString(data[offset:])
			r.Topics[i].Partitions[j].Metadata = metadata
			offset += n

			r.Topics[i].Partitions[j].ErrorCode = decodeInt16(data[offset:])
			offset += 2
		}
	}

	return nil
}
