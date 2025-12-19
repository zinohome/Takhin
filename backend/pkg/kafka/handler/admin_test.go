// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestCreateTopics(t *testing.T) {
	dir := t.TempDir()

	// Create topic manager
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Create handler
	cfg := &config.Config{}
	h := New(cfg, topicMgr)

	// Create request
	req := &protocol.CreateTopicsRequest{
		Header: &protocol.RequestHeader{
			APIKey:        protocol.CreateTopicsKey,
			APIVersion:    0,
			CorrelationID: 1,
			ClientID:      "test-client",
		},
		Topics: []protocol.CreatableTopic{
			{
				Name:              "test-topic",
				NumPartitions:     3,
				ReplicationFactor: 1,
			},
		},
		TimeoutMs:    5000,
		ValidateOnly: false,
	}

	// Encode request manually
	var buf bytes.Buffer
	req.Header.Encode(&buf)
	protocol.WriteArray(&buf, len(req.Topics))
	for _, topic := range req.Topics {
		protocol.WriteString(&buf, topic.Name)
		protocol.WriteInt32(&buf, topic.NumPartitions)
		protocol.WriteInt16(&buf, topic.ReplicationFactor)
		protocol.WriteArray(&buf, 0) // assignments
		protocol.WriteArray(&buf, 0) // configs
	}
	protocol.WriteInt32(&buf, req.TimeoutMs)
	protocol.WriteBool(&buf, req.ValidateOnly)

	// Handle request
	resp, err := h.HandleRequest(buf.Bytes())
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Decode response header (correlation ID)
	respReader := bytes.NewReader(resp)
	correlationID, err := protocol.ReadInt32(respReader)
	require.NoError(t, err)
	assert.Equal(t, int32(1), correlationID)

	// Decode response body
	throttleTime, _ := protocol.ReadInt32(respReader)
	assert.Equal(t, int32(0), throttleTime)

	// Read topic results
	numResults, _ := protocol.ReadInt32(respReader)
	assert.Equal(t, int32(1), numResults)

	topicName, _ := protocol.ReadString(respReader)
	assert.Equal(t, "test-topic", topicName)

	errorCode, _ := protocol.ReadInt16(respReader)
	assert.Equal(t, int16(protocol.None), errorCode)

	// Verify topic was created
	topics := topicMgr.ListTopics()
	assert.Contains(t, topics, "test-topic")
}

func TestCreateTopicsValidateOnly(t *testing.T) {
	dir := t.TempDir()

	// Create topic manager
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Create handler
	cfg := &config.Config{}
	h := New(cfg, topicMgr)

	// Create request with validate-only
	req := &protocol.CreateTopicsRequest{
		Header: &protocol.RequestHeader{
			APIKey:        protocol.CreateTopicsKey,
			APIVersion:    0,
			CorrelationID: 1,
			ClientID:      "test-client",
		},
		Topics: []protocol.CreatableTopic{
			{
				Name:              "validate-topic",
				NumPartitions:     2,
				ReplicationFactor: 1,
			},
		},
		TimeoutMs:    5000,
		ValidateOnly: true,
	}

	// Encode request
	var buf bytes.Buffer
	req.Header.Encode(&buf)
	protocol.WriteArray(&buf, len(req.Topics))
	for _, topic := range req.Topics {
		protocol.WriteString(&buf, topic.Name)
		protocol.WriteInt32(&buf, topic.NumPartitions)
		protocol.WriteInt16(&buf, topic.ReplicationFactor)
		protocol.WriteArray(&buf, 0)
		protocol.WriteArray(&buf, 0)
	}
	protocol.WriteInt32(&buf, req.TimeoutMs)
	protocol.WriteBool(&buf, req.ValidateOnly)

	// Handle request
	resp, err := h.HandleRequest(buf.Bytes())
	require.NoError(t, err)

	// Decode response
	respReader := bytes.NewReader(resp)
	protocol.ReadInt32(respReader)  // correlation ID
	protocol.ReadInt32(respReader)  // throttle time
	protocol.ReadInt32(respReader)  // num results
	protocol.ReadString(respReader) // topic name
	errorCode, _ := protocol.ReadInt16(respReader)

	// Should succeed validation
	assert.Equal(t, int16(protocol.None), errorCode)

	// Verify topic was NOT created (validate-only)
	topics := topicMgr.ListTopics()
	assert.NotContains(t, topics, "validate-topic")
}

func TestCreateTopicsAlreadyExists(t *testing.T) {
	dir := t.TempDir()

	// Create topic manager
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Pre-create topic
	err := topicMgr.CreateTopic("existing-topic", 1)
	require.NoError(t, err)

	// Create handler
	cfg := &config.Config{}
	h := New(cfg, topicMgr)

	// Try to create same topic again
	req := &protocol.CreateTopicsRequest{
		Header: &protocol.RequestHeader{
			APIKey:        protocol.CreateTopicsKey,
			APIVersion:    0,
			CorrelationID: 1,
			ClientID:      "test-client",
		},
		Topics: []protocol.CreatableTopic{
			{
				Name:              "existing-topic",
				NumPartitions:     3,
				ReplicationFactor: 1,
			},
		},
		TimeoutMs:    5000,
		ValidateOnly: false,
	}

	// Encode request
	var buf bytes.Buffer
	req.Header.Encode(&buf)
	protocol.WriteArray(&buf, len(req.Topics))
	for _, topic := range req.Topics {
		protocol.WriteString(&buf, topic.Name)
		protocol.WriteInt32(&buf, topic.NumPartitions)
		protocol.WriteInt16(&buf, topic.ReplicationFactor)
		protocol.WriteArray(&buf, 0)
		protocol.WriteArray(&buf, 0)
	}
	protocol.WriteInt32(&buf, req.TimeoutMs)
	protocol.WriteBool(&buf, req.ValidateOnly)

	// Handle request
	resp, err := h.HandleRequest(buf.Bytes())
	require.NoError(t, err)

	// Decode response
	respReader := bytes.NewReader(resp)
	protocol.ReadInt32(respReader)  // correlation ID
	protocol.ReadInt32(respReader)  // throttle time
	protocol.ReadInt32(respReader)  // num results
	protocol.ReadString(respReader) // topic name
	errorCode, _ := protocol.ReadInt16(respReader)

	// Should get TopicAlreadyExists error
	assert.Equal(t, int16(protocol.TopicAlreadyExists), errorCode)
}

func TestDeleteTopics(t *testing.T) {
	dir := t.TempDir()

	// Create topic manager
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Pre-create topic
	err := topicMgr.CreateTopic("delete-me", 2)
	require.NoError(t, err)

	// Create handler
	cfg := &config.Config{}
	h := New(cfg, topicMgr)

	// Create delete request
	req := &protocol.DeleteTopicsRequest{
		Header: &protocol.RequestHeader{
			APIKey:        protocol.DeleteTopicsKey,
			APIVersion:    0,
			CorrelationID: 1,
			ClientID:      "test-client",
		},
		TopicNames: []string{"delete-me"},
		TimeoutMs:  5000,
	}

	// Encode request
	var buf bytes.Buffer
	req.Header.Encode(&buf)
	protocol.WriteArray(&buf, len(req.TopicNames))
	for _, name := range req.TopicNames {
		protocol.WriteString(&buf, name)
	}
	protocol.WriteInt32(&buf, req.TimeoutMs)

	// Handle request
	resp, err := h.HandleRequest(buf.Bytes())
	require.NoError(t, err)

	// Decode response
	respReader := bytes.NewReader(resp)
	protocol.ReadInt32(respReader) // correlation ID
	protocol.ReadInt32(respReader) // throttle time
	numResults, _ := protocol.ReadInt32(respReader)
	assert.Equal(t, int32(1), numResults)

	topicName, _ := protocol.ReadString(respReader)
	assert.Equal(t, "delete-me", topicName)

	errorCode, _ := protocol.ReadInt16(respReader)
	assert.Equal(t, int16(protocol.None), errorCode)

	// Verify topic was deleted
	topics := topicMgr.ListTopics()
	assert.NotContains(t, topics, "delete-me")
}

func TestDeleteTopicsNotFound(t *testing.T) {
	dir := t.TempDir()

	// Create topic manager
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Create handler
	cfg := &config.Config{}
	h := New(cfg, topicMgr)

	// Try to delete non-existent topic
	req := &protocol.DeleteTopicsRequest{
		Header: &protocol.RequestHeader{
			APIKey:        protocol.DeleteTopicsKey,
			APIVersion:    0,
			CorrelationID: 1,
			ClientID:      "test-client",
		},
		TopicNames: []string{"nonexistent"},
		TimeoutMs:  5000,
	}

	// Encode request
	var buf bytes.Buffer
	req.Header.Encode(&buf)
	protocol.WriteArray(&buf, len(req.TopicNames))
	for _, name := range req.TopicNames {
		protocol.WriteString(&buf, name)
	}
	protocol.WriteInt32(&buf, req.TimeoutMs)

	// Handle request
	resp, err := h.HandleRequest(buf.Bytes())
	require.NoError(t, err)

	// Decode response
	respReader := bytes.NewReader(resp)
	protocol.ReadInt32(respReader)  // correlation ID
	protocol.ReadInt32(respReader)  // throttle time
	protocol.ReadInt32(respReader)  // num results
	protocol.ReadString(respReader) // topic name
	errorCode, _ := protocol.ReadInt16(respReader)

	// Should get UnknownTopicOrPartition error
	assert.Equal(t, int16(protocol.UnknownTopicOrPartition), errorCode)
}

func TestDescribeConfigs(t *testing.T) {
	dir := t.TempDir()

	// Create topic manager and create a topic first
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	err := topicMgr.CreateTopic("config-topic", 1)
	require.NoError(t, err)

	// Create handler
	cfg := &config.Config{}
	h := New(cfg, topicMgr)

	// Create describe configs request
	req := &protocol.DescribeConfigsRequest{
		Header: &protocol.RequestHeader{
			APIKey:        protocol.DescribeConfigsKey,
			APIVersion:    0,
			CorrelationID: 1,
			ClientID:      "test-client",
		},
		Resources: []protocol.DescribeConfigsResource{
			{
				ResourceType: protocol.ResourceTypeTopic,
				ResourceName: "config-topic",
				ConfigNames:  nil, // nil means return all configs
			},
		},
	}

	// Encode request
	var buf bytes.Buffer
	req.Header.Encode(&buf)
	protocol.WriteArray(&buf, len(req.Resources))
	for _, resource := range req.Resources {
		protocol.WriteInt8(&buf, int8(resource.ResourceType))
		protocol.WriteString(&buf, resource.ResourceName)
		if resource.ConfigNames == nil {
			protocol.WriteArray(&buf, -1) // null array
		} else {
			protocol.WriteArray(&buf, len(resource.ConfigNames))
			for _, name := range resource.ConfigNames {
				protocol.WriteString(&buf, name)
			}
		}
	}

	// Handle request
	resp, err := h.HandleRequest(buf.Bytes())
	require.NoError(t, err)

	// Decode response
	respReader := bytes.NewReader(resp)
	protocol.ReadInt32(respReader) // correlation ID
	protocol.ReadInt32(respReader) // throttle time
	numResults, _ := protocol.ReadInt32(respReader)
	assert.Equal(t, int32(1), numResults)

	errorCode, _ := protocol.ReadInt16(respReader)
	assert.Equal(t, int16(protocol.None), errorCode)

	errorMsg, _ := protocol.ReadString(respReader)
	assert.Empty(t, errorMsg)

	resourceType, _ := protocol.ReadInt8(respReader)
	assert.Equal(t, int8(protocol.ResourceTypeTopic), resourceType)

	resourceName, _ := protocol.ReadString(respReader)
	assert.Equal(t, "config-topic", resourceName)

	numConfigs, _ := protocol.ReadInt32(respReader)
	assert.Greater(t, numConfigs, int32(0))

	// Verify we get expected configs
	configNames := make(map[string]bool)
	for i := int32(0); i < numConfigs; i++ {
		name, _ := protocol.ReadString(respReader)
		configNames[name] = true
		protocol.ReadString(respReader) // value
		protocol.ReadBool(respReader)   // readonly
		protocol.ReadBool(respReader)   // default
		protocol.ReadBool(respReader)   // sensitive
	}

	assert.True(t, configNames["compression.type"])
	assert.True(t, configNames["cleanup.policy"])
}

func TestDescribeConfigsTopicNotFound(t *testing.T) {
	dir := t.TempDir()

	// Create topic manager
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Create handler
	cfg := &config.Config{}
	h := New(cfg, topicMgr)

	// Try to describe configs for non-existent topic
	req := &protocol.DescribeConfigsRequest{
		Header: &protocol.RequestHeader{
			APIKey:        protocol.DescribeConfigsKey,
			APIVersion:    0,
			CorrelationID: 1,
			ClientID:      "test-client",
		},
		Resources: []protocol.DescribeConfigsResource{
			{
				ResourceType: protocol.ResourceTypeTopic,
				ResourceName: "nonexistent",
				ConfigNames:  nil,
			},
		},
	}

	// Encode request
	var buf bytes.Buffer
	req.Header.Encode(&buf)
	protocol.WriteArray(&buf, len(req.Resources))
	for _, resource := range req.Resources {
		protocol.WriteInt8(&buf, int8(resource.ResourceType))
		protocol.WriteString(&buf, resource.ResourceName)
		protocol.WriteArray(&buf, -1) // null
	}

	// Handle request
	resp, err := h.HandleRequest(buf.Bytes())
	require.NoError(t, err)

	// Decode response
	respReader := bytes.NewReader(resp)
	protocol.ReadInt32(respReader) // correlation ID
	protocol.ReadInt32(respReader) // throttle time
	protocol.ReadInt32(respReader) // num results

	errorCode, _ := protocol.ReadInt16(respReader)
	assert.Equal(t, int16(protocol.UnknownTopicOrPartition), errorCode)
}

func TestAdminAPIEndToEnd(t *testing.T) {
	dir := t.TempDir()

	// Create topic manager
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Create handler
	cfg := &config.Config{}
	h := New(cfg, topicMgr)

	// Step 1: Create topic
	createReq := &protocol.CreateTopicsRequest{
		Header: &protocol.RequestHeader{
			APIKey:        protocol.CreateTopicsKey,
			APIVersion:    0,
			CorrelationID: 1,
			ClientID:      "test-client",
		},
		Topics: []protocol.CreatableTopic{
			{
				Name:              "e2e-topic",
				NumPartitions:     2,
				ReplicationFactor: 1,
			},
		},
		TimeoutMs:    5000,
		ValidateOnly: false,
	}

	var buf bytes.Buffer
	createReq.Header.Encode(&buf)
	protocol.WriteArray(&buf, len(createReq.Topics))
	for _, topic := range createReq.Topics {
		protocol.WriteString(&buf, topic.Name)
		protocol.WriteInt32(&buf, topic.NumPartitions)
		protocol.WriteInt16(&buf, topic.ReplicationFactor)
		protocol.WriteArray(&buf, 0)
		protocol.WriteArray(&buf, 0)
	}
	protocol.WriteInt32(&buf, createReq.TimeoutMs)
	protocol.WriteBool(&buf, createReq.ValidateOnly)

	createResp, err := h.HandleRequest(buf.Bytes())
	require.NoError(t, err)
	require.NotNil(t, createResp)

	// Verify topic created
	topics := topicMgr.ListTopics()
	assert.Contains(t, topics, "e2e-topic")

	// Step 2: Describe configs
	describeReq := &protocol.DescribeConfigsRequest{
		Header: &protocol.RequestHeader{
			APIKey:        protocol.DescribeConfigsKey,
			APIVersion:    0,
			CorrelationID: 2,
			ClientID:      "test-client",
		},
		Resources: []protocol.DescribeConfigsResource{
			{
				ResourceType: protocol.ResourceTypeTopic,
				ResourceName: "e2e-topic",
				ConfigNames:  nil,
			},
		},
	}

	buf.Reset()
	describeReq.Header.Encode(&buf)
	protocol.WriteArray(&buf, len(describeReq.Resources))
	for _, resource := range describeReq.Resources {
		protocol.WriteInt8(&buf, int8(resource.ResourceType))
		protocol.WriteString(&buf, resource.ResourceName)
		protocol.WriteArray(&buf, -1)
	}

	describeResp, err := h.HandleRequest(buf.Bytes())
	require.NoError(t, err)
	require.NotNil(t, describeResp)

	// Step 3: Delete topic
	deleteReq := &protocol.DeleteTopicsRequest{
		Header: &protocol.RequestHeader{
			APIKey:        protocol.DeleteTopicsKey,
			APIVersion:    0,
			CorrelationID: 3,
			ClientID:      "test-client",
		},
		TopicNames: []string{"e2e-topic"},
		TimeoutMs:  5000,
	}

	buf.Reset()
	deleteReq.Header.Encode(&buf)
	protocol.WriteArray(&buf, len(deleteReq.TopicNames))
	for _, name := range deleteReq.TopicNames {
		protocol.WriteString(&buf, name)
	}
	protocol.WriteInt32(&buf, deleteReq.TimeoutMs)

	deleteResp, err := h.HandleRequest(buf.Bytes())
	require.NoError(t, err)
	require.NotNil(t, deleteResp)

	// Verify topic deleted
	topics = topicMgr.ListTopics()
	assert.NotContains(t, topics, "e2e-topic")
}
