// Copyright 2025 Takhin Data, Inc.

package console

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// TierStatsResponse represents tier statistics
type TierStatsResponse struct {
	HotSegments     int     `json:"hot_segments"`
	WarmSegments    int     `json:"warm_segments"`
	ColdSegments    int     `json:"cold_segments"`
	PromotionCount  int64   `json:"promotion_count"`
	DemotionCount   int64   `json:"demotion_count"`
	CacheHits       int64   `json:"cache_hits"`
	CacheMisses     int64   `json:"cache_misses"`
	TrackedSegments int     `json:"tracked_segments"`
}

// AccessStatsResponse represents access pattern statistics
type AccessStatsResponse struct {
	SegmentPath    string  `json:"segment_path"`
	AccessCount    int64   `json:"access_count"`
	LastAccessAt   string  `json:"last_access_at"`
	FirstAccessAt  string  `json:"first_access_at"`
	ReadBytes      int64   `json:"read_bytes"`
	AverageReadHz  float64 `json:"average_read_hz"`
}

// CostAnalysisResponse represents cost analysis data
type CostAnalysisResponse struct {
	HotStorageGB          float64 `json:"hot_storage_gb"`
	ColdStorageGB         float64 `json:"cold_storage_gb"`
	HotStorageCostMonthly string  `json:"hot_storage_cost_monthly"`
	ColdStorageCostMonthly string `json:"cold_storage_cost_monthly"`
	TotalCostMonthly      string  `json:"total_cost_monthly"`
	CostSavingsPct        string  `json:"cost_savings_pct"`
	RetrievalCostPerRestore string `json:"retrieval_cost_per_restore"`
}

// handleGetTierStats returns tier manager statistics
// @Summary Get tier statistics
// @Description Returns hot/warm/cold tier statistics and promotion/demotion counts
// @Tags tiered-storage
// @Produce json
// @Success 200 {object} TierStatsResponse
// @Router /api/v1/tiers/stats [get]
func (s *Server) handleGetTierStats(w http.ResponseWriter, r *http.Request) {
	if s.tierManager == nil {
		s.respondError(w, http.StatusServiceUnavailable, "tier management not enabled")
		return
	}

	stats := s.tierManager.GetTierStats()
	
	response := TierStatsResponse{
		HotSegments:     stats["hot_segments"].(int),
		WarmSegments:    stats["warm_segments"].(int),
		ColdSegments:    stats["cold_segments"].(int),
		PromotionCount:  stats["promotion_count"].(int64),
		DemotionCount:   stats["demotion_count"].(int64),
		CacheHits:       stats["cache_hits"].(int64),
		CacheMisses:     stats["cache_misses"].(int64),
		TrackedSegments: stats["tracked_segments"].(int),
	}

	s.respondJSON(w, http.StatusOK, response)
}

// handleGetAccessStats returns access statistics for a segment
// @Summary Get segment access statistics
// @Description Returns access pattern statistics for a specific segment
// @Tags tiered-storage
// @Produce json
// @Param segment_path path string true "Segment path"
// @Success 200 {object} AccessStatsResponse
// @Router /api/v1/tiers/access/{segment_path} [get]
func (s *Server) handleGetAccessStats(w http.ResponseWriter, r *http.Request) {
	if s.tierManager == nil {
		s.respondError(w, http.StatusServiceUnavailable, "tier management not enabled")
		return
	}

	segmentPath := chi.URLParam(r, "segment_path")
	if segmentPath == "" {
		s.respondError(w, http.StatusBadRequest, "segment_path is required")
		return
	}

	stats := s.tierManager.GetAccessStats(segmentPath)
	if stats == nil {
		s.respondError(w, http.StatusNotFound, "no access statistics for segment")
		return
	}

	response := AccessStatsResponse{
		SegmentPath:   stats.SegmentPath,
		AccessCount:   stats.AccessCount,
		LastAccessAt:  stats.LastAccessAt.Format("2006-01-02T15:04:05Z07:00"),
		FirstAccessAt: stats.FirstAccessAt.Format("2006-01-02T15:04:05Z07:00"),
		ReadBytes:     stats.ReadBytes,
		AverageReadHz: stats.AverageReadHz,
	}

	s.respondJSON(w, http.StatusOK, response)
}

// handleGetCostAnalysis returns cost analysis for tiered storage
// @Summary Get cost analysis
// @Description Returns cost analysis comparing hot and cold storage
// @Tags tiered-storage
// @Produce json
// @Success 200 {object} CostAnalysisResponse
// @Router /api/v1/tiers/cost-analysis [get]
func (s *Server) handleGetCostAnalysis(w http.ResponseWriter, r *http.Request) {
	if s.tierManager == nil {
		s.respondError(w, http.StatusServiceUnavailable, "tier management not enabled")
		return
	}

	analysis := s.tierManager.GetCostAnalysis()
	
	response := CostAnalysisResponse{
		HotStorageGB:           analysis["hot_storage_gb"].(float64),
		ColdStorageGB:          analysis["cold_storage_gb"].(float64),
		HotStorageCostMonthly:  analysis["hot_storage_cost_monthly"].(string),
		ColdStorageCostMonthly: analysis["cold_storage_cost_monthly"].(string),
		TotalCostMonthly:       analysis["total_cost_monthly"].(string),
		CostSavingsPct:         analysis["cost_savings_pct"].(string),
		RetrievalCostPerRestore: analysis["retrieval_cost_per_restore"].(string),
	}

	s.respondJSON(w, http.StatusOK, response)
}

// handleEvaluateTiers triggers tier evaluation
// @Summary Trigger tier evaluation
// @Description Manually trigger tier evaluation and promotion/demotion
// @Tags tiered-storage
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/v1/tiers/evaluate [post]
func (s *Server) handleEvaluateTiers(w http.ResponseWriter, r *http.Request) {
	if s.tierManager == nil {
		s.respondError(w, http.StatusServiceUnavailable, "tier management not enabled")
		return
	}

	if err := s.tierManager.EvaluateAndApplyTiers(r.Context()); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{
		"message": "tier evaluation completed successfully",
	})
}

// setupTierRoutes configures tier management routes
func (s *Server) setupTierRoutes(r chi.Router) {
	r.Get("/api/v1/tiers/stats", s.handleGetTierStats)
	r.Get("/api/v1/tiers/access/{segment_path}", s.handleGetAccessStats)
	r.Get("/api/v1/tiers/cost-analysis", s.handleGetCostAnalysis)
	r.Post("/api/v1/tiers/evaluate", s.handleEvaluateTiers)
}
