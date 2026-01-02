// API Response Types matching backend Console API

// Health Types
export interface HealthCheck {
  status: HealthStatus
  version: string
  timestamp: number
  components: ComponentHealth
}

export type HealthStatus = 'healthy' | 'degraded' | 'unhealthy'

export interface ComponentHealth {
  storage: ComponentStatus
  coordinator: ComponentStatus
}

export interface ComponentStatus {
  status: HealthStatus
  message?: string
}

// Topic Types
export interface TopicSummary {
  name: string
  partitionCount: number
  partitions?: PartitionInfo[]
}

export interface TopicDetail {
  name: string
  partitionCount: number
  partitions: PartitionInfo[]
}

export interface PartitionInfo {
  id: number
  highWaterMark: number
}

export interface CreateTopicRequest {
  name: string
  partitions: number
}

export interface CreateTopicResponse {
  name: string
  partitions: string
}

// Message Types
export interface Message {
  partition: number
  offset: number
  key: string
  value: string
  timestamp: number
}

export interface ProduceMessageRequest {
  partition: number
  key: string
  value: string
}

export interface ProduceMessageResponse {
  partition: number
  offset: number
}

// Consumer Group Types
export interface ConsumerGroupSummary {
  groupId: string
  state: string
  members: number
}

export interface ConsumerGroupDetail {
  groupId: string
  state: string
  protocolType: string
  protocol: string
  members: ConsumerGroupMember[]
  offsetCommits: ConsumerGroupOffsetCommit[]
}

export interface ConsumerGroupMember {
  memberId: string
  clientId: string
  clientHost: string
  partitions: number[]
}

export interface ConsumerGroupOffsetCommit {
  topic: string
  partition: number
  offset: number
  metadata: string
}

// API Error Type
export interface ApiError {
  error: string
}

// Query Parameters
export interface GetMessagesParams {
  partition: number
  offset: number
  limit?: number
}

// Monitoring Types
export interface MonitoringMetrics {
  throughput: ThroughputMetrics
  latency: LatencyMetrics
  topicStats: TopicStats[]
  consumerLags: ConsumerGroupLag[]
  clusterHealth: ClusterHealthMetrics
  timestamp: number
}

export interface ThroughputMetrics {
  produceRate: number
  fetchRate: number
  produceBytes: number
  fetchBytes: number
}

export interface LatencyMetrics {
  produceP50: number
  produceP95: number
  produceP99: number
  fetchP50: number
  fetchP95: number
  fetchP99: number
}

export interface TopicStats {
  name: string
  partitions: number
  totalMessages: number
  totalBytes: number
  produceRate: number
  fetchRate: number
}

export interface ConsumerGroupLag {
  groupId: string
  totalLag: number
  topicLags: TopicLag[]
}

export interface TopicLag {
  topic: string
  totalLag: number
  partitionLags: PartitionLag[]
}

export interface PartitionLag {
  partition: number
  currentOffset: number
  logEndOffset: number
  lag: number
}

export interface ClusterHealthMetrics {
  activeConnections: number
  totalTopics: number
  totalPartitions: number
  totalConsumers: number
  diskUsageBytes: number
  memoryUsageBytes: number
  goroutineCount: number
}
