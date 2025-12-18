// Copyright 2025 Takhin Data, Inc.

package console

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestConsoleAPI(t *testing.T) {
	dir := t.TempDir()
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	coord := coordinator.NewCoordinator()
	coord.Start()

	authConfig := AuthConfig{Enabled: false}
	server := NewServer(":8080", topicMgr, coord, authConfig)

	t.Run("health check", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("create and list topics", func(t *testing.T) {
		reqBody := CreateTopicRequest{
			Name:       "test-topic",
			Partitions: 3,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/topics", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// List topics
		req = httptest.NewRequest("GET", "/api/topics", nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("consumer groups", func(t *testing.T) {
		// Create a consumer group with offset commit
		testGroup := coord.GetOrCreateGroup("test-group", "consumer")
		testGroup.CommitOffset("test-topic", 0, 100, "test-metadata")

		// List consumer groups
		req := httptest.NewRequest("GET", "/api/consumer-groups", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var groups []ConsumerGroupSummary
		json.NewDecoder(w.Body).Decode(&groups)
		assert.Len(t, groups, 1)
		assert.Equal(t, "test-group", groups[0].GroupID)

		// Get consumer group details
		req = httptest.NewRequest("GET", "/api/consumer-groups/test-group", nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var detail ConsumerGroupDetail
		json.NewDecoder(w.Body).Decode(&detail)
		assert.Equal(t, "test-group", detail.GroupID)
		assert.Equal(t, "consumer", detail.ProtocolType)
		assert.Len(t, detail.OffsetCommits, 1)
	})
}

func TestConsoleAPIErrors(t *testing.T) {
	dir := t.TempDir()
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	coord := coordinator.NewCoordinator()
	coord.Start()

	authConfig := AuthConfig{Enabled: false}
	server := NewServer(":8080", topicMgr, coord, authConfig)

	t.Run("create topic with empty name", func(t *testing.T) {
		reqBody := CreateTopicRequest{
			Name:       "",
			Partitions: 3,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/topics", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]string
		json.NewDecoder(w.Body).Decode(&resp)
		assert.Contains(t, resp["error"], "topic name is required")
	})

	t.Run("create topic with invalid partitions", func(t *testing.T) {
		reqBody := CreateTopicRequest{
			Name:       "invalid-topic",
			Partitions: 0,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/topics", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("get non-existent topic", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/topics/non-existent", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("delete non-existent topic", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/topics/non-existent", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("get messages from non-existent topic", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/topics/non-existent/messages?partition=0&offset=0&limit=10", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("get messages with invalid parameters", func(t *testing.T) {
		// Create a topic first
		topicMgr.CreateTopic("test-topic-2", 3)

		// Invalid partition
		req := httptest.NewRequest("GET", "/api/topics/test-topic-2/messages?partition=-1&offset=0&limit=10", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Invalid offset
		req = httptest.NewRequest("GET", "/api/topics/test-topic-2/messages?partition=0&offset=-1&limit=10", nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Invalid limit
		req = httptest.NewRequest("GET", "/api/topics/test-topic-2/messages?partition=0&offset=0&limit=0", nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("produce message to non-existent topic", func(t *testing.T) {
		reqBody := ProduceMessageRequest{
			Partition: 0,
			Key:       "key1",
			Value:     "value1",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/topics/non-existent/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("get non-existent consumer group", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/consumer-groups/non-existent", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestConsoleAPIMessages(t *testing.T) {
	dir := t.TempDir()
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	coord := coordinator.NewCoordinator()
	coord.Start()

	authConfig := AuthConfig{Enabled: false}
	server := NewServer(":8080", topicMgr, coord, authConfig)

	// Create a topic
	topicMgr.CreateTopic("messages-topic", 3)

	t.Run("produce and consume multiple messages", func(t *testing.T) {
		// Produce 10 messages
		for i := 0; i < 10; i++ {
			reqBody := ProduceMessageRequest{
				Partition: 0,
				Key:       "key",
				Value:     "value-" + string(rune('0'+i)),
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/api/topics/messages-topic/messages", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusCreated, w.Code)
		}

		// Read first 5 messages
		req := httptest.NewRequest("GET", "/api/topics/messages-topic/messages?partition=0&offset=0&limit=5", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var messages []Message
		json.NewDecoder(w.Body).Decode(&messages)
		assert.Len(t, messages, 5)
		assert.Equal(t, int64(0), messages[0].Offset)
		assert.Equal(t, int64(4), messages[4].Offset)

		// Read next 5 messages
		req = httptest.NewRequest("GET", "/api/topics/messages-topic/messages?partition=0&offset=5&limit=5", nil)
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		json.NewDecoder(w.Body).Decode(&messages)
		assert.Len(t, messages, 5)
		assert.Equal(t, int64(5), messages[0].Offset)
	})

	t.Run("produce messages to different partitions", func(t *testing.T) {
		// Produce to partition 0
		reqBody := ProduceMessageRequest{
			Partition: 0,
			Key:       "p0-key",
			Value:     "p0-value",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/topics/messages-topic/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Produce to partition 1
		reqBody.Partition = 1
		reqBody.Key = "p1-key"
		reqBody.Value = "p1-value"
		body, _ = json.Marshal(reqBody)
		req = httptest.NewRequest("POST", "/api/topics/messages-topic/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Produce to partition 2
		reqBody.Partition = 2
		reqBody.Key = "p2-key"
		reqBody.Value = "p2-value"
		body, _ = json.Marshal(reqBody)
		req = httptest.NewRequest("POST", "/api/topics/messages-topic/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("read from empty partition", func(t *testing.T) {
		// Create new topic
		topicMgr.CreateTopic("empty-topic", 1)

		req := httptest.NewRequest("GET", "/api/topics/empty-topic/messages?partition=0&offset=0&limit=10", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var messages []Message
		json.NewDecoder(w.Body).Decode(&messages)
		assert.Len(t, messages, 0)
	})
}
