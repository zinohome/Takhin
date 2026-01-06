// Copyright 2025 Takhin Data, Inc.

package console

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/takhin-data/takhin/pkg/coordinator"
)

// HTTPConsumer represents an HTTP-based consumer instance
type HTTPConsumer struct {
	ID              string
	GroupID         string
	Topics          []string
	Assignments     map[string][]int32 // topic -> partitions
	Offsets         map[string]map[int32]int64 // topic -> partition -> offset
	LastHeartbeat   time.Time
	SessionTimeout  time.Duration
	mu              sync.RWMutex
	cancelHeartbeat context.CancelFunc
}

// ConsumerManager manages HTTP consumers
type ConsumerManager struct {
	consumers map[string]*HTTPConsumer
	mu        sync.RWMutex
}

func NewConsumerManager() *ConsumerManager {
	return &ConsumerManager{
		consumers: make(map[string]*HTTPConsumer),
	}
}

func (cm *ConsumerManager) CreateConsumer(groupID string, topics []string, sessionTimeout time.Duration) *HTTPConsumer {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	consumer := &HTTPConsumer{
		ID:             uuid.New().String(),
		GroupID:        groupID,
		Topics:         topics,
		Assignments:    make(map[string][]int32),
		Offsets:        make(map[string]map[int32]int64),
		LastHeartbeat:  time.Now(),
		SessionTimeout: sessionTimeout,
	}

	cm.consumers[consumer.ID] = consumer
	return consumer
}

func (cm *ConsumerManager) GetConsumer(consumerID string) (*HTTPConsumer, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	consumer, exists := cm.consumers[consumerID]
	return consumer, exists
}

func (cm *ConsumerManager) DeleteConsumer(consumerID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if consumer, exists := cm.consumers[consumerID]; exists {
		if consumer.cancelHeartbeat != nil {
			consumer.cancelHeartbeat()
		}
		delete(cm.consumers, consumerID)
	}
}

func (cm *ConsumerManager) UpdateHeartbeat(consumerID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	if consumer, exists := cm.consumers[consumerID]; exists {
		consumer.mu.Lock()
		consumer.LastHeartbeat = time.Now()
		consumer.mu.Unlock()
		return true
	}
	return false
}

// Request/Response types

// SubscribeRequest represents consumer subscription request
type SubscribeRequest struct {
	GroupID         string   `json:"group_id"`
	Topics          []string `json:"topics"`
	AutoOffsetReset string   `json:"auto_offset_reset,omitempty"` // earliest, latest
	SessionTimeout  int      `json:"session_timeout_ms,omitempty"` // default 30000
}

// SubscribeResponse represents subscription response
type SubscribeResponse struct {
	ConsumerID string            `json:"consumer_id"`
	GroupID    string            `json:"group_id"`
	Topics     []string          `json:"topics"`
	Assignment map[string][]int32 `json:"assignment"` // topic -> partitions
}

// ConsumeRequest represents consume polling request
type ConsumeRequest struct {
	MaxRecords    int `json:"max_records,omitempty"`     // default 500
	TimeoutMs     int `json:"timeout_ms,omitempty"`      // default 30000
	MaxBytesTotal int `json:"max_bytes_total,omitempty"` // default 1MB
}

// ConsumeResponse represents consume response
type ConsumeResponse struct {
	Records   []ConsumerRecord `json:"records"`
	Timestamp int64            `json:"timestamp"`
}

// ConsumerRecord represents a consumed record
type ConsumerRecord struct {
	Topic     string            `json:"topic"`
	Partition int32             `json:"partition"`
	Offset    int64             `json:"offset"`
	Timestamp int64             `json:"timestamp"`
	Key       []byte            `json:"key"`
	Value     []byte            `json:"value"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// CommitRequest represents offset commit request
type CommitRequest struct {
	Offsets map[string]map[int32]int64 `json:"offsets"` // topic -> partition -> offset
}

// CommitResponse represents commit response
type CommitResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// SeekRequest represents seek position request
type SeekRequest struct {
	Topic     string `json:"topic"`
	Partition int32  `json:"partition"`
	Offset    int64  `json:"offset"`
}

// SeekResponse represents seek response
type SeekResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// AssignmentRequest represents manual partition assignment
type AssignmentRequest struct {
	Topics map[string][]int32 `json:"topics"` // topic -> partitions
}

// PositionResponse represents current position
type PositionResponse struct {
	Offsets map[string]map[int32]int64 `json:"offsets"` // topic -> partition -> offset
}

// Handler methods

// handleSubscribe godoc
// @Summary      Subscribe to topics
// @Description  Subscribe consumer to topics and join consumer group
// @Tags         Consumer
// @Accept       json
// @Produce      json
// @Param        request  body      SubscribeRequest  true  "Subscribe request"
// @Success      200      {object}  SubscribeResponse
// @Failure      400      {object}  map[string]string
// @Router       /api/consumers/subscribe [post]
func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.GroupID == "" {
		s.respondError(w, http.StatusBadRequest, "group_id is required")
		return
	}

	if len(req.Topics) == 0 {
		s.respondError(w, http.StatusBadRequest, "topics are required")
		return
	}

	sessionTimeout := 30 * time.Second
	if req.SessionTimeout > 0 {
		sessionTimeout = time.Duration(req.SessionTimeout) * time.Millisecond
	}

	// Create consumer instance
	if s.consumerManager == nil {
		s.consumerManager = NewConsumerManager()
	}
	consumer := s.consumerManager.CreateConsumer(req.GroupID, req.Topics, sessionTimeout)

	// Join consumer group
	memberID := consumer.ID
	protocols := []coordinator.MemberProtocol{
		{Name: "range", Metadata: []byte("{}")},
	}

	_, member, isNew, err := s.coordinator.JoinGroup(
		req.GroupID,
		memberID,
		"http-consumer",
		r.RemoteAddr,
		"consumer",
		protocols,
		int32(sessionTimeout.Milliseconds()),
		int32(sessionTimeout.Milliseconds()),
	)
	if err != nil {
		s.consumerManager.DeleteConsumer(consumer.ID)
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to join group: %v", err))
		return
	}

	_ = isNew
	_ = member

	// Assign partitions (simple round-robin for now)
	assignments := s.assignPartitions(consumer.ID, req.Topics)
	consumer.mu.Lock()
	consumer.Assignments = assignments
	// Initialize offsets based on auto.offset.reset
	for topic, partitions := range assignments {
		if consumer.Offsets[topic] == nil {
			consumer.Offsets[topic] = make(map[int32]int64)
		}
		for _, partition := range partitions {
			if req.AutoOffsetReset == "earliest" {
				consumer.Offsets[topic][partition] = 0
			} else {
				// Latest - get current end offset
				if t, ok := s.topicManager.GetTopic(topic); ok {
					if logInstance := t.Partitions[partition]; logInstance != nil {
						consumer.Offsets[topic][partition] = logInstance.HighWaterMark()
					}
				}
			}
		}
	}
	consumer.mu.Unlock()

	// Start heartbeat monitoring
	ctx, cancel := context.WithCancel(context.Background())
	consumer.cancelHeartbeat = cancel
	go s.monitorConsumerHeartbeat(ctx, consumer)

	s.respondJSON(w, http.StatusOK, SubscribeResponse{
		ConsumerID: consumer.ID,
		GroupID:    req.GroupID,
		Topics:     req.Topics,
		Assignment: assignments,
	})
}

// handleConsume godoc
// @Summary      Poll for records
// @Description  Long-poll for new records from assigned partitions
// @Tags         Consumer
// @Accept       json
// @Produce      json
// @Param        consumer_id  path      string         true  "Consumer ID"
// @Param        request      body      ConsumeRequest  false  "Consume options"
// @Success      200          {object}  ConsumeResponse
// @Failure      404          {object}  map[string]string
// @Router       /api/consumers/{consumer_id}/consume [post]
func (s *Server) handleConsume(w http.ResponseWriter, r *http.Request) {
	consumerID := chi.URLParam(r, "consumer_id")

	consumer, exists := s.consumerManager.GetConsumer(consumerID)
	if !exists {
		s.respondError(w, http.StatusNotFound, "consumer not found")
		return
	}

	var req ConsumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.MaxRecords = 500
		req.TimeoutMs = 30000
		req.MaxBytesTotal = 1024 * 1024
	}

	if req.MaxRecords <= 0 {
		req.MaxRecords = 500
	}
	if req.TimeoutMs <= 0 {
		req.TimeoutMs = 30000
	}
	if req.MaxBytesTotal <= 0 {
		req.MaxBytesTotal = 1024 * 1024
	}

	// Update heartbeat
	s.consumerManager.UpdateHeartbeat(consumerID)

	// Poll for records with timeout
	records := s.pollRecords(consumer, req.MaxRecords, req.MaxBytesTotal, time.Duration(req.TimeoutMs)*time.Millisecond)

	s.respondJSON(w, http.StatusOK, ConsumeResponse{
		Records:   records,
		Timestamp: time.Now().UnixMilli(),
	})
}

// handleCommit godoc
// @Summary      Commit offsets
// @Description  Commit consumer offsets to coordinator
// @Tags         Consumer
// @Accept       json
// @Produce      json
// @Param        consumer_id  path      string        true  "Consumer ID"
// @Param        request      body      CommitRequest  true  "Commit request"
// @Success      200          {object}  CommitResponse
// @Failure      404          {object}  map[string]string
// @Router       /api/consumers/{consumer_id}/commit [post]
func (s *Server) handleCommit(w http.ResponseWriter, r *http.Request) {
	consumerID := chi.URLParam(r, "consumer_id")

	consumer, exists := s.consumerManager.GetConsumer(consumerID)
	if !exists {
		s.respondError(w, http.StatusNotFound, "consumer not found")
		return
	}

	var req CommitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update consumer offsets
	consumer.mu.Lock()
	for topic, partitions := range req.Offsets {
		if consumer.Offsets[topic] == nil {
			consumer.Offsets[topic] = make(map[int32]int64)
		}
		for partition, offset := range partitions {
			consumer.Offsets[topic][partition] = offset
		}
	}
	consumer.mu.Unlock()

	// Commit to coordinator
	group, exists := s.coordinator.GetGroup(consumer.GroupID)
	if exists {
		for topic, partitions := range req.Offsets {
			for partition, offset := range partitions {
				group.CommitOffset(topic, partition, offset, "")
			}
		}
	}

	s.respondJSON(w, http.StatusOK, CommitResponse{
		Success: true,
		Message: "offsets committed",
	})
}

// handleSeek godoc
// @Summary      Seek to offset
// @Description  Seek consumer to specific offset for partition
// @Tags         Consumer
// @Accept       json
// @Produce      json
// @Param        consumer_id  path      string       true  "Consumer ID"
// @Param        request      body      SeekRequest  true  "Seek request"
// @Success      200          {object}  SeekResponse
// @Failure      404          {object}  map[string]string
// @Router       /api/consumers/{consumer_id}/seek [post]
func (s *Server) handleSeek(w http.ResponseWriter, r *http.Request) {
	consumerID := chi.URLParam(r, "consumer_id")

	consumer, exists := s.consumerManager.GetConsumer(consumerID)
	if !exists {
		s.respondError(w, http.StatusNotFound, "consumer not found")
		return
	}

	var req SeekRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate topic and partition assignment
	consumer.mu.Lock()
	partitions, hasAssignment := consumer.Assignments[req.Topic]
	assigned := false
	for _, p := range partitions {
		if p == req.Partition {
			assigned = true
			break
		}
	}
	if !hasAssignment || !assigned {
		consumer.mu.Unlock()
		s.respondError(w, http.StatusBadRequest, "partition not assigned to consumer")
		return
	}

	if consumer.Offsets[req.Topic] == nil {
		consumer.Offsets[req.Topic] = make(map[int32]int64)
	}
	consumer.Offsets[req.Topic][req.Partition] = req.Offset
	consumer.mu.Unlock()

	s.respondJSON(w, http.StatusOK, SeekResponse{
		Success: true,
		Message: "offset updated",
	})
}

// handleAssignment godoc
// @Summary      Manual partition assignment
// @Description  Manually assign partitions to consumer (bypasses group coordination)
// @Tags         Consumer
// @Accept       json
// @Produce      json
// @Param        consumer_id  path      string            true  "Consumer ID"
// @Param        request      body      AssignmentRequest  true  "Assignment request"
// @Success      200          {object}  SubscribeResponse
// @Failure      404          {object}  map[string]string
// @Router       /api/consumers/{consumer_id}/assignment [put]
func (s *Server) handleAssignment(w http.ResponseWriter, r *http.Request) {
	consumerID := chi.URLParam(r, "consumer_id")

	consumer, exists := s.consumerManager.GetConsumer(consumerID)
	if !exists {
		s.respondError(w, http.StatusNotFound, "consumer not found")
		return
	}

	var req AssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	consumer.mu.Lock()
	consumer.Assignments = req.Topics
	// Initialize offsets for new assignments
	for topic, partitions := range req.Topics {
		if consumer.Offsets[topic] == nil {
			consumer.Offsets[topic] = make(map[int32]int64)
		}
		for _, partition := range partitions {
			if _, exists := consumer.Offsets[topic][partition]; !exists {
				consumer.Offsets[topic][partition] = 0
			}
		}
	}
	consumer.mu.Unlock()

	s.respondJSON(w, http.StatusOK, SubscribeResponse{
		ConsumerID: consumer.ID,
		GroupID:    consumer.GroupID,
		Topics:     consumer.Topics,
		Assignment: consumer.Assignments,
	})
}

// handlePosition godoc
// @Summary      Get current positions
// @Description  Get current offset positions for all assigned partitions
// @Tags         Consumer
// @Produce      json
// @Param        consumer_id  path      string  true  "Consumer ID"
// @Success      200          {object}  PositionResponse
// @Failure      404          {object}  map[string]string
// @Router       /api/consumers/{consumer_id}/position [get]
func (s *Server) handlePosition(w http.ResponseWriter, r *http.Request) {
	consumerID := chi.URLParam(r, "consumer_id")

	consumer, exists := s.consumerManager.GetConsumer(consumerID)
	if !exists {
		s.respondError(w, http.StatusNotFound, "consumer not found")
		return
	}

	consumer.mu.RLock()
	offsets := make(map[string]map[int32]int64)
	for topic, partitions := range consumer.Offsets {
		offsets[topic] = make(map[int32]int64)
		for partition, offset := range partitions {
			offsets[topic][partition] = offset
		}
	}
	consumer.mu.RUnlock()

	s.respondJSON(w, http.StatusOK, PositionResponse{
		Offsets: offsets,
	})
}

// handleUnsubscribe godoc
// @Summary      Unsubscribe consumer
// @Description  Unsubscribe and close consumer instance
// @Tags         Consumer
// @Produce      json
// @Param        consumer_id  path      string  true  "Consumer ID"
// @Success      200          {object}  map[string]string
// @Failure      404          {object}  map[string]string
// @Router       /api/consumers/{consumer_id} [delete]
func (s *Server) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	consumerID := chi.URLParam(r, "consumer_id")

	consumer, exists := s.consumerManager.GetConsumer(consumerID)
	if !exists {
		s.respondError(w, http.StatusNotFound, "consumer not found")
		return
	}

	// Leave consumer group
	if group, exists := s.coordinator.GetGroup(consumer.GroupID); exists {
		group.RemoveMember(consumer.ID)
	}

	// Delete consumer
	s.consumerManager.DeleteConsumer(consumerID)

	s.respondJSON(w, http.StatusOK, map[string]string{
		"message": "consumer closed",
	})
}

// Helper methods

func (s *Server) assignPartitions(consumerID string, topics []string) map[string][]int32 {
	assignments := make(map[string][]int32)

	for _, topicName := range topics {
		topic, ok := s.topicManager.GetTopic(topicName)
		if !ok {
			continue
		}

		// Get all partitions
		partitions := make([]int32, 0)
		for partitionID := range topic.Partitions {
			partitions = append(partitions, partitionID)
		}

		// Assign all partitions (simple assignment - should use group coordinator for real)
		assignments[topicName] = partitions
	}

	return assignments
}

func (s *Server) pollRecords(consumer *HTTPConsumer, maxRecords int, maxBytes int, timeout time.Duration) []ConsumerRecord {
	records := make([]ConsumerRecord, 0, maxRecords)
	totalBytes := 0

	startTime := time.Now()
	deadline := startTime.Add(timeout)

	consumer.mu.RLock()
	assignments := consumer.Assignments
	offsets := make(map[string]map[int32]int64)
	for topic, partitionOffsets := range consumer.Offsets {
		offsets[topic] = make(map[int32]int64)
		for partition, offset := range partitionOffsets {
			offsets[topic][partition] = offset
		}
	}
	consumer.mu.RUnlock()

	// Poll loop with timeout
	for {
		if len(records) >= maxRecords || totalBytes >= maxBytes {
			break
		}

		if time.Now().After(deadline) {
			break
		}

		foundRecords := false

		for topic, partitions := range assignments {
			topicObj, ok := s.topicManager.GetTopic(topic)
			if !ok {
				continue
			}

			for _, partition := range partitions {
				if len(records) >= maxRecords || totalBytes >= maxBytes {
					break
				}

				offset := offsets[topic][partition]
				logInstance := topicObj.Partitions[partition]
				if logInstance == nil {
					continue
				}

				// Try to read record
				record, err := logInstance.Read(offset)
				if err != nil {
					continue
				}

				// Convert to consumer record
				consumerRecord := ConsumerRecord{
					Topic:     topic,
					Partition: partition,
					Offset:    offset,
					Timestamp: record.Timestamp,
					Key:       record.Key,
					Value:     record.Value,
				}

				records = append(records, consumerRecord)
				totalBytes += len(record.Key) + len(record.Value)
				offsets[topic][partition] = offset + 1
				foundRecords = true
			}
		}

		if !foundRecords {
			// No records available, wait a bit before retrying
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Update consumer offsets
	consumer.mu.Lock()
	for topic, partitionOffsets := range offsets {
		if consumer.Offsets[topic] == nil {
			consumer.Offsets[topic] = make(map[int32]int64)
		}
		for partition, offset := range partitionOffsets {
			consumer.Offsets[topic][partition] = offset
		}
	}
	consumer.mu.Unlock()

	return records
}

func (s *Server) monitorConsumerHeartbeat(ctx context.Context, consumer *HTTPConsumer) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			consumer.mu.RLock()
			lastHeartbeat := consumer.LastHeartbeat
			sessionTimeout := consumer.SessionTimeout
			consumer.mu.RUnlock()

			if time.Since(lastHeartbeat) > sessionTimeout {
				// Session expired, remove consumer
				s.logger.Warn("consumer session expired", "consumer_id", consumer.ID, "group_id", consumer.GroupID)
				s.consumerManager.DeleteConsumer(consumer.ID)
				if group, exists := s.coordinator.GetGroup(consumer.GroupID); exists {
					group.RemoveMember(consumer.ID)
				}
				return
			}
		}
	}
}
