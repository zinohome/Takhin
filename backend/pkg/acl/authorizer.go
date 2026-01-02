// Copyright 2025 Takhin Data, Inc.

package acl

import (
	"strings"
)

// Authorizer checks permissions for resource access
type Authorizer struct {
	store   *Store
	enabled bool
}

// NewAuthorizer creates a new ACL authorizer
func NewAuthorizer(store *Store, enabled bool) *Authorizer {
	return &Authorizer{
		store:   store,
		enabled: enabled,
	}
}

// Authorize checks if a principal has permission to perform an operation on a resource
func (a *Authorizer) Authorize(principal, host string, resourceType ResourceType, resourceName string, operation Operation) bool {
	// If ACL is disabled, allow everything
	if !a.enabled {
		return true
	}

	// Get all ACL entries
	entries := a.store.GetAll()

	// Check for explicit DENY first (deny takes precedence)
	for _, entry := range entries {
		if a.matches(entry, principal, host, resourceType, resourceName, operation) {
			if entry.PermissionType == PermissionTypeDeny {
				return false
			}
		}
	}

	// Then check for ALLOW
	for _, entry := range entries {
		if a.matches(entry, principal, host, resourceType, resourceName, operation) {
			if entry.PermissionType == PermissionTypeAllow {
				return true
			}
		}
	}

	// Default deny if no matching ALLOW found
	return false
}

// matches checks if an ACL entry matches the authorization request
func (a *Authorizer) matches(entry Entry, principal, host string, resourceType ResourceType, resourceName string, operation Operation) bool {
	// Check principal (supports wildcard *)
	if !matchesPrincipal(entry.Principal, principal) {
		return false
	}

	// Check host (supports wildcard *)
	if !matchesHost(entry.Host, host) {
		return false
	}

	// Check resource type
	if entry.ResourceType != resourceType {
		return false
	}

	// Check resource name with pattern matching
	if !matchesResourceName(entry.ResourceName, entry.PatternType, resourceName) {
		return false
	}

	// Check operation (OperationAll matches any operation)
	if entry.Operation != OperationAll && entry.Operation != operation {
		return false
	}

	return true
}

// matchesPrincipal checks if a principal matches the pattern
func matchesPrincipal(pattern, principal string) bool {
	if pattern == "*" {
		return true
	}
	return pattern == principal
}

// matchesHost checks if a host matches the pattern
func matchesHost(pattern, host string) bool {
	if pattern == "*" {
		return true
	}
	return pattern == host
}

// matchesResourceName checks if a resource name matches the ACL pattern
func matchesResourceName(pattern string, patternType PatternType, resourceName string) bool {
	switch patternType {
	case PatternTypeLiteral:
		return pattern == resourceName
	case PatternTypePrefixed:
		return strings.HasPrefix(resourceName, pattern)
	default:
		return false
	}
}
