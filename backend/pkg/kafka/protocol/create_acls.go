// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// CreateAclsRequest represents a CreateAcls API request
type CreateAclsRequest struct {
	Creations []*AclCreation
}

// AclCreation represents a single ACL creation
type AclCreation struct {
	ResourceType   int8
	ResourceName   string
	PatternType    int8
	Principal      string
	Host           string
	Operation      int8
	PermissionType int8
}

// CreateAclsResponse represents a CreateAcls API response
type CreateAclsResponse struct {
	ThrottleTimeMs int32
	Results        []*AclCreationResult
}

// AclCreationResult represents the result of creating a single ACL
type AclCreationResult struct {
	ErrorCode    ErrorCode
	ErrorMessage *string
}

// DecodeCreateAclsRequest decodes a CreateAcls request
func DecodeCreateAclsRequest(r io.Reader, version int16) (*CreateAclsRequest, error) {
	req := &CreateAclsRequest{}

	// Read array length
	var arrayLen int32
	if err := binary.Read(r, binary.BigEndian, &arrayLen); err != nil {
		return nil, fmt.Errorf("read array length: %w", err)
	}

	req.Creations = make([]*AclCreation, arrayLen)
	for i := int32(0); i < arrayLen; i++ {
		creation := &AclCreation{}

		// Read resource type
		if err := binary.Read(r, binary.BigEndian, &creation.ResourceType); err != nil {
			return nil, fmt.Errorf("read resource type: %w", err)
		}

		// Read resource name
		name, err := ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("read resource name: %w", err)
		}
		creation.ResourceName = name

		// Read pattern type (version 1+)
		if version >= 1 {
			if err := binary.Read(r, binary.BigEndian, &creation.PatternType); err != nil {
				return nil, fmt.Errorf("read pattern type: %w", err)
			}
		} else {
			creation.PatternType = 2 // Literal
		}

		// Read principal
		principal, err := ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("read principal: %w", err)
		}
		creation.Principal = principal

		// Read host
		host, err := ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("read host: %w", err)
		}
		creation.Host = host

		// Read operation
		if err := binary.Read(r, binary.BigEndian, &creation.Operation); err != nil {
			return nil, fmt.Errorf("read operation: %w", err)
		}

		// Read permission type
		if err := binary.Read(r, binary.BigEndian, &creation.PermissionType); err != nil {
			return nil, fmt.Errorf("read permission type: %w", err)
		}

		req.Creations[i] = creation
	}

	return req, nil
}

// WriteCreateAclsResponse writes a CreateAcls response
func WriteCreateAclsResponse(w io.Writer, header *RequestHeader, resp *CreateAclsResponse) error {
	// Write correlation ID
	if err := binary.Write(w, binary.BigEndian, header.CorrelationID); err != nil {
		return fmt.Errorf("write correlation ID: %w", err)
	}

	// Write throttle time
	if err := binary.Write(w, binary.BigEndian, resp.ThrottleTimeMs); err != nil {
		return fmt.Errorf("write throttle time: %w", err)
	}

	// Write results array length
	if err := binary.Write(w, binary.BigEndian, int32(len(resp.Results))); err != nil {
		return fmt.Errorf("write results length: %w", err)
	}

	// Write each result
	for _, result := range resp.Results {
		// Write error code
		if err := binary.Write(w, binary.BigEndian, result.ErrorCode); err != nil {
			return fmt.Errorf("write error code: %w", err)
		}

		// Write error message
		if err := WriteNullableString(w, result.ErrorMessage); err != nil {
			return fmt.Errorf("write error message: %w", err)
		}
	}

	return nil
}
