// Copyright 2025 Takhin Data, Inc.

package console

// Topic types

// TopicSummary represents a brief overview of a topic
type TopicSummary struct {
	Name           string          `json:"name"`
	PartitionCount int             `json:"partitionCount"`
	Partitions     []PartitionInfo `json:"partitions,omitempty"`
}

// TopicDetail represents detailed information about a topic
type TopicDetail struct {
	Name           string          `json:"name"`
	PartitionCount int             `json:"partitionCount"`
	Partitions     []PartitionInfo `json:"partitions"`
}

// PartitionInfo represents partition information
type PartitionInfo struct {
	ID            int32 `json:"id"`
	HighWaterMark int64 `json:"highWaterMark"`
}

// CreateTopicRequest represents a request to create a topic
type CreateTopicRequest struct {
	Name       string `json:"name"`
	Partitions int32  `json:"partitions"`
}

// Message types

// Message represents a Kafka message
type Message struct {
	Partition int32  `json:"partition"`
	Offset    int64  `json:"offset"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Timestamp int64  `json:"timestamp"`
}

// ProduceMessageRequest represents a request to produce a message
type ProduceMessageRequest struct {
	Partition int32  `json:"partition"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

// Consumer Group types

// ConsumerGroupSummary represents a brief overview of a consumer group
type ConsumerGroupSummary struct {
	GroupID string `json:"groupId"`
	State   string `json:"state"`
	Members int    `json:"members"`
}

// ConsumerGroupDetail represents detailed information about a consumer group
type ConsumerGroupDetail struct {
	GroupID       string                      `json:"groupId"`
	State         string                      `json:"state"`
	ProtocolType  string                      `json:"protocolType"`
	Protocol      string                      `json:"protocol"`
	Members       []ConsumerGroupMember       `json:"members"`
	OffsetCommits []ConsumerGroupOffsetCommit `json:"offsetCommits"`
}

// ConsumerGroupMember represents a member of a consumer group
type ConsumerGroupMember struct {
	MemberID   string  `json:"memberId"`
	ClientID   string  `json:"clientId"`
	ClientHost string  `json:"clientHost"`
	Partitions []int32 `json:"partitions"`
}

// ConsumerGroupOffsetCommit represents an offset commit for a consumer group
type ConsumerGroupOffsetCommit struct {
	Topic     string `json:"topic"`
	Partition int32  `json:"partition"`
	Offset    int64  `json:"offset"`
	Metadata  string `json:"metadata"`
}

// Monitoring types

// MonitoringMetrics represents real-time cluster metrics
type MonitoringMetrics struct {
	Throughput    ThroughputMetrics    `json:"throughput"`
	Latency       LatencyMetrics       `json:"latency"`
	TopicStats    []TopicStats         `json:"topicStats"`
	ConsumerLags  []ConsumerGroupLag   `json:"consumerLags"`
	ClusterHealth ClusterHealthMetrics `json:"clusterHealth"`
	Timestamp     int64                `json:"timestamp"`
}

// ThroughputMetrics represents produce/fetch throughput
type ThroughputMetrics struct {
	ProduceRate  float64 `json:"produceRate"`
	FetchRate    float64 `json:"fetchRate"`
	ProduceBytes float64 `json:"produceBytes"`
	FetchBytes   float64 `json:"fetchBytes"`
}

// LatencyMetrics represents request latency percentiles
type LatencyMetrics struct {
	ProduceP50 float64 `json:"produceP50"`
	ProduceP95 float64 `json:"produceP95"`
	ProduceP99 float64 `json:"produceP99"`
	FetchP50   float64 `json:"fetchP50"`
	FetchP95   float64 `json:"fetchP95"`
	FetchP99   float64 `json:"fetchP99"`
}

// TopicStats represents statistics for a topic
type TopicStats struct {
	Name          string  `json:"name"`
	Partitions    int     `json:"partitions"`
	TotalMessages int64   `json:"totalMessages"`
	TotalBytes    int64   `json:"totalBytes"`
	ProduceRate   float64 `json:"produceRate"`
	FetchRate     float64 `json:"fetchRate"`
}

// ConsumerGroupLag represents lag information for a consumer group
type ConsumerGroupLag struct {
	GroupID   string     `json:"groupId"`
	TotalLag  int64      `json:"totalLag"`
	TopicLags []TopicLag `json:"topicLags"`
}

// TopicLag represents lag per topic
type TopicLag struct {
	Topic         string         `json:"topic"`
	TotalLag      int64          `json:"totalLag"`
	PartitionLags []PartitionLag `json:"partitionLags"`
}

// PartitionLag represents lag for a partition
type PartitionLag struct {
	Partition     int32 `json:"partition"`
	CurrentOffset int64 `json:"currentOffset"`
	LogEndOffset  int64 `json:"logEndOffset"`
	Lag           int64 `json:"lag"`
}

// ClusterHealthMetrics represents overall cluster health
type ClusterHealthMetrics struct {
	ActiveConnections int   `json:"activeConnections"`
	TotalTopics       int   `json:"totalTopics"`
	TotalPartitions   int   `json:"totalPartitions"`
	TotalConsumers    int   `json:"totalConsumers"`
	DiskUsageBytes    int64 `json:"diskUsageBytes"`
	MemoryUsageBytes  int64 `json:"memoryUsageBytes"`
	GoroutineCount    int   `json:"goroutineCount"`
}
