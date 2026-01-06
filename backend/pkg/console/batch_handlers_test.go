// Copyright 2025 Takhin Data, Inc.

package console

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestBatchCreateTopics(t *testing.T) {
	tests := []struct {
		name           string
		request        BatchCreateTopicsRequest
		existingTopics []string
		expectedStatus int
		validateResult func(t *testing.T, result BatchOperationResult)
	}{
		{
			name: "successful batch create",
			request: BatchCreateTopicsRequest{
				Topics: []CreateTopicRequest{
					{Name: "batch-topic-1", Partitions: 3},
					{Name: "batch-topic-2", Partitions: 5},
					{Name: "batch-topic-3", Partitions: 1},
				},
			},
			expectedStatus: http.StatusOK,
			validateResult: func(t *testing.T, result BatchOperationResult) {
				assert.Equal(t, 3, result.TotalRequested)
				assert.Equal(t, 3, result.Successful)
				assert.Equal(t, 0, result.Failed)
				assert.Len(t, result.Results, 3)
				assert.Len(t, result.Errors, 0)
				
				for _, r := range result.Results {
					assert.True(t, r.Success)
					assert.Empty(t, r.Error)
				}
			},
		},
		{
			name: "topic already exists - rollback all",
			request: BatchCreateTopicsRequest{
				Topics: []CreateTopicRequest{
					{Name: "existing-topic", Partitions: 3},
					{Name: "new-topic", Partitions: 5},
				},
			},
			existingTopics: []string{"existing-topic"},
			expectedStatus: http.StatusBadRequest,
			validateResult: func(t *testing.T, result BatchOperationResult) {
				assert.Equal(t, 2, result.TotalRequested)
				assert.Equal(t, 0, result.Successful)
				assert.Equal(t, 1, result.Failed)
				assert.Greater(t, len(result.Errors), 0)
			},
		},
		{
			name: "empty topic name",
			request: BatchCreateTopicsRequest{
				Topics: []CreateTopicRequest{
					{Name: "", Partitions: 3},
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateResult: func(t *testing.T, result BatchOperationResult) {
				// Should fail validation before result is created
			},
		},
		{
			name: "invalid partitions",
			request: BatchCreateTopicsRequest{
				Topics: []CreateTopicRequest{
					{Name: "invalid-topic", Partitions: 0},
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateResult: func(t *testing.T, result BatchOperationResult) {
				// Should fail validation before result is created
			},
		},
		{
			name: "duplicate names in request",
			request: BatchCreateTopicsRequest{
				Topics: []CreateTopicRequest{
					{Name: "duplicate", Partitions: 3},
					{Name: "duplicate", Partitions: 5},
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateResult: func(t *testing.T, result BatchOperationResult) {
				// Should fail validation before result is created
			},
		},
		{
			name:           "empty request",
			request:        BatchCreateTopicsRequest{Topics: []CreateTopicRequest{}},
			expectedStatus: http.StatusBadRequest,
			validateResult: func(t *testing.T, result BatchOperationResult) {
				// Should fail validation before result is created
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tmpDir := t.TempDir()
			mgr := topic.NewManager(tmpDir, 1024*1024)
			coord := coordinator.NewCoordinator()
			coord.Start()
			
			// Create existing topics if specified
			for _, topicName := range tt.existingTopics {
				err := mgr.CreateTopic(topicName, 1)
				require.NoError(t, err)
			}

			server := NewServer(":8080", mgr, coord, nil, AuthConfig{Enabled: false})

			// Make request
			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/topics/batch", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			// Assert status
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Validate result if status is OK or mixed results
			if w.Code == http.StatusOK || w.Code == http.StatusBadRequest {
				var result BatchOperationResult
				if err := json.NewDecoder(w.Body).Decode(&result); err == nil {
					tt.validateResult(t, result)
					
					// Verify topics were created on success
					if result.Successful > 0 {
						for _, res := range result.Results {
							if res.Success {
								_, exists := mgr.GetTopic(res.Resource)
								assert.True(t, exists, "topic %s should exist", res.Resource)
							}
						}
					}
				} else if tt.expectedStatus == http.StatusBadRequest {
					// Validation errors return error message instead of result
					tt.validateResult(t, BatchOperationResult{})
				}
			}
		})
	}
}

func TestBatchDeleteTopics(t *testing.T) {
	tests := []struct {
		name           string
		request        BatchDeleteTopicsRequest
		existingTopics []string
		expectedStatus int
		validateResult func(t *testing.T, result BatchOperationResult)
	}{
		{
			name: "successful batch delete",
			request: BatchDeleteTopicsRequest{
				Topics: []string{"topic-1", "topic-2", "topic-3"},
			},
			existingTopics: []string{"topic-1", "topic-2", "topic-3"},
			expectedStatus: http.StatusOK,
			validateResult: func(t *testing.T, result BatchOperationResult) {
				assert.Equal(t, 3, result.TotalRequested)
				assert.Equal(t, 3, result.Successful)
				assert.Equal(t, 0, result.Failed)
				assert.Len(t, result.Results, 3)
				assert.Len(t, result.Errors, 0)
			},
		},
		{
			name: "topic not found - abort all",
			request: BatchDeleteTopicsRequest{
				Topics: []string{"topic-1", "non-existent"},
			},
			existingTopics: []string{"topic-1"},
			expectedStatus: http.StatusBadRequest,
			validateResult: func(t *testing.T, result BatchOperationResult) {
				assert.Equal(t, 2, result.TotalRequested)
				assert.Equal(t, 0, result.Successful)
				assert.Equal(t, 1, result.Failed)
				assert.Greater(t, len(result.Errors), 0)
			},
		},
		{
			name: "empty topic name in request",
			request: BatchDeleteTopicsRequest{
				Topics: []string{"topic-1", ""},
			},
			existingTopics: []string{"topic-1"},
			expectedStatus: http.StatusBadRequest,
			validateResult: func(t *testing.T, result BatchOperationResult) {
				// Should fail validation
			},
		},
		{
			name: "duplicate names in request",
			request: BatchDeleteTopicsRequest{
				Topics: []string{"topic-1", "topic-1"},
			},
			existingTopics: []string{"topic-1"},
			expectedStatus: http.StatusBadRequest,
			validateResult: func(t *testing.T, result BatchOperationResult) {
				// Should fail validation
			},
		},
		{
			name:           "empty request",
			request:        BatchDeleteTopicsRequest{Topics: []string{}},
			expectedStatus: http.StatusBadRequest,
			validateResult: func(t *testing.T, result BatchOperationResult) {
				// Should fail validation
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tmpDir := t.TempDir()
			mgr := topic.NewManager(tmpDir, 1024*1024)
			coord := coordinator.NewCoordinator()
			coord.Start()
			
			// Create existing topics
			for _, topicName := range tt.existingTopics {
				err := mgr.CreateTopic(topicName, 1)
				require.NoError(t, err)
			}

			server := NewServer(":8080", mgr, coord, nil, AuthConfig{Enabled: false})

			// Make request
			body, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodDelete, "/api/topics/batch", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			// Assert status
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Validate result
			if w.Code == http.StatusOK || w.Code == http.StatusBadRequest {
				var result BatchOperationResult
				if err := json.NewDecoder(w.Body).Decode(&result); err == nil {
					tt.validateResult(t, result)
					
					// Verify topics were deleted on success
					if result.Successful > 0 {
						for _, res := range result.Results {
							if res.Success {
								_, exists := mgr.GetTopic(res.Resource)
								assert.False(t, exists, "topic %s should not exist", res.Resource)
							}
						}
					}
				} else if tt.expectedStatus == http.StatusBadRequest {
					// Validation errors return error message
					tt.validateResult(t, BatchOperationResult{})
				}
			}
		})
	}
}

func TestBatchCreateRollback(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := topic.NewManager(tmpDir, 1024*1024)
	coord := coordinator.NewCoordinator()
	coord.Start()
	
	// Create one existing topic
	err := mgr.CreateTopic("existing", 1)
	require.NoError(t, err)

	server := NewServer(":8080", mgr, coord, nil, AuthConfig{Enabled: false})

	// Attempt batch create where one topic already exists
	request := BatchCreateTopicsRequest{
		Topics: []CreateTopicRequest{
			{Name: "new-1", Partitions: 3},
			{Name: "existing", Partitions: 5},
			{Name: "new-2", Partitions: 2},
		},
	}

	body, err := json.Marshal(request)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/topics/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	// Should fail with bad request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Verify no new topics were created
	_, exists := mgr.GetTopic("new-1")
	assert.False(t, exists, "new-1 should not exist after rollback")
	
	_, exists = mgr.GetTopic("new-2")
	assert.False(t, exists, "new-2 should not exist after rollback")
	
	// Existing topic should still be there
	_, exists = mgr.GetTopic("existing")
	assert.True(t, exists, "existing topic should still exist")
}

func TestBatchDeletePartialFailure(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := topic.NewManager(tmpDir, 1024*1024)
	coord := coordinator.NewCoordinator()
	coord.Start()
	
	// Create some topics
	err := mgr.CreateTopic("topic-1", 1)
	require.NoError(t, err)
	err = mgr.CreateTopic("topic-2", 1)
	require.NoError(t, err)

	server := NewServer(":8080", mgr, coord, nil, AuthConfig{Enabled: false})

	// Try to delete existing and non-existing topics
	request := BatchDeleteTopicsRequest{
		Topics: []string{"topic-1", "non-existent", "topic-2"},
	}

	body, err := json.Marshal(request)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/api/topics/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	// Should fail with bad request (abort on first non-existent topic)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var result BatchOperationResult
	err = json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, 3, result.TotalRequested)
	assert.Equal(t, 0, result.Successful)
	assert.Greater(t, result.Failed, 0)

	// Original topics should still exist
	_, exists := mgr.GetTopic("topic-1")
	assert.True(t, exists, "topic-1 should still exist")
	
	_, exists = mgr.GetTopic("topic-2")
	assert.True(t, exists, "topic-2 should still exist")
}
