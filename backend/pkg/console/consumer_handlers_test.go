// Copyright 2025 Takhin Data, Inc.

package console

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func setupConsumerTestServer(t *testing.T) *Server {
	tmpDir := t.TempDir()

	topicMgr := topic.NewManager(tmpDir, 1024*1024)
	coord := coordinator.NewCoordinator()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tmpDir,
		},
	}

	server := NewServer(":0", topicMgr, coord, nil, AuthConfig{Enabled: false}, nil, cfg)

	// Create test topic with partitions
	err := topicMgr.CreateTopic("test-topic", 3)
	assert.NoError(t, err)

	// Write some test data
	testTopic, ok := topicMgr.GetTopic("test-topic")
	assert.True(t, ok)
	for i := 0; i < 3; i++ {
		partition := testTopic.Partitions[int32(i)]
		for j := 0; j < 10; j++ {
			_, err := partition.Append([]byte("key"), []byte("value"))
			assert.NoError(t, err)
		}
	}

	return server
}

func TestHandleSubscribe(t *testing.T) {
	server := setupConsumerTestServer(t)

	tests := []struct {
		name           string
		request        SubscribeRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name: "valid subscription",
			request: SubscribeRequest{
				GroupID:        "test-group",
				Topics:         []string{"test-topic"},
				SessionTimeout: 30000,
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "subscription with auto offset reset",
			request: SubscribeRequest{
				GroupID:         "test-group-2",
				Topics:          []string{"test-topic"},
				AutoOffsetReset: "earliest",
				SessionTimeout:  30000,
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "missing group id",
			request: SubscribeRequest{
				Topics: []string{"test-topic"},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "missing topics",
			request: SubscribeRequest{
				GroupID: "test-group",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/api/consumers/subscribe", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleSubscribe(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectError {
				var resp SubscribeResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.NotEmpty(t, resp.ConsumerID)
				assert.Equal(t, tt.request.GroupID, resp.GroupID)
				assert.Equal(t, tt.request.Topics, resp.Topics)
				assert.NotEmpty(t, resp.Assignment)
			}
		})
	}
}

func TestHandleConsume(t *testing.T) {
	server := setupConsumerTestServer(t)

	// First subscribe
	subscribeReq := SubscribeRequest{
		GroupID:         "test-group",
		Topics:          []string{"test-topic"},
		AutoOffsetReset: "earliest",
		SessionTimeout:  30000,
	}
	body, _ := json.Marshal(subscribeReq)
	req := httptest.NewRequest("POST", "/api/consumers/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.handleSubscribe(w, req)

	var subscribeResp SubscribeResponse
	json.NewDecoder(w.Body).Decode(&subscribeResp)
	consumerID := subscribeResp.ConsumerID

	tests := []struct {
		name           string
		consumerID     string
		request        ConsumeRequest
		expectedStatus int
		expectRecords  bool
	}{
		{
			name:       "consume with defaults",
			consumerID: consumerID,
			request: ConsumeRequest{
				MaxRecords: 5,
				TimeoutMs:  1000,
			},
			expectedStatus: http.StatusOK,
			expectRecords:  true,
		},
		{
			name:       "consume with max bytes",
			consumerID: consumerID,
			request: ConsumeRequest{
				MaxRecords:    10,
				TimeoutMs:     1000,
				MaxBytesTotal: 100,
			},
			expectedStatus: http.StatusOK,
			expectRecords:  true,
		},
		{
			name:           "invalid consumer id",
			consumerID:     "invalid-id",
			request:        ConsumeRequest{},
			expectedStatus: http.StatusNotFound,
			expectRecords:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/api/consumers/"+tt.consumerID+"/consume", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			
			// Use chi context to set URL params
			rctx := &testRequestContext{params: map[string]string{"consumer_id": tt.consumerID}}
			req = req.WithContext(rctx)

			w := httptest.NewRecorder()
			server.handleConsume(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectRecords {
				var resp ConsumeResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.NotEmpty(t, resp.Records)
				assert.NotZero(t, resp.Timestamp)

				// Verify record structure
				for _, record := range resp.Records {
					assert.Equal(t, "test-topic", record.Topic)
					assert.NotNil(t, record.Value)
				}
			}
		})
	}
}

func TestHandleCommit(t *testing.T) {
	server := setupConsumerTestServer(t)

	// Subscribe first
	subscribeReq := SubscribeRequest{
		GroupID:         "test-group",
		Topics:          []string{"test-topic"},
		AutoOffsetReset: "earliest",
	}
	body, _ := json.Marshal(subscribeReq)
	req := httptest.NewRequest("POST", "/api/consumers/subscribe", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.handleSubscribe(w, req)

	var subscribeResp SubscribeResponse
	json.NewDecoder(w.Body).Decode(&subscribeResp)
	consumerID := subscribeResp.ConsumerID

	tests := []struct {
		name           string
		consumerID     string
		request        CommitRequest
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:       "valid commit",
			consumerID: consumerID,
			request: CommitRequest{
				Offsets: map[string]map[int32]int64{
					"test-topic": {
						0: 5,
						1: 10,
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "invalid consumer",
			consumerID:     "invalid-id",
			request:        CommitRequest{},
			expectedStatus: http.StatusNotFound,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/api/consumers/"+tt.consumerID+"/commit", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rctx := &testRequestContext{params: map[string]string{"consumer_id": tt.consumerID}}
			req = req.WithContext(rctx)

			w := httptest.NewRecorder()
			server.handleCommit(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectSuccess {
				var resp CommitResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.True(t, resp.Success)
			}
		})
	}
}

func TestHandleSeek(t *testing.T) {
	server := setupConsumerTestServer(t)

	// Subscribe first
	subscribeReq := SubscribeRequest{
		GroupID: "test-group",
		Topics:  []string{"test-topic"},
	}
	body, _ := json.Marshal(subscribeReq)
	req := httptest.NewRequest("POST", "/api/consumers/subscribe", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.handleSubscribe(w, req)

	var subscribeResp SubscribeResponse
	json.NewDecoder(w.Body).Decode(&subscribeResp)
	consumerID := subscribeResp.ConsumerID

	tests := []struct {
		name           string
		consumerID     string
		request        SeekRequest
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:       "valid seek",
			consumerID: consumerID,
			request: SeekRequest{
				Topic:     "test-topic",
				Partition: 0,
				Offset:    5,
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:       "seek to beginning",
			consumerID: consumerID,
			request: SeekRequest{
				Topic:     "test-topic",
				Partition: 1,
				Offset:    0,
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name:           "invalid consumer",
			consumerID:     "invalid-id",
			request:        SeekRequest{},
			expectedStatus: http.StatusNotFound,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/api/consumers/"+tt.consumerID+"/seek", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rctx := &testRequestContext{params: map[string]string{"consumer_id": tt.consumerID}}
			req = req.WithContext(rctx)

			w := httptest.NewRecorder()
			server.handleSeek(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectSuccess {
				var resp SeekResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.True(t, resp.Success)
			}
		})
	}
}

func TestHandleAssignment(t *testing.T) {
	server := setupConsumerTestServer(t)

	// Subscribe first
	subscribeReq := SubscribeRequest{
		GroupID: "test-group",
		Topics:  []string{"test-topic"},
	}
	body, _ := json.Marshal(subscribeReq)
	req := httptest.NewRequest("POST", "/api/consumers/subscribe", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.handleSubscribe(w, req)

	var subscribeResp SubscribeResponse
	json.NewDecoder(w.Body).Decode(&subscribeResp)
	consumerID := subscribeResp.ConsumerID

	tests := []struct {
		name           string
		consumerID     string
		request        AssignmentRequest
		expectedStatus int
	}{
		{
			name:       "manual assignment",
			consumerID: consumerID,
			request: AssignmentRequest{
				Topics: map[string][]int32{
					"test-topic": {0, 1},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "multi-topic assignment",
			consumerID: consumerID,
			request: AssignmentRequest{
				Topics: map[string][]int32{
					"test-topic": {0, 1, 2},
				},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("PUT", "/api/consumers/"+tt.consumerID+"/assignment", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rctx := &testRequestContext{params: map[string]string{"consumer_id": tt.consumerID}}
			req = req.WithContext(rctx)

			w := httptest.NewRecorder()
			server.handleAssignment(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandlePosition(t *testing.T) {
	server := setupConsumerTestServer(t)

	// Subscribe and consume some records
	subscribeReq := SubscribeRequest{
		GroupID:         "test-group",
		Topics:          []string{"test-topic"},
		AutoOffsetReset: "earliest",
	}
	body, _ := json.Marshal(subscribeReq)
	req := httptest.NewRequest("POST", "/api/consumers/subscribe", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.handleSubscribe(w, req)

	var subscribeResp SubscribeResponse
	json.NewDecoder(w.Body).Decode(&subscribeResp)
	consumerID := subscribeResp.ConsumerID

	// Get position
	req = httptest.NewRequest("GET", "/api/consumers/"+consumerID+"/position", nil)
	rctx := &testRequestContext{params: map[string]string{"consumer_id": consumerID}}
	req = req.WithContext(rctx)

	w = httptest.NewRecorder()
	server.handlePosition(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp PositionResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Offsets)
	assert.Contains(t, resp.Offsets, "test-topic")
}

func TestHandleUnsubscribe(t *testing.T) {
	server := setupConsumerTestServer(t)

	// Subscribe first
	subscribeReq := SubscribeRequest{
		GroupID: "test-group",
		Topics:  []string{"test-topic"},
	}
	body, _ := json.Marshal(subscribeReq)
	req := httptest.NewRequest("POST", "/api/consumers/subscribe", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.handleSubscribe(w, req)

	var subscribeResp SubscribeResponse
	json.NewDecoder(w.Body).Decode(&subscribeResp)
	consumerID := subscribeResp.ConsumerID

	// Unsubscribe
	req = httptest.NewRequest("DELETE", "/api/consumers/"+consumerID, nil)
	rctx := &testRequestContext{params: map[string]string{"consumer_id": consumerID}}
	req = req.WithContext(rctx)

	w = httptest.NewRecorder()
	server.handleUnsubscribe(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify consumer is removed
	_, exists := server.consumerManager.GetConsumer(consumerID)
	assert.False(t, exists)
}

func TestConsumerSessionTimeout(t *testing.T) {
	t.Skip("Session timeout test requires 7s wait time - skipping in CI")
	
	server := setupConsumerTestServer(t)

	// Subscribe with short timeout
	subscribeReq := SubscribeRequest{
		GroupID:        "test-group",
		Topics:         []string{"test-topic"},
		SessionTimeout: 1000, // 1 second
	}
	body, _ := json.Marshal(subscribeReq)
	req := httptest.NewRequest("POST", "/api/consumers/subscribe", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.handleSubscribe(w, req)

	var subscribeResp SubscribeResponse
	json.NewDecoder(w.Body).Decode(&subscribeResp)
	consumerID := subscribeResp.ConsumerID

	// Verify consumer exists
	_, exists := server.consumerManager.GetConsumer(consumerID)
	assert.True(t, exists)

	// Wait for session timeout (1s + monitor interval 5s + buffer)
	time.Sleep(7 * time.Second)

	// Consumer should be removed
	_, exists = server.consumerManager.GetConsumer(consumerID)
	assert.False(t, exists)
}

func TestLongPolling(t *testing.T) {
	server := setupConsumerTestServer(t)

	// Subscribe
	subscribeReq := SubscribeRequest{
		GroupID:         "test-group",
		Topics:          []string{"test-topic"},
		AutoOffsetReset: "latest", // Start at end
	}
	body, _ := json.Marshal(subscribeReq)
	req := httptest.NewRequest("POST", "/api/consumers/subscribe", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.handleSubscribe(w, req)

	var subscribeResp SubscribeResponse
	json.NewDecoder(w.Body).Decode(&subscribeResp)
	consumerID := subscribeResp.ConsumerID

	// Consume with timeout (should return empty initially)
	consumeReq := ConsumeRequest{
		MaxRecords: 10,
		TimeoutMs:  500, // Short timeout for test
	}
	body, _ = json.Marshal(consumeReq)
	req = httptest.NewRequest("POST", "/api/consumers/"+consumerID+"/consume", bytes.NewReader(body))
	rctx := &testRequestContext{params: map[string]string{"consumer_id": consumerID}}
	req = req.WithContext(rctx)

	start := time.Now()
	w = httptest.NewRecorder()
	server.handleConsume(w, req)
	duration := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code)
	// Should wait approximately the timeout duration
	assert.True(t, duration >= 400*time.Millisecond, "should wait for timeout")

	var resp ConsumeResponse
	json.NewDecoder(w.Body).Decode(&resp)
	// May be empty or have records depending on timing
}

// Helper for chi URL params in tests
type testRequestContext struct {
	context.Context
	params map[string]string
}

func (c *testRequestContext) Value(key interface{}) interface{} {
	if key == chi.RouteCtxKey {
		rctx := chi.NewRouteContext()
		for k, v := range c.params {
			rctx.URLParams.Add(k, v)
		}
		return rctx
	}
	return c.Context.Value(key)
}
