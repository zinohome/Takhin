export interface Topic {
  name: string
  partitions: number
  replicationFactor: number
  configs?: Record<string, string>
}

export interface Broker {
  id: number
  host: string
  port: number
  rack?: string
}

export interface Partition {
  id: number
  leader: number
  replicas: number[]
  isr: number[]
}

export interface ConsumerGroup {
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
  highWaterMark: number
  lag: number
  metadata: string
}

export interface ResetOffsetsRequest {
  strategy: 'earliest' | 'latest' | 'specific'
  offsets?: Record<string, Record<number, number>>
}

export interface ApiResponse<T> {
  data: T
  message?: string
}

export interface ApiError {
  message: string
  code?: string
}
