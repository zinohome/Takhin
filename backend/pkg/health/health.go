// Copyright 2025 Takhin Data, Inc.

package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/takhin-data/takhin/pkg/logger"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

// Status represents the health status of a component
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// Check represents the overall health of the system
type Check struct {
	Status     Status                  `json:"status"`
	Version    string                  `json:"version"`
	Uptime     string                  `json:"uptime"`
	Timestamp  time.Time               `json:"timestamp"`
	Components map[string]Component    `json:"components"`
	SystemInfo SystemInfo              `json:"system_info"`
}

// Component represents the health of a single component
type Component struct {
	Status  Status                 `json:"status"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// SystemInfo contains system-level information
type SystemInfo struct {
	GoVersion     string  `json:"go_version"`
	NumGoroutines int     `json:"num_goroutines"`
	NumCPU        int     `json:"num_cpu"`
	MemoryMB      float64 `json:"memory_mb"`
}

// Checker manages health checks
type Checker struct {
	startTime    time.Time
	version      string
	topicManager *topic.Manager
	logger       logger.Logger
	mu           sync.RWMutex
}

// NewChecker creates a new health checker
func NewChecker(version string, topicManager *topic.Manager) *Checker {
	return &Checker{
		startTime:    time.Now(),
		version:      version,
		topicManager: topicManager,
		logger:       *logger.Default().WithComponent("health"),
	}
}

// Check performs a comprehensive health check
func (c *Checker) Check() *Check {
	c.mu.RLock()
	defer c.mu.RUnlock()

	components := make(map[string]Component)

	// Check topic manager
	topicHealth := c.checkTopicManager()
	components["storage"] = topicHealth

	// Determine overall status
	overallStatus := c.determineOverallStatus(components)

	return &Check{
		Status:     overallStatus,
		Version:    c.version,
		Uptime:     c.getUptime(),
		Timestamp:  time.Now(),
		Components: components,
		SystemInfo: c.getSystemInfo(),
	}
}

// checkTopicManager checks the health of the topic manager
func (c *Checker) checkTopicManager() Component {
	if c.topicManager == nil {
		return Component{
			Status:  StatusUnhealthy,
			Message: "storage not initialized",
		}
	}

	topics := c.topicManager.ListTopics()
	totalPartitions := 0
	totalSize := int64(0)

	for _, topicName := range topics {
		topic, exists := c.topicManager.GetTopic(topicName)
		if exists {
			totalPartitions += topic.NumPartitions()
			size, _ := topic.Size()
			totalSize += size
		}
	}

	return Component{
		Status:  StatusHealthy,
		Message: "operating normally",
		Details: map[string]interface{}{
			"num_topics":     len(topics),
			"num_partitions": totalPartitions,
			"total_size_mb":  float64(totalSize) / (1024 * 1024),
		},
	}
}

// determineOverallStatus determines the overall health status
func (c *Checker) determineOverallStatus(components map[string]Component) Status {
	hasUnhealthy := false
	hasDegraded := false

	for _, component := range components {
		switch component.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return StatusUnhealthy
	}
	if hasDegraded {
		return StatusDegraded
	}
	return StatusHealthy
}

// getUptime returns the uptime as a human-readable string
func (c *Checker) getUptime() string {
	duration := time.Since(c.startTime)

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// getSystemInfo returns system-level information
func (c *Checker) getSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemInfo{
		GoVersion:     runtime.Version(),
		NumGoroutines: runtime.NumGoroutine(),
		NumCPU:        runtime.NumCPU(),
		MemoryMB:      float64(m.Alloc) / (1024 * 1024),
	}
}

// ReadinessCheck performs a readiness check
func (c *Checker) ReadinessCheck() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.topicManager != nil
}

// LivenessCheck performs a liveness check
func (c *Checker) LivenessCheck() bool {
	return true // If we can respond, we're alive
}

// Server provides HTTP endpoints for health checks
type Server struct {
	checker *Checker
	server  *http.Server
	logger  logger.Logger
}

// NewServer creates a new health check HTTP server
func NewServer(addr string, checker *Checker) *Server {
	mux := http.NewServeMux()

	s := &Server{
		checker: checker,
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
		logger: *logger.Default().WithComponent("health-server"),
	}

	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/health/ready", s.handleReadiness)
	mux.HandleFunc("/health/live", s.handleLiveness)

	return s
}

// Start starts the health check server
func (s *Server) Start() error {
	s.logger.Info("starting health check server", "address", s.server.Addr)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("health server error", "error", err)
		}
	}()

	return nil
}

// Stop stops the health check server
func (s *Server) Stop() error {
	s.logger.Info("stopping health check server")
	return s.server.Close()
}

// handleHealth handles comprehensive health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := s.checker.Check()

	statusCode := http.StatusOK
	if health.Status == StatusDegraded {
		statusCode = http.StatusOK // 200 for degraded but functional
	} else if health.Status == StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

// handleReadiness handles Kubernetes readiness probe requests
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	ready := s.checker.ReadinessCheck()

	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]bool{"ready": ready})
}

// handleLiveness handles Kubernetes liveness probe requests
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	alive := s.checker.LivenessCheck()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"alive": alive})
}
