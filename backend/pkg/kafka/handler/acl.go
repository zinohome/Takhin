// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"io"

	"github.com/takhin-data/takhin/pkg/acl"
	"github.com/takhin-data/takhin/pkg/kafka/protocol"
)

// HandleCreateAcls handles CreateAcls requests
func (h *Handler) HandleCreateAcls(r io.Reader, apiVersion int16) ([]byte, error) {
	// Decode request
	numCreations, err := protocol.ReadInt32(r)
	if err != nil {
		return nil, err
	}

	creations := make([]protocol.AclCreation, numCreations)
	for i := int32(0); i < numCreations; i++ {
		resourceType, err := protocol.ReadInt8(r)
		if err != nil {
			return nil, err
		}

		resourceName, err := protocol.ReadString(r)
		if err != nil {
			return nil, err
		}

		patternType, err := protocol.ReadInt8(r)
		if err != nil {
			return nil, err
		}

		principal, err := protocol.ReadString(r)
		if err != nil {
			return nil, err
		}

		host, err := protocol.ReadString(r)
		if err != nil {
			return nil, err
		}

		operation, err := protocol.ReadInt8(r)
		if err != nil {
			return nil, err
		}

		permissionType, err := protocol.ReadInt8(r)
		if err != nil {
			return nil, err
		}

		creations[i] = protocol.AclCreation{
			ResourceType:   resourceType,
			ResourceName:   resourceName,
			PatternType:    patternType,
			Principal:      principal,
			Host:           host,
			Operation:      operation,
			PermissionType: permissionType,
		}
	}

	// Process ACL creations
	results := make([]protocol.AclCreationResult, len(creations))
	for i, creation := range creations {
		entry := acl.Entry{
			Principal:      creation.Principal,
			Host:           creation.Host,
			ResourceType:   acl.ResourceType(creation.ResourceType),
			ResourceName:   creation.ResourceName,
			PatternType:    acl.PatternType(creation.PatternType),
			Operation:      acl.Operation(creation.Operation),
			PermissionType: acl.PermissionType(creation.PermissionType),
		}

		if err := h.aclStore.Add(entry); err != nil {
			errMsg := err.Error()
			results[i] = protocol.AclCreationResult{
				ErrorCode:    protocol.InvalidRequest,
				ErrorMessage: &errMsg,
			}
		} else {
			results[i] = protocol.AclCreationResult{
				ErrorCode:    protocol.None,
				ErrorMessage: nil,
			}
		}
	}

	// Save ACLs to disk
	if err := h.aclStore.Save(); err != nil {
		h.logger.Error("failed to save ACLs", "error", err)
	}

	// Encode response
	var buf bytes.Buffer
	protocol.WriteInt32(&buf, 0) // ThrottleTimeMs
	protocol.WriteInt32(&buf, int32(len(results)))
	for _, result := range results {
		protocol.WriteInt16(&buf, int16(result.ErrorCode))
		protocol.WriteNullableString(&buf, result.ErrorMessage)
	}

	return buf.Bytes(), nil
}

// HandleDescribeAcls handles DescribeAcls requests
func (h *Handler) HandleDescribeAcls(r io.Reader, apiVersion int16) ([]byte, error) {
	// Decode request
	resourceTypeFilter, err := protocol.ReadInt8(r)
	if err != nil {
		return nil, err
	}

	resourceNameFilter, err := protocol.ReadNullableString(r)
	if err != nil {
		return nil, err
	}

	patternTypeFilter, err := protocol.ReadInt8(r)
	if err != nil {
		return nil, err
	}

	principalFilter, err := protocol.ReadNullableString(r)
	if err != nil {
		return nil, err
	}

	hostFilter, err := protocol.ReadNullableString(r)
	if err != nil {
		return nil, err
	}

	operation, err := protocol.ReadInt8(r)
	if err != nil {
		return nil, err
	}

	permissionType, err := protocol.ReadInt8(r)
	if err != nil {
		return nil, err
	}

	// Build filter
	filter := acl.Filter{
		ResourceFilter: acl.ResourceFilter{
			ResourceType: acl.ResourceType(resourceTypeFilter),
			ResourceName: resourceNameFilter,
		},
		AccessFilter: acl.AccessFilter{
			Principal: principalFilter,
			Host:      hostFilter,
		},
	}

	if patternTypeFilter >= 0 {
		pt := acl.PatternType(patternTypeFilter)
		filter.ResourceFilter.PatternType = &pt
	}

	if operation >= 0 {
		op := acl.Operation(operation)
		filter.AccessFilter.Operation = &op
	}

	if permissionType >= 0 {
		pt := acl.PermissionType(permissionType)
		filter.AccessFilter.PermissionType = &pt
	}

	// Get matching ACLs
	entries := h.aclStore.List(filter)

	// Group by resource
	resourceMap := make(map[string]*protocol.AclResource)
	for _, entry := range entries {
		key := string(entry.ResourceType) + ":" + entry.ResourceName + ":" + string(entry.PatternType)

		resource, exists := resourceMap[key]
		if !exists {
			resource = &protocol.AclResource{
				ResourceType: int8(entry.ResourceType),
				ResourceName: entry.ResourceName,
				PatternType:  int8(entry.PatternType),
				Acls:         []protocol.AclDescription{},
			}
			resourceMap[key] = resource
		}

		resource.Acls = append(resource.Acls, protocol.AclDescription{
			Principal:      entry.Principal,
			Host:           entry.Host,
			Operation:      int8(entry.Operation),
			PermissionType: int8(entry.PermissionType),
		})
	}

	// Convert map to slice
	resources := make([]protocol.AclResource, 0, len(resourceMap))
	for _, resource := range resourceMap {
		resources = append(resources, *resource)
	}

	// Encode response
	var buf bytes.Buffer
	protocol.WriteInt32(&buf, 0) // ThrottleTimeMs
	protocol.WriteInt16(&buf, int16(protocol.None))
	protocol.WriteNullableString(&buf, nil)
	protocol.WriteInt32(&buf, int32(len(resources)))

	for _, resource := range resources {
		protocol.WriteInt8(&buf, resource.ResourceType)
		protocol.WriteString(&buf, resource.ResourceName)
		protocol.WriteInt8(&buf, resource.PatternType)
		protocol.WriteInt32(&buf, int32(len(resource.Acls)))
		for _, aclDesc := range resource.Acls {
			protocol.WriteString(&buf, aclDesc.Principal)
			protocol.WriteString(&buf, aclDesc.Host)
			protocol.WriteInt8(&buf, aclDesc.Operation)
			protocol.WriteInt8(&buf, aclDesc.PermissionType)
		}
	}

	return buf.Bytes(), nil
}

// HandleDeleteAcls handles DeleteAcls requests
func (h *Handler) HandleDeleteAcls(r io.Reader, apiVersion int16) ([]byte, error) {
	// Decode request
	numFilters, err := protocol.ReadInt32(r)
	if err != nil {
		return nil, err
	}

	filters := make([]acl.Filter, numFilters)
	for i := int32(0); i < numFilters; i++ {
		resourceTypeFilter, err := protocol.ReadInt8(r)
		if err != nil {
			return nil, err
		}

		resourceNameFilter, err := protocol.ReadNullableString(r)
		if err != nil {
			return nil, err
		}

		patternTypeFilter, err := protocol.ReadInt8(r)
		if err != nil {
			return nil, err
		}

		principalFilter, err := protocol.ReadNullableString(r)
		if err != nil {
			return nil, err
		}

		hostFilter, err := protocol.ReadNullableString(r)
		if err != nil {
			return nil, err
		}

		operation, err := protocol.ReadInt8(r)
		if err != nil {
			return nil, err
		}

		permissionType, err := protocol.ReadInt8(r)
		if err != nil {
			return nil, err
		}

		filter := acl.Filter{
			ResourceFilter: acl.ResourceFilter{
				ResourceType: acl.ResourceType(resourceTypeFilter),
				ResourceName: resourceNameFilter,
			},
			AccessFilter: acl.AccessFilter{
				Principal: principalFilter,
				Host:      hostFilter,
			},
		}

		if patternTypeFilter >= 0 {
			pt := acl.PatternType(patternTypeFilter)
			filter.ResourceFilter.PatternType = &pt
		}

		if operation >= 0 {
			op := acl.Operation(operation)
			filter.AccessFilter.Operation = &op
		}

		if permissionType >= 0 {
			pt := acl.PermissionType(permissionType)
			filter.AccessFilter.PermissionType = &pt
		}

		filters[i] = filter
	}

	// Process deletions
	results := make([]protocol.DeleteAclsResult, len(filters))
	for i, filter := range filters {
		// Get matching ACLs before deletion
		matching := h.aclStore.List(filter)

		// Delete matching ACLs
		deleted, err := h.aclStore.Delete(filter)
		if err != nil {
			errMsg := err.Error()
			results[i] = protocol.DeleteAclsResult{
				ErrorCode:    protocol.InvalidRequest,
				ErrorMessage: &errMsg,
				MatchingAcls: []protocol.DeleteAclsMatch{},
			}
			continue
		}

		// Build match results
		matches := make([]protocol.DeleteAclsMatch, len(matching))
		for j, entry := range matching {
			matches[j] = protocol.DeleteAclsMatch{
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

		results[i] = protocol.DeleteAclsResult{
			ErrorCode:    protocol.None,
			ErrorMessage: nil,
			MatchingAcls: matches,
		}

		h.logger.Info("deleted ACL entries", "count", deleted, "filter", filter)
	}

	// Save ACLs to disk
	if err := h.aclStore.Save(); err != nil {
		h.logger.Error("failed to save ACLs", "error", err)
	}

	// Encode response
	var buf bytes.Buffer
	protocol.WriteInt32(&buf, 0) // ThrottleTimeMs
	protocol.WriteInt32(&buf, int32(len(results)))

	for _, result := range results {
		protocol.WriteInt16(&buf, int16(result.ErrorCode))
		protocol.WriteNullableString(&buf, result.ErrorMessage)
		protocol.WriteInt32(&buf, int32(len(result.MatchingAcls)))
		for _, match := range result.MatchingAcls {
			protocol.WriteInt16(&buf, int16(match.ErrorCode))
			protocol.WriteNullableString(&buf, match.ErrorMessage)
			protocol.WriteInt8(&buf, match.ResourceType)
			protocol.WriteString(&buf, match.ResourceName)
			protocol.WriteInt8(&buf, match.PatternType)
			protocol.WriteString(&buf, match.Principal)
			protocol.WriteString(&buf, match.Host)
			protocol.WriteInt8(&buf, match.Operation)
			protocol.WriteInt8(&buf, match.PermissionType)
		}
	}

	return buf.Bytes(), nil
}
