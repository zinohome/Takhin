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

// TestMultiBrokerReplicaAssignment 测试多 broker 环境的副本分配
func TestMultiBrokerReplicaAssignment(t *testing.T) {
	tests := []struct {
		name              string
		clusterBrokers    []int
		numPartitions     int32
		replicationFactor int16
		expectedLeaders   []int32 // 期望的 leader 分布
	}{
		{
			name:              "3 brokers, 3 partitions, RF=3",
			clusterBrokers:    []int{1, 2, 3},
			numPartitions:     3,
			replicationFactor: 3,
			expectedLeaders:   []int32{1, 2, 3}, // Round-robin leaders
		},
		{
			name:              "3 brokers, 6 partitions, RF=2",
			clusterBrokers:    []int{1, 2, 3},
			numPartitions:     6,
			replicationFactor: 2,
			expectedLeaders:   []int32{1, 2, 3, 1, 2, 3}, // Balanced leaders
		},
		{
			name:              "5 brokers, 10 partitions, RF=3",
			clusterBrokers:    []int{1, 2, 3, 4, 5},
			numPartitions:     10,
			replicationFactor: 3,
			expectedLeaders:   []int32{1, 2, 3, 4, 5, 1, 2, 3, 4, 5},
		},
		{
			name:              "single broker, 3 partitions, RF=1",
			clusterBrokers:    []int{1},
			numPartitions:     3,
			replicationFactor: 1,
			expectedLeaders:   []int32{1, 1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Kafka: config.KafkaConfig{
					BrokerID:       tt.clusterBrokers[0],
					AdvertisedHost: "localhost",
					AdvertisedPort: 9092,
					ClusterBrokers: tt.clusterBrokers,
				},
				Replication: config.ReplicationConfig{
					DefaultReplicationFactor: tt.replicationFactor,
				},
			}

			topicMgr := topic.NewManager(t.TempDir(), 1024*1024)
			h := New(cfg, topicMgr)

			// 创建 topic
			var buf bytes.Buffer

			header := &protocol.RequestHeader{
				APIKey:        protocol.CreateTopicsKey,
				APIVersion:    0,
				CorrelationID: 1,
				ClientID:      "test-client",
			}
			header.Encode(&buf)

			// Encode CreateTopics request
			protocol.WriteArray(&buf, 1) // 1 topic
			protocol.WriteString(&buf, "test-topic")
			protocol.WriteInt32(&buf, tt.numPartitions)
			protocol.WriteInt16(&buf, tt.replicationFactor)
			protocol.WriteArray(&buf, 0)     // no manual assignments
			protocol.WriteArray(&buf, 0)     // no configs
			protocol.WriteInt32(&buf, 30000) // timeout
			protocol.WriteBool(&buf, false)  // not validate-only

			resp, err := h.HandleRequest(buf.Bytes())
			require.NoError(t, err)
			require.Greater(t, len(resp), 0)

			// 验证副本分配
			createdTopic, exists := topicMgr.GetTopic("test-topic")
			require.True(t, exists)

			// 验证 leader 分布
			for partID := int32(0); partID < tt.numPartitions; partID++ {
				replicas := createdTopic.GetReplicas(partID)
				require.NotNil(t, replicas, "partition %d should have replicas", partID)
				require.Len(t, replicas, int(tt.replicationFactor), "partition %d should have %d replicas", partID, tt.replicationFactor)

				// Leader is first replica
				leader := replicas[0]
				assert.Equal(t, tt.expectedLeaders[partID], leader, "partition %d leader mismatch", partID)

				// 验证没有重复副本
				seen := make(map[int32]bool)
				for _, replicaID := range replicas {
					assert.False(t, seen[replicaID], "partition %d has duplicate replica %d", partID, replicaID)
					seen[replicaID] = true
				}

				// 验证所有副本都在集群中
				for _, replicaID := range replicas {
					found := false
					for _, brokerID := range tt.clusterBrokers {
						if int32(brokerID) == replicaID {
							found = true
							break
						}
					}
					assert.True(t, found, "replica %d not in cluster broker list", replicaID)
				}
			}
		})
	}
}

// TestReplicaDistribution 测试副本分布的均衡性
func TestReplicaDistribution(t *testing.T) {
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			AdvertisedHost: "localhost",
			AdvertisedPort: 9092,
			ClusterBrokers: []int{1, 2, 3, 4, 5},
		},
		Replication: config.ReplicationConfig{
			DefaultReplicationFactor: 3,
		},
	}

	topicMgr := topic.NewManager(t.TempDir(), 1024*1024)
	h := New(cfg, topicMgr)

	// 创建包含大量分区的 topic
	var buf bytes.Buffer

	header := &protocol.RequestHeader{
		APIKey:        protocol.CreateTopicsKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}
	header.Encode(&buf)

	numPartitions := int32(50)
	replicationFactor := int16(3)

	protocol.WriteArray(&buf, 1) // 1 topic
	protocol.WriteString(&buf, "test-topic")
	protocol.WriteInt32(&buf, numPartitions)
	protocol.WriteInt16(&buf, replicationFactor)
	protocol.WriteArray(&buf, 0)     // no manual assignments
	protocol.WriteArray(&buf, 0)     // no configs
	protocol.WriteInt32(&buf, 30000) // timeout
	protocol.WriteBool(&buf, false)  // not validate-only

	resp, err := h.HandleRequest(buf.Bytes())
	require.NoError(t, err)
	require.Greater(t, len(resp), 0)

	// 统计每个 broker 作为 leader 的次数
	leaderCounts := make(map[int32]int)
	replicaCounts := make(map[int32]int)

	createdTopic, exists := topicMgr.GetTopic("test-topic")
	require.True(t, exists)

	for partID := int32(0); partID < numPartitions; partID++ {
		replicas := createdTopic.GetReplicas(partID)
		require.NotNil(t, replicas)

		// Leader is first replica
		leader := replicas[0]
		leaderCounts[leader]++

		// Count all replicas
		for _, replicaID := range replicas {
			replicaCounts[replicaID]++
		}
	}

	// 验证 leader 分布均衡（每个 broker 应该有接近的 leader 数量）
	expectedLeadersPerBroker := float64(numPartitions) / float64(len(cfg.Kafka.ClusterBrokers))
	tolerance := expectedLeadersPerBroker * 0.2 // 20% tolerance

	for brokerID, count := range leaderCounts {
		diff := float64(count) - expectedLeadersPerBroker
		if diff < 0 {
			diff = -diff
		}
		assert.LessOrEqual(t, diff, tolerance,
			"broker %d has %d leaders, expected ~%.1f (tolerance ±%.1f)",
			brokerID, count, expectedLeadersPerBroker, tolerance)
	}

	// 验证副本分布均衡
	expectedReplicasPerBroker := float64(numPartitions*int32(replicationFactor)) / float64(len(cfg.Kafka.ClusterBrokers))
	replicaTolerance := expectedReplicasPerBroker * 0.1 // 10% tolerance

	for brokerID, count := range replicaCounts {
		diff := float64(count) - expectedReplicasPerBroker
		if diff < 0 {
			diff = -diff
		}
		assert.LessOrEqual(t, diff, replicaTolerance,
			"broker %d has %d replicas, expected ~%.1f (tolerance ±%.1f)",
			brokerID, count, expectedReplicasPerBroker, replicaTolerance)
	}

	t.Logf("Leader distribution: %v", leaderCounts)
	t.Logf("Replica distribution: %v", replicaCounts)
}

// TestBuildBrokerList 测试 buildBrokerList 辅助函数
func TestBuildBrokerList(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.Config
		expectedResult []int32
	}{
		{
			name: "with cluster brokers configured",
			config: &config.Config{
				Kafka: config.KafkaConfig{
					BrokerID:       1,
					ClusterBrokers: []int{1, 2, 3, 4, 5},
				},
			},
			expectedResult: []int32{1, 2, 3, 4, 5},
		},
		{
			name: "without cluster brokers (single broker)",
			config: &config.Config{
				Kafka: config.KafkaConfig{
					BrokerID:       2,
					ClusterBrokers: []int{},
				},
			},
			expectedResult: []int32{2},
		},
		{
			name: "nil cluster brokers",
			config: &config.Config{
				Kafka: config.KafkaConfig{
					BrokerID:       3,
					ClusterBrokers: nil,
				},
			},
			expectedResult: []int32{3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildBrokerList(tt.config)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// TestReplicationFactorExceedsBrokers 测试副本因子超过 broker 数量的错误处理
func TestReplicationFactorExceedsBrokers(t *testing.T) {
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			AdvertisedHost: "localhost",
			AdvertisedPort: 9092,
			ClusterBrokers: []int{1, 2}, // Only 2 brokers
		},
		Replication: config.ReplicationConfig{
			DefaultReplicationFactor: 3, // But RF=3
		},
	}

	topicMgr := topic.NewManager(t.TempDir(), 1024*1024)
	h := New(cfg, topicMgr)

	// 创建 topic
	var buf bytes.Buffer

	header := &protocol.RequestHeader{
		APIKey:        protocol.CreateTopicsKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}
	header.Encode(&buf)

	protocol.WriteArray(&buf, 1) // 1 topic
	protocol.WriteString(&buf, "test-topic")
	protocol.WriteInt32(&buf, 3) // 3 partitions
	protocol.WriteInt16(&buf, 3) // RF=3 (exceeds brokers)
	protocol.WriteArray(&buf, 0) // no manual assignments
	protocol.WriteArray(&buf, 0) // no configs
	protocol.WriteInt32(&buf, 30000)
	protocol.WriteBool(&buf, false)

	resp, err := h.HandleRequest(buf.Bytes())
	require.NoError(t, err)

	// Topic 应该被创建但副本分配会失败
	// 验证没有崩溃并返回响应
	require.Greater(t, len(resp), 0)
}
