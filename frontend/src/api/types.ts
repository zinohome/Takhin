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

// Configuration Types
export interface ClusterConfig {
  brokerId: number
  listeners: string[]
  advertisedHost: string
  advertisedPort: number
  maxMessageBytes: number
  maxConnections: number
  requestTimeoutMs: number
  connectionTimeoutMs: number
  dataDir: string
  logSegmentSize: number
  logRetentionHours: number
  logRetentionBytes: number
  metricsEnabled: boolean
  metricsPort: number
}

export interface TopicConfig {
  name: string
  compressionType: string
  cleanupPolicy: string
  retentionMs: number
  segmentMs: number
  maxMessageBytes: number
  minInSyncReplicas: number
  customConfigs?: Record<string, string>
}

export interface UpdateClusterConfigRequest {
  maxMessageBytes?: number
  maxConnections?: number
  requestTimeoutMs?: number
  connectionTimeoutMs?: number
  logRetentionHours?: number
}

export interface UpdateTopicConfigRequest {
  compressionType?: string
  cleanupPolicy?: string
  retentionMs?: number
  segmentMs?: number
  maxMessageBytes?: number
  minInSyncReplicas?: number
}

export interface BatchUpdateTopicConfigsRequest {
  topics: string[]
  config: UpdateTopicConfigRequest
}

export interface ConfigChange {
  key: string
  oldValue: string
  newValue: string
  timestamp: number
  user?: string
}

export interface ConfigHistory {
  resourceType: string
  resourceName: string
  changes: ConfigChange[]
}
