// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// DescribeAclsRequest represents a DescribeAcls API request
type DescribeAclsRequest struct {
	ResourceTypeFilter   int8
	ResourceNameFilter   *string
	PatternTypeFilter    int8
	PrincipalFilter      *string
	HostFilter           *string
	Operation            int8
	PermissionType       int8
}

// DescribeAclsResponse represents a DescribeAcls API response
type DescribeAclsResponse struct {
	ThrottleTimeMs int32
	ErrorCode      ErrorCode
	ErrorMessage   *string
	Resources      []*DescribeAclsResource
}

// DescribeAclsResource represents ACLs for a resource
type DescribeAclsResource struct {
	ResourceType int8
	ResourceName string
	PatternType  int8
	Acls         []*AclDescription
}

// AclDescription describes a single ACL
type AclDescription struct {
	Principal      string
	Host           string
	Operation      int8
	PermissionType int8
}

// DecodeDescribeAclsRequest decodes a DescribeAcls request
func DecodeDescribeAclsRequest(r io.Reader, version int16) (*DescribeAclsRequest, error) {
	req := &DescribeAclsRequest{}

	// Read resource type filter
	if err := binary.Read(r, binary.BigEndian, &req.ResourceTypeFilter); err != nil {
		return nil, fmt.Errorf("read resource type filter: %w", err)
	}

	// Read resource name filter
	name, err := ReadNullableString(r)
	if err != nil {
		return nil, fmt.Errorf("read resource name filter: %w", err)
	}
	req.ResourceNameFilter = name

	// Read pattern type filter (version 1+)
	if version >= 1 {
		if err := binary.Read(r, binary.BigEndian, &req.PatternTypeFilter); err != nil {
			return nil, fmt.Errorf("read pattern type filter: %w", err)
		}
	} else {
		req.PatternTypeFilter = 2 // Literal
	}

	// Read principal filter
	principal, err := ReadNullableString(r)
	if err != nil {
		return nil, fmt.Errorf("read principal filter: %w", err)
	}
	req.PrincipalFilter = principal

	// Read host filter
	host, err := ReadNullableString(r)
	if err != nil {
		return nil, fmt.Errorf("read host filter: %w", err)
	}
	req.HostFilter = host

	// Read operation
	if err := binary.Read(r, binary.BigEndian, &req.Operation); err != nil {
		return nil, fmt.Errorf("read operation: %w", err)
	}

	// Read permission type
	if err := binary.Read(r, binary.BigEndian, &req.PermissionType); err != nil {
		return nil, fmt.Errorf("read permission type: %w", err)
	}

	return req, nil
}

// WriteDescribeAclsResponse writes a DescribeAcls response
func WriteDescribeAclsResponse(w io.Writer, header *RequestHeader, resp *DescribeAclsResponse) error {
	// Write correlation ID
	if err := binary.Write(w, binary.BigEndian, header.CorrelationID); err != nil {
		return fmt.Errorf("write correlation ID: %w", err)
	}

	// Write throttle time
	if err := binary.Write(w, binary.BigEndian, resp.ThrottleTimeMs); err != nil {
		return fmt.Errorf("write throttle time: %w", err)
	}

	// Write error code
	if err := binary.Write(w, binary.BigEndian, resp.ErrorCode); err != nil {
		return fmt.Errorf("write error code: %w", err)
	}

	// Write error message
	if err := WriteNullableString(w, resp.ErrorMessage); err != nil {
		return fmt.Errorf("write error message: %w", err)
	}

	// Write resources array length
	if err := binary.Write(w, binary.BigEndian, int32(len(resp.Resources))); err != nil {
		return fmt.Errorf("write resources length: %w", err)
	}

	// Write each resource
	for _, resource := range resp.Resources {
		// Write resource type
		if err := binary.Write(w, binary.BigEndian, resource.ResourceType); err != nil {
			return fmt.Errorf("write resource type: %w", err)
		}

		// Write resource name
		if err := WriteString(w, resource.ResourceName); err != nil {
			return fmt.Errorf("write resource name: %w", err)
		}

		// Write pattern type
		if err := binary.Write(w, binary.BigEndian, resource.PatternType); err != nil {
			return fmt.Errorf("write pattern type: %w", err)
		}

		// Write ACLs array length
		if err := binary.Write(w, binary.BigEndian, int32(len(resource.Acls))); err != nil {
			return fmt.Errorf("write acls length: %w", err)
		}

		// Write each ACL
		for _, acl := range resource.Acls {
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
