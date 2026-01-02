// Copyright 2025 Takhin Data, Inc.

package grpcapi

// Stub types for proto messages
// These will be replaced by actual generated code from takhin.proto

type CreateTopicRequest struct {
	Name              string
	NumPartitions     int32
	ReplicationFactor int32
	Configs           map[string]string
}

type CreateTopicResponse struct {
	Success bool
	Error   string
}

type DeleteTopicRequest struct {
	Name string
}

type DeleteTopicResponse struct {
	Success bool
	Error   string
}

type ListTopicsRequest struct{}

type ListTopicsResponse struct {
	Topics []string
}

type GetTopicRequest struct {
	Name string
}

type GetTopicResponse struct {
	Topic *TopicInfo
	Error string
}

type DescribeTopicsRequest struct {
	Topics []string
}

type DescribeTopicsResponse struct {
	Topics []*TopicInfo
}

type TopicInfo struct {
	Name              string
	NumPartitions     int32
	ReplicationFactor int32
	Partitions        map[int32]*PartitionInfo
}

type PartitionInfo struct {
	PartitionId     int32
	BeginningOffset int64
	EndOffset       int64
	Leader          int32
	Replicas        []int32
	Isr             []int32
}

type ProduceMessageRequest struct {
	Topic        string
	Partition    int32
	Record       *Record
	RequiredAcks int32
	TimeoutMs    int32
}

type ProduceMessageResponse struct {
	Topic     string
	Partition int32
	Offset    int64
	Timestamp int64
	Error     string
}

type Record struct {
	Key       []byte
	Value     []byte
	Headers   []*RecordHeader
	Timestamp int64
	Offset    int64
	Partition int32
}

type RecordHeader struct {
	Key   string
	Value []byte
}

type ConsumeMessagesRequest struct {
	Topic     string
	Partition int32
	Offset    int64
	MaxBytes  int32
	MinBytes  int32
	MaxWaitMs int32
}

type ConsumeMessagesResponse struct {
	Topic         string
	Partition     int32
	HighWatermark int64
	Records       []*Record
	Error         string
}

type CommitOffsetRequest struct {
	GroupId string
	Offsets []*TopicPartitionOffset
}

type CommitOffsetResponse struct {
	Success bool
	Error   string
}

type TopicPartitionOffset struct {
	Topic     string
	Partition int32
	Offset    int64
}

type ListConsumerGroupsRequest struct{}

type ListConsumerGroupsResponse struct {
	Groups []*ConsumerGroupInfo
}

type ConsumerGroupInfo struct {
	GroupId      string
	State        string
	ProtocolType string
	MemberCount  int32
}

type DescribeConsumerGroupRequest struct {
	GroupId string
}

type DescribeConsumerGroupResponse struct {
	GroupId      string
	State        string
	ProtocolType string
	Members      []*ConsumerGroupMember
	Error        string
}

type ConsumerGroupMember struct {
	MemberId    string
	ClientId    string
	ClientHost  string
	Assignments []*TopicPartitionOffset
}

type DeleteConsumerGroupRequest struct {
	GroupId string
}

type DeleteConsumerGroupResponse struct {
	Success bool
	Error   string
}

type GetPartitionOffsetsRequest struct {
	Topic     string
	Partition int32
}

type GetPartitionOffsetsResponse struct {
	Topic           string
	Partition       int32
	BeginningOffset int64
	EndOffset       int64
	Error           string
}

type HealthCheckRequest struct{}

type HealthCheckResponse struct {
	Status        string
	Version       string
	UptimeSeconds int64
}

// Streaming interfaces (stubs)
type TakhinService_ProduceMessageStreamServer interface {
	Send(*ProduceMessageResponse) error
	Recv() (*ProduceMessageRequest, error)
}

type TakhinService_ConsumeMessagesServer interface {
	Send(*ConsumeMessagesResponse) error
}
