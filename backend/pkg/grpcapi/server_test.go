// Copyright 2025 Takhin Data, Inc.

package grpcapi

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func setupTestServer(t *testing.T) *Server {
	dataDir := t.TempDir()
	topicManager := topic.NewManager(dataDir, 1024*1024)
	coord := coordinator.NewCoordinator()
	
	coord.Start()

	return NewServer(topicManager, coord, "test-1.0.0")
}

func TestCreateTopic(t *testing.T) {
	server := setupTestServer(t)

	tests := []struct {
		name      string
		req       *CreateTopicRequest
		wantError bool
	}{
		{
			name: "valid topic",
			req: &CreateTopicRequest{
				Name:              "test-topic",
				NumPartitions:     3,
				ReplicationFactor: 1,
			},
			wantError: false,
		},
		{
			name: "empty topic name",
			req: &CreateTopicRequest{
				Name:          "",
				NumPartitions: 1,
			},
			wantError: true,
		},
		{
			name: "default partitions",
			req: &CreateTopicRequest{
				Name:          "default-topic",
				NumPartitions: 0,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := server.CreateTopic(context.Background(), tt.req)
			require.NoError(t, err)

			if tt.wantError {
				assert.False(t, resp.Success)
				assert.NotEmpty(t, resp.Error)
			} else {
				assert.True(t, resp.Success)
				assert.Empty(t, resp.Error)
			}
		})
	}
}

func TestListTopics(t *testing.T) {
	server := setupTestServer(t)

	// Create some topics
	topics := []string{"topic1", "topic2", "topic3"}
	for _, topicName := range topics {
		_, err := server.CreateTopic(context.Background(), &CreateTopicRequest{
			Name:          topicName,
			NumPartitions: 1,
		})
		require.NoError(t, err)
	}

	// List topics
	resp, err := server.ListTopics(context.Background(), &ListTopicsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.Topics, len(topics))

	// Verify all topics are present
	topicSet := make(map[string]bool)
	for _, topic := range resp.Topics {
		topicSet[topic] = true
	}
	for _, topic := range topics {
		assert.True(t, topicSet[topic], "Topic %s not found", topic)
	}
}

func TestGetTopic(t *testing.T) {
	server := setupTestServer(t)

	// Create topic
	topicName := "test-topic"
	numPartitions := int32(5)
	_, err := server.CreateTopic(context.Background(), &CreateTopicRequest{
		Name:          topicName,
		NumPartitions: numPartitions,
	})
	require.NoError(t, err)

	// Get topic
	resp, err := server.GetTopic(context.Background(), &GetTopicRequest{
		Name: topicName,
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Error)
	assert.NotNil(t, resp.Topic)
	assert.Equal(t, topicName, resp.Topic.Name)
	assert.Equal(t, numPartitions, resp.Topic.NumPartitions)

	// Get non-existent topic
	resp, err = server.GetTopic(context.Background(), &GetTopicRequest{
		Name: "non-existent",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Error)
}

func TestProduceMessage(t *testing.T) {
	server := setupTestServer(t)

	// Create topic
	topicName := "produce-topic"
	_, err := server.CreateTopic(context.Background(), &CreateTopicRequest{
		Name:          topicName,
		NumPartitions: 3,
	})
	require.NoError(t, err)

	tests := []struct {
		name      string
		req       *ProduceMessageRequest
		wantError bool
	}{
		{
			name: "valid message",
			req: &ProduceMessageRequest{
				Topic:     topicName,
				Partition: 0,
				Record: &Record{
					Key:   []byte("key1"),
					Value: []byte("value1"),
				},
			},
			wantError: false,
		},
		{
			name: "auto partition selection",
			req: &ProduceMessageRequest{
				Topic:     topicName,
				Partition: -1,
				Record: &Record{
					Key:   []byte("key2"),
					Value: []byte("value2"),
				},
			},
			wantError: false,
		},
		{
			name: "invalid topic",
			req: &ProduceMessageRequest{
				Topic:     "non-existent",
				Partition: 0,
				Record: &Record{
					Value: []byte("value"),
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := server.ProduceMessage(context.Background(), tt.req)
			require.NoError(t, err)

			if tt.wantError {
				assert.NotEmpty(t, resp.Error)
			} else {
				assert.Empty(t, resp.Error)
				assert.Equal(t, tt.req.Topic, resp.Topic)
				assert.GreaterOrEqual(t, resp.Offset, int64(0))
			}
		})
	}
}

func TestDeleteTopic(t *testing.T) {
	server := setupTestServer(t)

	// Create topic
	topicName := "delete-topic"
	_, err := server.CreateTopic(context.Background(), &CreateTopicRequest{
		Name:          topicName,
		NumPartitions: 1,
	})
	require.NoError(t, err)

	// Delete topic
	resp, err := server.DeleteTopic(context.Background(), &DeleteTopicRequest{
		Name: topicName,
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Empty(t, resp.Error)

	// Verify topic is deleted
	getResp, err := server.GetTopic(context.Background(), &GetTopicRequest{
		Name: topicName,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, getResp.Error)
}

func TestListConsumerGroups(t *testing.T) {
	server := setupTestServer(t)

	// List consumer groups (should be empty initially)
	resp, err := server.ListConsumerGroups(context.Background(), &ListConsumerGroupsRequest{})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.Groups)
}

func TestGetPartitionOffsets(t *testing.T) {
	server := setupTestServer(t)

	// Create topic
	topicName := "offset-topic"
	_, err := server.CreateTopic(context.Background(), &CreateTopicRequest{
		Name:          topicName,
		NumPartitions: 1,
	})
	require.NoError(t, err)

	// Produce some messages
	for i := 0; i < 10; i++ {
		_, err := server.ProduceMessage(context.Background(), &ProduceMessageRequest{
			Topic:     topicName,
			Partition: 0,
			Record: &Record{
				Value: []byte("test message"),
			},
		})
		require.NoError(t, err)
	}

	// Get partition offsets
	resp, err := server.GetPartitionOffsets(context.Background(), &GetPartitionOffsetsRequest{
		Topic:     topicName,
		Partition: 0,
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Error)
	assert.Equal(t, topicName, resp.Topic)
	assert.Equal(t, int32(0), resp.Partition)
	assert.Equal(t, int64(0), resp.BeginningOffset)
	assert.Equal(t, int64(10), resp.EndOffset)
}

func TestHealthCheck(t *testing.T) {
	server := setupTestServer(t)

	// Wait a bit for uptime
	time.Sleep(200 * time.Millisecond)

	resp, err := server.HealthCheck(context.Background(), &HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, "healthy", resp.Status)
	assert.Equal(t, "test-1.0.0", resp.Version)
	assert.GreaterOrEqual(t, resp.UptimeSeconds, int64(0))
}

func TestDescribeTopics(t *testing.T) {
	server := setupTestServer(t)

	// Create topics
	topics := []string{"topic1", "topic2"}
	for _, topicName := range topics {
		_, err := server.CreateTopic(context.Background(), &CreateTopicRequest{
			Name:          topicName,
			NumPartitions: 2,
		})
		require.NoError(t, err)
	}

	// Describe topics
	resp, err := server.DescribeTopics(context.Background(), &DescribeTopicsRequest{
		Topics: topics,
	})
	require.NoError(t, err)
	assert.Len(t, resp.Topics, len(topics))

	for _, topicInfo := range resp.Topics {
		assert.Contains(t, topics, topicInfo.Name)
		assert.Equal(t, int32(2), topicInfo.NumPartitions)
	}
}
