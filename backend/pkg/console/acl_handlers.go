// Copyright 2025 Takhin Data, Inc.

package console

import (
	"encoding/json"
	"net/http"

	"github.com/takhin-data/takhin/pkg/acl"
)

// AclEntryRequest represents an ACL entry for API requests
type AclEntryRequest struct {
	Principal      string `json:"principal"`
	Host           string `json:"host"`
	ResourceType   string `json:"resource_type"`
	ResourceName   string `json:"resource_name"`
	PatternType    string `json:"pattern_type"`
	Operation      string `json:"operation"`
	PermissionType string `json:"permission_type"`
}

// AclEntryResponse represents an ACL entry for API responses
type AclEntryResponse struct {
	Principal      string `json:"principal"`
	Host           string `json:"host"`
	ResourceType   string `json:"resource_type"`
	ResourceName   string `json:"resource_name"`
	PatternType    string `json:"pattern_type"`
	Operation      string `json:"operation"`
	PermissionType string `json:"permission_type"`
}

// AclFilterRequest represents an ACL filter for API requests
type AclFilterRequest struct {
	ResourceType   *string `json:"resource_type,omitempty"`
	ResourceName   *string `json:"resource_name,omitempty"`
	PatternType    *string `json:"pattern_type,omitempty"`
	Principal      *string `json:"principal,omitempty"`
	Host           *string `json:"host,omitempty"`
	Operation      *string `json:"operation,omitempty"`
	PermissionType *string `json:"permission_type,omitempty"`
}

// handleListAcls godoc
// @Summary List ACLs
// @Description Get all ACL entries, optionally filtered
// @Tags ACL
// @Accept json
// @Produce json
// @Param resource_type query string false "Resource type filter (Topic, Group, Cluster)"
// @Param resource_name query string false "Resource name filter"
// @Param principal query string false "Principal filter"
// @Success 200 {object} map[string]interface{} "ACL entries"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/acls [get]
func (s *Server) handleListAcls(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	resourceType := r.URL.Query().Get("resource_type")
	resourceName := r.URL.Query().Get("resource_name")
	principal := r.URL.Query().Get("principal")

	// Build filter
	var entries []acl.Entry
	if resourceType == "" && resourceName == "" && principal == "" {
		// No filter, get all
		entries = s.aclStore.GetAll()
	} else {
		filter := acl.Filter{}
		
		if resourceType != "" {
			filter.ResourceFilter.ResourceType = parseResourceType(resourceType)
			if resourceName != "" {
				filter.ResourceFilter.ResourceName = &resourceName
			}
		}
		
		if principal != "" {
			filter.AccessFilter.Principal = &principal
		}
		
		entries = s.aclStore.List(filter)
	}

	// Convert to response format
	response := make([]AclEntryResponse, len(entries))
	for i, entry := range entries {
		response[i] = AclEntryResponse{
			Principal:      entry.Principal,
			Host:           entry.Host,
			ResourceType:   entry.ResourceType.String(),
			ResourceName:   entry.ResourceName,
			PatternType:    entry.PatternType.String(),
			Operation:      entry.Operation.String(),
			PermissionType: entry.PermissionType.String(),
		}
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"acls":  response,
		"count": len(response),
	})
}

// handleCreateAcl godoc
// @Summary Create ACL
// @Description Create a new ACL entry
// @Tags ACL
// @Accept json
// @Produce json
// @Param acl body AclEntryRequest true "ACL entry to create"
// @Success 201 {object} map[string]interface{} "ACL created"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 409 {object} map[string]interface{} "ACL already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/acls [post]
func (s *Server) handleCreateAcl(w http.ResponseWriter, r *http.Request) {
	var req AclEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.Principal == "" || req.Host == "" || req.ResourceType == "" ||
		req.ResourceName == "" || req.PatternType == "" ||
		req.Operation == "" || req.PermissionType == "" {
		s.respondError(w, http.StatusBadRequest, "Missing required fields")
		return
	}

	// Create ACL entry
	entry := acl.Entry{
		Principal:      req.Principal,
		Host:           req.Host,
		ResourceType:   parseResourceType(req.ResourceType),
		ResourceName:   req.ResourceName,
		PatternType:    parsePatternType(req.PatternType),
		Operation:      parseOperation(req.Operation),
		PermissionType: parsePermissionType(req.PermissionType),
	}

	// Add to store
	if err := s.aclStore.Add(entry); err != nil {
		s.respondError(w, http.StatusConflict, err.Error())
		return
	}

	// Save to disk
	if err := s.aclStore.Save(); err != nil {
		s.logger.Error("failed to save ACLs", "error", err)
		s.respondError(w, http.StatusInternalServerError, "Failed to persist ACL")
		return
	}

	s.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "ACL created successfully",
		"acl": AclEntryResponse{
			Principal:      entry.Principal,
			Host:           entry.Host,
			ResourceType:   entry.ResourceType.String(),
			ResourceName:   entry.ResourceName,
			PatternType:    entry.PatternType.String(),
			Operation:      entry.Operation.String(),
			PermissionType: entry.PermissionType.String(),
		},
	})
}

// handleDeleteAcls godoc
// @Summary Delete ACLs
// @Description Delete ACL entries matching the filter
// @Tags ACL
// @Accept json
// @Produce json
// @Param filter body AclFilterRequest true "Filter for ACLs to delete"
// @Success 200 {object} map[string]interface{} "ACLs deleted"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/acls [delete]
func (s *Server) handleDeleteAcls(w http.ResponseWriter, r *http.Request) {
	var req AclFilterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Build filter
	filter := acl.Filter{}
	
	if req.ResourceType != nil {
		filter.ResourceFilter.ResourceType = parseResourceType(*req.ResourceType)
	}
	
	if req.ResourceName != nil {
		filter.ResourceFilter.ResourceName = req.ResourceName
	}
	
	if req.PatternType != nil {
		pt := parsePatternType(*req.PatternType)
		filter.ResourceFilter.PatternType = &pt
	}
	
	if req.Principal != nil {
		filter.AccessFilter.Principal = req.Principal
	}
	
	if req.Host != nil {
		filter.AccessFilter.Host = req.Host
	}
	
	if req.Operation != nil {
		op := parseOperation(*req.Operation)
		filter.AccessFilter.Operation = &op
	}
	
	if req.PermissionType != nil {
		pt := parsePermissionType(*req.PermissionType)
		filter.AccessFilter.PermissionType = &pt
	}

	// Delete matching ACLs
	deleted, err := s.aclStore.Delete(filter)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Save to disk
	if err := s.aclStore.Save(); err != nil {
		s.logger.Error("failed to save ACLs", "error", err)
		s.respondError(w, http.StatusInternalServerError, "Failed to persist changes")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "ACLs deleted successfully",
		"deleted": deleted,
	})
}

// Helper functions to parse string values to ACL types

func parseResourceType(s string) acl.ResourceType {
	switch s {
	case "Topic":
		return acl.ResourceTypeTopic
	case "Group":
		return acl.ResourceTypeGroup
	case "Cluster":
		return acl.ResourceTypeCluster
	default:
		return acl.ResourceTypeTopic
	}
}

func parsePatternType(s string) acl.PatternType {
	switch s {
	case "Literal":
		return acl.PatternTypeLiteral
	case "Prefixed":
		return acl.PatternTypePrefixed
	default:
		return acl.PatternTypeLiteral
	}
}

func parseOperation(s string) acl.Operation {
	switch s {
	case "All":
		return acl.OperationAll
	case "Read":
		return acl.OperationRead
	case "Write":
		return acl.OperationWrite
	case "Create":
		return acl.OperationCreate
	case "Delete":
		return acl.OperationDelete
	case "Alter":
		return acl.OperationAlter
	case "Describe":
		return acl.OperationDescribe
	case "ClusterAction":
		return acl.OperationClusterAction
	default:
		return acl.OperationRead
	}
}

func parsePermissionType(s string) acl.PermissionType {
	switch s {
	case "Allow":
		return acl.PermissionTypeAllow
	case "Deny":
		return acl.PermissionTypeDeny
	default:
		return acl.PermissionTypeAllow
	}
}
