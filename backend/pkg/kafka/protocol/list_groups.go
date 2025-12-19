// Copyright 2025 Takhin Data
// Licensed under the Apache License, Version 2.0

package protocol

import (
	"encoding/binary"
)

// ListGroups API (Key: 16) - 列出所有消费者组

// ListGroupsRequest 请求结构
type ListGroupsRequest struct {
	Header *RequestHeader
}

// ListGroupsResponse 响应结构
type ListGroupsResponse struct {
	ThrottleTimeMs int32
	ErrorCode      ErrorCode
	Groups         []ListedGroup
}

// ListedGroup 组列表项
type ListedGroup struct {
	GroupID      string
	ProtocolType string
}

// DecodeListGroupsRequest 解码请求
func DecodeListGroupsRequest(data []byte, version int16) (*ListGroupsRequest, error) {
	// ListGroups 请求没有body，只有header
	return &ListGroupsRequest{}, nil
}

// EncodeListGroupsResponse 编码响应
func EncodeListGroupsResponse(resp *ListGroupsResponse, version int16) []byte {
	buf := make([]byte, 0, 1024)

	// ThrottleTimeMs
	throttle := make([]byte, 4)
	binary.BigEndian.PutUint32(throttle, uint32(resp.ThrottleTimeMs))
	buf = append(buf, throttle...)

	// ErrorCode
	errCode := make([]byte, 2)
	binary.BigEndian.PutUint16(errCode, uint16(resp.ErrorCode))
	buf = append(buf, errCode...)

	// Groups array length
	groupsLen := make([]byte, 4)
	binary.BigEndian.PutUint32(groupsLen, uint32(len(resp.Groups)))
	buf = append(buf, groupsLen...)

	for _, group := range resp.Groups {
		// GroupID
		groupIDLen := make([]byte, 2)
		binary.BigEndian.PutUint16(groupIDLen, uint16(len(group.GroupID)))
		buf = append(buf, groupIDLen...)
		buf = append(buf, []byte(group.GroupID)...)

		// ProtocolType
		protocolTypeLen := make([]byte, 2)
		binary.BigEndian.PutUint16(protocolTypeLen, uint16(len(group.ProtocolType)))
		buf = append(buf, protocolTypeLen...)
		buf = append(buf, []byte(group.ProtocolType)...)
	}

	return buf
}
