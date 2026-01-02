import apiClient from './client'
import type { ConsumerGroup, ConsumerGroupDetail, ResetOffsetsRequest } from '../types'

export const consumerGroupsApi = {
  list: async (): Promise<ConsumerGroup[]> => {
    const response = await apiClient.get<ConsumerGroup[]>('/consumer-groups')
    return response.data
  },

  get: async (groupId: string): Promise<ConsumerGroupDetail> => {
    const response = await apiClient.get<ConsumerGroupDetail>(`/consumer-groups/${groupId}`)
    return response.data
  },

  resetOffsets: async (
    groupId: string,
    request: ResetOffsetsRequest
  ): Promise<{ message: string }> => {
    const response = await apiClient.post<{ message: string }>(
      `/consumer-groups/${groupId}/reset-offsets`,
      request
    )
    return response.data
  },
}
