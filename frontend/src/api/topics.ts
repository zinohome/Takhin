import apiClient from './client'

export interface PartitionInfo {
  id: number
  highWaterMark: number
}

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

export interface CreateTopicRequest {
  name: string
  partitions: number
}

export const topicApi = {
  list: async (): Promise<TopicSummary[]> => {
    const response = await apiClient.get<TopicSummary[]>('/topics')
    return response.data
  },

  get: async (name: string): Promise<TopicDetail> => {
    const response = await apiClient.get<TopicDetail>(`/topics/${name}`)
    return response.data
  },

  create: async (request: CreateTopicRequest): Promise<void> => {
    await apiClient.post('/topics', request)
  },

  delete: async (name: string): Promise<void> => {
    await apiClient.delete(`/topics/${name}`)
  },
}
