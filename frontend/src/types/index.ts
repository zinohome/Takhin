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

export interface Message {
  offset: number
  timestamp: number
  key: string | null
  value: string
  headers?: Record<string, string>
  partition: number
}

export interface MessageQueryParams {
  topic: string
  partition: number
  startOffset?: number
  endOffset?: number
  startTime?: number
  endTime?: number
  key?: string
  value?: string
  limit?: number
}

export interface MessageListResponse {
  messages: Message[]
  totalCount: number
  hasMore: boolean
}

export interface ApiResponse<T> {
  data: T
  message?: string
}

export interface ApiError {
  message: string
  code?: string
}
