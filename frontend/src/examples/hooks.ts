/**
 * React Hooks for Takhin API
 * Provides reusable hooks for common API operations
 */

import { useState, useEffect, useCallback } from 'react'
import {
  takhinApi,
  TakhinApiError,
} from '../api'
import type {
  TopicSummary,
  TopicDetail,
  Message,
  ConsumerGroupSummary,
  ConsumerGroupDetail,
  HealthCheck,
} from '../api'

// Generic hook for API calls with loading and error states
function useApiCall<T>(
  apiFunction: () => Promise<T>,
  dependencies: unknown[] = []
) {
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const result = await apiFunction()
      setData(result)
    } catch (err) {
      const message =
        err instanceof TakhinApiError ? err.message : 'An error occurred'
      setError(message)
    } finally {
      setLoading(false)
    }
  }, [apiFunction])

  useEffect(() => {
    fetchData()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, dependencies)

  return { data, loading, error }
}

/**
 * Hook to fetch and manage topics
 */
export function useTopics() {
  const [topics, setTopics] = useState<TopicSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchTopics = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await takhinApi.listTopics()
      setTopics(data)
    } catch (err) {
      const message =
        err instanceof TakhinApiError ? err.message : 'Failed to fetch topics'
      setError(message)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchTopics()
  }, [fetchTopics])

  const createTopic = useCallback(
    async (name: string, partitions: number) => {
      try {
        await takhinApi.createTopic({ name, partitions })
        await fetchTopics()
        return true
      } catch (err) {
        const message =
          err instanceof TakhinApiError ? err.message : 'Failed to create topic'
        setError(message)
        return false
      }
    },
    [fetchTopics]
  )

  const deleteTopic = useCallback(
    async (name: string) => {
      try {
        await takhinApi.deleteTopic(name)
        await fetchTopics()
        return true
      } catch (err) {
        const message =
          err instanceof TakhinApiError ? err.message : 'Failed to delete topic'
        setError(message)
        return false
      }
    },
    [fetchTopics]
  )

  return {
    topics,
    loading,
    error,
    refresh: fetchTopics,
    createTopic,
    deleteTopic,
  }
}

/**
 * Hook to fetch topic details
 */
export function useTopic(topicName: string) {
  return useApiCall(
    () => takhinApi.getTopic(topicName),
    [topicName]
  ) as {
    data: TopicDetail | null
    loading: boolean
    error: string | null
  }
}

/**
 * Hook to fetch and produce messages
 */
export function useMessages(
  topicName: string,
  partition: number,
  offset: number,
  limit?: number
) {
  const [messages, setMessages] = useState<Message[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchMessages = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await takhinApi.getMessages(topicName, {
        partition,
        offset,
        limit,
      })
      setMessages(data)
    } catch (err) {
      const message =
        err instanceof TakhinApiError ? err.message : 'Failed to fetch messages'
      setError(message)
    } finally {
      setLoading(false)
    }
  }, [topicName, partition, offset, limit])

  useEffect(() => {
    fetchMessages()
  }, [fetchMessages])

  const produceMessage = useCallback(
    async (key: string, value: string) => {
      try {
        const result = await takhinApi.produceMessage(topicName, {
          partition,
          key,
          value,
        })
        await fetchMessages()
        return result
      } catch (err) {
        const message =
          err instanceof TakhinApiError
            ? err.message
            : 'Failed to produce message'
        setError(message)
        return null
      }
    },
    [topicName, partition, fetchMessages]
  )

  return {
    messages,
    loading,
    error,
    refresh: fetchMessages,
    produceMessage,
  }
}

/**
 * Hook to fetch consumer groups
 */
export function useConsumerGroups() {
  const [groups, setGroups] = useState<ConsumerGroupSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchGroups = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await takhinApi.listConsumerGroups()
      setGroups(data)
    } catch (err) {
      const message =
        err instanceof TakhinApiError
          ? err.message
          : 'Failed to fetch consumer groups'
      setError(message)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchGroups()
  }, [fetchGroups])

  return {
    groups,
    loading,
    error,
    refresh: fetchGroups,
  }
}

/**
 * Hook to fetch consumer group details
 */
export function useConsumerGroup(groupId: string) {
  return useApiCall(
    () => takhinApi.getConsumerGroup(groupId),
    [groupId]
  ) as {
    data: ConsumerGroupDetail | null
    loading: boolean
    error: string | null
  }
}

/**
 * Hook to monitor health status
 */
export function useHealth(pollInterval?: number) {
  const [health, setHealth] = useState<HealthCheck | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchHealth = useCallback(async () => {
    try {
      setError(null)
      const data = await takhinApi.getHealth()
      setHealth(data)
    } catch (err) {
      const message =
        err instanceof TakhinApiError ? err.message : 'Failed to fetch health'
      setError(message)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchHealth()

    if (pollInterval && pollInterval > 0) {
      const intervalId = setInterval(fetchHealth, pollInterval)
      return () => clearInterval(intervalId)
    }
  }, [fetchHealth, pollInterval])

  return {
    health,
    loading,
    error,
    refresh: fetchHealth,
  }
}

/**
 * Hook to check readiness
 */
export function useReadiness() {
  return useApiCall(() => takhinApi.getReadiness()) as {
    data: { ready: boolean } | null
    loading: boolean
    error: string | null
  }
}

/**
 * Hook for pagination
 */
export function usePagination<T>(
  fetchFunction: (page: number, pageSize: number) => Promise<T[]>,
  initialPageSize: number = 10
) {
  const [data, setData] = useState<T[]>([])
  const [page, setPage] = useState(0)
  const [pageSize, setPageSize] = useState(initialPageSize)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [hasMore, setHasMore] = useState(true)

  const fetchData = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const result = await fetchFunction(page, pageSize)
      setData(result)
      setHasMore(result.length === pageSize)
    } catch (err) {
      const message =
        err instanceof TakhinApiError ? err.message : 'Failed to fetch data'
      setError(message)
    } finally {
      setLoading(false)
    }
  }, [fetchFunction, page, pageSize])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  const nextPage = useCallback(() => {
    if (hasMore) {
      setPage(prev => prev + 1)
    }
  }, [hasMore])

  const prevPage = useCallback(() => {
    if (page > 0) {
      setPage(prev => prev - 1)
    }
  }, [page])

  const goToPage = useCallback((newPage: number) => {
    setPage(newPage)
  }, [])

  return {
    data,
    loading,
    error,
    page,
    pageSize,
    hasMore,
    setPageSize,
    nextPage,
    prevPage,
    goToPage,
    refresh: fetchData,
  }
}

/**
 * Hook for polling data at intervals
 */
export function usePolling<T>(
  fetchFunction: () => Promise<T>,
  interval: number = 5000,
  enabled: boolean = true
) {
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = useCallback(async () => {
    try {
      setError(null)
      const result = await fetchFunction()
      setData(result)
    } catch (err) {
      const message =
        err instanceof TakhinApiError ? err.message : 'Failed to fetch data'
      setError(message)
    } finally {
      setLoading(false)
    }
  }, [fetchFunction])

  useEffect(() => {
    if (!enabled) return

    fetchData()

    const intervalId = setInterval(fetchData, interval)

    return () => {
      clearInterval(intervalId)
    }
  }, [fetchData, interval, enabled])

  return {
    data,
    loading,
    error,
    refresh: fetchData,
  }
}
