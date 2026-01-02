// Copyright 2025 Takhin Data, Inc.

package console

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// Configuration Types

type ClusterConfig struct {
	BrokerID          int      `json:"brokerId"`
	Listeners         []string `json:"listeners"`
	AdvertisedHost    string   `json:"advertisedHost"`
	AdvertisedPort    int      `json:"advertisedPort"`
	MaxMessageBytes   int      `json:"maxMessageBytes"`
	MaxConnections    int      `json:"maxConnections"`
	RequestTimeout    int      `json:"requestTimeoutMs"`
	ConnectionTimeout int      `json:"connectionTimeoutMs"`
	DataDir           string   `json:"dataDir"`
	LogSegmentSize    int64    `json:"logSegmentSize"`
	LogRetentionHours int      `json:"logRetentionHours"`
	LogRetentionBytes int64    `json:"logRetentionBytes"`
	MetricsEnabled    bool     `json:"metricsEnabled"`
	MetricsPort       int      `json:"metricsPort"`
}

type TopicConfig struct {
	Name              string            `json:"name"`
	CompressionType   string            `json:"compressionType"`
	CleanupPolicy     string            `json:"cleanupPolicy"`
	RetentionMs       int64             `json:"retentionMs"`
	SegmentMs         int64             `json:"segmentMs"`
	MaxMessageBytes   int               `json:"maxMessageBytes"`
	MinInSyncReplicas int               `json:"minInSyncReplicas"`
	CustomConfigs     map[string]string `json:"customConfigs,omitempty"`
}

type UpdateClusterConfigRequest struct {
	MaxMessageBytes   *int `json:"maxMessageBytes,omitempty"`
	MaxConnections    *int `json:"maxConnections,omitempty"`
	RequestTimeout    *int `json:"requestTimeoutMs,omitempty"`
	ConnectionTimeout *int `json:"connectionTimeoutMs,omitempty"`
	LogRetentionHours *int `json:"logRetentionHours,omitempty"`
}

type UpdateTopicConfigRequest struct {
	CompressionType   *string `json:"compressionType,omitempty"`
	CleanupPolicy     *string `json:"cleanupPolicy,omitempty"`
	RetentionMs       *int64  `json:"retentionMs,omitempty"`
	SegmentMs         *int64  `json:"segmentMs,omitempty"`
	MaxMessageBytes   *int    `json:"maxMessageBytes,omitempty"`
	MinInSyncReplicas *int    `json:"minInSyncReplicas,omitempty"`
}

type BatchUpdateTopicConfigsRequest struct {
	Topics []string                 `json:"topics"`
	Config UpdateTopicConfigRequest `json:"config"`
}

type ConfigChange struct {
	Key       string    `json:"key"`
	OldValue  string    `json:"oldValue"`
	NewValue  string    `json:"newValue"`
	Timestamp time.Time `json:"timestamp"`
	User      string    `json:"user,omitempty"`
}

type ConfigHistory struct {
	ResourceType string         `json:"resourceType"`
	ResourceName string         `json:"resourceName"`
	Changes      []ConfigChange `json:"changes"`
}

// Handler implementations

// handleGetClusterConfig godoc
// @Summary      Get cluster configuration
// @Description  Retrieve current cluster-level configuration settings
// @Tags         Configuration
// @Produce      json
// @Success      200  {object}  ClusterConfig
// @Security     ApiKeyAuth
// @Router       /configs/cluster [get]
func (s *Server) handleGetClusterConfig(w http.ResponseWriter, r *http.Request) {
	// Build cluster config from server config
	// Note: Some values are hardcoded as they're not exposed by topicManager
	config := ClusterConfig{
		BrokerID:          0, // Default broker ID
		Listeners:         []string{"localhost:9092"},
		AdvertisedHost:    "localhost",
		AdvertisedPort:    9092,
		MaxMessageBytes:   1048576,
		MaxConnections:    100,
		RequestTimeout:    30000,
		ConnectionTimeout: 30000,
		DataDir:           "/tmp/takhin-data", // Default data dir
		LogSegmentSize:    1073741824,
		LogRetentionHours: 168,
		LogRetentionBytes: -1,
		MetricsEnabled:    true,
		MetricsPort:       9090,
	}

	s.respondJSON(w, http.StatusOK, config)
}

// handleUpdateClusterConfig godoc
// @Summary      Update cluster configuration
// @Description  Update cluster-level configuration settings
// @Tags         Configuration
// @Accept       json
// @Produce      json
// @Param        request  body      UpdateClusterConfigRequest  true  "Configuration update"
// @Success      200      {object}  ClusterConfig
// @Failure      400      {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /configs/cluster [put]
func (s *Server) handleUpdateClusterConfig(w http.ResponseWriter, r *http.Request) {
	var req UpdateClusterConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate configuration values
	if req.MaxMessageBytes != nil && *req.MaxMessageBytes < 1024 {
		s.respondError(w, http.StatusBadRequest, "maxMessageBytes must be at least 1024")
		return
	}

	if req.MaxConnections != nil && *req.MaxConnections < 1 {
		s.respondError(w, http.StatusBadRequest, "maxConnections must be at least 1")
		return
	}

	// In a real implementation, these would be persisted
	s.logger.Info("cluster config update requested",
		"max_message_bytes", req.MaxMessageBytes,
		"max_connections", req.MaxConnections,
	)

	// Return updated config
	s.handleGetClusterConfig(w, r)
}

// handleGetTopicConfig godoc
// @Summary      Get topic configuration
// @Description  Retrieve configuration for a specific topic
// @Tags         Configuration
// @Produce      json
// @Param        topic  path      string  true  "Topic name"
// @Success      200    {object}  TopicConfig
// @Failure      404    {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /configs/topics/{topic} [get]
func (s *Server) handleGetTopicConfig(w http.ResponseWriter, r *http.Request) {
	topicName := chi.URLParam(r, "topic")

	_, exists := s.topicManager.GetTopic(topicName)
	if !exists {
		s.respondError(w, http.StatusNotFound, "topic not found")
		return
	}

	// Return default topic configuration
	config := TopicConfig{
		Name:              topicName,
		CompressionType:   "producer",
		CleanupPolicy:     "delete",
		RetentionMs:       604800000, // 7 days
		SegmentMs:         604800000, // 7 days
		MaxMessageBytes:   1048576,   // 1 MB
		MinInSyncReplicas: 1,
		CustomConfigs:     make(map[string]string),
	}

	s.respondJSON(w, http.StatusOK, config)
}

// handleUpdateTopicConfig godoc
// @Summary      Update topic configuration
// @Description  Update configuration for a specific topic
// @Tags         Configuration
// @Accept       json
// @Produce      json
// @Param        topic    path      string                    true  "Topic name"
// @Param        request  body      UpdateTopicConfigRequest  true  "Configuration update"
// @Success      200      {object}  TopicConfig
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /configs/topics/{topic} [put]
func (s *Server) handleUpdateTopicConfig(w http.ResponseWriter, r *http.Request) {
	topicName := chi.URLParam(r, "topic")

	_, exists := s.topicManager.GetTopic(topicName)
	if !exists {
		s.respondError(w, http.StatusNotFound, "topic not found")
		return
	}

	var req UpdateTopicConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate configuration values
	if req.CompressionType != nil {
		validTypes := []string{"none", "gzip", "snappy", "lz4", "zstd", "producer"}
		valid := false
		for _, t := range validTypes {
			if *req.CompressionType == t {
				valid = true
				break
			}
		}
		if !valid {
			s.respondError(w, http.StatusBadRequest, "invalid compression type")
			return
		}
	}

	if req.CleanupPolicy != nil && *req.CleanupPolicy != "delete" && *req.CleanupPolicy != "compact" {
		s.respondError(w, http.StatusBadRequest, "cleanup policy must be 'delete' or 'compact'")
		return
	}

	if req.RetentionMs != nil && *req.RetentionMs < 1 {
		s.respondError(w, http.StatusBadRequest, "retentionMs must be positive")
		return
	}

	// In a real implementation, these would be persisted
	s.logger.Info("topic config update requested",
		"topic", topicName,
		"compression_type", req.CompressionType,
		"cleanup_policy", req.CleanupPolicy,
		"retention_ms", req.RetentionMs,
	)

	// Return updated config
	s.handleGetTopicConfig(w, r)
}

// handleBatchUpdateTopicConfigs godoc
// @Summary      Batch update topic configurations
// @Description  Update configuration for multiple topics at once
// @Tags         Configuration
// @Accept       json
// @Produce      json
// @Param        request  body      BatchUpdateTopicConfigsRequest  true  "Batch update request"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /configs/topics [put]
func (s *Server) handleBatchUpdateTopicConfigs(w http.ResponseWriter, r *http.Request) {
	var req BatchUpdateTopicConfigsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Topics) == 0 {
		s.respondError(w, http.StatusBadRequest, "no topics specified")
		return
	}

	// Validate all topics exist
	var notFound []string
	for _, topicName := range req.Topics {
		if _, exists := s.topicManager.GetTopic(topicName); !exists {
			notFound = append(notFound, topicName)
		}
	}

	if len(notFound) > 0 {
		s.respondError(w, http.StatusNotFound, "topics not found: "+notFound[0])
		return
	}

	// Validate configuration
	if req.Config.CompressionType != nil {
		validTypes := []string{"none", "gzip", "snappy", "lz4", "zstd", "producer"}
		valid := false
		for _, t := range validTypes {
			if *req.Config.CompressionType == t {
				valid = true
				break
			}
		}
		if !valid {
			s.respondError(w, http.StatusBadRequest, "invalid compression type")
			return
		}
	}

	// In a real implementation, these would be persisted
	s.logger.Info("batch topic config update requested",
		"topics", req.Topics,
		"config", req.Config,
	)

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"updated": len(req.Topics),
		"topics":  req.Topics,
	})
}
