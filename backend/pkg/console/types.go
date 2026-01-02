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
	Topic          string `json:"topic"`
	Partition      int32  `json:"partition"`
	Offset         int64  `json:"offset"`
	HighWaterMark  int64  `json:"highWaterMark"`
	Lag            int64  `json:"lag"`
	Metadata       string `json:"metadata"`
}

// ResetOffsetsRequest represents a request to reset consumer group offsets
type ResetOffsetsRequest struct {
	Strategy  string                       `json:"strategy"` // "earliest", "latest", or "specific"
	Offsets   map[string]map[int32]int64   `json:"offsets,omitempty"` // For "specific" strategy
}
