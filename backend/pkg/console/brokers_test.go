// Copyright 2025 Takhin Data, Inc.

package console

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestHandleListBrokers(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			AdvertisedHost: "localhost",
			AdvertisedPort: 9092,
		},
		Server: config.ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Storage: config.StorageConfig{
			DataDir: tmpDir,
		},
	}

	topicMgr := topic.NewManager(tmpDir, 1024*1024)
	coord := coordinator.NewCoordinator()

	server := NewServer(":8080", topicMgr, coord, nil, AuthConfig{Enabled: false}, nil, cfg)

	// Create some test topics
	err := topicMgr.CreateTopic("test-topic-1", 3)
	assert.NoError(t, err)
	err = topicMgr.CreateTopic("test-topic-2", 2)
	assert.NoError(t, err)

	// Test
	req := httptest.NewRequest("GET", "/api/brokers", nil)
	w := httptest.NewRecorder()
	server.handleListBrokers(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var brokers []BrokerInfo
	err = json.NewDecoder(w.Body).Decode(&brokers)
	assert.NoError(t, err)
	assert.Len(t, brokers, 1)
	assert.Equal(t, int32(1), brokers[0].ID)
	assert.Equal(t, "localhost", brokers[0].Host)
	assert.Equal(t, int32(9092), brokers[0].Port)
	assert.True(t, brokers[0].IsController)
	assert.Equal(t, 2, brokers[0].TopicCount)
	assert.Equal(t, 5, brokers[0].PartitionCount) // 3 + 2 partitions
	assert.Equal(t, "online", brokers[0].Status)
}

func TestHandleGetBroker(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			AdvertisedHost: "localhost",
			AdvertisedPort: 9092,
		},
		Server: config.ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Storage: config.StorageConfig{
			DataDir: tmpDir,
		},
	}

	topicMgr := topic.NewManager(tmpDir, 1024*1024)
	coord := coordinator.NewCoordinator()

	server := NewServer(":8080", topicMgr, coord, nil, AuthConfig{Enabled: false}, nil, cfg)

	// Create test topic
	err := topicMgr.CreateTopic("test-topic", 4)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		brokerID       string
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:           "Valid broker ID",
			brokerID:       "1",
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var broker BrokerInfo
				err := json.Unmarshal(body, &broker)
				assert.NoError(t, err)
				assert.Equal(t, int32(1), broker.ID)
				assert.Equal(t, "localhost", broker.Host)
				assert.Equal(t, int32(9092), broker.Port)
				assert.True(t, broker.IsController)
				assert.Equal(t, 1, broker.TopicCount)
				assert.Equal(t, 4, broker.PartitionCount)
			},
		},
		{
			name:           "Invalid broker ID",
			brokerID:       "2",
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body []byte) {
				var errResp map[string]string
				err := json.Unmarshal(body, &errResp)
				assert.NoError(t, err)
				assert.Contains(t, errResp["error"], "broker not found")
			},
		},
		{
			name:           "Non-numeric broker ID",
			brokerID:       "abc",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body []byte) {
				var errResp map[string]string
				err := json.Unmarshal(body, &errResp)
				assert.NoError(t, err)
				assert.Contains(t, errResp["error"], "invalid broker ID")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/brokers/"+tt.brokerID, nil)
			w := httptest.NewRecorder()

			// Setup chi context
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.brokerID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			server.handleGetBroker(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkBody(t, w.Body.Bytes())
		})
	}
}

func TestHandleGetClusterStats(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID: 1,
		},
		Storage: config.StorageConfig{
			DataDir: tmpDir,
		},
	}

	topicMgr := topic.NewManager(tmpDir, 1024*1024)
	coord := coordinator.NewCoordinator()

	server := NewServer(":8080", topicMgr, coord, nil, AuthConfig{Enabled: false}, nil, cfg)

	// Create test topics and add messages
	err := topicMgr.CreateTopic("test-topic-1", 2)
	assert.NoError(t, err)
	err = topicMgr.CreateTopic("test-topic-2", 3)
	assert.NoError(t, err)

	// Add some messages to test-topic-1
	testTopic, exists := topicMgr.GetTopic("test-topic-1")
	assert.True(t, exists)
	for i := 0; i < 10; i++ {
		_, err := testTopic.Append(0, []byte("key"), []byte("value"))
		assert.NoError(t, err)
	}

	// Test
	req := httptest.NewRequest("GET", "/api/cluster/stats", nil)
	w := httptest.NewRecorder()
	server.handleGetClusterStats(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var stats ClusterStats
	err = json.NewDecoder(w.Body).Decode(&stats)
	assert.NoError(t, err)
	assert.Equal(t, 1, stats.BrokerCount)
	assert.Equal(t, 2, stats.TopicCount)
	assert.Equal(t, 5, stats.PartitionCount) // 2 + 3 partitions
	assert.Equal(t, int64(10), stats.TotalMessages)
	assert.Greater(t, stats.TotalSizeBytes, int64(0))
	assert.Equal(t, 1, stats.ReplicationFactor)
}
