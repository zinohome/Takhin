// Copyright 2025 Takhin Data, Inc.

package acl

// ResourceType represents the type of resource
type ResourceType int8

const (
	ResourceTypeTopic   ResourceType = 2
	ResourceTypeGroup   ResourceType = 3
	ResourceTypeCluster ResourceType = 4
)

// Operation represents an ACL operation
type Operation int8

const (
	OperationUnknown       Operation = 0
	OperationAll           Operation = 1
	OperationRead          Operation = 2
	OperationWrite         Operation = 3
	OperationCreate        Operation = 4
	OperationDelete        Operation = 5
	OperationAlter         Operation = 6
	OperationDescribe      Operation = 7
	OperationClusterAction Operation = 8
)

// PermissionType represents allow or deny
type PermissionType int8

const (
	PermissionTypeAllow PermissionType = 2
	PermissionTypeDeny  PermissionType = 3
)

// PatternType represents ACL pattern matching type
type PatternType int8

const (
	PatternTypeLiteral  PatternType = 0
	PatternTypePrefixed PatternType = 1
)

// Entry represents an ACL entry
type Entry struct {
	Principal      string         // User:alice, User:*, etc.
	Host           string         // IP address or * for any
	ResourceType   ResourceType   // Topic, Group, Cluster
	ResourceName   string         // Resource name (topic name, group name)
	PatternType    PatternType    // Literal or Prefixed
	Operation      Operation      // Read, Write, Delete, etc.
	PermissionType PermissionType // Allow or Deny
}

// ResourceFilter represents a filter for querying ACLs
type ResourceFilter struct {
	ResourceType ResourceType
	ResourceName *string
	PatternType  *PatternType
}

// AccessFilter represents a filter for querying ACLs by access control
type AccessFilter struct {
	Principal      *string
	Host           *string
	Operation      *Operation
	PermissionType *PermissionType
}

// Filter represents a complete ACL filter
type Filter struct {
	ResourceFilter ResourceFilter
	AccessFilter   AccessFilter
}

// String returns the string representation of ResourceType
func (rt ResourceType) String() string {
	switch rt {
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

// String returns the string representation of Operation
func (op Operation) String() string {
	switch op {
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
	case OperationClusterAction:
		return "ClusterAction"
	default:
		return "Unknown"
	}
}

// String returns the string representation of PermissionType
func (pt PermissionType) String() string {
	switch pt {
	case PermissionTypeAllow:
		return "Allow"
	case PermissionTypeDeny:
		return "Deny"
	default:
		return "Unknown"
	}
}

// String returns the string representation of PatternType
func (pt PatternType) String() string {
	switch pt {
	case PatternTypeLiteral:
		return "Literal"
	case PatternTypePrefixed:
		return "Prefixed"
	default:
		return "Unknown"
	}
}
