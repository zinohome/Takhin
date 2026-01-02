// Copyright 2025 Takhin Data, Inc.

package grpcapi

import (
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/logger"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

// Server implements the Takhin gRPC service
type Server struct {
	topicManager *topic.Manager
	coordinator  *coordinator.Coordinator
	logger       *logger.Logger
	startTime    time.Time
	version      string
}

// NewServer creates a new gRPC API server
func NewServer(topicManager *topic.Manager, coord *coordinator.Coordinator, version string) *Server {
	return &Server{
		topicManager: topicManager,
		coordinator:  coord,
		logger:       logger.Default().WithComponent("grpc-api"),
		startTime:    time.Now(),
		version:      version,
	}
}

// CreateTopic creates a new topic
func (s *Server) CreateTopic(ctx context.Context, req *CreateTopicRequest) (*CreateTopicResponse, error) {
	s.logger.Info("CreateTopic", "topic", req.Name, "partitions", req.NumPartitions)

	if req.Name == "" {
		return &CreateTopicResponse{
			Success: false,
			Error:   "topic name is required",
		}, nil
	}

	if req.NumPartitions <= 0 {
		req.NumPartitions = 1
	}

	if req.ReplicationFactor <= 0 {
		req.ReplicationFactor = 1
	}

	err := s.topicManager.CreateTopic(req.Name, req.NumPartitions)
	if err != nil {
		s.logger.Error("Failed to create topic", "topic", req.Name, "error", err)
		return &CreateTopicResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &CreateTopicResponse{
		Success: true,
	}, nil
}

// DeleteTopic deletes a topic
func (s *Server) DeleteTopic(ctx context.Context, req *DeleteTopicRequest) (*DeleteTopicResponse, error) {
	s.logger.Info("DeleteTopic", "topic", req.Name)

	if req.Name == "" {
		return &DeleteTopicResponse{
			Success: false,
			Error:   "topic name is required",
		}, nil
	}

	err := s.topicManager.DeleteTopic(req.Name)
	if err != nil {
		s.logger.Error("Failed to delete topic", "topic", req.Name, "error", err)
		return &DeleteTopicResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &DeleteTopicResponse{
		Success: true,
	}, nil
}

// ListTopics lists all topics
func (s *Server) ListTopics(ctx context.Context, req *ListTopicsRequest) (*ListTopicsResponse, error) {
	s.logger.Debug("ListTopics")

	topics := s.topicManager.ListTopics()
	return &ListTopicsResponse{
		Topics: topics,
	}, nil
}

// GetTopic gets topic metadata
func (s *Server) GetTopic(ctx context.Context, req *GetTopicRequest) (*GetTopicResponse, error) {
	s.logger.Debug("GetTopic", "topic", req.Name)

	if req.Name == "" {
		return &GetTopicResponse{
			Error: "topic name is required",
		}, nil
	}

	t, ok := s.topicManager.GetTopic(req.Name)
	if !ok {
		return &GetTopicResponse{
			Error: fmt.Sprintf("topic '%s' not found", req.Name),
		}, nil
	}

	topicInfo := s.buildTopicInfo(t)
	return &GetTopicResponse{
		Topic: topicInfo,
	}, nil
}

// DescribeTopics describes multiple topics
func (s *Server) DescribeTopics(ctx context.Context, req *DescribeTopicsRequest) (*DescribeTopicsResponse, error) {
	s.logger.Debug("DescribeTopics", "count", len(req.Topics))

	var topicInfos []*TopicInfo
	for _, topicName := range req.Topics {
		t, ok := s.topicManager.GetTopic(topicName)
		if ok {
			topicInfos = append(topicInfos, s.buildTopicInfo(t))
		}
	}

	return &DescribeTopicsResponse{
		Topics: topicInfos,
	}, nil
}

// ProduceMessage produces a single message
func (s *Server) ProduceMessage(ctx context.Context, req *ProduceMessageRequest) (*ProduceMessageResponse, error) {
	s.logger.Debug("ProduceMessage", "topic", req.Topic, "partition", req.Partition)

	if req.Topic == "" {
		return &ProduceMessageResponse{
			Error: "topic is required",
		}, nil
	}

	t, ok := s.topicManager.GetTopic(req.Topic)
	if !ok {
		return &ProduceMessageResponse{
			Error: fmt.Sprintf("topic '%s' not found", req.Topic),
		}, nil
	}

	// Select partition
	partition := req.Partition
	if partition == -1 {
		// Simple round-robin partition selection
		partition = 0
		if len(t.Partitions) > 0 {
			partition = int32(time.Now().UnixNano() % int64(len(t.Partitions)))
		}
	}

	// Validate partition
	logPartition := t.Partitions[partition]
	if logPartition == nil {
		return &ProduceMessageResponse{
			Error: fmt.Sprintf("partition %d not found", partition),
		}, nil
	}

	// Create record batch  
	// Append to log (key-value interface)
	offset, err := logPartition.Append(req.Record.Key, req.Record.Value)
	if err != nil {
		s.logger.Error("Failed to append to log", "topic", req.Topic, "partition", partition, "error", err)
		return &ProduceMessageResponse{
			Error: err.Error(),
		}, nil
	}

	return &ProduceMessageResponse{
		Topic:     req.Topic,
		Partition: partition,
		Offset:    offset,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// ProduceMessageStream handles streaming produce requests
func (s *Server) ProduceMessageStream(stream TakhinService_ProduceMessageStreamServer) error {
	s.logger.Debug("ProduceMessageStream started")

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			s.logger.Debug("ProduceMessageStream ended")
			return nil
		}
		if err != nil {
			s.logger.Error("ProduceMessageStream receive error", "error", err)
			return status.Errorf(codes.Unknown, "failed to receive: %v", err)
		}

		// Process the message
		resp, err := s.ProduceMessage(context.Background(), req)
		if err != nil {
			return err
		}

		// Send response
		if err := stream.Send(resp); err != nil {
			s.logger.Error("ProduceMessageStream send error", "error", err)
			return status.Errorf(codes.Unknown, "failed to send: %v", err)
		}
	}
}

// ConsumeMessages streams messages from a topic partition
func (s *Server) ConsumeMessages(req *ConsumeMessagesRequest, stream TakhinService_ConsumeMessagesServer) error {
	s.logger.Debug("ConsumeMessages", "topic", req.Topic, "partition", req.Partition, "offset", req.Offset)

	if req.Topic == "" {
		return status.Errorf(codes.InvalidArgument, "topic is required")
	}

	t, ok := s.topicManager.GetTopic(req.Topic)
	if !ok {
		return status.Errorf(codes.NotFound, "topic '%s' not found", req.Topic)
	}

	logPartition := t.Partitions[req.Partition]
	if logPartition == nil {
		return status.Errorf(codes.NotFound, "partition %d not found", req.Partition)
	}

	// Determine starting offset
	offset := req.Offset
	if offset == -1 {
		offset = 0 // Earliest
	} else if offset == -2 {
		offset = logPartition.HighWaterMark() // Latest
	}

	// Read records in a loop (simplified streaming)
	var pbRecords []*Record
	maxBytes := req.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 1024 * 1024 // 1MB default
	}
	
	// Read up to 1000 records or maxBytes
	currentOffset := offset
	bytesRead := 0
	for i := 0; i < 1000 && int32(bytesRead) < maxBytes; i++ {
		record, err := logPartition.Read(currentOffset)
		if err != nil {
			// No more records
			break
		}
		
		pbRecords = append(pbRecords, &Record{
			Key:       record.Key,
			Value:     record.Value,
			Offset:    record.Offset,
			Partition: req.Partition,
			Timestamp: record.Timestamp,
		})
		
		bytesRead += len(record.Key) + len(record.Value)
		currentOffset++
	}

	resp := &ConsumeMessagesResponse{
		Topic:         req.Topic,
		Partition:     req.Partition,
		HighWatermark: logPartition.HighWaterMark(),
		Records:       pbRecords,
	}

	if err := stream.Send(resp); err != nil {
		s.logger.Error("ConsumeMessages send error", "error", err)
		return status.Errorf(codes.Unknown, "failed to send: %v", err)
	}

	return nil
}

// CommitOffset commits consumer group offsets
func (s *Server) CommitOffset(ctx context.Context, req *CommitOffsetRequest) (*CommitOffsetResponse, error) {
	s.logger.Debug("CommitOffset", "group", req.GroupId, "offsets", len(req.Offsets))

	if req.GroupId == "" {
		return &CommitOffsetResponse{
			Success: false,
			Error:   "group_id is required",
		}, nil
	}

	// Commit each offset
	for _, tpo := range req.Offsets {
		err := s.coordinator.CommitOffset(req.GroupId, tpo.Topic, tpo.Partition, tpo.Offset, "")
		if err != nil {
			s.logger.Error("Failed to commit offset", "group", req.GroupId, "topic", tpo.Topic, "error", err)
			return &CommitOffsetResponse{
				Success: false,
				Error:   err.Error(),
			}, nil
		}
	}

	return &CommitOffsetResponse{
		Success: true,
	}, nil
}

// ListConsumerGroups lists all consumer groups
func (s *Server) ListConsumerGroups(ctx context.Context, req *ListConsumerGroupsRequest) (*ListConsumerGroupsResponse, error) {
	s.logger.Debug("ListConsumerGroups")

	groups := s.coordinator.ListGroups()
	var groupInfos []*ConsumerGroupInfo

	for _, groupID := range groups {
		group, ok := s.coordinator.GetGroup(groupID)
		if ok {
			groupInfos = append(groupInfos, &ConsumerGroupInfo{
				GroupId:      groupID,
				State:        string(group.State),
				ProtocolType: group.ProtocolType,
				MemberCount:  int32(len(group.Members)),
			})
		}
	}

	return &ListConsumerGroupsResponse{
		Groups: groupInfos,
	}, nil
}

// DescribeConsumerGroup describes a consumer group
func (s *Server) DescribeConsumerGroup(ctx context.Context, req *DescribeConsumerGroupRequest) (*DescribeConsumerGroupResponse, error) {
	s.logger.Debug("DescribeConsumerGroup", "group", req.GroupId)

	if req.GroupId == "" {
		return &DescribeConsumerGroupResponse{
			Error: "group_id is required",
		}, nil
	}

	group, ok := s.coordinator.GetGroup(req.GroupId)
	if !ok {
		return &DescribeConsumerGroupResponse{
			Error: fmt.Sprintf("group '%s' not found", req.GroupId),
		}, nil
	}

	var members []*ConsumerGroupMember
	for _, member := range group.Members {
		members = append(members, &ConsumerGroupMember{
			MemberId:   member.ID,
			ClientId:   member.ClientID,
			ClientHost: member.ClientHost,
		})
	}

	return &DescribeConsumerGroupResponse{
		GroupId:      req.GroupId,
		State:        string(group.State),
		ProtocolType: group.ProtocolType,
		Members:      members,
	}, nil
}

// DeleteConsumerGroup deletes a consumer group
func (s *Server) DeleteConsumerGroup(ctx context.Context, req *DeleteConsumerGroupRequest) (*DeleteConsumerGroupResponse, error) {
	s.logger.Info("DeleteConsumerGroup", "group", req.GroupId)

	if req.GroupId == "" {
		return &DeleteConsumerGroupResponse{
			Success: false,
			Error:   "group_id is required",
		}, nil
	}

	s.coordinator.DeleteGroup(req.GroupId)

	return &DeleteConsumerGroupResponse{
		Success: true,
	}, nil
}

// GetPartitionOffsets gets partition offset information
func (s *Server) GetPartitionOffsets(ctx context.Context, req *GetPartitionOffsetsRequest) (*GetPartitionOffsetsResponse, error) {
	s.logger.Debug("GetPartitionOffsets", "topic", req.Topic, "partition", req.Partition)

	if req.Topic == "" {
		return &GetPartitionOffsetsResponse{
			Error: "topic is required",
		}, nil
	}

	t, ok := s.topicManager.GetTopic(req.Topic)
	if !ok {
		return &GetPartitionOffsetsResponse{
			Error: fmt.Sprintf("topic '%s' not found", req.Topic),
		}, nil
	}

	logPartition := t.Partitions[req.Partition]
	if logPartition == nil {
		return &GetPartitionOffsetsResponse{
			Error: fmt.Sprintf("partition %d not found", req.Partition),
		}, nil
	}

	return &GetPartitionOffsetsResponse{
		Topic:           req.Topic,
		Partition:       req.Partition,
		BeginningOffset: 0,
		EndOffset:       logPartition.HighWaterMark(),
	}, nil
}

// HealthCheck performs a health check
func (s *Server) HealthCheck(ctx context.Context, req *HealthCheckRequest) (*HealthCheckResponse, error) {
	uptime := time.Since(s.startTime).Seconds()

	return &HealthCheckResponse{
		Status:        "healthy",
		Version:       s.version,
		UptimeSeconds: int64(uptime),
	}, nil
}

// Helper methods

func (s *Server) buildTopicInfo(t *topic.Topic) *TopicInfo {
	partitions := make(map[int32]*PartitionInfo)
	for partID, logPartition := range t.Partitions {
		replicas := t.GetReplicas(partID)
		isr := t.GetISR(partID)

		partitions[partID] = &PartitionInfo{
			PartitionId:     partID,
			BeginningOffset: 0,
			EndOffset:       logPartition.HighWaterMark(),
			Leader:          0, // Single node for now
			Replicas:        replicas,
			Isr:             isr,
		}
	}

	return &TopicInfo{
		Name:              t.Name,
		NumPartitions:     int32(len(t.Partitions)),
		ReplicationFactor: int32(t.ReplicationFactor),
		Partitions:        partitions,
	}
}
