// Copyright 2025 Takhin Data, Inc.

package protocol

// HeartbeatRequest is sent periodically by group members
type HeartbeatRequest struct {
	GroupID         string // Consumer group ID
	GenerationID    int32  // Group generation ID
	MemberID        string // Member ID
	GroupInstanceID string // Static member ID (v3+)
}

// HeartbeatResponse indicates if the member should rejoin
type HeartbeatResponse struct {
	ErrorCode int16 // Error code
}

// Encode encodes the HeartbeatRequest
func (r *HeartbeatRequest) Encode() []byte {
	buf := make([]byte, 0, 128)

	// Group ID
	buf = append(buf, encodeString(r.GroupID)...)

	// Generation ID
	buf = append(buf, encodeInt32(r.GenerationID)...)

	// Member ID
	buf = append(buf, encodeString(r.MemberID)...)

	// Group instance ID (v3+)
	buf = append(buf, encodeNullableString(r.GroupInstanceID)...)

	return buf
}

// Decode decodes the HeartbeatRequest
func (r *HeartbeatRequest) Decode(data []byte) error {
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
		instanceID, _ := decodeNullableString(data[offset:])
		r.GroupInstanceID = instanceID
	}

	return nil
}

// Encode encodes the HeartbeatResponse
func (r *HeartbeatResponse) Encode() []byte {
	buf := make([]byte, 0, 16)

	// Error code
	buf = append(buf, encodeInt16(r.ErrorCode)...)

	return buf
}

// Decode decodes the HeartbeatResponse
func (r *HeartbeatResponse) Decode(data []byte) error {
	// Error code
	r.ErrorCode = decodeInt16(data)

	return nil
}
