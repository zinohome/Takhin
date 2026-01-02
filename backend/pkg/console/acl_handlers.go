// Copyright 2025 Takhin Data, Inc.

package console

import (
	"encoding/json"
	"net/http"

	"github.com/takhin-data/takhin/pkg/acl"
)

// ACLManager interface for ACL operations (similar to handler.Authorizer)
type ACLManager interface {
	IsEnabled() bool
	AddACL(entry interface{}) error
	DeleteACL(filter interface{}) (int, error)
	ListACL(filter interface{}) []interface{}
	Stats() interface{}
}

// CreateACLRequest represents a request to create an ACL
type CreateACLRequest struct {
	Principal      string `json:"principal"`
	Host           string `json:"host"`
	ResourceType   int8   `json:"resource_type"`
	ResourceName   string `json:"resource_name"`
	PatternType    int8   `json:"pattern_type"`
	Operation      int8   `json:"operation"`
	PermissionType int8   `json:"permission_type"`
}

// ACLResponse represents an ACL entry
type ACLResponse struct {
	Principal      string `json:"principal"`
	Host           string `json:"host"`
	ResourceType   string `json:"resource_type"`
	ResourceName   string `json:"resource_name"`
	PatternType    string `json:"pattern_type"`
	Operation      string `json:"operation"`
	PermissionType string `json:"permission_type"`
}

// ACLStatsResponse represents ACL statistics
type ACLStatsResponse struct {
	Enabled        bool  `json:"enabled"`
	TotalACLs      int   `json:"total_acls"`
	AllowCount     int64 `json:"allow_count"`
	DenyCount      int64 `json:"deny_count"`
	CacheHitCount  int64 `json:"cache_hit_count"`
	CacheMissCount int64 `json:"cache_miss_count"`
}

// handleCreateACL handles POST /api/acls
// @Summary Create ACL
// @Description Create a new ACL entry
// @Tags ACL
// @Accept json
// @Produce json
// @Param acl body CreateACLRequest true "ACL to create"
// @Success 201 {object} ACLResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/acls [post]
func (s *Server) handleCreateACL(w http.ResponseWriter, r *http.Request) {
	if s.aclManager == nil {
		s.respondError(w, http.StatusNotImplemented, "ACL not enabled")
		return
	}

	var req CreateACLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create ACL entry
	entry, err := acl.NewEntry(
		req.Principal,
		req.Host,
		acl.ResourceType(req.ResourceType),
		req.ResourceName,
		acl.ResourcePatternType(req.PatternType),
		acl.Operation(req.Operation),
		acl.PermissionType(req.PermissionType),
	)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.aclManager.AddACL(entry); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := ACLResponse{
		Principal:      entry.Principal,
		Host:           entry.Host,
		ResourceType:   entry.ResourceType.String(),
		ResourceName:   entry.ResourceName,
		PatternType:    entry.PatternType.String(),
		Operation:      entry.Operation.String(),
		PermissionType: entry.PermissionType.String(),
	}

	s.respondJSON(w, http.StatusCreated, resp)
}

// handleListACLs handles GET /api/acls
// @Summary List ACLs
// @Description List all ACL entries
// @Tags ACL
// @Produce json
// @Param principal query string false "Filter by principal"
// @Param resource_type query int false "Filter by resource type"
// @Param resource_name query string false "Filter by resource name"
// @Success 200 {array} ACLResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/acls [get]
func (s *Server) handleListACLs(w http.ResponseWriter, r *http.Request) {
	if s.aclManager == nil {
		s.respondError(w, http.StatusNotImplemented, "ACL not enabled")
		return
	}

	// Build filter from query parameters
	filter := &acl.Filter{}
	if principal := r.URL.Query().Get("principal"); principal != "" {
		filter.Principal = &principal
	}
	if resourceName := r.URL.Query().Get("resource_name"); resourceName != "" {
		filter.ResourceName = &resourceName
	}

	// List ACLs
	entriesInterface := s.aclManager.ListACL(filter)

	// Convert to response format
	responses := make([]ACLResponse, 0, len(entriesInterface))
	for _, ei := range entriesInterface {
		if entry, ok := ei.(*acl.Entry); ok {
			responses = append(responses, ACLResponse{
				Principal:      entry.Principal,
				Host:           entry.Host,
				ResourceType:   entry.ResourceType.String(),
				ResourceName:   entry.ResourceName,
				PatternType:    entry.PatternType.String(),
				Operation:      entry.Operation.String(),
				PermissionType: entry.PermissionType.String(),
			})
		}
	}

	s.respondJSON(w, http.StatusOK, responses)
}

// handleDeleteACL handles DELETE /api/acls
// @Summary Delete ACLs
// @Description Delete ACL entries matching the filter
// @Tags ACL
// @Produce json
// @Param principal query string false "Filter by principal"
// @Param resource_type query int false "Filter by resource type"
// @Param resource_name query string false "Filter by resource name"
// @Success 200 {object} map[string]int "deleted_count"
// @Failure 500 {object} ErrorResponse
// @Router /api/acls [delete]
func (s *Server) handleDeleteACL(w http.ResponseWriter, r *http.Request) {
	if s.aclManager == nil {
		s.respondError(w, http.StatusNotImplemented, "ACL not enabled")
		return
	}

	// Build filter from query parameters
	filter := &acl.Filter{}
	if principal := r.URL.Query().Get("principal"); principal != "" {
		filter.Principal = &principal
	}
	if resourceName := r.URL.Query().Get("resource_name"); resourceName != "" {
		filter.ResourceName = &resourceName
	}

	// Delete ACLs
	deleted, err := s.aclManager.DeleteACL(filter)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]int{"deleted_count": deleted})
}

// handleACLStats handles GET /api/acls/stats
// @Summary Get ACL statistics
// @Description Get ACL authorization statistics
// @Tags ACL
// @Produce json
// @Success 200 {object} ACLStatsResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/acls/stats [get]
func (s *Server) handleACLStats(w http.ResponseWriter, r *http.Request) {
	if s.aclManager == nil {
		s.respondError(w, http.StatusNotImplemented, "ACL not enabled")
		return
	}

	statsInterface := s.aclManager.Stats()
	if stats, ok := statsInterface.(acl.AuthStats); ok {
		resp := ACLStatsResponse{
			Enabled:        stats.Enabled,
			TotalACLs:      stats.TotalACLs,
			AllowCount:     stats.AllowCount,
			DenyCount:      stats.DenyCount,
			CacheHitCount:  stats.CacheHitCount,
			CacheMissCount: stats.CacheMissCount,
		}
		s.respondJSON(w, http.StatusOK, resp)
		return
	}

	s.respondError(w, http.StatusInternalServerError, "failed to get stats")
}
