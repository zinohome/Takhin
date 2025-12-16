// Copyright 2025 Takhin Data, Inc.

package protocol

// LeaveGroupRequest is sent when a member wants to leave
type LeaveGroupRequest struct {
	GroupID  string             // Consumer group ID
	MemberID string             // Member ID (v0-v2)
	Members  []LeaveGroupMember // Members leaving (v3+)
}

// LeaveGroupMember represents a member leaving the group
type LeaveGroupMember struct {
	MemberID        string // Member ID
	GroupInstanceID string // Static member ID (v3+)
	Reason          string // Leave reason (v5+)
}

// LeaveGroupResponse indicates if the leave was successful
type LeaveGroupResponse struct {
	ErrorCode int16                      // Error code
	Members   []LeaveGroupMemberResponse // Member results (v3+)
}

// LeaveGroupMemberResponse represents a member leave result
type LeaveGroupMemberResponse struct {
	MemberID        string // Member ID
	GroupInstanceID string // Static member ID (v3+)
	ErrorCode       int16  // Error code
}

// Encode encodes the LeaveGroupRequest
func (r *LeaveGroupRequest) Encode() []byte {
	buf := make([]byte, 0, 256)

	// Group ID
	buf = append(buf, encodeString(r.GroupID)...)

	// Member ID (v0-v2)
	buf = append(buf, encodeString(r.MemberID)...)

	// Members array (v3+)
	buf = append(buf, encodeInt32(int32(len(r.Members)))...)
	for _, member := range r.Members {
		buf = append(buf, encodeString(member.MemberID)...)
		buf = append(buf, encodeNullableString(member.GroupInstanceID)...)
		buf = append(buf, encodeNullableString(member.Reason)...)
	}

	return buf
}

// Decode decodes the LeaveGroupRequest
func (r *LeaveGroupRequest) Decode(data []byte) error {
	offset := 0

	// Group ID
	groupID, n := decodeString(data[offset:])
	r.GroupID = groupID
	offset += n

	// Member ID (v0-v2)
	if offset < len(data) {
		memberID, n := decodeString(data[offset:])
		r.MemberID = memberID
		offset += n
	}

	// Members array (v3+)
	if offset < len(data) {
		numMembers := decodeInt32(data[offset:])
		offset += 4

		r.Members = make([]LeaveGroupMember, numMembers)
		for i := int32(0); i < numMembers; i++ {
			memberID, n := decodeString(data[offset:])
			r.Members[i].MemberID = memberID
			offset += n

			instanceID, n := decodeNullableString(data[offset:])
			r.Members[i].GroupInstanceID = instanceID
			offset += n

			reason, n := decodeNullableString(data[offset:])
			r.Members[i].Reason = reason
			offset += n
		}
	}

	return nil
}

// Encode encodes the LeaveGroupResponse
func (r *LeaveGroupResponse) Encode() []byte {
	buf := make([]byte, 0, 128)

	// Error code
	buf = append(buf, encodeInt16(r.ErrorCode)...)

	// Members array (v3+)
	buf = append(buf, encodeInt32(int32(len(r.Members)))...)
	for _, member := range r.Members {
		buf = append(buf, encodeString(member.MemberID)...)
		buf = append(buf, encodeNullableString(member.GroupInstanceID)...)
		buf = append(buf, encodeInt16(member.ErrorCode)...)
	}

	return buf
}

// Decode decodes the LeaveGroupResponse
func (r *LeaveGroupResponse) Decode(data []byte) error {
	offset := 0

	// Error code
	r.ErrorCode = decodeInt16(data[offset:])
	offset += 2

	// Members array (v3+)
	if offset < len(data) {
		numMembers := decodeInt32(data[offset:])
		offset += 4

		r.Members = make([]LeaveGroupMemberResponse, numMembers)
		for i := int32(0); i < numMembers; i++ {
			memberID, n := decodeString(data[offset:])
			r.Members[i].MemberID = memberID
			offset += n

			instanceID, n := decodeNullableString(data[offset:])
			r.Members[i].GroupInstanceID = instanceID
			offset += n

			r.Members[i].ErrorCode = decodeInt16(data[offset:])
			offset += 2
		}
	}

	return nil
}
