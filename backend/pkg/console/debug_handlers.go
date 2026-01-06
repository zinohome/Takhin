// Copyright 2025 Takhin Data, Inc.

package console

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/takhin-data/takhin/pkg/debug"
)

// DebugBundleRequest represents a debug bundle generation request
type DebugBundleRequest struct {
	IncludeLogs      bool  `json:"include_logs"`
	IncludeConfig    bool  `json:"include_config"`
	IncludeMetrics   bool  `json:"include_metrics"`
	IncludeSystem    bool  `json:"include_system"`
	IncludeStorage   bool  `json:"include_storage"`
	LogsMaxSizeMB    int64 `json:"logs_max_size_mb,omitempty"`
	LogsSinceHours   int   `json:"logs_since_hours,omitempty"`
	StorageMaxSizeMB int64 `json:"storage_max_size_mb,omitempty"`
}

// DebugBundleResponse represents the response after generating a debug bundle
type DebugBundleResponse struct {
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

// handleGenerateDebugBundle godoc
// @Summary      Generate debug bundle
// @Description  Collect system diagnostics and logs into a compressed bundle
// @Tags         Debug
// @Accept       json
// @Produce      json
// @Param        request  body      DebugBundleRequest  true  "Debug bundle options"
// @Success      200      {object}  DebugBundleResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Router       /debug/bundle [post]
func (s *Server) handleGenerateDebugBundle(w http.ResponseWriter, r *http.Request) {
	var req DebugBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.LogsMaxSizeMB == 0 {
		req.LogsMaxSizeMB = 100
	}
	if req.LogsSinceHours == 0 {
		req.LogsSinceHours = 24
	}
	if req.StorageMaxSizeMB == 0 {
		req.StorageMaxSizeMB = 50
	}

	opts := &debug.BundleOptions{
		IncludeLogs:      req.IncludeLogs,
		IncludeConfig:    req.IncludeConfig,
		IncludeMetrics:   req.IncludeMetrics,
		IncludeSystem:    req.IncludeSystem,
		IncludeStorage:   req.IncludeStorage,
		LogsMaxSizeMB:    req.LogsMaxSizeMB,
		LogsSince:        time.Duration(req.LogsSinceHours) * time.Hour,
		StorageMaxSizeMB: req.StorageMaxSizeMB,
	}

	bundle := debug.NewBundle(s.config, s.logger)
	path, err := bundle.Generate(r.Context(), opts)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to generate debug bundle")
		return
	}

	s.respondJSON(w, http.StatusOK, DebugBundleResponse{
		Path:      path,
		CreatedAt: time.Now(),
	})
}

// handleDownloadDebugBundle godoc
// @Summary      Download debug bundle
// @Description  Download a previously generated debug bundle
// @Tags         Debug
// @Produce      application/gzip
// @Param        path  query     string  true  "Path to the debug bundle"
// @Success      200   {file}    binary
// @Failure      400   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /debug/bundle/download [get]
func (s *Server) handleDownloadDebugBundle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		s.respondError(w, http.StatusBadRequest, "path parameter is required")
		return
	}

	http.ServeFile(w, r, path)
}

// handleDebugSystemInfo godoc
// @Summary      Get system information
// @Description  Retrieve current system diagnostic information
// @Tags         Debug
// @Produce      json
// @Success      200  {object}  debug.SystemInfo
// @Failure      500  {object}  ErrorResponse
// @Router       /debug/system [get]
func (s *Server) handleDebugSystemInfo(w http.ResponseWriter, r *http.Request) {
	bundle := debug.NewBundle(s.config, s.logger)

	tmpDir := "/tmp/takhin-debug-temp"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to create temp directory")
		return
	}
	defer os.RemoveAll(tmpDir)

	if err := bundle.CollectSystemInfo(r.Context(), tmpDir); err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to collect system info")
		return
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "system-info.json"))
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to read system info")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
