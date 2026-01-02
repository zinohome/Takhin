// Copyright 2025 Takhin Data, Inc.

package acl

import (
	"fmt"
	"strings"
)

// ResourceType represents the type of resource
type ResourceType int8

const (
	ResourceTypeUnknown ResourceType = 0
	ResourceTypeTopic   ResourceType = 2
	ResourceTypeGroup   ResourceType = 3
	ResourceTypeCluster ResourceType = 4
)

func (r ResourceType) String() string {
	switch r {
	case ResourceTypeTopic:
		return "Topic"
	case ResourceTypeGroup:
		return "Group"
	case ResourceTypeCluster:
		return "Cluster"
	default:
		return "Unknown"
	}
}

// ResourcePatternType represents how the resource name is matched
type ResourcePatternType int8

const (
	PatternTypeUnknown ResourcePatternType = 0
	PatternTypeLiteral ResourcePatternType = 2
	PatternTypePrefix  ResourcePatternType = 3
)

func (p ResourcePatternType) String() string {
	switch p {
	case PatternTypeLiteral:
		return "Literal"
	case PatternTypePrefix:
		return "Prefix"
	default:
		return "Unknown"
	}
}

// Operation represents an ACL operation
type Operation int8

const (
	OperationUnknown  Operation = 0
	OperationAll      Operation = 1
	OperationRead     Operation = 2
	OperationWrite    Operation = 3
	OperationCreate   Operation = 4
	OperationDelete   Operation = 5
	OperationAlter    Operation = 6
	OperationDescribe Operation = 7
)

func (o Operation) String() string {
	switch o {
	case OperationAll:
		return "All"
	case OperationRead:
		return "Read"
	case OperationWrite:
		return "Write"
	case OperationCreate:
		return "Create"
	case OperationDelete:
		return "Delete"
	case OperationAlter:
		return "Alter"
	case OperationDescribe:
		return "Describe"
	default:
		return "Unknown"
	}
}

// PermissionType represents allow or deny
type PermissionType int8

const (
	PermissionTypeUnknown PermissionType = 0
	PermissionTypeAllow   PermissionType = 2
	PermissionTypeDeny    PermissionType = 3
)

func (p PermissionType) String() string {
	switch p {
	case PermissionTypeAllow:
		return "Allow"
	case PermissionTypeDeny:
		return "Deny"
	default:
		return "Unknown"
	}
}

// Entry represents an ACL entry
type Entry struct {
	Principal      string              // User:username or User:*
	Host           string              // IP address or *
	ResourceType   ResourceType        // Topic, Group, Cluster
	ResourceName   string              // Name of the resource
	PatternType    ResourcePatternType // Literal or Prefix
	Operation      Operation           // Read, Write, etc.
	PermissionType PermissionType      // Allow or Deny
}

// NewEntry creates a new ACL entry with validation
func NewEntry(principal, host string, resourceType ResourceType, resourceName string,
	patternType ResourcePatternType, operation Operation, permission PermissionType) (*Entry, error) {

	if principal == "" {
		return nil, fmt.Errorf("principal cannot be empty")
	}
	if !strings.HasPrefix(principal, "User:") {
		return nil, fmt.Errorf("principal must start with 'User:' prefix")
	}
	if host == "" {
		host = "*"
	}
	if resourceType == ResourceTypeUnknown {
		return nil, fmt.Errorf("invalid resource type")
	}
	if resourceName == "" {
		return nil, fmt.Errorf("resource name cannot be empty")
	}
	if patternType == PatternTypeUnknown {
		return nil, fmt.Errorf("invalid pattern type")
	}
	if operation == OperationUnknown {
		return nil, fmt.Errorf("invalid operation")
	}
	if permission == PermissionTypeUnknown {
		return nil, fmt.Errorf("invalid permission type")
	}

	return &Entry{
		Principal:      principal,
		Host:           host,
		ResourceType:   resourceType,
		ResourceName:   resourceName,
		PatternType:    patternType,
		Operation:      operation,
		PermissionType: permission,
	}, nil
}

// Key returns a unique key for this ACL entry
func (e *Entry) Key() string {
	return fmt.Sprintf("%s|%s|%d|%s|%d|%d|%d",
		e.Principal, e.Host, e.ResourceType, e.ResourceName,
		e.PatternType, e.Operation, e.PermissionType)
}

// Matches checks if this ACL entry matches the given request
func (e *Entry) Matches(principal, host string, resourceType ResourceType,
	resourceName string, operation Operation) bool {

	// Check principal match
	if e.Principal != principal && e.Principal != "User:*" {
		return false
	}

	// Check host match
	if e.Host != host && e.Host != "*" {
		return false
	}

	// Check resource type match
	if e.ResourceType != resourceType {
		return false
	}

	// Check resource name match based on pattern type
	switch e.PatternType {
	case PatternTypeLiteral:
		if e.ResourceName != resourceName && e.ResourceName != "*" {
			return false
		}
	case PatternTypePrefix:
		if !strings.HasPrefix(resourceName, e.ResourceName) {
			return false
		}
	default:
		return false
	}

	// Check operation match
	if e.Operation != operation && e.Operation != OperationAll {
		return false
	}

	return true
}

// Resource represents an ACL resource
type Resource struct {
	Type        ResourceType
	Name        string
	PatternType ResourcePatternType
}

// Filter represents criteria for filtering ACLs
type Filter struct {
	Principal      *string
	Host           *string
	ResourceType   *ResourceType
	ResourceName   *string
	PatternType    *ResourcePatternType
	Operation      *Operation
	PermissionType *PermissionType
}

// Matches checks if an ACL entry matches this filter
func (f *Filter) Matches(entry *Entry) bool {
	if f.Principal != nil && *f.Principal != entry.Principal {
		return false
	}
	if f.Host != nil && *f.Host != entry.Host {
		return false
	}
	if f.ResourceType != nil && *f.ResourceType != entry.ResourceType {
		return false
	}
	if f.ResourceName != nil && *f.ResourceName != entry.ResourceName {
		return false
	}
	if f.PatternType != nil && *f.PatternType != entry.PatternType {
		return false
	}
	if f.Operation != nil && *f.Operation != entry.Operation {
		return false
	}
	if f.PermissionType != nil && *f.PermissionType != entry.PermissionType {
		return false
	}
	return true
}
