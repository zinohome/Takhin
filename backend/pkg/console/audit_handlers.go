// Copyright 2025 Takhin Data, Inc.

package console

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/takhin-data/takhin/pkg/audit"
)

// AuditEventResponse represents an audit event in API responses
type AuditEventResponse struct {
	Timestamp    string                 `json:"timestamp"`
	EventID      string                 `json:"event_id"`
	EventType    string                 `json:"event_type"`
	Severity     string                 `json:"severity"`
	Principal    string                 `json:"principal"`
	Host         string                 `json:"host"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	ResourceType string                 `json:"resource_type,omitempty"`
	ResourceName string                 `json:"resource_name,omitempty"`
	Operation    string                 `json:"operation"`
	Result       string                 `json:"result"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Error        string                 `json:"error,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
	APIVersion   string                 `json:"api_version,omitempty"`
	Duration     int64                  `json:"duration_ms,omitempty"`
}

// AuditQueryRequest represents a query for audit logs
type AuditQueryRequest struct {
	StartTime    *time.Time `json:"start_time,omitempty"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	EventTypes   []string   `json:"event_types,omitempty"`
	Principals   []string   `json:"principals,omitempty"`
	ResourceType string     `json:"resource_type,omitempty"`
	ResourceName string     `json:"resource_name,omitempty"`
	Result       string     `json:"result,omitempty"`
	Severity     string     `json:"severity,omitempty"`
	Limit        int        `json:"limit,omitempty"`
	Offset       int        `json:"offset,omitempty"`
}

// AuditQueryResponse represents the response for audit log queries
type AuditQueryResponse struct {
	Events     []AuditEventResponse `json:"events"`
	TotalCount int                  `json:"total_count"`
	Limit      int                  `json:"limit"`
	Offset     int                  `json:"offset"`
}

// handleQueryAuditLogs queries audit logs
// @Summary Query audit logs
// @Description Query audit logs with filters
// @Tags audit
// @Accept json
// @Produce json
// @Param query body AuditQueryRequest true "Query parameters"
// @Success 200 {object} AuditQueryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/audit/logs [post]
func (s *Server) handleQueryAuditLogs(w http.ResponseWriter, r *http.Request) {
	if s.auditLogger == nil {
		s.respondError(w, http.StatusServiceUnavailable, "audit logging not enabled")
		return
	}

	var req AuditQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert to audit filter
	filter := audit.Filter{
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		ResourceType: req.ResourceType,
		ResourceName: req.ResourceName,
		Result:       req.Result,
		Limit:        req.Limit,
		Offset:       req.Offset,
	}

	// Convert event types
	if len(req.EventTypes) > 0 {
		filter.EventTypes = make([]audit.EventType, len(req.EventTypes))
		for i, et := range req.EventTypes {
			filter.EventTypes[i] = audit.EventType(et)
		}
	}

	// Convert principals
	if len(req.Principals) > 0 {
		filter.Principals = req.Principals
	}

	// Convert severity
	if req.Severity != "" {
		filter.Severity = audit.Severity(req.Severity)
	}

	// Query audit logs
	events, err := s.auditLogger.Query(filter)
	if err != nil {
		s.logger.Error("failed to query audit logs", "error", err)
		s.respondError(w, http.StatusInternalServerError, "failed to query audit logs")
		return
	}

	// Convert to response format
	respEvents := make([]AuditEventResponse, len(events))
	for i, event := range events {
		respEvents[i] = AuditEventResponse{
			Timestamp:    event.Timestamp.Format(time.RFC3339),
			EventID:      event.EventID,
			EventType:    string(event.EventType),
			Severity:     string(event.Severity),
			Principal:    event.Principal,
			Host:         event.Host,
			UserAgent:    event.UserAgent,
			ResourceType: event.ResourceType,
			ResourceName: event.ResourceName,
			Operation:    event.Operation,
			Result:       event.Result,
			Metadata:     event.Metadata,
			Error:        event.Error,
			RequestID:    event.RequestID,
			SessionID:    event.SessionID,
			APIVersion:   event.APIVersion,
			Duration:     event.Duration,
		}
	}

	resp := AuditQueryResponse{
		Events:     respEvents,
		TotalCount: len(respEvents),
		Limit:      req.Limit,
		Offset:     req.Offset,
	}

	s.respondJSON(w, http.StatusOK, resp)
}

// handleGetAuditStats gets audit log statistics
// @Summary Get audit statistics
// @Description Get statistics about audit logs
// @Tags audit
// @Produce json
// @Param start_time query string false "Start time (RFC3339)"
// @Param end_time query string false "End time (RFC3339)"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /api/audit/stats [get]
func (s *Server) handleGetAuditStats(w http.ResponseWriter, r *http.Request) {
	if s.auditLogger == nil {
		s.respondError(w, http.StatusServiceUnavailable, "audit logging not enabled")
		return
	}

	// Parse time range
	var startTime, endTime *time.Time
	if st := r.URL.Query().Get("start_time"); st != "" {
		t, err := time.Parse(time.RFC3339, st)
		if err == nil {
			startTime = &t
		}
	}
	if et := r.URL.Query().Get("end_time"); et != "" {
		t, err := time.Parse(time.RFC3339, et)
		if err == nil {
			endTime = &t
		}
	}

	// Query all events in time range
	filter := audit.Filter{
		StartTime: startTime,
		EndTime:   endTime,
	}

	events, err := s.auditLogger.Query(filter)
	if err != nil {
		s.logger.Error("failed to query audit logs", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to query audit logs")
		return
	}

	// Calculate statistics
	stats := map[string]interface{}{
		"total_events": len(events),
		"by_type":      make(map[string]int),
		"by_severity":  make(map[string]int),
		"by_principal": make(map[string]int),
		"by_result":    make(map[string]int),
	}

	for _, event := range events {
		stats["by_type"].(map[string]int)[string(event.EventType)]++
		stats["by_severity"].(map[string]int)[string(event.Severity)]++
		stats["by_principal"].(map[string]int)[event.Principal]++
		stats["by_result"].(map[string]int)[event.Result]++
	}

	s.respondJSON(w, http.StatusOK, stats)
}

// handleGetAuditEvent gets a specific audit event by ID
// @Summary Get audit event
// @Description Get a specific audit event by ID
// @Tags audit
// @Produce json
// @Param event_id path string true "Event ID"
// @Success 200 {object} AuditEventResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/audit/events/{event_id} [get]
func (s *Server) handleGetAuditEvent(w http.ResponseWriter, r *http.Request) {
	if s.auditLogger == nil {
		s.respondError(w, http.StatusServiceUnavailable, "audit logging not enabled")
		return
	}

	eventID := chi.URLParam(r, "event_id")
	if eventID == "" {
		s.respondError(w, http.StatusBadRequest, "event_id is required")
		return
	}

	// Query all events and find by ID
	events, err := s.auditLogger.Query(audit.Filter{})
	if err != nil {
		s.logger.Error("failed to query audit logs", "error", err)
		s.respondError(w, http.StatusInternalServerError, "failed to query audit logs")
		return
	}

	// Find event by ID
	for _, event := range events {
		if event.EventID == eventID {
			resp := AuditEventResponse{
				Timestamp:    event.Timestamp.Format(time.RFC3339),
				EventID:      event.EventID,
				EventType:    string(event.EventType),
				Severity:     string(event.Severity),
				Principal:    event.Principal,
				Host:         event.Host,
				UserAgent:    event.UserAgent,
				ResourceType: event.ResourceType,
				ResourceName: event.ResourceName,
				Operation:    event.Operation,
				Result:       event.Result,
				Metadata:     event.Metadata,
				Error:        event.Error,
				RequestID:    event.RequestID,
				SessionID:    event.SessionID,
				APIVersion:   event.APIVersion,
				Duration:     event.Duration,
			}
			s.respondJSON(w, http.StatusOK, resp)
			return
		}
	}

	s.respondError(w, http.StatusNotFound, "event not found")
}

// handleExportAuditLogs exports audit logs
// @Summary Export audit logs
// @Description Export audit logs in various formats
// @Tags audit
// @Produce json,text/csv
// @Param format query string false "Export format (json, csv)" default(json)
// @Param start_time query string false "Start time (RFC3339)"
// @Param end_time query string false "End time (RFC3339)"
// @Param limit query int false "Limit" default(1000)
// @Success 200 {file} file
// @Failure 500 {object} ErrorResponse
// @Router /api/audit/export [get]
func (s *Server) handleExportAuditLogs(w http.ResponseWriter, r *http.Request) {
	if s.auditLogger == nil {
		s.respondError(w, http.StatusServiceUnavailable, "audit logging not enabled")
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	// Parse time range
	var startTime, endTime *time.Time
	if st := r.URL.Query().Get("start_time"); st != "" {
		t, err := time.Parse(time.RFC3339, st)
		if err == nil {
			startTime = &t
		}
	}
	if et := r.URL.Query().Get("end_time"); et != "" {
		t, err := time.Parse(time.RFC3339, et)
		if err == nil {
			endTime = &t
		}
	}

	// Parse limit
	limit := 1000
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	// Query events
	filter := audit.Filter{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     limit,
	}

	events, err := s.auditLogger.Query(filter)
	if err != nil {
		s.logger.Error("failed to query audit logs", "error", err)
		s.respondError(w, http.StatusInternalServerError, "failed to query audit logs")
		return
	}

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=audit-logs.json")
		json.NewEncoder(w).Encode(events)
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=audit-logs.csv")
		// Write CSV header
		w.Write([]byte("timestamp,event_id,event_type,severity,principal,host,resource_type,resource_name,operation,result\n"))
		// Write events
		for _, event := range events {
			line := event.Timestamp.Format(time.RFC3339) + "," +
				event.EventID + "," +
				string(event.EventType) + "," +
				string(event.Severity) + "," +
				event.Principal + "," +
				event.Host + "," +
				event.ResourceType + "," +
				event.ResourceName + "," +
				event.Operation + "," +
				event.Result + "\n"
			w.Write([]byte(line))
		}
	default:
		s.respondError(w, http.StatusBadRequest, "unsupported format")
	}
}
