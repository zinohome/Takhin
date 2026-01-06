// Copyright 2025 Takhin Data, Inc.

package console

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"github.com/takhin-data/takhin/pkg/acl"
	"github.com/takhin-data/takhin/pkg/audit"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/logger"
	"github.com/takhin-data/takhin/pkg/storage/topic"

	_ "github.com/takhin-data/takhin/docs/swagger" // Import swagger docs
)

// Server represents the Console HTTP API server
type Server struct {
	router        *chi.Mux
	logger        *logger.Logger
	topicManager  *topic.Manager
	coordinator   *coordinator.Coordinator
	aclStore      *acl.Store
	authConfig    AuthConfig
	addr          string
	healthChecker *HealthChecker
	wsHub         *WebSocketHub
	auditLogger   *audit.Logger
	config        *config.Config
}

// NewServer creates a new Console API server
func NewServer(addr string, topicManager *topic.Manager, coord *coordinator.Coordinator, aclStore *acl.Store, authConfig AuthConfig, auditLogger *audit.Logger, cfg *config.Config) *Server {
	wsHub := NewWebSocketHub()
	
	s := &Server{
		router:        chi.NewRouter(),
		logger:        logger.Default().WithComponent("console-api"),
		topicManager:  topicManager,
		coordinator:   coord,
		aclStore:      aclStore,
		authConfig:    authConfig,
		addr:          addr,
		healthChecker: NewHealthChecker("1.0.0", topicManager, coord),
		wsHub:         wsHub,
		auditLogger:   auditLogger,
		config:        cfg,
	}

	go wsHub.Run()

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware configures HTTP middleware
func (s *Server) setupMiddleware() {
	// Basic middleware
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	// Authentication middleware
	s.router.Use(AuthMiddleware(s.authConfig))

	// CORS middleware
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.1:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes() {
	// Swagger UI
	s.router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Health check endpoints (no auth required)
	s.router.Get("/api/health", s.handleHealth)
	s.router.Get("/api/health/ready", s.handleReadiness)
	s.router.Get("/api/health/live", s.handleLiveness)

	// Topic routes
	s.router.Route("/api/topics", func(r chi.Router) {
		r.Get("/", s.handleListTopics)
		r.Post("/", s.handleCreateTopic)
		r.Post("/batch", s.handleBatchCreateTopics)
		r.Delete("/batch", s.handleBatchDeleteTopics)
		r.Get("/{topic}", s.handleGetTopic)
		r.Delete("/{topic}", s.handleDeleteTopic)
	})

	// Message routes
	s.router.Route("/api/topics/{topic}/messages", func(r chi.Router) {
		r.Get("/", s.handleGetMessages)
		r.Post("/", s.handleProduceMessage)
	})

	// Consumer Group routes
	s.router.Route("/api/consumer-groups", func(r chi.Router) {
		r.Get("/", s.handleListConsumerGroups)
		r.Get("/{group}", s.handleGetConsumerGroup)
	})

	// Monitoring routes
	s.router.Route("/api/monitoring", func(r chi.Router) {
		r.Get("/metrics", s.handleMonitoringMetrics)
		r.Get("/ws", s.handleMonitoringWebSocket)
	})

	// ACL routes
	s.router.Route("/api/acls", func(r chi.Router) {
		r.Get("/", s.handleListAcls)
		r.Post("/", s.handleCreateAcl)
		r.Delete("/", s.handleDeleteAcls)
	})

	// Configuration routes
	s.router.Route("/api/configs", func(r chi.Router) {
		r.Get("/cluster", s.handleGetClusterConfig)
		r.Put("/cluster", s.handleUpdateClusterConfig)
		r.Get("/topics/{topic}", s.handleGetTopicConfig)
		r.Put("/topics/{topic}", s.handleUpdateTopicConfig)
		r.Put("/topics", s.handleBatchUpdateTopicConfigs)
	})

	// Audit routes
	s.router.Route("/api/audit", func(r chi.Router) {
		r.Post("/logs", s.handleQueryAuditLogs)
		r.Get("/stats", s.handleGetAuditStats)
		r.Get("/events/{event_id}", s.handleGetAuditEvent)
		r.Get("/export", s.handleExportAuditLogs)
	})

	// Debug routes
	s.router.Route("/api/debug", func(r chi.Router) {
		r.Post("/bundle", s.handleGenerateDebugBundle)
		r.Get("/bundle/download", s.handleDownloadDebugBundle)
		r.Get("/system", s.handleDebugSystemInfo)
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("starting console API server", "addr", s.addr)
	return http.ListenAndServe(s.addr, s.router)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	s.logger.Info("shutting down console API server")
	if s.wsHub != nil {
		s.wsHub.Stop()
	}
}

// handleHealth godoc
// @Summary      Health check
// @Description  Check comprehensive health status of all components
// @Tags         Health
// @Produce      json
// @Success      200  {object}  HealthCheck
// @Router       /health [get]
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := s.healthChecker.Check()

	statusCode := http.StatusOK
	if health.Status == HealthStatusDegraded {
		statusCode = http.StatusOK // 200 for degraded but functional
	} else if health.Status == HealthStatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	s.respondJSON(w, statusCode, health)
}

// handleReadiness godoc
// @Summary      Readiness check
// @Description  Check if the service is ready to accept traffic (Kubernetes readiness probe)
// @Tags         Health
// @Produce      json
// @Success      200  {object}  map[string]bool
// @Failure      503  {object}  map[string]bool
// @Router       /health/ready [get]
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	ready := s.healthChecker.ReadinessCheck()

	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}

	s.respondJSON(w, statusCode, map[string]bool{
		"ready": ready,
	})
}

// handleLiveness godoc
// @Summary      Liveness check
// @Description  Check if the service is alive (Kubernetes liveness probe)
// @Tags         Health
// @Produce      json
// @Success      200  {object}  map[string]bool
// @Router       /health/live [get]
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	alive := s.healthChecker.LivenessCheck()
	s.respondJSON(w, http.StatusOK, map[string]bool{
		"alive": alive,
	})
}

// Topic handlers

// handleListTopics godoc
// @Summary      List all topics
// @Description  Get a list of all topics with their partition information
// @Tags         Topics
// @Produce      json
// @Success      200  {array}   TopicSummary
// @Security     ApiKeyAuth
// @Router       /topics [get]
func (s *Server) handleListTopics(w http.ResponseWriter, r *http.Request) {
	topics := s.topicManager.ListTopics()

	var response []TopicSummary
	for _, topicName := range topics {
		topic, exists := s.topicManager.GetTopic(topicName)
		if !exists {
			continue
		}

		var partitions []PartitionInfo
		for partID := range topic.Partitions {
			hwm, _ := topic.HighWaterMark(partID)
			partitions = append(partitions, PartitionInfo{
				ID:            partID,
				HighWaterMark: hwm,
			})
		}

		response = append(response, TopicSummary{
			Name:           topicName,
			PartitionCount: len(topic.Partitions),
			Partitions:     partitions,
		})
	}

	s.respondJSON(w, http.StatusOK, response)
}

// handleGetTopic godoc
// @Summary      Get topic details
// @Description  Get detailed information about a specific topic
// @Tags         Topics
// @Produce      json
// @Param        topic  path      string  true  "Topic name"
// @Success      200    {object}  TopicDetail
// @Failure      404    {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /topics/{topic} [get]
func (s *Server) handleGetTopic(w http.ResponseWriter, r *http.Request) {
	topicName := chi.URLParam(r, "topic")

	topic, exists := s.topicManager.GetTopic(topicName)
	if !exists {
		s.respondError(w, http.StatusNotFound, "topic not found")
		return
	}

	var partitions []PartitionInfo
	for partID := range topic.Partitions {
		hwm, _ := topic.HighWaterMark(partID)
		partitions = append(partitions, PartitionInfo{
			ID:            partID,
			HighWaterMark: hwm,
		})
	}

	response := TopicDetail{
		Name:           topicName,
		PartitionCount: len(topic.Partitions),
		Partitions:     partitions,
	}

	s.respondJSON(w, http.StatusOK, response)
}

// handleCreateTopic godoc
// @Summary      Create a new topic
// @Description  Create a new topic with the specified number of partitions
// @Tags         Topics
// @Accept       json
// @Produce      json
// @Param        request  body      CreateTopicRequest  true  "Topic creation request"
// @Success      201      {object}  map[string]string
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /topics [post]
func (s *Server) handleCreateTopic(w http.ResponseWriter, r *http.Request) {
	var req CreateTopicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		s.respondError(w, http.StatusBadRequest, "topic name is required")
		return
	}

	if req.Partitions <= 0 {
		s.respondError(w, http.StatusBadRequest, "partitions must be greater than 0")
		return
	}

	if err := s.topicManager.CreateTopic(req.Name, req.Partitions); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.BroadcastTopicCreated(req.Name, req.Partitions)

	s.respondJSON(w, http.StatusCreated, map[string]string{
		"name":       req.Name,
		"partitions": strconv.Itoa(int(req.Partitions)),
	})
}

// handleDeleteTopic godoc
// @Summary      Delete a topic
// @Description  Delete a topic and all its data
// @Tags         Topics
// @Param        topic  path  string  true  "Topic name"
// @Success      204    "No Content"
// @Failure      500    {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /topics/{topic} [delete]
func (s *Server) handleDeleteTopic(w http.ResponseWriter, r *http.Request) {
	topicName := chi.URLParam(r, "topic")

	if err := s.topicManager.DeleteTopic(topicName); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.BroadcastTopicDeleted(topicName)

	s.respondJSON(w, http.StatusOK, map[string]string{
		"message": "topic deleted successfully",
	})
}

// Message handlers

// handleGetMessages godoc
// @Summary      Get messages from a topic
// @Description  Fetch messages from a specific topic partition
// @Tags         Messages
// @Produce      json
// @Param        topic      path      string  true   "Topic name"
// @Param        partition  query     int     true   "Partition ID"
// @Param        offset     query     int     true   "Starting offset"
// @Param        limit      query     int     false  "Maximum number of messages to return" default(100)
// @Success      200        {array}   Message
// @Failure      400        {object}  map[string]string
// @Failure      404        {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /topics/{topic}/messages [get]
func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	topicName := chi.URLParam(r, "topic")
	partitionStr := r.URL.Query().Get("partition")
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")

	partition := int32(0)
	if partitionStr != "" {
		p, err := strconv.ParseInt(partitionStr, 10, 32)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, "invalid partition")
			return
		}
		partition = int32(p)
	}

	if partition < 0 {
		s.respondError(w, http.StatusBadRequest, "partition must be non-negative")
		return
	}

	offset := int64(0)
	if offsetStr != "" {
		o, err := strconv.ParseInt(offsetStr, 10, 64)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, "invalid offset")
			return
		}
		offset = o
	}

	if offset < 0 {
		s.respondError(w, http.StatusBadRequest, "offset must be non-negative")
		return
	}

	limit := 100
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = l
	}

	if limit <= 0 {
		s.respondError(w, http.StatusBadRequest, "limit must be greater than 0")
		return
	}

	topic, exists := s.topicManager.GetTopic(topicName)
	if !exists {
		s.respondError(w, http.StatusNotFound, "topic not found")
		return
	}

	var messages []Message
	for i := 0; i < limit; i++ {
		record, err := topic.Read(partition, offset+int64(i))
		if err != nil {
			break
		}

		messages = append(messages, Message{
			Partition: partition,
			Offset:    offset + int64(i),
			Key:       string(record.Key),
			Value:     string(record.Value),
			Timestamp: record.Timestamp,
		})
	}

	s.respondJSON(w, http.StatusOK, messages)
}

// handleProduceMessage godoc
// @Summary      Produce a message to a topic
// @Description  Send a message to a specific topic partition
// @Tags         Messages
// @Accept       json
// @Produce      json
// @Param        topic    path      string                  true  "Topic name"
// @Param        message  body      ProduceMessageRequest   true  "Message to produce"
// @Success      201      {object}  map[string]interface{}
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /topics/{topic}/messages [post]
func (s *Server) handleProduceMessage(w http.ResponseWriter, r *http.Request) {
	topicName := chi.URLParam(r, "topic")

	var req ProduceMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	topic, exists := s.topicManager.GetTopic(topicName)
	if !exists {
		s.respondError(w, http.StatusNotFound, "topic not found")
		return
	}

	partition := req.Partition
	if partition < 0 {
		partition = 0
	}

	offset, err := topic.Append(partition, []byte(req.Key), []byte(req.Value))
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"partition": partition,
		"offset":    offset,
	})
}

// Consumer Group handlers

// handleListConsumerGroups godoc
// @Summary      List all consumer groups
// @Description  Get a list of all consumer groups with their status
// @Tags         Consumer Groups
// @Produce      json
// @Success      200  {array}   ConsumerGroupSummary
// @Security     ApiKeyAuth
// @Router       /consumer-groups [get]
func (s *Server) handleListConsumerGroups(w http.ResponseWriter, r *http.Request) {
	groupIDs := s.coordinator.ListGroups()

	groups := make([]ConsumerGroupSummary, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		group, exists := s.coordinator.GetGroup(groupID)
		if !exists {
			continue
		}

		summary := ConsumerGroupSummary{
			GroupID: groupID,
			State:   string(group.State),
			Members: len(group.Members),
		}
		groups = append(groups, summary)
	}

	s.respondJSON(w, http.StatusOK, groups)
}

// handleGetConsumerGroup godoc
// @Summary      Get consumer group details
// @Description  Get detailed information about a specific consumer group
// @Tags         Consumer Groups
// @Produce      json
// @Param        group  path      string  true  "Consumer group ID"
// @Success      200    {object}  ConsumerGroupDetail
// @Failure      404    {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /consumer-groups/{group} [get]
func (s *Server) handleGetConsumerGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "group")

	group, exists := s.coordinator.GetGroup(groupID)
	if !exists {
		s.respondError(w, http.StatusNotFound, "consumer group not found: "+groupID)
		return
	}

	// Convert members
	members := make([]ConsumerGroupMember, 0, len(group.Members))
	for _, member := range group.Members {
		// Extract partition assignments from member assignment
		partitions := []int32{} // TODO: Parse assignment bytes

		members = append(members, ConsumerGroupMember{
			MemberID:   member.ID,
			ClientID:   member.ClientID,
			ClientHost: member.ClientHost,
			Partitions: partitions,
		})
	}

	// Convert offsets
	offsetCommits := make([]ConsumerGroupOffsetCommit, 0)
	for topic, partitions := range group.OffsetCommits {
		for partition, offset := range partitions {
			offsetCommits = append(offsetCommits, ConsumerGroupOffsetCommit{
				Topic:     topic,
				Partition: partition,
				Offset:    offset.Offset,
				Metadata:  offset.Metadata,
			})
		}
	}

	detail := ConsumerGroupDetail{
		GroupID:       groupID,
		State:         string(group.State),
		ProtocolType:  group.ProtocolType,
		Protocol:      group.ProtocolName,
		Members:       members,
		OffsetCommits: offsetCommits,
	}

	s.respondJSON(w, http.StatusOK, detail)
}

// Helper methods

func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{
		"error": message,
	})
}
