// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"fmt"
	"io"

	"github.com/takhin-data/takhin/pkg/acl"
	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/logger"
)

// handleCreateAcls handles CreateAcls requests
func (h *Handler) handleCreateAcls(reader io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeCreateAclsRequest(reader, header.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}

	logger.Info("create ACLs request",
		"component", "kafka-handler",
		"count", len(req.Creations),
	)

	// Check if authorizer is available
	if h.authorizer == nil {
		errMsg := "ACL not enabled"
		resp := &protocol.CreateAclsResponse{
			ThrottleTimeMs: 0,
			Results:        make([]*protocol.AclCreationResult, len(req.Creations)),
		}
		for i := range req.Creations {
			resp.Results[i] = &protocol.AclCreationResult{
				ErrorCode:    protocol.SecurityDisabled,
				ErrorMessage: &errMsg,
			}
		}
		return encodeCreateAclsResponse(header, resp)
	}

	// Process each ACL creation
	results := make([]*protocol.AclCreationResult, len(req.Creations))
	for i, creation := range req.Creations {
		// Convert protocol types to ACL types
		entry, err := acl.NewEntry(
			creation.Principal,
			creation.Host,
			acl.ResourceType(creation.ResourceType),
			creation.ResourceName,
			acl.ResourcePatternType(creation.PatternType),
			acl.Operation(creation.Operation),
			acl.PermissionType(creation.PermissionType),
		)

		if err != nil {
			errMsg := err.Error()
			results[i] = &protocol.AclCreationResult{
				ErrorCode:    protocol.InvalidRequest,
				ErrorMessage: &errMsg,
			}
			continue
		}

		// Add ACL using interface
		if err := h.authorizer.AddACL(entry); err != nil {
			errMsg := err.Error()
			results[i] = &protocol.AclCreationResult{
				ErrorCode:    protocol.InvalidRequest,
				ErrorMessage: &errMsg,
			}
			continue
		}

		results[i] = &protocol.AclCreationResult{
			ErrorCode:    protocol.None,
			ErrorMessage: nil,
		}
	}

	resp := &protocol.CreateAclsResponse{
		ThrottleTimeMs: 0,
		Results:        results,
	}

	return encodeCreateAclsResponse(header, resp)
}

// handleDescribeAcls handles DescribeAcls requests
func (h *Handler) handleDescribeAcls(reader io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeDescribeAclsRequest(reader, header.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}

	logger.Info("describe ACLs request", "component", "kafka-handler")

	// Check if authorizer is available
	if h.authorizer == nil {
		errMsg := "ACL not enabled"
		resp := &protocol.DescribeAclsResponse{
			ThrottleTimeMs: 0,
			ErrorCode:      protocol.SecurityDisabled,
			ErrorMessage:   &errMsg,
			Resources:      []*protocol.DescribeAclsResource{},
		}
		return encodeDescribeAclsResponse(header, resp)
	}

	// Build filter
	filter := &acl.Filter{}
	if req.PrincipalFilter != nil {
		filter.Principal = req.PrincipalFilter
	}
	if req.HostFilter != nil {
		filter.Host = req.HostFilter
	}
	if req.ResourceTypeFilter != 0 {
		rt := acl.ResourceType(req.ResourceTypeFilter)
		filter.ResourceType = &rt
	}
	if req.ResourceNameFilter != nil {
		filter.ResourceName = req.ResourceNameFilter
	}
	if req.PatternTypeFilter != 0 {
		pt := acl.ResourcePatternType(req.PatternTypeFilter)
		filter.PatternType = &pt
	}
	if req.Operation != 0 {
		op := acl.Operation(req.Operation)
		filter.Operation = &op
	}
	if req.PermissionType != 0 {
		perm := acl.PermissionType(req.PermissionType)
		filter.PermissionType = &perm
	}

	// List ACLs
	entriesInterface := h.authorizer.ListACL(filter)

	// Convert to entries
	entries := make([]*acl.Entry, 0, len(entriesInterface))
	for _, ei := range entriesInterface {
		if entry, ok := ei.(*acl.Entry); ok {
			entries = append(entries, entry)
		}
	}

	// Group by resource
	resourceMap := make(map[string]*protocol.DescribeAclsResource)
	for _, entry := range entries {
		key := fmt.Sprintf("%d|%s|%d", entry.ResourceType, entry.ResourceName, entry.PatternType)
		resource, ok := resourceMap[key]
		if !ok {
			resource = &protocol.DescribeAclsResource{
				ResourceType: int8(entry.ResourceType),
				ResourceName: entry.ResourceName,
				PatternType:  int8(entry.PatternType),
				Acls:         []*protocol.AclDescription{},
			}
			resourceMap[key] = resource
		}

		resource.Acls = append(resource.Acls, &protocol.AclDescription{
			Principal:      entry.Principal,
			Host:           entry.Host,
			Operation:      int8(entry.Operation),
			PermissionType: int8(entry.PermissionType),
		})
	}

	// Convert map to slice
	resources := make([]*protocol.DescribeAclsResource, 0, len(resourceMap))
	for _, resource := range resourceMap {
		resources = append(resources, resource)
	}

	resp := &protocol.DescribeAclsResponse{
		ThrottleTimeMs: 0,
		ErrorCode:      protocol.None,
		ErrorMessage:   nil,
		Resources:      resources,
	}

	return encodeDescribeAclsResponse(header, resp)
}

// handleDeleteAcls handles DeleteAcls requests
func (h *Handler) handleDeleteAcls(reader io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeDeleteAclsRequest(reader, header.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}

	logger.Info("delete ACLs request",
		"component", "kafka-handler",
		"filter_count", len(req.Filters),
	)

	// Check if authorizer is available
	if h.authorizer == nil {
		errMsg := "ACL not enabled"
		resp := &protocol.DeleteAclsResponse{
			ThrottleTimeMs: 0,
			FilterResults:  make([]*protocol.DeleteAclsFilterResult, len(req.Filters)),
		}
		for i := range req.Filters {
			resp.FilterResults[i] = &protocol.DeleteAclsFilterResult{
				ErrorCode:    protocol.SecurityDisabled,
				ErrorMessage: &errMsg,
				MatchingAcls: []*protocol.DeleteAclsMatchingAcl{},
			}
		}
		return encodeDeleteAclsResponse(header, resp)
	}

	// Process each filter
	filterResults := make([]*protocol.DeleteAclsFilterResult, len(req.Filters))
	for i, filterReq := range req.Filters {
		// Build filter
		filter := &acl.Filter{}
		if filterReq.PrincipalFilter != nil {
			filter.Principal = filterReq.PrincipalFilter
		}
		if filterReq.HostFilter != nil {
			filter.Host = filterReq.HostFilter
		}
		if filterReq.ResourceTypeFilter != 0 {
			rt := acl.ResourceType(filterReq.ResourceTypeFilter)
			filter.ResourceType = &rt
		}
		if filterReq.ResourceNameFilter != nil {
			filter.ResourceName = filterReq.ResourceNameFilter
		}
		if filterReq.PatternTypeFilter != 0 {
			pt := acl.ResourcePatternType(filterReq.PatternTypeFilter)
			filter.PatternType = &pt
		}
		if filterReq.Operation != 0 {
			op := acl.Operation(filterReq.Operation)
			filter.Operation = &op
		}
		if filterReq.PermissionType != 0 {
			perm := acl.PermissionType(filterReq.PermissionType)
			filter.PermissionType = &perm
		}

		// List matching ACLs before deletion
		matchingEntriesInterface := h.authorizer.ListACL(filter)

		// Convert to entries
		matchingEntries := make([]*acl.Entry, 0, len(matchingEntriesInterface))
		for _, ei := range matchingEntriesInterface {
			if entry, ok := ei.(*acl.Entry); ok {
				matchingEntries = append(matchingEntries, entry)
			}
		}

		// Delete ACLs
		_, err := h.authorizer.DeleteACL(filter)
		if err != nil {
			errMsg := err.Error()
			filterResults[i] = &protocol.DeleteAclsFilterResult{
				ErrorCode:    protocol.InvalidRequest,
				ErrorMessage: &errMsg,
				MatchingAcls: []*protocol.DeleteAclsMatchingAcl{},
			}
			continue
		}

		// Build matching ACLs result
		matchingAcls := make([]*protocol.DeleteAclsMatchingAcl, len(matchingEntries))
		for j, entry := range matchingEntries {
			matchingAcls[j] = &protocol.DeleteAclsMatchingAcl{
				ErrorCode:      protocol.None,
				ErrorMessage:   nil,
				ResourceType:   int8(entry.ResourceType),
				ResourceName:   entry.ResourceName,
				PatternType:    int8(entry.PatternType),
				Principal:      entry.Principal,
				Host:           entry.Host,
				Operation:      int8(entry.Operation),
				PermissionType: int8(entry.PermissionType),
			}
		}

		filterResults[i] = &protocol.DeleteAclsFilterResult{
			ErrorCode:    protocol.None,
			ErrorMessage: nil,
			MatchingAcls: matchingAcls,
		}
	}

	resp := &protocol.DeleteAclsResponse{
		ThrottleTimeMs: 0,
		FilterResults:  filterResults,
	}

	return encodeDeleteAclsResponse(header, resp)
}

// encodeCreateAclsResponse encodes the CreateAcls response
func encodeCreateAclsResponse(header *protocol.RequestHeader, resp *protocol.CreateAclsResponse) ([]byte, error) {
	var buf bytes.Buffer
	if err := protocol.WriteCreateAclsResponse(&buf, header, resp); err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}
	return buf.Bytes(), nil
}

// encodeDescribeAclsResponse encodes the DescribeAcls response
func encodeDescribeAclsResponse(header *protocol.RequestHeader, resp *protocol.DescribeAclsResponse) ([]byte, error) {
	var buf bytes.Buffer
	if err := protocol.WriteDescribeAclsResponse(&buf, header, resp); err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}
	return buf.Bytes(), nil
}

// encodeDeleteAclsResponse encodes the DeleteAcls response
func encodeDeleteAclsResponse(header *protocol.RequestHeader, resp *protocol.DeleteAclsResponse) ([]byte, error) {
	var buf bytes.Buffer
	if err := protocol.WriteDeleteAclsResponse(&buf, header, resp); err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}
	return buf.Bytes(), nil
}
