// Copyright 2025 Takhin Data, Inc.

package protocol

// FindCoordinatorRequest is used to find the coordinator for a group or transaction
type FindCoordinatorRequest struct {
	Key     string // Group ID or Transaction ID
	KeyType int8   // 0 = Group, 1 = Transaction
}

// FindCoordinatorResponse contains the coordinator information
type FindCoordinatorResponse struct {
	ErrorCode    int16  // Error code
	ErrorMessage string // Error message (v1+)
	NodeID       int32  // Coordinator node ID
	Host         string // Coordinator host
	Port         int32  // Coordinator port
}

// Encode encodes the FindCoordinatorRequest
func (r *FindCoordinatorRequest) Encode() []byte {
	buf := make([]byte, 0, 256)

	// Key
	buf = append(buf, encodeString(r.Key)...)

	// KeyType (v1+)
	buf = append(buf, byte(r.KeyType))

	return buf
}

// Decode decodes the FindCoordinatorRequest
func (r *FindCoordinatorRequest) Decode(data []byte) error {
	offset := 0

	// Key
	key, n := decodeString(data[offset:])
	r.Key = key
	offset += n

	// KeyType (v1+)
	if offset < len(data) {
		r.KeyType = int8(data[offset])
		offset++
	}

	return nil
}

// Encode encodes the FindCoordinatorResponse
func (r *FindCoordinatorResponse) Encode() []byte {
	buf := make([]byte, 0, 256)

	// Error code
	buf = append(buf, encodeInt16(r.ErrorCode)...)

	// Error message (v1+)
	buf = append(buf, encodeNullableString(r.ErrorMessage)...)

	// Node ID
	buf = append(buf, encodeInt32(r.NodeID)...)

	// Host
	buf = append(buf, encodeString(r.Host)...)

	// Port
	buf = append(buf, encodeInt32(r.Port)...)

	return buf
}

// Decode decodes the FindCoordinatorResponse
func (r *FindCoordinatorResponse) Decode(data []byte) error {
	offset := 0

	// Error code
	r.ErrorCode = decodeInt16(data[offset:])
	offset += 2

	// Error message (v1+)
	if offset < len(data) {
		msg, n := decodeNullableString(data[offset:])
		r.ErrorMessage = msg
		offset += n
	}

	// Node ID
	r.NodeID = decodeInt32(data[offset:])
	offset += 4

	// Host
	host, n := decodeString(data[offset:])
	r.Host = host
	offset += n

	// Port
	r.Port = decodeInt32(data[offset:])

	return nil
}
