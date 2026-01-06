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
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestHandleProduceBatch(t *testing.T) {
	tests := []struct {
		name           string
		topicName      string
		request        ProduceRequest
		queryParams    map[string]string
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:      "produce single message",
			topicName: "test-topic",
			request: ProduceRequest{
				Records: []ProducerRecord{
					{
						Key:   "key1",
						Value: map[string]interface{}{"data": "value1"},
					},
				},
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp ProduceResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Len(t, resp.Offsets, 1)
				assert.Equal(t, int32(0), resp.Offsets[0].Partition)
				assert.GreaterOrEqual(t, resp.Offsets[0].Offset, int64(0))
			},
		},
		{
			name:      "produce multiple messages",
			topicName: "test-topic",
			request: ProduceRequest{
				Records: []ProducerRecord{
					{Value: map[string]interface{}{"data": "value1"}},
					{Value: map[string]interface{}{"data": "value2"}},
					{Value: map[string]interface{}{"data": "value3"}},
				},
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp ProduceResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Len(t, resp.Offsets, 3)
				for i, offset := range resp.Offsets {
					assert.Equal(t, int64(i), offset.Offset)
				}
			},
		},
		{
			name:      "produce with specific partition",
			topicName: "test-topic",
			request: ProduceRequest{
				Records: []ProducerRecord{
					{
						Partition: int32Ptr(1),
						Value:     "test-value",
					},
				},
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp ProduceResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Len(t, resp.Offsets, 1)
				assert.Equal(t, int32(1), resp.Offsets[0].Partition)
			},
		},
		{
			name:      "produce with string format",
			topicName: "test-topic",
			request: ProduceRequest{
				Records: []ProducerRecord{
					{
						Key:   "string-key",
						Value: "string-value",
					},
				},
			},
			queryParams: map[string]string{
				"key.format":   "string",
				"value.format": "string",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "produce async",
			topicName: "test-topic",
			request: ProduceRequest{
				Records: []ProducerRecord{
					{Value: "async-value"},
				},
			},
			queryParams: map[string]string{
				"async": "true",
			},
			expectedStatus: http.StatusAccepted,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp AsyncProduceResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)
				assert.NotEmpty(t, resp.RequestID)
				assert.Equal(t, "pending", resp.Status)
			},
		},
		{
			name:      "topic not found",
			topicName: "non-existent-topic",
			request: ProduceRequest{
				Records: []ProducerRecord{{Value: "test"}},
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "empty records",
			topicName: "test-topic",
			request: ProduceRequest{
				Records: []ProducerRecord{},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			server := setupTestServer(t)
			
			// Create topic if needed
			if tt.topicName == "test-topic" {
				err := server.topicManager.CreateTopic(tt.topicName, 2)
				require.NoError(t, err)
			}

			// Create request
			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/topics/"+tt.topicName+"/produce", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			// Add query parameters
			if len(tt.queryParams) > 0 {
				q := req.URL.Query()
				for k, v := range tt.queryParams {
					q.Add(k, v)
				}
				req.URL.RawQuery = q.Encode()
			}

			// Add route context
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("topic", tt.topicName)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()

			// Execute
			server.handleProduceBatch(rec, req)

			// Verify
			assert.Equal(t, tt.expectedStatus, rec.Code)
			
			if tt.validateResp != nil {
				tt.validateResp(t, rec)
			}
		})
	}
}

func TestHandleProduceStatus(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func() string
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "get pending status",
			setupRequest: func() string {
				requestID := generateRequestID()
				asyncRequestsMu.Lock()
				asyncRequests[requestID] = &asyncProduceRequest{
					id:        requestID,
					status:    "pending",
					createdAt: time.Now(),
				}
				asyncRequestsMu.Unlock()
				return requestID
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp ProduceStatusResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Equal(t, "pending", resp.Status)
			},
		},
		{
			name: "get completed status",
			setupRequest: func() string {
				requestID := generateRequestID()
				asyncRequestsMu.Lock()
				asyncRequests[requestID] = &asyncProduceRequest{
					id:     requestID,
					status: "completed",
					offsets: []ProducedRecordMetadata{
						{Partition: 0, Offset: 0},
					},
					createdAt: time.Now(),
				}
				asyncRequestsMu.Unlock()
				return requestID
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp ProduceStatusResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Equal(t, "completed", resp.Status)
				assert.Len(t, resp.Offsets, 1)
			},
		},
		{
			name: "request not found",
			setupRequest: func() string {
				return "non-existent-request-id"
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := setupTestServer(t)
			requestID := tt.setupRequest()

			req := httptest.NewRequest(http.MethodGet, "/api/produce/status/"+requestID, nil)
			
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("requestId", requestID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			server.handleProduceStatus(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			
			if tt.validateResp != nil {
				tt.validateResp(t, rec)
			}
		})
	}
}

func TestSerializeData(t *testing.T) {
	ctx := &producerContext{}

	tests := []struct {
		name        string
		data        interface{}
		format      DataFormat
		expectError bool
	}{
		{
			name:        "json object",
			data:        map[string]interface{}{"key": "value"},
			format:      FormatJSON,
			expectError: false,
		},
		{
			name:        "json string",
			data:        "test-string",
			format:      FormatJSON,
			expectError: false,
		},
		{
			name:        "string format",
			data:        "test-value",
			format:      FormatString,
			expectError: false,
		},
		{
			name:        "binary format valid base64",
			data:        "aGVsbG8=", // "hello" in base64
			format:      FormatBinary,
			expectError: false,
		},
		{
			name:        "binary format invalid base64",
			data:        "not-valid-base64!!!",
			format:      FormatBinary,
			expectError: true,
		},
		{
			name:        "avro without codec",
			data:        map[string]interface{}{"key": "value"},
			format:      FormatAvro,
			expectError: true,
		},
		{
			name:        "nil data",
			data:        nil,
			format:      FormatJSON,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ctx.serializeData(tt.data, tt.format, nil)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.data == nil {
					assert.Nil(t, result)
				} else {
					assert.NotNil(t, result)
				}
			}
		})
	}
}

func TestCompressData(t *testing.T) {
	testData := []byte("test data for compression")

	tests := []struct {
		name            string
		compressionType string
		expectError     bool
	}{
		{
			name:            "no compression",
			compressionType: "none",
			expectError:     false,
		},
		{
			name:            "gzip compression",
			compressionType: "gzip",
			expectError:     false,
		},
		{
			name:            "snappy compression",
			compressionType: "snappy",
			expectError:     false,
		},
		{
			name:            "lz4 compression",
			compressionType: "lz4",
			expectError:     false,
		},
		{
			name:            "zstd compression",
			compressionType: "zstd",
			expectError:     false,
		},
		{
			name:            "unsupported compression",
			compressionType: "invalid",
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &producerContext{compressionType: tt.compressionType}
			result, err := ctx.compressData(testData)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				
				if tt.compressionType != "none" {
					// Compressed data should exist
					assert.True(t, len(result) > 0)
				}
			}
		})
	}
}

func TestCleanupAsyncRequests(t *testing.T) {
	// Setup old and new requests
	oldRequestID := generateRequestID()
	newRequestID := generateRequestID()
	
	asyncRequestsMu.Lock()
	asyncRequests[oldRequestID] = &asyncProduceRequest{
		id:        oldRequestID,
		status:    "completed",
		createdAt: time.Now().Add(-1 * time.Hour), // 1 hour old
	}
	asyncRequests[newRequestID] = &asyncProduceRequest{
		id:        newRequestID,
		status:    "pending",
		createdAt: time.Now(),
	}
	asyncRequestsMu.Unlock()

	// Cleanup requests older than 30 minutes
	cleanupAsyncRequests(30 * time.Minute)

	asyncRequestsMu.RLock()
	_, oldExists := asyncRequests[oldRequestID]
	_, newExists := asyncRequests[newRequestID]
	asyncRequestsMu.RUnlock()

	assert.False(t, oldExists, "old request should be cleaned up")
	assert.True(t, newExists, "new request should remain")

	// Cleanup
	asyncRequestsMu.Lock()
	delete(asyncRequests, newRequestID)
	asyncRequestsMu.Unlock()
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateRequestID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2, "request IDs should be unique")
	assert.Contains(t, id1, "req_")
	assert.Contains(t, id2, "req_")
}

func TestProduceWithHeaders(t *testing.T) {
	server := setupTestServer(t)
	
	// Create topic
	topicName := "test-headers-topic"
	err := server.topicManager.CreateTopic(topicName, 1)
	require.NoError(t, err)

	// Produce with headers
	request := ProduceRequest{
		Records: []ProducerRecord{
			{
				Key:   "test-key",
				Value: "test-value",
				Headers: []Header{
					{Key: "header1", Value: "value1"},
					{Key: "header2", Value: "value2"},
				},
			},
		},
	}

	body, err := json.Marshal(request)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/topics/"+topicName+"/produce", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("topic", topicName)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	server.handleProduceBatch(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	
	var resp ProduceResponse
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Len(t, resp.Offsets, 1)
}

// Helper functions

func setupTestServer(t *testing.T) *Server {
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: t.TempDir(),
		},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, 1024*1024)
	coord := coordinator.NewCoordinator()

	return &Server{
		topicManager: topicMgr,
		coordinator:  coord,
		config:       cfg,
		logger:       nil,
	}
}

func int32Ptr(v int32) *int32 {
	return &v
}
