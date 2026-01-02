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
