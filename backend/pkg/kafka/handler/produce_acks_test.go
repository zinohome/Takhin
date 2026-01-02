package handler

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/kafka/protocol"
)

// TestProduceAcks0 tests produce with acks=0 (fire and forget)
func TestProduceAcks0(t *testing.T) {
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID: 1,
		},
		Storage: config.StorageConfig{
			DataDir: t.TempDir(),
		},
	}

	handler, cleanup := setupHandler(t, cfg)
	defer cleanup()

	// Create topic
	createTopic(t, handler, "test-topic", 1, 3)

	// Produce with acks=0
	req := &protocol.ProduceRequest{
		Acks:      0,
		TimeoutMs: 5000,
		Topics: []protocol.ProduceRequestTopic{
			{
				TopicName: "test-topic",
				Partitions: []protocol.ProduceRequestPartition{
					{
						PartitionIndex: 0,
						RecordBatch: &protocol.RecordBatch{
							BaseOffset: 0,
							Records: []protocol.Record{
								{Value: []byte("message-acks-0")},
							},
						},
					},
				},
			},
		},
	}

	resp, err := handler.handleProduceRequest(req, &protocol.RequestHeader{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Responses, 1)
	require.Len(t, resp.Responses[0].PartitionResponses, 1)

	partResp := resp.Responses[0].PartitionResponses[0]
	assert.Equal(t, protocol.None, partResp.ErrorCode, "acks=0 should succeed")
	assert.Equal(t, int64(0), partResp.BaseOffset)
}

// TestProduceAcks1 tests produce with acks=1 (leader acknowledgment)
func TestProduceAcks1(t *testing.T) {
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID: 1,
		},
		Storage: config.StorageConfig{
			DataDir: t.TempDir(),
		},
	}

	handler, cleanup := setupHandler(t, cfg)
	defer cleanup()

	// Create topic
	createTopic(t, handler, "test-topic", 1, 3)

	// Produce with acks=1
	req := &protocol.ProduceRequest{
		Acks:      1,
		TimeoutMs: 5000,
		Topics: []protocol.ProduceRequestTopic{
			{
				TopicName: "test-topic",
				Partitions: []protocol.ProduceRequestPartition{
					{
						PartitionIndex: 0,
						RecordBatch: &protocol.RecordBatch{
							BaseOffset: 0,
							Records: []protocol.Record{
								{Value: []byte("message-acks-1")},
							},
						},
					},
				},
			},
		},
	}

	resp, err := handler.handleProduceRequest(req, &protocol.RequestHeader{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Responses, 1)
	require.Len(t, resp.Responses[0].PartitionResponses, 1)

	partResp := resp.Responses[0].PartitionResponses[0]
	assert.Equal(t, protocol.None, partResp.ErrorCode, "acks=1 should succeed")
	assert.Equal(t, int64(0), partResp.BaseOffset)
}

// TestProduceAcksAllSingleBroker tests produce with acks=-1 in single-broker setup
func TestProduceAcksAllSingleBroker(t *testing.T) {
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID: 1,
		},
		Storage: config.StorageConfig{
			DataDir: t.TempDir(),
		},
	}

	handler, cleanup := setupHandler(t, cfg)
	defer cleanup()

	// Create topic with RF=3
	createTopic(t, handler, "test-topic", 1, 3)

	// Produce with acks=-1 (all ISR)
	req := &protocol.ProduceRequest{
		Acks:      -1,
		TimeoutMs: 5000,
		Topics: []protocol.ProduceRequestTopic{
			{
				TopicName: "test-topic",
				Partitions: []protocol.ProduceRequestPartition{
					{
						PartitionIndex: 0,
						RecordBatch: &protocol.RecordBatch{
							BaseOffset: 0,
							Records: []protocol.Record{
								{Value: []byte("message-acks-all")},
							},
						},
					},
				},
			},
		},
	}

	// In single-broker setup, ISR only contains leader
	// So acks=-1 should succeed immediately
	resp, err := handler.handleProduceRequest(req, &protocol.RequestHeader{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Responses, 1)
	require.Len(t, resp.Responses[0].PartitionResponses, 1)

	partResp := resp.Responses[0].PartitionResponses[0]
	assert.Equal(t, protocol.None, partResp.ErrorCode, "acks=-1 should succeed in single-broker")
	assert.Equal(t, int64(0), partResp.BaseOffset)
}

// TestProduceAcksAllWithISRWait tests acks=-1 waiting for ISR acknowledgment
func TestProduceAcksAllWithISRWait(t *testing.T) {
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			ClusterBrokers: []int32{1, 2, 3},
		},
		Storage: config.StorageConfig{
			DataDir: t.TempDir(),
		},
	}

	handler, cleanup := setupHandler(t, cfg)
	defer cleanup()

	// Create topic with RF=3
	createTopic(t, handler, "test-topic", 1, 3)

	// Get the topic to manipulate ISR
	topic, _ := handler.topicManager.GetTopic("test-topic")
	require.NotNil(t, topic)

	// Manually set ISR to include followers
	topic.SetISR(0, []int32{1, 2, 3})

	// Start produce in goroutine (will wait for ISR acks)
	doneCh := make(chan *protocol.ProduceResponse, 1)
	errCh := make(chan error, 1)

	go func() {
		req := &protocol.ProduceRequest{
			Acks:      -1,
			TimeoutMs: 2000, // 2 second timeout
			Topics: []protocol.ProduceRequestTopic{
				{
					TopicName: "test-topic",
					Partitions: []protocol.ProduceRequestPartition{
						{
							PartitionIndex: 0,
							RecordBatch: &protocol.RecordBatch{
								BaseOffset: 0,
								Records: []protocol.Record{
									{Value: []byte("wait-for-isr")},
								},
							},
						},
					},
				},
			},
		}

		resp, err := handler.handleProduceRequest(req, &protocol.RequestHeader{})
		if err != nil {
			errCh <- err
		} else {
			doneCh <- resp
		}
	}()

	// Wait a bit to ensure produce is waiting
	time.Sleep(100 * time.Millisecond)

	// Verify that produce is waiting
	waitCount := handler.produceWaiter.GetWaitingCount()
	assert.Greater(t, waitCount, 0, "should have waiting produce requests")

	// Simulate follower fetch that advances HWM
	// Follower 2 fetches and advances LEO
	topic.UpdateFollowerLEO(0, 2, 1) // LEO = 1 (offset 0 replicated)
	topic.UpdateFollowerLEO(0, 3, 1) // LEO = 1 (offset 0 replicated)

	// Get current HWM (minimum LEO among ISR)
	hwm := topic.GetHWM(0)

	// Notify HWM advancement
	handler.produceWaiter.NotifyHWMAdvanced("test-topic", 0, hwm)

	// Wait for produce to complete
	select {
	case resp := <-doneCh:
		require.NotNil(t, resp)
		require.Len(t, resp.Responses, 1)
		partResp := resp.Responses[0].PartitionResponses[0]
		assert.Equal(t, protocol.None, partResp.ErrorCode, "acks=-1 should succeed after HWM advances")
		assert.Equal(t, int64(0), partResp.BaseOffset)

	case err := <-errCh:
		t.Fatalf("produce failed: %v", err)

	case <-time.After(3 * time.Second):
		t.Fatal("produce did not complete within timeout")
	}
}

// TestProduceAcksAllTimeout tests acks=-1 timeout when ISR doesn't acknowledge
func TestProduceAcksAllTimeout(t *testing.T) {
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			ClusterBrokers: []int32{1, 2, 3},
		},
		Storage: config.StorageConfig{
			DataDir: t.TempDir(),
		},
	}

	handler, cleanup := setupHandler(t, cfg)
	defer cleanup()

	// Create topic
	createTopic(t, handler, "test-topic", 1, 3)

	// Get the topic to manipulate ISR
	topic, _ := handler.topicManager.GetTopic("test-topic")
	require.NotNil(t, topic)

	// Set ISR to include followers
	topic.SetISR(0, []int32{1, 2, 3})

	// Produce with acks=-1 and short timeout
	req := &protocol.ProduceRequest{
		Acks:      -1,
		TimeoutMs: 500, // 500ms timeout
		Topics: []protocol.ProduceRequestTopic{
			{
				TopicName: "test-topic",
				Partitions: []protocol.ProduceRequestPartition{
					{
						PartitionIndex: 0,
						RecordBatch: &protocol.RecordBatch{
							BaseOffset: 0,
							Records: []protocol.Record{
								{Value: []byte("timeout-test")},
							},
						},
					},
				},
			},
		},
	}

	// Don't simulate follower fetches - let it timeout
	start := time.Now()
	resp, err := handler.handleProduceRequest(req, &protocol.RequestHeader{})
	duration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Responses, 1)
	require.Len(t, resp.Responses[0].PartitionResponses, 1)

	partResp := resp.Responses[0].PartitionResponses[0]
	assert.Equal(t, protocol.RequestTimeout, partResp.ErrorCode, "should timeout waiting for ISR")
	assert.GreaterOrEqual(t, duration, 500*time.Millisecond, "should wait at least timeout duration")
	assert.Less(t, duration, 1*time.Second, "should not wait too long beyond timeout")
}

// TestProduceAcksAllNotEnoughReplicas tests NotEnoughReplicas error
func TestProduceAcksAllNotEnoughReplicas(t *testing.T) {
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			ClusterBrokers: []int32{1, 2, 3},
		},
		Storage: config.StorageConfig{
			DataDir: t.TempDir(),
		},
	}

	handler, cleanup := setupHandler(t, cfg)
	defer cleanup()

	// Create topic
	createTopic(t, handler, "test-topic", 1, 3)

	// Get the topic to manipulate ISR
	topic, _ := handler.topicManager.GetTopic("test-topic")
	require.NotNil(t, topic)

	// Set ISR to empty (simulating all followers down)
	topic.SetISR(0, []int32{}) // Empty ISR

	// Produce with acks=-1
	req := &protocol.ProduceRequest{
		Acks:      -1,
		TimeoutMs: 1000,
		Topics: []protocol.ProduceRequestTopic{
			{
				TopicName: "test-topic",
				Partitions: []protocol.ProduceRequestPartition{
					{
						PartitionIndex: 0,
						RecordBatch: &protocol.RecordBatch{
							BaseOffset: 0,
							Records: []protocol.Record{
								{Value: []byte("not-enough-replicas")},
							},
						},
					},
				},
			},
		},
	}

	resp, err := handler.handleProduceRequest(req, &protocol.RequestHeader{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Responses, 1)
	require.Len(t, resp.Responses[0].PartitionResponses, 1)

	partResp := resp.Responses[0].PartitionResponses[0]
	assert.Equal(t, protocol.NotEnoughReplicas, partResp.ErrorCode, "should return NotEnoughReplicas error")
}

// TestProduceAcksConcurrent tests concurrent produce requests with acks=-1
func TestProduceAcksConcurrent(t *testing.T) {
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			ClusterBrokers: []int32{1, 2, 3},
		},
		Storage: config.StorageConfig{
			DataDir: t.TempDir(),
		},
	}

	handler, cleanup := setupHandler(t, cfg)
	defer cleanup()

	// Create topic
	createTopic(t, handler, "test-topic", 1, 3)

	// Get the topic
	topic, _ := handler.topicManager.GetTopic("test-topic")
	require.NotNil(t, topic)

	// Set ISR to include followers
	topic.SetISR(0, []int32{1, 2, 3})

	// Start multiple concurrent producers
	numProducers := 5
	doneCh := make(chan int, numProducers)

	for i := 0; i < numProducers; i++ {
		go func(id int) {
			req := &protocol.ProduceRequest{
				Acks:      -1,
				TimeoutMs: 3000,
				Topics: []protocol.ProduceRequestTopic{
					{
						TopicName: "test-topic",
						Partitions: []protocol.ProduceRequestPartition{
							{
								PartitionIndex: 0,
								RecordBatch: &protocol.RecordBatch{
									BaseOffset: 0,
									Records: []protocol.Record{
										{Value: []byte("concurrent")},
									},
								},
							},
						},
					},
				},
			}

			resp, err := handler.handleProduceRequest(req, &protocol.RequestHeader{})
			if err == nil && resp != nil {
				doneCh <- id
			}
		}(i)
	}

	// Wait a bit for producers to start waiting
	time.Sleep(100 * time.Millisecond)

	// Simulate followers catching up
	topic.UpdateFollowerLEO(0, 2, 10)
	topic.UpdateFollowerLEO(0, 3, 10)

	hwm := topic.GetHWM(0)
	handler.produceWaiter.NotifyHWMAdvanced("test-topic", 0, hwm)

	// Wait for all producers to complete
	completed := 0
	timeout := time.After(5 * time.Second)
	for completed < numProducers {
		select {
		case <-doneCh:
			completed++
		case <-timeout:
			t.Fatalf("only %d/%d producers completed", completed, numProducers)
		}
	}

	assert.Equal(t, numProducers, completed, "all concurrent producers should complete")
}

// Helper function to setup handler
func setupHandler(t *testing.T, cfg *config.Config) (*Handler, func()) {
	handler := New(cfg)
	cleanup := func() {
		if handler != nil {
			handler.Close()
		}
	}
	return handler, cleanup
}

// Helper function to create topic
func createTopic(t *testing.T, handler *Handler, name string, partitions int32, replicationFactor int32) {
	req := &protocol.CreateTopicsRequest{
		Topics: []protocol.CreateTopicsRequestTopic{
			{
				Name:              name,
				NumPartitions:     partitions,
				ReplicationFactor: replicationFactor,
			},
		},
	}

	var buf bytes.Buffer
	err := req.Encode(&buf, 0)
	require.NoError(t, err)

	header := &protocol.RequestHeader{
		RequestAPIKey:     protocol.CreateTopics,
		RequestAPIVersion: 0,
	}

	_, err = handler.handleCreateTopics(&buf, header)
	require.NoError(t, err)
}

// Helper function to handle produce request
func (h *Handler) handleProduceRequest(req *protocol.ProduceRequest, header *protocol.RequestHeader) (*protocol.ProduceResponse, error) {
	var buf bytes.Buffer
	if err := req.Encode(&buf, 0); err != nil {
		return nil, err
	}

	respBytes, err := h.handleProduce(&buf, header)
	if err != nil {
		return nil, err
	}

	// Decode response
	respBuf := bytes.NewReader(respBytes)

	// Skip response header
	respHeader := &protocol.ResponseHeader{}
	if err := respHeader.Decode(respBuf); err != nil {
		return nil, err
	}

	// Decode produce response
	resp := &protocol.ProduceResponse{}
	if err := resp.Decode(respBuf, 0); err != nil {
		return nil, err
	}

	return resp, nil
}
