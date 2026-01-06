// Copyright 2025 Takhin Data, Inc.

package console

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// GetTieredStorageStats godoc
// @Summary Get tiered storage statistics
// @Description Returns statistics about tiered storage including hot/warm/cold segment counts
// @Tags tiered-storage
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/v1/tiered/stats [get]
func (s *Server) GetTieredStorageStats(w http.ResponseWriter, r *http.Request) {
	if s.tieredStorage == nil {
		s.respondError(w, http.StatusNotImplemented, "tiered storage not enabled")
		return
	}
	stats := s.tieredStorage.GetStats()
	s.respondJSON(w, http.StatusOK, stats)
}

// ArchiveSegment godoc
// @Summary Archive a segment to S3
// @Description Manually archive a specific segment to S3 cold storage
// @Tags tiered-storage
// @Accept json
// @Produce json
// @Param request body ArchiveRequest true "Archive request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tiered/archive [post]
func (s *Server) ArchiveSegment(w http.ResponseWriter, r *http.Request) {
	if s.tieredStorage == nil {
		s.respondError(w, http.StatusNotImplemented, "tiered storage not enabled")
		return
	}

	var req ArchiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SegmentPath == "" {
		s.respondError(w, http.StatusBadRequest, "segment_path is required")
		return
	}

	if err := s.tieredStorage.ArchiveSegment(r.Context(), req.SegmentPath); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "segment archived successfully",
	})
}

// RestoreSegment godoc
// @Summary Restore a segment from S3
// @Description Restore a specific segment from S3 cold storage to local disk
// @Tags tiered-storage
// @Accept json
// @Produce json
// @Param request body RestoreRequest true "Restore request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tiered/restore [post]
func (s *Server) RestoreSegment(w http.ResponseWriter, r *http.Request) {
	if s.tieredStorage == nil {
		s.respondError(w, http.StatusNotImplemented, "tiered storage not enabled")
		return
	}

	var req RestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SegmentPath == "" {
		s.respondError(w, http.StatusBadRequest, "segment_path is required")
		return
	}

	if err := s.tieredStorage.RestoreSegment(r.Context(), req.SegmentPath); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "segment restored successfully",
	})
}

// GetSegmentStatus godoc
// @Summary Get segment storage status
// @Description Get the current storage policy and archival status for a segment
// @Tags tiered-storage
// @Accept json
// @Produce json
// @Param segment_path path string true "Segment path"
// @Success 200 {object} SegmentStatusResponse
// @Failure 400 {object} map[string]string
// @Router /api/v1/tiered/segments/{segment_path} [get]
func (s *Server) GetSegmentStatus(w http.ResponseWriter, r *http.Request) {
	if s.tieredStorage == nil {
		s.respondError(w, http.StatusNotImplemented, "tiered storage not enabled")
		return
	}

	segmentPath := chi.URLParam(r, "segment_path")
	if segmentPath == "" {
		s.respondError(w, http.StatusBadRequest, "segment_path is required")
		return
	}

	policy := s.tieredStorage.GetSegmentPolicy(segmentPath)
	isArchived := s.tieredStorage.IsSegmentArchived(segmentPath)

	response := SegmentStatusResponse{
		SegmentPath: segmentPath,
		Policy:      string(policy),
		IsArchived:  isArchived,
	}

	s.respondJSON(w, http.StatusOK, response)
}

type ArchiveRequest struct {
	SegmentPath string `json:"segment_path"`
}

type RestoreRequest struct {
	SegmentPath string `json:"segment_path"`
}

type SegmentStatusResponse struct {
	SegmentPath string `json:"segment_path"`
	Policy      string `json:"policy"`
	IsArchived  bool   `json:"is_archived"`
}
