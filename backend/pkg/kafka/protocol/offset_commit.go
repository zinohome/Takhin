// Copyright 2025 Takhin Data, Inc.

package protocol

// OffsetCommitRequest commits offsets for a consumer group
type OffsetCommitRequest struct {
	GroupID         string                     // Consumer group ID
	GenerationID    int32                      // Group generation ID (v1+)
	MemberID        string                     // Member ID (v1+)
	GroupInstanceID string                     // Static member ID (v7+)
	RetentionTime   int64                      // Retention time in ms (v2-v4, -1 = default)
	Topics          []OffsetCommitRequestTopic // Topics and partitions
}

// OffsetCommitRequestTopic represents a topic in the commit request
type OffsetCommitRequestTopic struct {
	Name       string                         // Topic name
	Partitions []OffsetCommitRequestPartition // Partitions
}

// OffsetCommitRequestPartition represents a partition offset to commit
type OffsetCommitRequestPartition struct {
	PartitionIndex int32  // Partition index
	Offset         int64  // Offset to commit
	LeaderEpoch    int32  // Leader epoch (v6+)
	Metadata       string // Custom metadata
}

// OffsetCommitResponse contains commit results
type OffsetCommitResponse struct {
	Topics []OffsetCommitResponseTopic // Topics and results
}

// OffsetCommitResponseTopic represents a topic in the commit response
type OffsetCommitResponseTopic struct {
	Name       string                          // Topic name
	Partitions []OffsetCommitResponsePartition // Partitions
}

// OffsetCommitResponsePartition represents a partition commit result
type OffsetCommitResponsePartition struct {
	PartitionIndex int32 // Partition index
	ErrorCode      int16 // Error code
}

// Encode encodes the OffsetCommitRequest
func (r *OffsetCommitRequest) Encode() []byte {
	buf := make([]byte, 0, 512)

	// Group ID
	buf = append(buf, encodeString(r.GroupID)...)

	// Generation ID (v1+)
	buf = append(buf, encodeInt32(r.GenerationID)...)

	// Member ID (v1+)
	buf = append(buf, encodeString(r.MemberID)...)

	// Group instance ID (v7+)
	buf = append(buf, encodeNullableString(r.GroupInstanceID)...)

	// Retention time (v2-v4)
	buf = append(buf, encodeInt64(r.RetentionTime)...)

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
		}
	}

	return buf
}

// Decode decodes the OffsetCommitRequest
func (r *OffsetCommitRequest) Decode(data []byte) error {
	offset := 0

	// Group ID
	groupID, n := decodeString(data[offset:])
	r.GroupID = groupID
	offset += n

	// Generation ID (v1+)
	r.GenerationID = decodeInt32(data[offset:])
	offset += 4

	// Member ID (v1+)
	memberID, n := decodeString(data[offset:])
	r.MemberID = memberID
	offset += n

	// Group instance ID (v7+)
	if offset < len(data) {
		instanceID, n := decodeNullableString(data[offset:])
		r.GroupInstanceID = instanceID
		offset += n
	}

	// Retention time (v2-v4)
	if offset < len(data) {
		r.RetentionTime = decodeInt64(data[offset:])
		offset += 8
	}

	// Topics array
	numTopics := decodeInt32(data[offset:])
	offset += 4

	r.Topics = make([]OffsetCommitRequestTopic, numTopics)
	for i := int32(0); i < numTopics; i++ {
		name, n := decodeString(data[offset:])
		r.Topics[i].Name = name
		offset += n

		// Partitions array
		numPartitions := decodeInt32(data[offset:])
		offset += 4

		r.Topics[i].Partitions = make([]OffsetCommitRequestPartition, numPartitions)
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
		}
	}

	return nil
}

// Encode encodes the OffsetCommitResponse
func (r *OffsetCommitResponse) Encode() []byte {
	buf := make([]byte, 0, 256)

	// Topics array
	buf = append(buf, encodeInt32(int32(len(r.Topics)))...)
	for _, topic := range r.Topics {
		buf = append(buf, encodeString(topic.Name)...)

		// Partitions array
		buf = append(buf, encodeInt32(int32(len(topic.Partitions)))...)
		for _, partition := range topic.Partitions {
			buf = append(buf, encodeInt32(partition.PartitionIndex)...)
			buf = append(buf, encodeInt16(partition.ErrorCode)...)
		}
	}

	return buf
}

// Decode decodes the OffsetCommitResponse
func (r *OffsetCommitResponse) Decode(data []byte) error {
	offset := 0

	// Topics array
	numTopics := decodeInt32(data[offset:])
	offset += 4

	r.Topics = make([]OffsetCommitResponseTopic, numTopics)
	for i := int32(0); i < numTopics; i++ {
		name, n := decodeString(data[offset:])
		r.Topics[i].Name = name
		offset += n

		// Partitions array
		numPartitions := decodeInt32(data[offset:])
		offset += 4

		r.Topics[i].Partitions = make([]OffsetCommitResponsePartition, numPartitions)
		for j := int32(0); j < numPartitions; j++ {
			r.Topics[i].Partitions[j].PartitionIndex = decodeInt32(data[offset:])
			offset += 4

			r.Topics[i].Partitions[j].ErrorCode = decodeInt16(data[offset:])
			offset += 2
		}
	}

	return nil
}
