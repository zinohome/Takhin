import axios from 'axios'
import type { AxiosInstance, AxiosRequestConfig } from 'axios'
import { authService } from './auth'
import { handleApiError } from './errors'
import type {
  HealthCheck,
  TopicSummary,
  TopicDetail,
  CreateTopicRequest,
  CreateTopicResponse,
  Message,
  ProduceMessageRequest,
  ProduceMessageResponse,
  ConsumerGroupSummary,
  ConsumerGroupDetail,
  GetMessagesParams,
  MonitoringMetrics,
} from './types'

export class TakhinApiClient {
  private client: AxiosInstance

  constructor(baseURL: string = '/api', timeout: number = 10000) {
    this.client = axios.create({
      baseURL,
      timeout,
      headers: {
        'Content-Type': 'application/json',
      },
    })

    this.setupInterceptors()
  }

  private setupInterceptors(): void {
    // Request interceptor to add auth token
    this.client.interceptors.request.use(
      config => {
        const authHeader = authService.getAuthHeader()
        if (authHeader) {
          config.headers.Authorization = authHeader
        }
        return config
      },
      error => {
        return Promise.reject(error)
      }
    )

    // Response interceptor for error handling
    this.client.interceptors.response.use(
      response => response,
      error => {
        if (error.response?.status === 401) {
          authService.removeApiKey()
          window.dispatchEvent(new CustomEvent('auth:unauthorized'))
        }
        return Promise.reject(error)
      }
    )
  }

  // Health Endpoints

  async getHealth(): Promise<HealthCheck> {
    try {
      const response = await this.client.get<HealthCheck>('/health')
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }

  async getReadiness(): Promise<{ ready: boolean }> {
    try {
      const response = await this.client.get<{ ready: boolean }>('/health/ready')
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }

  async getLiveness(): Promise<{ alive: boolean }> {
    try {
      const response = await this.client.get<{ alive: boolean }>('/health/live')
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }

  // Topic Endpoints

  async listTopics(): Promise<TopicSummary[]> {
    try {
      const response = await this.client.get<TopicSummary[]>('/topics')
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }

  async getTopic(topicName: string): Promise<TopicDetail> {
    try {
      const response = await this.client.get<TopicDetail>(`/topics/${encodeURIComponent(topicName)}`)
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }

  async createTopic(request: CreateTopicRequest): Promise<CreateTopicResponse> {
    try {
      const response = await this.client.post<CreateTopicResponse>('/topics', request)
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }

  async deleteTopic(topicName: string): Promise<{ message: string }> {
    try {
      const response = await this.client.delete<{ message: string }>(
        `/topics/${encodeURIComponent(topicName)}`
      )
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }

  // Message Endpoints

  async getMessages(topicName: string, params: GetMessagesParams): Promise<Message[]> {
    try {
      const queryParams: Record<string, string> = {
        partition: params.partition.toString(),
        offset: params.offset.toString(),
      }
      
      if (params.limit !== undefined) {
        queryParams.limit = params.limit.toString()
      }

      const response = await this.client.get<Message[]>(
        `/topics/${encodeURIComponent(topicName)}/messages`,
        { params: queryParams }
      )
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }

  async produceMessage(
    topicName: string,
    message: ProduceMessageRequest
  ): Promise<ProduceMessageResponse> {
    try {
      const response = await this.client.post<ProduceMessageResponse>(
        `/topics/${encodeURIComponent(topicName)}/messages`,
        message
      )
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }

  // Consumer Group Endpoints

  async listConsumerGroups(): Promise<ConsumerGroupSummary[]> {
    try {
      const response = await this.client.get<ConsumerGroupSummary[]>('/consumer-groups')
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }

  async getConsumerGroup(groupId: string): Promise<ConsumerGroupDetail> {
    try {
      const response = await this.client.get<ConsumerGroupDetail>(
        `/consumer-groups/${encodeURIComponent(groupId)}`
      )
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }

  // Monitoring Endpoints

  async getMonitoringMetrics(): Promise<MonitoringMetrics> {
    try {
      const response = await this.client.get<MonitoringMetrics>('/monitoring/metrics')
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }

  connectMonitoringWebSocket(
    onMessage: (metrics: MonitoringMetrics) => void,
    onError?: (error: Event) => void,
    onClose?: () => void
  ): WebSocket {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const wsUrl = `${protocol}//${host}/api/monitoring/ws`

    const ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      console.log('WebSocket connected')
    }

    ws.onmessage = (event) => {
      try {
        const metrics = JSON.parse(event.data) as MonitoringMetrics
        onMessage(metrics)
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error)
      }
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
      if (onError) onError(error)
    }

    ws.onclose = () => {
      console.log('WebSocket disconnected')
      if (onClose) onClose()
    }

    return ws
  }

  // Custom request method for advanced use cases
  async request<T>(config: AxiosRequestConfig): Promise<T> {
    try {
      const response = await this.client.request<T>(config)
      return response.data
    } catch (error) {
      throw handleApiError(error)
    }
  }
}

// Export singleton instance
export const takhinApi = new TakhinApiClient()

// Export for custom instances
export default TakhinApiClient
