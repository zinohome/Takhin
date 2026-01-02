// Copyright 2025 Takhin Data, Inc.

package protocol

// CreateAcls API Key
const (
	CreateAclsKey  APIKey = 30
	DescribeAclsKey APIKey = 29
	DeleteAclsKey   APIKey = 31
)

// ACL-specific error codes
const (
	SecurityDisabledError ErrorCode = 54
)

// CreateAclsRequest represents a CreateAcls request
type CreateAclsRequest struct {
	Creations []AclCreation
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

// CreateAclsResponse represents a CreateAcls response
type CreateAclsResponse struct {
	ThrottleTimeMs int32
	Results        []AclCreationResult
}

// AclCreationResult represents the result of creating an ACL
type AclCreationResult struct {
	ErrorCode    ErrorCode
	ErrorMessage *string
}

// DescribeAclsRequest represents a DescribeAcls request
type DescribeAclsRequest struct {
	ResourceTypeFilter   int8
	ResourceNameFilter   *string
	PatternTypeFilter    int8
	PrincipalFilter      *string
	HostFilter           *string
	Operation            int8
	PermissionType       int8
}

// DescribeAclsResponse represents a DescribeAcls response
type DescribeAclsResponse struct {
	ThrottleTimeMs int32
	ErrorCode      ErrorCode
	ErrorMessage   *string
	Resources      []AclResource
}

// AclResource represents ACLs for a resource
type AclResource struct {
	ResourceType int8
	ResourceName string
	PatternType  int8
	Acls         []AclDescription
}

// AclDescription describes an ACL entry
type AclDescription struct {
	Principal      string
	Host           string
	Operation      int8
	PermissionType int8
}

// DeleteAclsRequest represents a DeleteAcls request
type DeleteAclsRequest struct {
	Filters []AclFilter
}

// AclFilter represents an ACL filter for deletion
type AclFilter struct {
	ResourceTypeFilter int8
	ResourceNameFilter *string
	PatternTypeFilter  int8
	PrincipalFilter    *string
	HostFilter         *string
	Operation          int8
	PermissionType     int8
}

// DeleteAclsResponse represents a DeleteAcls response
type DeleteAclsResponse struct {
	ThrottleTimeMs int32
	Results        []DeleteAclsResult
}

// DeleteAclsResult represents the result of deleting ACLs
type DeleteAclsResult struct {
	ErrorCode      ErrorCode
	ErrorMessage   *string
	MatchingAcls   []DeleteAclsMatch
}

// DeleteAclsMatch represents a matched ACL for deletion
type DeleteAclsMatch struct {
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
