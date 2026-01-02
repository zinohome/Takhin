import apiClient from './client'
import type { MessageQueryParams, MessageListResponse, ApiResponse } from '../types'

export const messagesApi = {
  async fetchMessages(params: MessageQueryParams): Promise<MessageListResponse> {
    const response = await apiClient.get<ApiResponse<MessageListResponse>>('/messages', {
      params,
    })
    return response.data.data
  },

  async exportMessages(params: MessageQueryParams, format: 'json' | 'csv' = 'json'): Promise<Blob> {
    const response = await apiClient.get('/messages/export', {
      params: { ...params, format },
      responseType: 'blob',
    })
    return response.data
  },
}
