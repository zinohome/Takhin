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

export interface ApiResponse<T> {
  data: T
  message?: string
}

export interface ApiError {
  message: string
  code?: string
}
