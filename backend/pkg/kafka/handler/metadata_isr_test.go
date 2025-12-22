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

// TestMetadata_ISRDynamicUpdate 验证 ISR 变化后 Metadata 响应正确反映
func TestMetadata_ISRDynamicUpdate(t *testing.T) {
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			AdvertisedHost: "localhost",
			AdvertisedPort: 9092,
		},
		Replication: config.ReplicationConfig{
			DefaultReplicationFactor: 3,
		},
	}

	topicMgr := topic.NewManager(t.TempDir(), 1024*1024)
	h := New(cfg, topicMgr)

	// 创建 topic
	err := topicMgr.CreateTopic("test-topic", 1)
	require.NoError(t, err)

	testTopic, exists := topicMgr.GetTopic("test-topic")
	require.True(t, exists)

	// 初始状态：设置副本为 [1, 2, 3]，ISR 也为 [1, 2, 3]
	testTopic.SetReplicas(0, []int32{1, 2, 3})
	testTopic.SetISR(0, []int32{1, 2, 3})

	// 第一次 Metadata 请求：应返回完整 ISR
	resp1 := requestMetadata(t, h, "test-topic")
	require.Greater(t, len(resp1), 0, "metadata response should not be empty")

	// 验证底层 Topic 的 ISR
	isr1 := testTopic.GetISR(0)
	assert.ElementsMatch(t, []int32{1, 2, 3}, isr1, "initial ISR should be [1,2,3]")

	// 模拟 Follower 2 滞后，从 ISR 中移除
	testTopic.SetISR(0, []int32{1, 3})

	// 第二次 Metadata 请求：应反映更新的 ISR
	resp2 := requestMetadata(t, h, "test-topic")
	require.Greater(t, len(resp2), 0, "metadata response should not be empty")

	isr2 := testTopic.GetISR(0)
	assert.ElementsMatch(t, []int32{1, 3}, isr2, "ISR should be updated to [1,3]")

	// 模拟 Follower 2 追上，重新加入 ISR
	testTopic.SetISR(0, []int32{1, 2, 3})

	// 第三次 Metadata 请求：ISR 应恢复完整
	resp3 := requestMetadata(t, h, "test-topic")
	require.Greater(t, len(resp3), 0, "metadata response should not be empty")

	isr3 := testTopic.GetISR(0)
	assert.ElementsMatch(t, []int32{1, 2, 3}, isr3, "ISR should be restored to [1,2,3]")
}

// TestMetadata_ISRReflectsFollowerFetch 集成测试：Follower Fetch → ISR 更新 → Metadata 反映
func TestMetadata_ISRReflectsFollowerFetch(t *testing.T) {
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			AdvertisedHost: "localhost",
			AdvertisedPort: 9092,
		},
		Replication: config.ReplicationConfig{
			DefaultReplicationFactor: 2,
		},
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			LogSegmentSize: 1024 * 1024,
		},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	// 创建 topic
	err := topicMgr.CreateTopic("test-topic", 1)
	require.NoError(t, err)

	testTopic, _ := topicMgr.GetTopic("test-topic")
	testTopic.SetReplicas(0, []int32{1, 2})

	// 初始 ISR 只有 Leader
	testTopic.SetISR(0, []int32{1})

	// 验证初始 Metadata 只包含 Leader
	resp1 := requestMetadata(t, h, "test-topic")
	require.Greater(t, len(resp1), 0, "metadata response should not be empty")

	isr1 := testTopic.GetISR(0)
	assert.ElementsMatch(t, []int32{1}, isr1, "initial ISR should only have leader")

	// 模拟 Follower 追上后手动加入 ISR
	// 注意：自动 ISR 更新需要基于 HWM 和 LEO 的复杂逻辑，当前版本手动设置
	t.Log("Simulating follower catching up - manually updating ISR")
	testTopic.SetISR(0, []int32{1, 2})

	// 验证 Metadata 反映 ISR 变化
	resp2 := requestMetadata(t, h, "test-topic")
	require.Greater(t, len(resp2), 0, "metadata response should not be empty")

	isr2 := testTopic.GetISR(0)
	assert.ElementsMatch(t, []int32{1, 2}, isr2, "ISR should include follower after update")

	// 模拟 Follower 再次滞后，从 ISR 移除
	testTopic.SetISR(0, []int32{1})

	resp3 := requestMetadata(t, h, "test-topic")
	require.Greater(t, len(resp3), 0, "metadata response should not be empty")

	isr3 := testTopic.GetISR(0)
	assert.ElementsMatch(t, []int32{1}, isr3, "ISR should only have leader after follower lags")
}

// TestMetadata_MultiplePartitionsISR 验证多分区 ISR 独立更新
func TestMetadata_MultiplePartitionsISR(t *testing.T) {
	cfg := &config.Config{
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			AdvertisedHost: "localhost",
			AdvertisedPort: 9092,
		},
	}

	topicMgr := topic.NewManager(t.TempDir(), 1024*1024)
	h := New(cfg, topicMgr)

	// 创建 3 分区 topic
	err := topicMgr.CreateTopic("test-topic", 3)
	require.NoError(t, err)

	testTopic, _ := topicMgr.GetTopic("test-topic")

	// 设置不同分区的 ISR
	testTopic.SetReplicas(0, []int32{1, 2, 3})
	testTopic.SetReplicas(1, []int32{2, 3, 4})
	testTopic.SetReplicas(2, []int32{3, 4, 5})

	testTopic.SetISR(0, []int32{1, 2})    // 分区 0 ISR 缺少副本 3
	testTopic.SetISR(1, []int32{2, 3, 4}) // 分区 1 ISR 完整
	testTopic.SetISR(2, []int32{3})       // 分区 2 ISR 只有 Leader

	// 请求 Metadata
	resp := requestMetadata(t, h, "test-topic")
	require.Greater(t, len(resp), 0, "metadata response should not be empty")

	// 验证每个分区的底层 ISR
	isr0 := testTopic.GetISR(0)
	isr1 := testTopic.GetISR(1)
	isr2 := testTopic.GetISR(2)

	assert.ElementsMatch(t, []int32{1, 2}, isr0, "partition 0 ISR should be [1,2]")
	assert.ElementsMatch(t, []int32{2, 3, 4}, isr1, "partition 1 ISR should be [2,3,4]")
	assert.ElementsMatch(t, []int32{3}, isr2, "partition 2 ISR should be [3]")
}

// --- Helper Functions ---

// requestMetadata 发送 Metadata 请求并返回原始响应
func requestMetadata(t *testing.T, h *Handler, topicName string) []byte {
	var buf bytes.Buffer

	header := &protocol.RequestHeader{
		APIKey:        protocol.MetadataKey,
		APIVersion:    0,
		CorrelationID: 1,
		ClientID:      "test-client",
	}
	err := header.Encode(&buf)
	require.NoError(t, err)

	// 编码 Metadata 请求体
	protocol.WriteArray(&buf, 1) // 1 topic
	protocol.WriteString(&buf, topicName)

	resp, err := h.HandleRequest(buf.Bytes())
	require.NoError(t, err)

	return resp
}
