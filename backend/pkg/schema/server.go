// Copyright 2025 Takhin Data, Inc.

package schema

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server provides REST API for schema registry
type Server struct {
	registry *Registry
	router   *chi.Mux
	addr     string
}

// NewServer creates a new schema registry server
func NewServer(cfg *Config, addr string) (*Server, error) {
	registry, err := NewRegistry(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry: %w", err)
	}

	s := &Server{
		registry: registry,
		addr:     addr,
	}

	s.setupRoutes()
	return s, nil
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	r.Get("/subjects", s.handleGetSubjects)
	r.Get("/subjects/{subject}/versions", s.handleGetVersions)
	r.Get("/subjects/{subject}/versions/{version}", s.handleGetSchemaByVersion)
	r.Post("/subjects/{subject}/versions", s.handleRegisterSchema)
	r.Delete("/subjects/{subject}/versions/{version}", s.handleDeleteVersion)
	r.Delete("/subjects/{subject}", s.handleDeleteSubject)
	r.Get("/schemas/ids/{id}", s.handleGetSchemaByID)
	r.Get("/config/{subject}", s.handleGetCompatibility)
	r.Put("/config/{subject}", s.handleSetCompatibility)
	r.Post("/compatibility/subjects/{subject}/versions/{version}", s.handleTestCompatibility)

	s.router = r
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return http.ListenAndServe(s.addr, s.router)
}

// Close closes the server
func (s *Server) Close() error {
	return s.registry.Close()
}

// handleGetSubjects returns all subjects
func (s *Server) handleGetSubjects(w http.ResponseWriter, r *http.Request) {
	subjects, err := s.registry.GetSubjects()
	if err != nil {
		s.respondError(w, err)
		return
	}

	s.respondJSON(w, http.StatusOK, subjects)
}

// handleGetVersions returns all versions for a subject
func (s *Server) handleGetVersions(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")

	versions, err := s.registry.GetAllVersions(subject)
	if err != nil {
		s.respondError(w, err)
		return
	}

	s.respondJSON(w, http.StatusOK, versions)
}

// handleGetSchemaByVersion returns a specific schema version
func (s *Server) handleGetSchemaByVersion(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")
	versionStr := chi.URLParam(r, "version")

	var schema *Schema
	var err error

	if versionStr == "latest" {
		schema, err = s.registry.GetLatestSchema(subject)
	} else {
		version, parseErr := strconv.Atoi(versionStr)
		if parseErr != nil {
			s.respondError(w, NewSchemaError(ErrCodeInvalidVersion, "invalid version"))
			return
		}
		schema, err = s.registry.GetSchemaBySubjectVersion(subject, version)
	}

	if err != nil {
		s.respondError(w, err)
		return
	}

	s.respondJSON(w, http.StatusOK, schema)
}

// RegisterRequest represents a schema registration request
type RegisterRequest struct {
	Schema     string            `json:"schema"`
	SchemaType SchemaType        `json:"schemaType"`
	References []SchemaReference `json:"references,omitempty"`
}

// handleRegisterSchema registers a new schema version
func (s *Server) handleRegisterSchema(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, NewSchemaError(ErrCodeInvalidSchema, "invalid request body"))
		return
	}

	if req.SchemaType == "" {
		req.SchemaType = SchemaTypeAvro
	}

	schema, err := s.registry.RegisterSchema(subject, req.Schema, req.SchemaType, req.References)
	if err != nil {
		s.respondError(w, err)
		return
	}

	response := map[string]interface{}{
		"id": schema.ID,
	}

	s.respondJSON(w, http.StatusOK, response)
}

// handleDeleteVersion deletes a schema version
func (s *Server) handleDeleteVersion(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")
	versionStr := chi.URLParam(r, "version")

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		s.respondError(w, NewSchemaError(ErrCodeInvalidVersion, "invalid version"))
		return
	}

	if err := s.registry.DeleteSchemaVersion(subject, version); err != nil {
		s.respondError(w, err)
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]int{"version": version})
}

// handleDeleteSubject deletes all versions of a subject
func (s *Server) handleDeleteSubject(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")

	versions, err := s.registry.DeleteSubject(subject)
	if err != nil {
		s.respondError(w, err)
		return
	}

	s.respondJSON(w, http.StatusOK, versions)
}

// handleGetSchemaByID returns a schema by ID
func (s *Server) handleGetSchemaByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		s.respondError(w, NewSchemaError(ErrCodeSchemaNotFound, "invalid schema ID"))
		return
	}

	schema, err := s.registry.GetSchemaByID(id)
	if err != nil {
		s.respondError(w, err)
		return
	}

	response := map[string]interface{}{
		"schema": schema.Schema,
	}

	s.respondJSON(w, http.StatusOK, response)
}

// CompatibilityRequest represents a compatibility configuration request
type CompatibilityRequest struct {
	Compatibility CompatibilityMode `json:"compatibility"`
}

// handleGetCompatibility returns compatibility mode for a subject
func (s *Server) handleGetCompatibility(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")

	mode, err := s.registry.GetCompatibility(subject)
	if err != nil {
		s.respondError(w, err)
		return
	}

	response := map[string]string{
		"compatibilityLevel": string(mode),
	}

	s.respondJSON(w, http.StatusOK, response)
}

// handleSetCompatibility sets compatibility mode for a subject
func (s *Server) handleSetCompatibility(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")

	var req CompatibilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, NewSchemaError(ErrCodeInvalidCompatibilityLevel, "invalid request body"))
		return
	}

	if !isValidCompatibilityMode(req.Compatibility) {
		s.respondError(w, NewSchemaError(ErrCodeInvalidCompatibilityLevel, "invalid compatibility level"))
		return
	}

	if err := s.registry.SetCompatibility(subject, req.Compatibility); err != nil {
		s.respondError(w, err)
		return
	}

	response := map[string]string{
		"compatibility": string(req.Compatibility),
	}

	s.respondJSON(w, http.StatusOK, response)
}

// TestCompatibilityRequest represents a compatibility test request
type TestCompatibilityRequest struct {
	Schema     string     `json:"schema"`
	SchemaType SchemaType `json:"schemaType"`
}

// handleTestCompatibility tests schema compatibility
func (s *Server) handleTestCompatibility(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")
	versionStr := chi.URLParam(r, "version")

	version := 0
	if versionStr != "latest" {
		var err error
		version, err = strconv.Atoi(versionStr)
		if err != nil {
			s.respondError(w, NewSchemaError(ErrCodeInvalidVersion, "invalid version"))
			return
		}
	}

	var req TestCompatibilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, NewSchemaError(ErrCodeInvalidSchema, "invalid request body"))
		return
	}

	if req.SchemaType == "" {
		req.SchemaType = SchemaTypeAvro
	}

	compatible, err := s.registry.TestCompatibility(subject, req.Schema, req.SchemaType, version)
	if err != nil {
		s.respondError(w, err)
		return
	}

	response := map[string]bool{
		"is_compatible": compatible,
	}

	s.respondJSON(w, http.StatusOK, response)
}

// respondJSON sends a JSON response
func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func (s *Server) respondError(w http.ResponseWriter, err error) {
	if schemaErr, ok := err.(*SchemaError); ok {
		status := http.StatusInternalServerError
		if schemaErr.ErrorCode >= 40000 && schemaErr.ErrorCode < 50000 {
			status = schemaErr.ErrorCode / 100
		}
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(schemaErr)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{
		"error": err.Error(),
	})
}

// isValidCompatibilityMode validates compatibility mode
func isValidCompatibilityMode(mode CompatibilityMode) bool {
	validModes := []CompatibilityMode{
		CompatibilityNone,
		CompatibilityBackward,
		CompatibilityBackwardTransit,
		CompatibilityForward,
		CompatibilityForwardTransit,
		CompatibilityFull,
		CompatibilityFullTransit,
	}

	modeStr := strings.ToUpper(string(mode))
	for _, valid := range validModes {
		if string(valid) == modeStr {
			return true
		}
	}

	return false
}
