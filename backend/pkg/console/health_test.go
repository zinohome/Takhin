// Copyright 2025 Takhin Data, Inc.

package console

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestHealthChecker_Basic(t *testing.T) {
	// Setup
	dataDir := t.TempDir()
	mgr := topic.NewManager(dataDir, 1024*1024)
	coord := coordinator.NewCoordinator()

	checker := NewHealthChecker("1.0.0-test", mgr, coord)

	// Initial check
	health := checker.Check()
	assert.Equal(t, HealthStatusHealthy, health.Status)
	assert.Equal(t, "1.0.0-test", health.Version)
	assert.NotEmpty(t, health.Uptime)
	assert.NotZero(t, health.Timestamp)

	// Check components
	assert.Contains(t, health.Components, "topic_manager")
	assert.Contains(t, health.Components, "coordinator")

	topicHealth := health.Components["topic_manager"]
	assert.Equal(t, HealthStatusHealthy, topicHealth.Status)
	assert.Contains(t, topicHealth.Details, "num_topics")
	assert.Equal(t, 0, topicHealth.Details["num_topics"])

	coordHealth := health.Components["coordinator"]
	assert.Equal(t, HealthStatusHealthy, coordHealth.Status)
	assert.Contains(t, coordHealth.Details, "num_groups")

	// Check system info
	assert.NotEmpty(t, health.SystemInfo.GoVersion)
	assert.Greater(t, health.SystemInfo.NumGoroutines, 0)
	assert.Greater(t, health.SystemInfo.NumCPU, 0)
	assert.Greater(t, health.SystemInfo.MemoryMB, 0.0)
}

func TestHealthChecker_WithTopics(t *testing.T) {
	// Setup
	dataDir := t.TempDir()
	mgr := topic.NewManager(dataDir, 1024*1024)
	coord := coordinator.NewCoordinator()

	// Create some topics
	err := mgr.CreateTopic("test-topic-1", 3)
	require.NoError(t, err)
	err = mgr.CreateTopic("test-topic-2", 2)
	require.NoError(t, err)

	checker := NewHealthChecker("1.0.0", mgr, coord)
	health := checker.Check()

	assert.Equal(t, HealthStatusHealthy, health.Status)

	topicHealth := health.Components["topic_manager"]
	assert.Equal(t, HealthStatusHealthy, topicHealth.Status)
	assert.Equal(t, 2, topicHealth.Details["num_topics"])
	assert.Equal(t, 5, topicHealth.Details["num_partitions"])
}

func TestHealthChecker_NilTopicManager(t *testing.T) {
	coord := coordinator.NewCoordinator()
	checker := NewHealthChecker("1.0.0", nil, coord)

	health := checker.Check()

	// Should be unhealthy due to nil topic manager
	assert.Equal(t, HealthStatusUnhealthy, health.Status)

	topicHealth := health.Components["topic_manager"]
	assert.Equal(t, HealthStatusUnhealthy, topicHealth.Status)
	assert.Contains(t, topicHealth.Message, "not initialized")
}

func TestHealthChecker_NilCoordinator(t *testing.T) {
	dataDir := t.TempDir()
	mgr := topic.NewManager(dataDir, 1024*1024)

	checker := NewHealthChecker("1.0.0", mgr, nil)
	health := checker.Check()

	// Should be unhealthy due to nil coordinator
	assert.Equal(t, HealthStatusUnhealthy, health.Status)

	coordHealth := health.Components["coordinator"]
	assert.Equal(t, HealthStatusUnhealthy, coordHealth.Status)
	assert.Contains(t, coordHealth.Message, "not initialized")
}

func TestHealthChecker_Uptime(t *testing.T) {
	dataDir := t.TempDir()
	mgr := topic.NewManager(dataDir, 1024*1024)
	coord := coordinator.NewCoordinator()

	checker := NewHealthChecker("1.0.0", mgr, coord)

	// Wait a bit to ensure uptime is measurable
	time.Sleep(1100 * time.Millisecond)

	// Check uptime format
	health1 := checker.Check()
	assert.Contains(t, health1.Uptime, "s") // Should contain seconds
	// Uptime should be at least 1 second
	assert.True(t, len(health1.Uptime) >= 2, "uptime should be at least '1s'")

	// Wait a bit and check again - uptime should increase
	prevUptime := health1.Uptime
	time.Sleep(1100 * time.Millisecond)
	health2 := checker.Check()
	// Second uptime should be different (and larger)
	assert.NotEqual(t, prevUptime, health2.Uptime)
}

func TestHealthChecker_ReadinessCheck(t *testing.T) {
	tests := []struct {
		name          string
		topicManager  *topic.Manager
		coordinator   *coordinator.Coordinator
		expectedReady bool
	}{
		{
			name:          "both initialized",
			topicManager:  topic.NewManager(t.TempDir(), 1024*1024),
			coordinator:   coordinator.NewCoordinator(),
			expectedReady: true,
		},
		{
			name:          "nil topic manager",
			topicManager:  nil,
			coordinator:   coordinator.NewCoordinator(),
			expectedReady: false,
		},
		{
			name:          "nil coordinator",
			topicManager:  topic.NewManager(t.TempDir(), 1024*1024),
			coordinator:   nil,
			expectedReady: false,
		},
		{
			name:          "both nil",
			topicManager:  nil,
			coordinator:   nil,
			expectedReady: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewHealthChecker("1.0.0", tt.topicManager, tt.coordinator)
			ready := checker.ReadinessCheck()
			assert.Equal(t, tt.expectedReady, ready)
		})
	}
}

func TestHealthChecker_LivenessCheck(t *testing.T) {
	dataDir := t.TempDir()
	mgr := topic.NewManager(dataDir, 1024*1024)
	coord := coordinator.NewCoordinator()

	checker := NewHealthChecker("1.0.0", mgr, coord)

	// Liveness should always be true if we can call it
	assert.True(t, checker.LivenessCheck())
}

func TestHealthChecker_SystemInfo(t *testing.T) {
	dataDir := t.TempDir()
	mgr := topic.NewManager(dataDir, 1024*1024)
	coord := coordinator.NewCoordinator()

	checker := NewHealthChecker("1.0.0", mgr, coord)
	health := checker.Check()

	// Validate system info
	sysInfo := health.SystemInfo
	assert.NotEmpty(t, sysInfo.GoVersion)
	assert.Contains(t, sysInfo.GoVersion, "go")
	assert.Greater(t, sysInfo.NumGoroutines, 0)
	assert.Greater(t, sysInfo.NumCPU, 0)
	assert.Greater(t, sysInfo.MemoryMB, 0.0)
}

func TestHealthChecker_ConcurrentAccess(t *testing.T) {
	dataDir := t.TempDir()
	mgr := topic.NewManager(dataDir, 1024*1024)
	coord := coordinator.NewCoordinator()

	checker := NewHealthChecker("1.0.0", mgr, coord)

	// Run multiple health checks concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				health := checker.Check()
				assert.NotNil(t, health)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
