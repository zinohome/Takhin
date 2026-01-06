// Copyright 2025 Takhin Data, Inc.

package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestChecker_Basic(t *testing.T) {
	dataDir := t.TempDir()
	mgr := topic.NewManager(dataDir, 1024*1024)

	checker := NewChecker("1.0.0-test", mgr)

	health := checker.Check()
	assert.Equal(t, StatusHealthy, health.Status)
	assert.Equal(t, "1.0.0-test", health.Version)
	assert.NotEmpty(t, health.Uptime)
	assert.NotZero(t, health.Timestamp)

	// Check components
	assert.Contains(t, health.Components, "storage")

	storageHealth := health.Components["storage"]
	assert.Equal(t, StatusHealthy, storageHealth.Status)
	assert.Contains(t, storageHealth.Details, "num_topics")
	assert.Equal(t, 0, storageHealth.Details["num_topics"])

	// Check system info
	assert.NotEmpty(t, health.SystemInfo.GoVersion)
	assert.Greater(t, health.SystemInfo.NumGoroutines, 0)
	assert.Greater(t, health.SystemInfo.NumCPU, 0)
	assert.Greater(t, health.SystemInfo.MemoryMB, 0.0)
}

func TestChecker_WithTopics(t *testing.T) {
	dataDir := t.TempDir()
	mgr := topic.NewManager(dataDir, 1024*1024)

	err := mgr.CreateTopic("test-topic-1", 3)
	require.NoError(t, err)
	err = mgr.CreateTopic("test-topic-2", 2)
	require.NoError(t, err)

	checker := NewChecker("1.0.0", mgr)
	health := checker.Check()

	assert.Equal(t, StatusHealthy, health.Status)

	storageHealth := health.Components["storage"]
	assert.Equal(t, StatusHealthy, storageHealth.Status)
	assert.Equal(t, 2, storageHealth.Details["num_topics"])
	assert.Equal(t, 5, storageHealth.Details["num_partitions"])
}

func TestChecker_NilTopicManager(t *testing.T) {
	checker := NewChecker("1.0.0", nil)
	health := checker.Check()

	assert.Equal(t, StatusUnhealthy, health.Status)

	storageHealth := health.Components["storage"]
	assert.Equal(t, StatusUnhealthy, storageHealth.Status)
	assert.Contains(t, storageHealth.Message, "not initialized")
}

func TestChecker_Uptime(t *testing.T) {
	dataDir := t.TempDir()
	mgr := topic.NewManager(dataDir, 1024*1024)

	checker := NewChecker("1.0.0", mgr)

	time.Sleep(1100 * time.Millisecond)

	health1 := checker.Check()
	assert.Contains(t, health1.Uptime, "s")
	assert.True(t, len(health1.Uptime) >= 2)

	prevUptime := health1.Uptime
	time.Sleep(1100 * time.Millisecond)
	health2 := checker.Check()
	assert.NotEqual(t, prevUptime, health2.Uptime)
}

func TestChecker_ReadinessCheck(t *testing.T) {
	tests := []struct {
		name          string
		topicManager  *topic.Manager
		expectedReady bool
	}{
		{
			name:          "initialized",
			topicManager:  topic.NewManager(t.TempDir(), 1024*1024),
			expectedReady: true,
		},
		{
			name:          "not initialized",
			topicManager:  nil,
			expectedReady: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewChecker("1.0.0", tt.topicManager)
			ready := checker.ReadinessCheck()
			assert.Equal(t, tt.expectedReady, ready)
		})
	}
}

func TestChecker_LivenessCheck(t *testing.T) {
	dataDir := t.TempDir()
	mgr := topic.NewManager(dataDir, 1024*1024)

	checker := NewChecker("1.0.0", mgr)
	assert.True(t, checker.LivenessCheck())
}

func TestChecker_ConcurrentAccess(t *testing.T) {
	dataDir := t.TempDir()
	mgr := topic.NewManager(dataDir, 1024*1024)

	checker := NewChecker("1.0.0", mgr)

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

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestServer_HandleHealth(t *testing.T) {
	dataDir := t.TempDir()
	mgr := topic.NewManager(dataDir, 1024*1024)
	checker := NewChecker("1.0.0", mgr)
	server := NewServer(":0", checker)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var health Check
	err := json.NewDecoder(w.Body).Decode(&health)
	require.NoError(t, err)
	assert.Equal(t, StatusHealthy, health.Status)
	assert.Equal(t, "1.0.0", health.Version)
}

func TestServer_HandleHealthUnhealthy(t *testing.T) {
	checker := NewChecker("1.0.0", nil)
	server := NewServer(":0", checker)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var health Check
	err := json.NewDecoder(w.Body).Decode(&health)
	require.NoError(t, err)
	assert.Equal(t, StatusUnhealthy, health.Status)
}

func TestServer_HandleReadiness(t *testing.T) {
	tests := []struct {
		name           string
		topicManager   *topic.Manager
		expectedStatus int
		expectedReady  bool
	}{
		{
			name:           "ready",
			topicManager:   topic.NewManager(t.TempDir(), 1024*1024),
			expectedStatus: http.StatusOK,
			expectedReady:  true,
		},
		{
			name:           "not ready",
			topicManager:   nil,
			expectedStatus: http.StatusServiceUnavailable,
			expectedReady:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewChecker("1.0.0", tt.topicManager)
			server := NewServer(":0", checker)

			req := httptest.NewRequest("GET", "/health/ready", nil)
			w := httptest.NewRecorder()

			server.handleReadiness(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]bool
			err := json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedReady, response["ready"])
		})
	}
}

func TestServer_HandleLiveness(t *testing.T) {
	dataDir := t.TempDir()
	mgr := topic.NewManager(dataDir, 1024*1024)
	checker := NewChecker("1.0.0", mgr)
	server := NewServer(":0", checker)

	req := httptest.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()

	server.handleLiveness(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]bool
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.True(t, response["alive"])
}

func TestServer_StartStop(t *testing.T) {
	dataDir := t.TempDir()
	mgr := topic.NewManager(dataDir, 1024*1024)
	checker := NewChecker("1.0.0", mgr)
	server := NewServer("localhost:0", checker)

	err := server.Start()
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = server.Stop()
	assert.NoError(t, err)
}
