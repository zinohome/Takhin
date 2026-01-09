// Copyright 2025 Takhin Data, Inc.

package console

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// BrokerInfo represents information about a broker
type BrokerInfo struct {
	ID             int32  `json:"id"`
	Host           string `json:"host"`
	Port           int32  `json:"port"`
	IsController   bool   `json:"isController"`
	TopicCount     int    `json:"topicCount"`
	PartitionCount int    `json:"partitionCount"`
	Status         string `json:"status"` // "online" or "offline"
}

// ClusterStats represents cluster-wide statistics
type ClusterStats struct {
	BrokerCount       int   `json:"brokerCount"`
	TopicCount        int   `json:"topicCount"`
	PartitionCount    int   `json:"partitionCount"`
	TotalMessages     int64 `json:"totalMessages"`
	TotalSizeBytes    int64 `json:"totalSizeBytes"`
	ReplicationFactor int   `json:"replicationFactor"`
}

// handleListBrokers godoc
// @Summary      List all brokers
// @Description  Get a list of all brokers in the cluster
// @Tags         Brokers
// @Produce      json
// @Success      200  {array}   BrokerInfo
// @Security     ApiKeyAuth
// @Router       /brokers [get]
func (s *Server) handleListBrokers(w http.ResponseWriter, r *http.Request) {
	// Get broker information from config
	brokerID := int32(s.config.Kafka.BrokerID)
	host := s.config.Kafka.AdvertisedHost
	port := int32(s.config.Kafka.AdvertisedPort)
	if host == "" {
		host = s.config.Server.Host
	}
	if port == 0 {
		port = int32(s.config.Server.Port)
	}

	// Count topics and partitions for this broker
	topics := s.topicManager.ListTopics()
	topicCount := len(topics)
	partitionCount := 0

	for _, topicName := range topics {
		topic, exists := s.topicManager.GetTopic(topicName)
		if exists {
			partitionCount += len(topic.Partitions)
		}
	}

	// For now, we only have single broker support
	// In a multi-broker setup, this would query the Raft cluster
	brokers := []BrokerInfo{
		{
			ID:             brokerID,
			Host:           host,
			Port:           port,
			IsController:   true, // Single broker is always controller
			TopicCount:     topicCount,
			PartitionCount: partitionCount,
			Status:         "online",
		},
	}

	s.respondJSON(w, http.StatusOK, brokers)
}

// handleGetBroker godoc
// @Summary      Get broker details
// @Description  Get detailed information about a specific broker
// @Tags         Brokers
// @Produce      json
// @Param        id   path      int  true  "Broker ID"
// @Success      200  {object}  BrokerInfo
// @Failure      404  {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /brokers/{id} [get]
func (s *Server) handleGetBroker(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid broker ID")
		return
	}

	brokerID := int32(s.config.Kafka.BrokerID)
	if int32(id) != brokerID {
		s.respondError(w, http.StatusNotFound, "broker not found")
		return
	}

	host := s.config.Kafka.AdvertisedHost
	port := int32(s.config.Kafka.AdvertisedPort)
	if host == "" {
		host = s.config.Server.Host
	}
	if port == 0 {
		port = int32(s.config.Server.Port)
	}

	// Count topics and partitions
	topics := s.topicManager.ListTopics()
	topicCount := len(topics)
	partitionCount := 0

	for _, topicName := range topics {
		topic, exists := s.topicManager.GetTopic(topicName)
		if exists {
			partitionCount += len(topic.Partitions)
		}
	}

	broker := BrokerInfo{
		ID:             brokerID,
		Host:           host,
		Port:           port,
		IsController:   true,
		TopicCount:     topicCount,
		PartitionCount: partitionCount,
		Status:         "online",
	}

	s.respondJSON(w, http.StatusOK, broker)
}

// handleGetClusterStats godoc
// @Summary      Get cluster statistics
// @Description  Get aggregated statistics for the entire cluster
// @Tags         Cluster
// @Produce      json
// @Success      200  {object}  ClusterStats
// @Security     ApiKeyAuth
// @Router       /cluster/stats [get]
func (s *Server) handleGetClusterStats(w http.ResponseWriter, r *http.Request) {
	// Get all topics
	topics := s.topicManager.ListTopics()
	topicCount := len(topics)
	partitionCount := 0
	totalMessages := int64(0)
	totalSizeBytes := int64(0)

	// Aggregate statistics from all topics
	for _, topicName := range topics {
		topic, exists := s.topicManager.GetTopic(topicName)
		if !exists {
			continue
		}

		partitionCount += len(topic.Partitions)

		// Sum up high water marks and sizes for all partitions
		for partID := range topic.Partitions {
			hwm, err := topic.HighWaterMark(partID)
			if err == nil {
				totalMessages += hwm
			}

			// Calculate size (this is an approximation)
			// In a real implementation, we'd track actual disk usage
			size, err := topic.PartitionSize(partID)
			if err == nil {
				totalSizeBytes += size
			}
		}
	}

	stats := ClusterStats{
		BrokerCount:       1, // Single broker for now
		TopicCount:        topicCount,
		PartitionCount:    partitionCount,
		TotalMessages:     totalMessages,
		TotalSizeBytes:    totalSizeBytes,
		ReplicationFactor: 1, // No replication in single broker mode
	}

	s.respondJSON(w, http.StatusOK, stats)
}
