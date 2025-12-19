// Copyright 2025 Takhin Data
// Licensed under the Apache License, Version 2.0

package protocol

import (
	"encoding/binary"
	"errors"
)

// AlterConfigs API (Key: 33) - 修改配置
// 用于修改 broker、topic 等资源的配置

// AlterConfigsRequest 请求结构
type AlterConfigsRequest struct {
	Header       *RequestHeader
	Resources    []AlterConfigsResource
	ValidateOnly bool // 仅验证，不实际修改
}

// AlterConfigsResource 资源配置
type AlterConfigsResource struct {
	ResourceType ResourceType
	ResourceName string
	Configs      []AlterableConfig
}

// AlterableConfig 可修改的配置项
type AlterableConfig struct {
	Name  string
	Value *string // null 表示删除配置
}

// AlterConfigsResponse 响应结构
type AlterConfigsResponse struct {
	ThrottleTimeMs int32
	Resources      []AlterConfigsResourceResponse
}

// AlterConfigsResourceResponse 资源配置响应
type AlterConfigsResourceResponse struct {
	ErrorCode    ErrorCode
	ErrorMessage *string
	ResourceType ResourceType
	ResourceName string
}

// DecodeAlterConfigsRequest 解码请求
func DecodeAlterConfigsRequest(data []byte, version int16) (*AlterConfigsRequest, error) {
	req := &AlterConfigsRequest{}
	offset := 0

	// Resources array
	if offset+4 > len(data) {
		return nil, errors.New("insufficient data for resources array length")
	}
	resourcesLen := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	req.Resources = make([]AlterConfigsResource, resourcesLen)
	for i := 0; i < resourcesLen; i++ {
		// ResourceType
		if offset+1 > len(data) {
			return nil, errors.New("insufficient data for resource type")
		}
		req.Resources[i].ResourceType = ResourceType(data[offset])
		offset += 1

		// ResourceName
		if offset+2 > len(data) {
			return nil, errors.New("insufficient data for resource name length")
		}
		nameLen := int(binary.BigEndian.Uint16(data[offset:]))
		offset += 2

		if offset+nameLen > len(data) {
			return nil, errors.New("insufficient data for resource name")
		}
		req.Resources[i].ResourceName = string(data[offset : offset+nameLen])
		offset += nameLen

		// Configs array
		if offset+4 > len(data) {
			return nil, errors.New("insufficient data for configs array length")
		}
		configsLen := int(binary.BigEndian.Uint32(data[offset:]))
		offset += 4

		req.Resources[i].Configs = make([]AlterableConfig, configsLen)
		for j := 0; j < configsLen; j++ {
			// Config name
			if offset+2 > len(data) {
				return nil, errors.New("insufficient data for config name length")
			}
			configNameLen := int(binary.BigEndian.Uint16(data[offset:]))
			offset += 2

			if offset+configNameLen > len(data) {
				return nil, errors.New("insufficient data for config name")
			}
			req.Resources[i].Configs[j].Name = string(data[offset : offset+configNameLen])
			offset += configNameLen

			// Config value (nullable)
			if offset+2 > len(data) {
				return nil, errors.New("insufficient data for config value length")
			}
			valueLen := int(int16(binary.BigEndian.Uint16(data[offset:])))
			offset += 2

			if valueLen == -1 {
				req.Resources[i].Configs[j].Value = nil
			} else {
				if offset+valueLen > len(data) {
					return nil, errors.New("insufficient data for config value")
				}
				value := string(data[offset : offset+valueLen])
				req.Resources[i].Configs[j].Value = &value
				offset += valueLen
			}
		}
	}

	// ValidateOnly
	if offset+1 > len(data) {
		return nil, errors.New("insufficient data for validate only")
	}
	req.ValidateOnly = data[offset] != 0
	offset += 1

	return req, nil
}

// EncodeAlterConfigsResponse 编码响应
func EncodeAlterConfigsResponse(resp *AlterConfigsResponse, version int16) []byte {
	buf := make([]byte, 0, 1024)

	// ThrottleTimeMs
	throttle := make([]byte, 4)
	binary.BigEndian.PutUint32(throttle, uint32(resp.ThrottleTimeMs))
	buf = append(buf, throttle...)

	// Resources array length
	resourcesLen := make([]byte, 4)
	binary.BigEndian.PutUint32(resourcesLen, uint32(len(resp.Resources)))
	buf = append(buf, resourcesLen...)

	for _, resource := range resp.Resources {
		// ErrorCode
		errCode := make([]byte, 2)
		binary.BigEndian.PutUint16(errCode, uint16(resource.ErrorCode))
		buf = append(buf, errCode...)

		// ErrorMessage (nullable)
		if resource.ErrorMessage != nil {
			msgLen := make([]byte, 2)
			binary.BigEndian.PutUint16(msgLen, uint16(len(*resource.ErrorMessage)))
			buf = append(buf, msgLen...)
			buf = append(buf, []byte(*resource.ErrorMessage)...)
		} else {
			nullLen := make([]byte, 2)
			binary.BigEndian.PutUint16(nullLen, 0xFFFF)
			buf = append(buf, nullLen...)
		}

		// ResourceType
		buf = append(buf, byte(resource.ResourceType))

		// ResourceName
		nameLen := make([]byte, 2)
		binary.BigEndian.PutUint16(nameLen, uint16(len(resource.ResourceName)))
		buf = append(buf, nameLen...)
		buf = append(buf, []byte(resource.ResourceName)...)
	}

	return buf
}

// EncodeAlterConfigsRequest 编码请求（用于测试）
func EncodeAlterConfigsRequest(req *AlterConfigsRequest, version int16) []byte {
	buf := make([]byte, 0, 1024)

	// Resources array length
	resourcesLen := make([]byte, 4)
	binary.BigEndian.PutUint32(resourcesLen, uint32(len(req.Resources)))
	buf = append(buf, resourcesLen...)

	for _, resource := range req.Resources {
		// ResourceType
		buf = append(buf, byte(resource.ResourceType))

		// ResourceName
		nameLen := make([]byte, 2)
		binary.BigEndian.PutUint16(nameLen, uint16(len(resource.ResourceName)))
		buf = append(buf, nameLen...)
		buf = append(buf, []byte(resource.ResourceName)...)

		// Configs array length
		configsLen := make([]byte, 4)
		binary.BigEndian.PutUint32(configsLen, uint32(len(resource.Configs)))
		buf = append(buf, configsLen...)

		for _, config := range resource.Configs {
			// Config name
			configNameLen := make([]byte, 2)
			binary.BigEndian.PutUint16(configNameLen, uint16(len(config.Name)))
			buf = append(buf, configNameLen...)
			buf = append(buf, []byte(config.Name)...)

			// Config value (nullable)
			if config.Value != nil {
				valueLen := make([]byte, 2)
				binary.BigEndian.PutUint16(valueLen, uint16(len(*config.Value)))
				buf = append(buf, valueLen...)
				buf = append(buf, []byte(*config.Value)...)
			} else {
				nullLen := make([]byte, 2)
				binary.BigEndian.PutUint16(nullLen, 0xFFFF)
				buf = append(buf, nullLen...)
			}
		}
	}

	// ValidateOnly
	if req.ValidateOnly {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}

	return buf
}

// DecodeAlterConfigsResponse 解码响应（用于测试）
func DecodeAlterConfigsResponse(data []byte, version int16) (*AlterConfigsResponse, error) {
	resp := &AlterConfigsResponse{}
	offset := 0

	// ThrottleTimeMs
	if offset+4 > len(data) {
		return nil, errors.New("insufficient data for throttle time")
	}
	resp.ThrottleTimeMs = int32(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	// Resources array length
	if offset+4 > len(data) {
		return nil, errors.New("insufficient data for resources array length")
	}
	resourcesLen := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	resp.Resources = make([]AlterConfigsResourceResponse, resourcesLen)
	for i := 0; i < resourcesLen; i++ {
		// ErrorCode
		if offset+2 > len(data) {
			return nil, errors.New("insufficient data for error code")
		}
		resp.Resources[i].ErrorCode = ErrorCode(binary.BigEndian.Uint16(data[offset:]))
		offset += 2

		// ErrorMessage (nullable)
		if offset+2 > len(data) {
			return nil, errors.New("insufficient data for error message length")
		}
		msgLen := int(int16(binary.BigEndian.Uint16(data[offset:])))
		offset += 2

		if msgLen == -1 {
			resp.Resources[i].ErrorMessage = nil
		} else {
			if offset+msgLen > len(data) {
				return nil, errors.New("insufficient data for error message")
			}
			errMsg := string(data[offset : offset+msgLen])
			resp.Resources[i].ErrorMessage = &errMsg
			offset += msgLen
		}

		// ResourceType
		if offset+1 > len(data) {
			return nil, errors.New("insufficient data for resource type")
		}
		resp.Resources[i].ResourceType = ResourceType(data[offset])
		offset += 1

		// ResourceName
		if offset+2 > len(data) {
			return nil, errors.New("insufficient data for resource name length")
		}
		nameLen := int(binary.BigEndian.Uint16(data[offset:]))
		offset += 2

		if offset+nameLen > len(data) {
			return nil, errors.New("insufficient data for resource name")
		}
		resp.Resources[i].ResourceName = string(data[offset : offset+nameLen])
		offset += nameLen
	}

	return resp, nil
}
