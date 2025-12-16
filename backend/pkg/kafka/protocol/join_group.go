// Copyright 2025 Takhin Data, Inc.

package protocol

// JoinGroupRequest is sent by a consumer to join a group
type JoinGroupRequest struct {
	GroupID          string          // Consumer group ID
	SessionTimeout   int32           // Session timeout in ms
	RebalanceTimeout int32           // Rebalance timeout in ms (v1+)
	MemberID         string          // Member ID (empty for new members)
	GroupInstanceID  string          // Static member ID (v5+)
	ProtocolType     string          // Protocol type (consumer)
	Protocols        []GroupProtocol // Supported protocols
}

// GroupProtocol represents a protocol supported by the member
type GroupProtocol struct {
	Name     string // Protocol name (range, roundrobin, etc.)
	Metadata []byte // Protocol metadata
}

// JoinGroupResponse contains the join result
type JoinGroupResponse struct {
	ErrorCode    int16             // Error code
	GenerationID int32             // Group generation ID
	ProtocolType string            // Selected protocol type (v7+)
	ProtocolName string            // Selected protocol name
	LeaderID     string            // Group leader member ID
	MemberID     string            // Assigned member ID
	Members      []JoinGroupMember // Group members (only sent to leader)
}

// JoinGroupMember represents a member in the group
type JoinGroupMember struct {
	MemberID        string // Member ID
	GroupInstanceID string // Static member ID (v5+)
	Metadata        []byte // Member metadata
}

// Encode encodes the JoinGroupRequest
func (r *JoinGroupRequest) Encode() []byte {
	buf := make([]byte, 0, 512)

	// Group ID
	buf = append(buf, encodeString(r.GroupID)...)

	// Session timeout
	buf = append(buf, encodeInt32(r.SessionTimeout)...)

	// Rebalance timeout (v1+)
	buf = append(buf, encodeInt32(r.RebalanceTimeout)...)

	// Member ID
	buf = append(buf, encodeString(r.MemberID)...)

	// Group instance ID (v5+)
	buf = append(buf, encodeNullableString(r.GroupInstanceID)...)

	// Protocol type
	buf = append(buf, encodeString(r.ProtocolType)...)

	// Protocols array
	buf = append(buf, encodeInt32(int32(len(r.Protocols)))...)
	for _, p := range r.Protocols {
		buf = append(buf, encodeString(p.Name)...)
		buf = append(buf, encodeBytes(p.Metadata)...)
	}

	return buf
}

// Decode decodes the JoinGroupRequest
func (r *JoinGroupRequest) Decode(data []byte) error {
	offset := 0

	// Group ID
	groupID, n := decodeString(data[offset:])
	r.GroupID = groupID
	offset += n

	// Session timeout
	r.SessionTimeout = decodeInt32(data[offset:])
	offset += 4

	// Rebalance timeout (v1+)
	if offset < len(data) {
		r.RebalanceTimeout = decodeInt32(data[offset:])
		offset += 4
	}

	// Member ID
	memberID, n := decodeString(data[offset:])
	r.MemberID = memberID
	offset += n

	// Group instance ID (v5+)
	if offset < len(data) {
		instanceID, n := decodeNullableString(data[offset:])
		r.GroupInstanceID = instanceID
		offset += n
	}

	// Protocol type
	protocolType, n := decodeString(data[offset:])
	r.ProtocolType = protocolType
	offset += n

	// Protocols array
	numProtocols := decodeInt32(data[offset:])
	offset += 4

	r.Protocols = make([]GroupProtocol, numProtocols)
	for i := int32(0); i < numProtocols; i++ {
		name, n := decodeString(data[offset:])
		r.Protocols[i].Name = name
		offset += n

		metadata, n := decodeBytes(data[offset:])
		r.Protocols[i].Metadata = metadata
		offset += n
	}

	return nil
}

// Encode encodes the JoinGroupResponse
func (r *JoinGroupResponse) Encode() []byte {
	buf := make([]byte, 0, 512)

	// Error code
	buf = append(buf, encodeInt16(r.ErrorCode)...)

	// Generation ID
	buf = append(buf, encodeInt32(r.GenerationID)...)

	// Protocol type (v7+)
	buf = append(buf, encodeNullableString(r.ProtocolType)...)

	// Protocol name
	buf = append(buf, encodeString(r.ProtocolName)...)

	// Leader ID
	buf = append(buf, encodeString(r.LeaderID)...)

	// Member ID
	buf = append(buf, encodeString(r.MemberID)...)

	// Members array
	buf = append(buf, encodeInt32(int32(len(r.Members)))...)
	for _, m := range r.Members {
		buf = append(buf, encodeString(m.MemberID)...)
		buf = append(buf, encodeNullableString(m.GroupInstanceID)...)
		buf = append(buf, encodeBytes(m.Metadata)...)
	}

	return buf
}

// Decode decodes the JoinGroupResponse
func (r *JoinGroupResponse) Decode(data []byte) error {
	offset := 0

	// Error code
	r.ErrorCode = decodeInt16(data[offset:])
	offset += 2

	// Generation ID
	r.GenerationID = decodeInt32(data[offset:])
	offset += 4

	// Protocol type (v7+)
	protocolType, n := decodeNullableString(data[offset:])
	r.ProtocolType = protocolType
	offset += n

	// Protocol name
	protocolName, n := decodeString(data[offset:])
	r.ProtocolName = protocolName
	offset += n

	// Leader ID
	leaderID, n := decodeString(data[offset:])
	r.LeaderID = leaderID
	offset += n

	// Member ID
	memberID, n := decodeString(data[offset:])
	r.MemberID = memberID
	offset += n

	// Members array
	numMembers := decodeInt32(data[offset:])
	offset += 4

	r.Members = make([]JoinGroupMember, numMembers)
	for i := int32(0); i < numMembers; i++ {
		memberID, n := decodeString(data[offset:])
		r.Members[i].MemberID = memberID
		offset += n

		instanceID, n := decodeNullableString(data[offset:])
		r.Members[i].GroupInstanceID = instanceID
		offset += n

		metadata, n := decodeBytes(data[offset:])
		r.Members[i].Metadata = metadata
		offset += n
	}

	return nil
}
