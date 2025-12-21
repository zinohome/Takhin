// Copyright 2025 Takhin Data, Inc.

package console

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents the overall health of the system
type HealthCheck struct {
	Status     HealthStatus               `json:"status"`
	Version    string                     `json:"version"`
	Uptime     string                     `json:"uptime"`
	Timestamp  time.Time                  `json:"timestamp"`
	Components map[string]ComponentHealth `json:"components"`
	SystemInfo SystemInfo                 `json:"system_info"`
}

// ComponentHealth represents the health of a single component
type ComponentHealth struct {
	Status  HealthStatus           `json:"status"`
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

// HealthChecker manages health checks for all components
type HealthChecker struct {
	startTime    time.Time
	version      string
	topicManager *topic.Manager
	coordinator  *coordinator.Coordinator
	mu           sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(version string, topicManager *topic.Manager, coord *coordinator.Coordinator) *HealthChecker {
	return &HealthChecker{
		startTime:    time.Now(),
		version:      version,
		topicManager: topicManager,
		coordinator:  coord,
	}
}

// Check performs a comprehensive health check
func (h *HealthChecker) Check() *HealthCheck {
	h.mu.RLock()
	defer h.mu.RUnlock()

	components := make(map[string]ComponentHealth)

	// Check topic manager
	topicHealth := h.checkTopicManager()
	components["topic_manager"] = topicHealth

	// Check coordinator
	coordHealth := h.checkCoordinator()
	components["coordinator"] = coordHealth

	// Determine overall status
	overallStatus := h.determineOverallStatus(components)

	return &HealthCheck{
		Status:     overallStatus,
		Version:    h.version,
		Uptime:     h.getUptime(),
		Timestamp:  time.Now(),
		Components: components,
		SystemInfo: h.getSystemInfo(),
	}
}

// checkTopicManager checks the health of the topic manager
func (h *HealthChecker) checkTopicManager() ComponentHealth {
	if h.topicManager == nil {
		return ComponentHealth{
			Status:  HealthStatusUnhealthy,
			Message: "topic manager not initialized",
		}
	}

	topics := h.topicManager.ListTopics()
	totalPartitions := 0
	totalSize := int64(0)

	for _, topicName := range topics {
		topic, exists := h.topicManager.GetTopic(topicName)
		if exists {
			totalPartitions += topic.NumPartitions()
			size, _ := topic.Size()
			totalSize += size
		}
	}

	return ComponentHealth{
		Status:  HealthStatusHealthy,
		Message: "operating normally",
		Details: map[string]interface{}{
			"num_topics":     len(topics),
			"num_partitions": totalPartitions,
			"total_size_mb":  float64(totalSize) / (1024 * 1024),
		},
	}
}

// checkCoordinator checks the health of the coordinator
func (h *HealthChecker) checkCoordinator() ComponentHealth {
	if h.coordinator == nil {
		return ComponentHealth{
			Status:  HealthStatusUnhealthy,
			Message: "coordinator not initialized",
		}
	}

	groups := h.coordinator.ListGroups()
	allGroups := h.coordinator.GetAllGroups()

	activeGroups := 0
	for _, group := range allGroups {
		if group.State != "Dead" && group.State != "Empty" {
			activeGroups++
		}
	}

	return ComponentHealth{
		Status:  HealthStatusHealthy,
		Message: "operating normally",
		Details: map[string]interface{}{
			"num_groups":        len(groups),
			"num_active_groups": activeGroups,
		},
	}
}

// determineOverallStatus determines the overall health status based on components
func (h *HealthChecker) determineOverallStatus(components map[string]ComponentHealth) HealthStatus {
	hasUnhealthy := false
	hasDegraded := false

	for _, component := range components {
		switch component.Status {
		case HealthStatusUnhealthy:
			hasUnhealthy = true
		case HealthStatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return HealthStatusUnhealthy
	}
	if hasDegraded {
		return HealthStatusDegraded
	}
	return HealthStatusHealthy
}

// getUptime returns the uptime as a human-readable string
func (h *HealthChecker) getUptime() string {
	duration := time.Since(h.startTime)

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
func (h *HealthChecker) getSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemInfo{
		GoVersion:     runtime.Version(),
		NumGoroutines: runtime.NumGoroutine(),
		NumCPU:        runtime.NumCPU(),
		MemoryMB:      float64(m.Alloc) / (1024 * 1024),
	}
}

// ReadinessCheck performs a readiness check (lighter than full health check)
func (h *HealthChecker) ReadinessCheck() bool {
	return h.topicManager != nil && h.coordinator != nil
}

// LivenessCheck performs a liveness check (minimal check)
func (h *HealthChecker) LivenessCheck() bool {
	return true // If we can respond, we're alive
}
