// Copyright 2025 Takhin Data, Inc.

package protocol

import (
"io"
)

// DescribeConfigsRequest represents a DescribeConfigs request (API Key 32)
type DescribeConfigsRequest struct {
	Header    *RequestHeader
	Resources []DescribeConfigsResource
}

// DescribeConfigsResource represents a resource to describe
type DescribeConfigsResource struct {
	ResourceType int8
	ResourceName string
	ConfigNames  []string
}

// Resource types
const (
ResourceTypeTopic  int8 = 2
ResourceTypeBroker int8 = 4
)

// DescribeConfigsResponse represents a DescribeConfigs response
type DescribeConfigsResponse struct {
	ThrottleTimeMs int32
	Results        []DescribeConfigsResult
}

// DescribeConfigsResult represents the result of describing configs
type DescribeConfigsResult struct {
	ErrorCode    ErrorCode
	ErrorMessage *string
	ResourceType int8
	ResourceName string
	Configs      []DescribeConfigsEntry
}

// DescribeConfigsEntry represents a configuration entry
type DescribeConfigsEntry struct {
	Name        string
	Value       *string
	ReadOnly    bool
	IsDefault   bool
	IsSensitive bool
}

// DecodeDescribeConfigsRequest decodes a DescribeConfigs request
func DecodeDescribeConfigsRequest(r io.Reader, header *RequestHeader) (*DescribeConfigsRequest, error) {
	req := &DescribeConfigsRequest{
		Header: header,
	}

	// Read resources array
	resourceCount, err := ReadArrayLength(r)
	if err != nil {
		return nil, err
	}

	req.Resources = make([]DescribeConfigsResource, resourceCount)
	for i := int32(0); i < resourceCount; i++ {
		// Read resource type
		resourceType, err := ReadInt8(r)
		if err != nil {
			return nil, err
		}

		// Read resource name
		resourceName, err := ReadString(r)
		if err != nil {
			return nil, err
		}

		// Read config names (nullable array)
		configCount, err := ReadArrayLength(r)
		if err != nil {
			return nil, err
		}

		var configNames []string
		if configCount > 0 {
			configNames = make([]string, configCount)
			for j := int32(0); j < configCount; j++ {
				configName, err := ReadString(r)
				if err != nil {
					return nil, err
				}
				configNames[j] = configName
			}
		}

		req.Resources[i] = DescribeConfigsResource{
			ResourceType: resourceType,
			ResourceName: resourceName,
			ConfigNames:  configNames,
		}
	}

	return req, nil
}

// Encode encodes the DescribeConfigs response
func (r *DescribeConfigsResponse) Encode(w io.Writer) error {
	// Write throttle time
	if err := WriteInt32(w, r.ThrottleTimeMs); err != nil {
		return err
	}

	// Write results array
	if err := WriteArray(w, len(r.Results)); err != nil {
		return err
	}

	for _, result := range r.Results {
		// Write error code
		if err := WriteInt16(w, int16(result.ErrorCode)); err != nil {
			return err
		}

		// Write error message
		if err := WriteNullableString(w, result.ErrorMessage); err != nil {
			return err
		}

		// Write resource type
		if err := WriteInt8(w, result.ResourceType); err != nil {
			return err
		}

		// Write resource name
		if err := WriteString(w, result.ResourceName); err != nil {
			return err
		}

		// Write configs array
		if err := WriteArray(w, len(result.Configs)); err != nil {
			return err
		}

		for _, config := range result.Configs {
			// Write config name
			if err := WriteString(w, config.Name); err != nil {
				return err
			}

			// Write config value
			if err := WriteNullableString(w, config.Value); err != nil {
				return err
			}

			// Write read only
			if err := WriteBool(w, config.ReadOnly); err != nil {
				return err
			}

			// Write is default
			if err := WriteBool(w, config.IsDefault); err != nil {
				return err
			}

			// Write is sensitive
			if err := WriteBool(w, config.IsSensitive); err != nil {
				return err
			}
		}
	}

	return nil
}
