// Copyright 2025 Takhin Data, Inc.

package protocol

// SyncGroupRequest is sent to synchronize group state
type SyncGroupRequest struct {
	GroupID         string                // Consumer group ID
	GenerationID    int32                 // Group generation ID
	MemberID        string                // Member ID
	GroupInstanceID string                // Static member ID (v3+)
	ProtocolType    string                // Protocol type (v5+)
	ProtocolName    string                // Protocol name (v5+)
	Assignments     []SyncGroupAssignment // Partition assignments (only from leader)
}

// SyncGroupAssignment represents a member's partition assignment
type SyncGroupAssignment struct {
	MemberID   string // Member ID
	Assignment []byte // Serialized assignment
}

// SyncGroupResponse contains the member's assignment
type SyncGroupResponse struct {
	ErrorCode    int16  // Error code
	ProtocolType string // Protocol type (v5+)
	ProtocolName string // Protocol name (v5+)
	Assignment   []byte // Member's partition assignment
}

// Encode encodes the SyncGroupRequest
func (r *SyncGroupRequest) Encode() []byte {
	buf := make([]byte, 0, 512)

	// Group ID
	buf = append(buf, encodeString(r.GroupID)...)

	// Generation ID
	buf = append(buf, encodeInt32(r.GenerationID)...)

	// Member ID
	buf = append(buf, encodeString(r.MemberID)...)

	// Group instance ID (v3+)
	buf = append(buf, encodeNullableString(r.GroupInstanceID)...)

	// Protocol type (v5+)
	buf = append(buf, encodeNullableString(r.ProtocolType)...)

	// Protocol name (v5+)
	buf = append(buf, encodeNullableString(r.ProtocolName)...)

	// Assignments array
	buf = append(buf, encodeInt32(int32(len(r.Assignments)))...)
	for _, a := range r.Assignments {
		buf = append(buf, encodeString(a.MemberID)...)
		buf = append(buf, encodeBytes(a.Assignment)...)
	}

	return buf
}

// Decode decodes the SyncGroupRequest
func (r *SyncGroupRequest) Decode(data []byte) error {
	offset := 0

	// Group ID
	groupID, n := decodeString(data[offset:])
	r.GroupID = groupID
	offset += n

	// Generation ID
	r.GenerationID = decodeInt32(data[offset:])
	offset += 4

	// Member ID
	memberID, n := decodeString(data[offset:])
	r.MemberID = memberID
	offset += n

	// Group instance ID (v3+)
	if offset < len(data) {
		instanceID, n := decodeNullableString(data[offset:])
		r.GroupInstanceID = instanceID
		offset += n
	}

	// Protocol type (v5+)
	if offset < len(data) {
		protocolType, n := decodeNullableString(data[offset:])
		r.ProtocolType = protocolType
		offset += n
	}

	// Protocol name (v5+)
	if offset < len(data) {
		protocolName, n := decodeNullableString(data[offset:])
		r.ProtocolName = protocolName
		offset += n
	}

	// Assignments array
	if offset < len(data) {
		numAssignments := decodeInt32(data[offset:])
		offset += 4

		r.Assignments = make([]SyncGroupAssignment, numAssignments)
		for i := int32(0); i < numAssignments; i++ {
			memberID, n := decodeString(data[offset:])
			r.Assignments[i].MemberID = memberID
			offset += n

			assignment, n := decodeBytes(data[offset:])
			r.Assignments[i].Assignment = assignment
			offset += n
		}
	}

	return nil
}

// Encode encodes the SyncGroupResponse
func (r *SyncGroupResponse) Encode() []byte {
	buf := make([]byte, 0, 256)

	// Error code
	buf = append(buf, encodeInt16(r.ErrorCode)...)

	// Protocol type (v5+)
	buf = append(buf, encodeNullableString(r.ProtocolType)...)

	// Protocol name (v5+)
	buf = append(buf, encodeNullableString(r.ProtocolName)...)

	// Assignment
	buf = append(buf, encodeBytes(r.Assignment)...)

	return buf
}

// Decode decodes the SyncGroupResponse
func (r *SyncGroupResponse) Decode(data []byte) error {
	offset := 0

	// Error code
	r.ErrorCode = decodeInt16(data[offset:])
	offset += 2

	// Protocol type (v5+)
	if offset < len(data) {
		protocolType, n := decodeNullableString(data[offset:])
		r.ProtocolType = protocolType
		offset += n
	}

	// Protocol name (v5+)
	if offset < len(data) {
		protocolName, n := decodeNullableString(data[offset:])
		r.ProtocolName = protocolName
		offset += n
	}

	// Assignment
	assignment, _ := decodeBytes(data[offset:])
	r.Assignment = assignment

	return nil
}
