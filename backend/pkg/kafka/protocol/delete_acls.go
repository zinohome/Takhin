// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// DeleteAclsRequest represents a DeleteAcls API request
type DeleteAclsRequest struct {
	Filters []*AclDeleteFilter
}

// AclDeleteFilter represents a filter for deleting ACLs
type AclDeleteFilter struct {
	ResourceTypeFilter int8
	ResourceNameFilter *string
	PatternTypeFilter  int8
	PrincipalFilter    *string
	HostFilter         *string
	Operation          int8
	PermissionType     int8
}

// DeleteAclsResponse represents a DeleteAcls API response
type DeleteAclsResponse struct {
	ThrottleTimeMs int32
	FilterResults  []*DeleteAclsFilterResult
}

// DeleteAclsFilterResult represents the result of a single delete filter
type DeleteAclsFilterResult struct {
	ErrorCode      ErrorCode
	ErrorMessage   *string
	MatchingAcls   []*DeleteAclsMatchingAcl
}

// DeleteAclsMatchingAcl represents a matching ACL that was deleted
type DeleteAclsMatchingAcl struct {
	ErrorCode      ErrorCode
	ErrorMessage   *string
	ResourceType   int8
	ResourceName   string
	PatternType    int8
	Principal      string
	Host           string
	Operation      int8
	PermissionType int8
}

// DecodeDeleteAclsRequest decodes a DeleteAcls request
func DecodeDeleteAclsRequest(r io.Reader, version int16) (*DeleteAclsRequest, error) {
	req := &DeleteAclsRequest{}

	// Read array length
	var arrayLen int32
	if err := binary.Read(r, binary.BigEndian, &arrayLen); err != nil {
		return nil, fmt.Errorf("read array length: %w", err)
	}

	req.Filters = make([]*AclDeleteFilter, arrayLen)
	for i := int32(0); i < arrayLen; i++ {
		filter := &AclDeleteFilter{}

		// Read resource type filter
		if err := binary.Read(r, binary.BigEndian, &filter.ResourceTypeFilter); err != nil {
			return nil, fmt.Errorf("read resource type filter: %w", err)
		}

		// Read resource name filter
		name, err := ReadNullableString(r)
		if err != nil {
			return nil, fmt.Errorf("read resource name filter: %w", err)
		}
		filter.ResourceNameFilter = name

		// Read pattern type filter (version 1+)
		if version >= 1 {
			if err := binary.Read(r, binary.BigEndian, &filter.PatternTypeFilter); err != nil {
				return nil, fmt.Errorf("read pattern type filter: %w", err)
			}
		} else {
			filter.PatternTypeFilter = 2 // Literal
		}

		// Read principal filter
		principal, err := ReadNullableString(r)
		if err != nil {
			return nil, fmt.Errorf("read principal filter: %w", err)
		}
		filter.PrincipalFilter = principal

		// Read host filter
		host, err := ReadNullableString(r)
		if err != nil {
			return nil, fmt.Errorf("read host filter: %w", err)
		}
		filter.HostFilter = host

		// Read operation
		if err := binary.Read(r, binary.BigEndian, &filter.Operation); err != nil {
			return nil, fmt.Errorf("read operation: %w", err)
		}

		// Read permission type
		if err := binary.Read(r, binary.BigEndian, &filter.PermissionType); err != nil {
			return nil, fmt.Errorf("read permission type: %w", err)
		}

		req.Filters[i] = filter
	}

	return req, nil
}

// WriteDeleteAclsResponse writes a DeleteAcls response
func WriteDeleteAclsResponse(w io.Writer, header *RequestHeader, resp *DeleteAclsResponse) error {
	// Write correlation ID
	if err := binary.Write(w, binary.BigEndian, header.CorrelationID); err != nil {
		return fmt.Errorf("write correlation ID: %w", err)
	}

	// Write throttle time
	if err := binary.Write(w, binary.BigEndian, resp.ThrottleTimeMs); err != nil {
		return fmt.Errorf("write throttle time: %w", err)
	}

	// Write filter results array length
	if err := binary.Write(w, binary.BigEndian, int32(len(resp.FilterResults))); err != nil {
		return fmt.Errorf("write filter results length: %w", err)
	}

	// Write each filter result
	for _, result := range resp.FilterResults {
		// Write error code
		if err := binary.Write(w, binary.BigEndian, result.ErrorCode); err != nil {
			return fmt.Errorf("write error code: %w", err)
		}

		// Write error message
		if err := WriteNullableString(w, result.ErrorMessage); err != nil {
			return fmt.Errorf("write error message: %w", err)
		}

		// Write matching ACLs array length
		if err := binary.Write(w, binary.BigEndian, int32(len(result.MatchingAcls))); err != nil {
			return fmt.Errorf("write matching acls length: %w", err)
		}

		// Write each matching ACL
		for _, acl := range result.MatchingAcls {
			// Write error code
			if err := binary.Write(w, binary.BigEndian, acl.ErrorCode); err != nil {
				return fmt.Errorf("write acl error code: %w", err)
			}

			// Write error message
			if err := WriteNullableString(w, acl.ErrorMessage); err != nil {
				return fmt.Errorf("write acl error message: %w", err)
			}

			// Write resource type
			if err := binary.Write(w, binary.BigEndian, acl.ResourceType); err != nil {
				return fmt.Errorf("write resource type: %w", err)
			}

			// Write resource name
			if err := WriteString(w, acl.ResourceName); err != nil {
				return fmt.Errorf("write resource name: %w", err)
			}

			// Write pattern type
			if err := binary.Write(w, binary.BigEndian, acl.PatternType); err != nil {
				return fmt.Errorf("write pattern type: %w", err)
			}

			// Write principal
			if err := WriteString(w, acl.Principal); err != nil {
				return fmt.Errorf("write principal: %w", err)
			}

			// Write host
			if err := WriteString(w, acl.Host); err != nil {
				return fmt.Errorf("write host: %w", err)
			}

			// Write operation
			if err := binary.Write(w, binary.BigEndian, acl.Operation); err != nil {
				return fmt.Errorf("write operation: %w", err)
			}

			// Write permission type
			if err := binary.Write(w, binary.BigEndian, acl.PermissionType); err != nil {
				return fmt.Errorf("write permission type: %w", err)
			}
		}
	}

	return nil
}
