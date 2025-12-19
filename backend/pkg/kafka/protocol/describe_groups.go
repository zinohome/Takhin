// Copyright 2025 Takhin Data
// Licensed under the Apache License, Version 2.0

package protocol

import (
	"encoding/binary"
	"errors"
)

// DescribeGroups API (Key: 15) - 查询消费者组详情

// DescribeGroupsRequest 请求结构
type DescribeGroupsRequest struct {
	Header                      *RequestHeader
	Groups                      []string // 要查询的组名列表
	IncludeAuthorizedOperations bool     // 是否包含授权操作(v3+)
}

// DescribeGroupsResponse 响应结构
type DescribeGroupsResponse struct {
	ThrottleTimeMs int32
	Groups         []DescribedGroup
}

// DescribedGroup 组详情
type DescribedGroup struct {
	ErrorCode            ErrorCode
	GroupID              string
	GroupState           string // Empty, PreparingRebalance, CompletingRebalance, Stable, Dead
	ProtocolType         string
	ProtocolData         string
	Members              []DescribedGroupMember
	AuthorizedOperations int32 // v3+
}

// DescribedGroupMember 组成员详情
type DescribedGroupMember struct {
	MemberID         string
	GroupInstanceID  *string // v4+
	ClientID         string
	ClientHost       string
	MemberMetadata   []byte
	MemberAssignment []byte
}

// DecodeDescribeGroupsRequest 解码请求
func DecodeDescribeGroupsRequest(data []byte, version int16) (*DescribeGroupsRequest, error) {
	req := &DescribeGroupsRequest{}
	offset := 0

	// Groups array
	if offset+4 > len(data) {
		return nil, errors.New("insufficient data for groups array length")
	}
	groupsLen := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	req.Groups = make([]string, groupsLen)
	for i := 0; i < groupsLen; i++ {
		if offset+2 > len(data) {
			return nil, errors.New("insufficient data for group name length")
		}
		nameLen := int(binary.BigEndian.Uint16(data[offset:]))
		offset += 2

		if offset+nameLen > len(data) {
			return nil, errors.New("insufficient data for group name")
		}
		req.Groups[i] = string(data[offset : offset+nameLen])
		offset += nameLen
	}

	// IncludeAuthorizedOperations (v3+)
	if version >= 3 {
		if offset+1 > len(data) {
			return nil, errors.New("insufficient data for include authorized operations")
		}
		req.IncludeAuthorizedOperations = data[offset] != 0
		offset += 1
	}

	return req, nil
}

// EncodeDescribeGroupsResponse 编码响应
func EncodeDescribeGroupsResponse(resp *DescribeGroupsResponse, version int16) []byte {
	buf := make([]byte, 0, 2048)

	// ThrottleTimeMs
	throttle := make([]byte, 4)
	binary.BigEndian.PutUint32(throttle, uint32(resp.ThrottleTimeMs))
	buf = append(buf, throttle...)

	// Groups array length
	groupsLen := make([]byte, 4)
	binary.BigEndian.PutUint32(groupsLen, uint32(len(resp.Groups)))
	buf = append(buf, groupsLen...)

	for _, group := range resp.Groups {
		// ErrorCode
		errCode := make([]byte, 2)
		binary.BigEndian.PutUint16(errCode, uint16(group.ErrorCode))
		buf = append(buf, errCode...)

		// GroupID
		groupIDLen := make([]byte, 2)
		binary.BigEndian.PutUint16(groupIDLen, uint16(len(group.GroupID)))
		buf = append(buf, groupIDLen...)
		buf = append(buf, []byte(group.GroupID)...)

		// GroupState
		stateLen := make([]byte, 2)
		binary.BigEndian.PutUint16(stateLen, uint16(len(group.GroupState)))
		buf = append(buf, stateLen...)
		buf = append(buf, []byte(group.GroupState)...)

		// ProtocolType
		protocolTypeLen := make([]byte, 2)
		binary.BigEndian.PutUint16(protocolTypeLen, uint16(len(group.ProtocolType)))
		buf = append(buf, protocolTypeLen...)
		buf = append(buf, []byte(group.ProtocolType)...)

		// ProtocolData
		protocolDataLen := make([]byte, 2)
		binary.BigEndian.PutUint16(protocolDataLen, uint16(len(group.ProtocolData)))
		buf = append(buf, protocolDataLen...)
		buf = append(buf, []byte(group.ProtocolData)...)

		// Members array length
		membersLen := make([]byte, 4)
		binary.BigEndian.PutUint32(membersLen, uint32(len(group.Members)))
		buf = append(buf, membersLen...)

		for _, member := range group.Members {
			// MemberID
			memberIDLen := make([]byte, 2)
			binary.BigEndian.PutUint16(memberIDLen, uint16(len(member.MemberID)))
			buf = append(buf, memberIDLen...)
			buf = append(buf, []byte(member.MemberID)...)

			// GroupInstanceID (v4+, nullable)
			if version >= 4 {
				if member.GroupInstanceID != nil {
					instanceIDLen := make([]byte, 2)
					binary.BigEndian.PutUint16(instanceIDLen, uint16(len(*member.GroupInstanceID)))
					buf = append(buf, instanceIDLen...)
					buf = append(buf, []byte(*member.GroupInstanceID)...)
				} else {
					// Null string (-1)
					nullLen := make([]byte, 2)
					binary.BigEndian.PutUint16(nullLen, 0xFFFF)
					buf = append(buf, nullLen...)
				}
			}

			// ClientID
			clientIDLen := make([]byte, 2)
			binary.BigEndian.PutUint16(clientIDLen, uint16(len(member.ClientID)))
			buf = append(buf, clientIDLen...)
			buf = append(buf, []byte(member.ClientID)...)

			// ClientHost
			clientHostLen := make([]byte, 2)
			binary.BigEndian.PutUint16(clientHostLen, uint16(len(member.ClientHost)))
			buf = append(buf, clientHostLen...)
			buf = append(buf, []byte(member.ClientHost)...)

			// MemberMetadata
			metadataLen := make([]byte, 4)
			binary.BigEndian.PutUint32(metadataLen, uint32(len(member.MemberMetadata)))
			buf = append(buf, metadataLen...)
			buf = append(buf, member.MemberMetadata...)

			// MemberAssignment
			assignmentLen := make([]byte, 4)
			binary.BigEndian.PutUint32(assignmentLen, uint32(len(member.MemberAssignment)))
			buf = append(buf, assignmentLen...)
			buf = append(buf, member.MemberAssignment...)
		}

		// AuthorizedOperations (v3+)
		if version >= 3 {
			authOps := make([]byte, 4)
			binary.BigEndian.PutUint32(authOps, uint32(group.AuthorizedOperations))
			buf = append(buf, authOps...)
		}
	}

	return buf
}
