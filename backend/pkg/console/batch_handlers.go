// Copyright 2025 Takhin Data, Inc.

package console

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Batch operation types

type BatchCreateTopicsRequest struct {
	Topics []CreateTopicRequest `json:"topics"`
}

type BatchDeleteTopicsRequest struct {
	Topics []string `json:"topics"`
}

type BatchOperationResult struct {
	TotalRequested int                        `json:"totalRequested"`
	Successful     int                        `json:"successful"`
	Failed         int                        `json:"failed"`
	Results        []SingleOperationResult    `json:"results"`
	Errors         []string                   `json:"errors,omitempty"`
}

type SingleOperationResult struct {
	Resource   string `json:"resource"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	Partitions int32  `json:"partitions,omitempty"`
}

// handleBatchCreateTopics godoc
// @Summary      Batch create topics
// @Description  Create multiple topics in a single transactional operation
// @Tags         Topics
// @Accept       json
// @Produce      json
// @Param        request  body      BatchCreateTopicsRequest  true  "Batch create request"
// @Success      200      {object}  BatchOperationResult
// @Failure      400      {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /topics/batch [post]
func (s *Server) handleBatchCreateTopics(w http.ResponseWriter, r *http.Request) {
	var req BatchCreateTopicsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Topics) == 0 {
		s.respondError(w, http.StatusBadRequest, "no topics specified")
		return
	}

	// Validate all requests first (fail-fast)
	for i, topic := range req.Topics {
		if topic.Name == "" {
			s.respondError(w, http.StatusBadRequest, fmt.Sprintf("topic[%d]: name is required", i))
			return
		}
		if topic.Partitions <= 0 {
			s.respondError(w, http.StatusBadRequest, fmt.Sprintf("topic[%d]: partitions must be greater than 0", i))
			return
		}
	}

	// Check for duplicate names in the request
	nameSet := make(map[string]bool)
	for _, topic := range req.Topics {
		if nameSet[topic.Name] {
			s.respondError(w, http.StatusBadRequest, fmt.Sprintf("duplicate topic name in request: %s", topic.Name))
			return
		}
		nameSet[topic.Name] = true
	}

	// Execute batch operation with transaction semantics
	result := s.executeBatchCreate(req.Topics)

	// Return 200 with detailed results even if some operations failed
	statusCode := http.StatusOK
	if result.Failed > 0 && result.Successful == 0 {
		statusCode = http.StatusBadRequest
	}

	s.respondJSON(w, statusCode, result)
}

func (s *Server) executeBatchCreate(topics []CreateTopicRequest) BatchOperationResult {
	result := BatchOperationResult{
		TotalRequested: len(topics),
		Results:        make([]SingleOperationResult, 0, len(topics)),
	}

	// Track created topics for rollback
	var createdTopics []string
	var mu sync.Mutex

	// Check for existing topics first
	for _, topic := range topics {
		if _, exists := s.topicManager.GetTopic(topic.Name); exists {
			result.Failed++
			result.Results = append(result.Results, SingleOperationResult{
				Resource: topic.Name,
				Success:  false,
				Error:    "topic already exists",
			})
			result.Errors = append(result.Errors, fmt.Sprintf("topic '%s' already exists", topic.Name))
		}
	}

	// If any topics exist, abort the entire batch
	if result.Failed > 0 {
		return result
	}

	// Reset for actual creation
	result.Failed = 0
	result.Results = make([]SingleOperationResult, 0, len(topics))
	result.Errors = nil

	// Attempt to create all topics
	for _, topic := range topics {
		err := s.topicManager.CreateTopic(topic.Name, topic.Partitions)
		
		mu.Lock()
		if err != nil {
			result.Failed++
			result.Results = append(result.Results, SingleOperationResult{
				Resource: topic.Name,
				Success:  false,
				Error:    err.Error(),
			})
			result.Errors = append(result.Errors, fmt.Sprintf("failed to create '%s': %s", topic.Name, err.Error()))
			mu.Unlock()
			
			// Rollback on first failure
			s.rollbackTopicCreation(createdTopics)
			return result
		}
		
		createdTopics = append(createdTopics, topic.Name)
		result.Successful++
		result.Results = append(result.Results, SingleOperationResult{
			Resource:   topic.Name,
			Success:    true,
			Partitions: topic.Partitions,
		})
		mu.Unlock()

		// Broadcast creation event
		s.BroadcastTopicCreated(topic.Name, topic.Partitions)
	}

	return result
}

func (s *Server) rollbackTopicCreation(topics []string) {
	s.logger.Warn("rolling back topic creation", "topics", topics)
	
	for _, topicName := range topics {
		if err := s.topicManager.DeleteTopic(topicName); err != nil {
			s.logger.Error("failed to rollback topic creation", "topic", topicName, "error", err)
		} else {
			s.BroadcastTopicDeleted(topicName)
		}
	}
}

// handleBatchDeleteTopics godoc
// @Summary      Batch delete topics
// @Description  Delete multiple topics in a single operation
// @Tags         Topics
// @Accept       json
// @Produce      json
// @Param        request  body      BatchDeleteTopicsRequest  true  "Batch delete request"
// @Success      200      {object}  BatchOperationResult
// @Failure      400      {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /topics/batch [delete]
func (s *Server) handleBatchDeleteTopics(w http.ResponseWriter, r *http.Request) {
	var req BatchDeleteTopicsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Topics) == 0 {
		s.respondError(w, http.StatusBadRequest, "no topics specified")
		return
	}

	// Check for duplicates
	nameSet := make(map[string]bool)
	for _, topicName := range req.Topics {
		if topicName == "" {
			s.respondError(w, http.StatusBadRequest, "empty topic name in request")
			return
		}
		if nameSet[topicName] {
			s.respondError(w, http.StatusBadRequest, fmt.Sprintf("duplicate topic name in request: %s", topicName))
			return
		}
		nameSet[topicName] = true
	}

	// Execute batch deletion
	result := s.executeBatchDelete(req.Topics)

	statusCode := http.StatusOK
	if result.Failed > 0 && result.Successful == 0 {
		statusCode = http.StatusBadRequest
	}

	s.respondJSON(w, statusCode, result)
}

func (s *Server) executeBatchDelete(topics []string) BatchOperationResult {
	result := BatchOperationResult{
		TotalRequested: len(topics),
		Results:        make([]SingleOperationResult, 0, len(topics)),
	}

	// Verify all topics exist first
	for _, topicName := range topics {
		if _, exists := s.topicManager.GetTopic(topicName); !exists {
			result.Failed++
			result.Results = append(result.Results, SingleOperationResult{
				Resource: topicName,
				Success:  false,
				Error:    "topic not found",
			})
			result.Errors = append(result.Errors, fmt.Sprintf("topic '%s' not found", topicName))
		}
	}

	// If any topics don't exist, abort the entire batch
	if result.Failed > 0 {
		return result
	}

	// Reset for actual deletion
	result.Failed = 0
	result.Results = make([]SingleOperationResult, 0, len(topics))
	result.Errors = nil

	// Delete all topics
	for _, topicName := range topics {
		err := s.topicManager.DeleteTopic(topicName)
		
		if err != nil {
			result.Failed++
			result.Results = append(result.Results, SingleOperationResult{
				Resource: topicName,
				Success:  false,
				Error:    err.Error(),
			})
			result.Errors = append(result.Errors, fmt.Sprintf("failed to delete '%s': %s", topicName, err.Error()))
		} else {
			result.Successful++
			result.Results = append(result.Results, SingleOperationResult{
				Resource: topicName,
				Success:  true,
			})
			
			// Broadcast deletion event
			s.BroadcastTopicDeleted(topicName)
		}
	}

	return result
}
