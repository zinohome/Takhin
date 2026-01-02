/**
 * Example usage of the Takhin API client
 * This file demonstrates common patterns and use cases
 */

import { takhinApi, authService, TakhinApiError } from '../api'

/**
 * Example 1: Authentication Setup
 */
export async function setupAuthentication(apiKey: string): Promise<boolean> {
  try {
    // Store API key
    authService.setApiKey(apiKey)

    // Verify authentication by checking health
    await takhinApi.getHealth()

    console.log('Authentication successful')
    return true
  } catch (error) {
    if (error instanceof TakhinApiError && error.statusCode === 401) {
      console.error('Invalid API key')
      authService.removeApiKey()
    }
    return false
  }
}

/**
 * Example 2: Topic Management
 */
export async function manageTopics() {
  try {
    // List all topics
    const topics = await takhinApi.listTopics()
    console.log('Topics:', topics)

    // Create a new topic
    const newTopic = await takhinApi.createTopic({
      name: 'example-topic',
      partitions: 3,
    })
    console.log('Created topic:', newTopic)

    // Get topic details
    const topicDetail = await takhinApi.getTopic('example-topic')
    console.log('Topic details:', topicDetail)

    // Delete topic (cleanup)
    await takhinApi.deleteTopic('example-topic')
    console.log('Topic deleted')
  } catch (error) {
    if (error instanceof TakhinApiError) {
      console.error('Topic management error:', error.message)
      if (error.statusCode === 404) {
        console.log('Topic not found')
      }
    }
  }
}

/**
 * Example 3: Message Production and Consumption
 */
export async function produceAndConsume(topicName: string) {
  try {
    // Produce messages
    const produceResults = []
    for (let i = 0; i < 10; i++) {
      const result = await takhinApi.produceMessage(topicName, {
        partition: 0,
        key: `key-${i}`,
        value: `message-${i}`,
      })
      produceResults.push(result)
    }
    console.log('Produced messages:', produceResults)

    // Consume messages
    const messages = await takhinApi.getMessages(topicName, {
      partition: 0,
      offset: 0,
      limit: 10,
    })
    console.log('Consumed messages:', messages)

    return { produced: produceResults.length, consumed: messages.length }
  } catch (error) {
    if (error instanceof TakhinApiError) {
      console.error('Message error:', error.message)
    }
    throw error
  }
}

/**
 * Example 4: Consumer Group Monitoring
 */
export async function monitorConsumerGroups() {
  try {
    // List all consumer groups
    const groups = await takhinApi.listConsumerGroups()
    console.log('Consumer groups:', groups)

    // Get details for each group
    const groupDetails = await Promise.all(
      groups.map(group => takhinApi.getConsumerGroup(group.groupId))
    )

    // Analyze group status
    const analysis = groupDetails.map(group => ({
      groupId: group.groupId,
      state: group.state,
      totalMembers: group.members.length,
      totalPartitions: group.offsetCommits.length,
      isHealthy: group.state === 'Stable' && group.members.length > 0,
    }))

    console.log('Consumer group analysis:', analysis)
    return analysis
  } catch (error) {
    if (error instanceof TakhinApiError) {
      console.error('Consumer group monitoring error:', error.message)
    }
    throw error
  }
}

/**
 * Example 5: Health Check Polling
 */
export async function healthCheckLoop(intervalMs: number = 30000): Promise<void> {
  const checkHealth = async () => {
    try {
      const health = await takhinApi.getHealth()
      console.log('Health status:', health.status)
      console.log('Components:', health.components)

      if (health.status === 'unhealthy') {
        console.error('System is unhealthy!')
        // Trigger alerts or notifications
      }
    } catch (error) {
      console.error('Health check failed:', error)
    }
  }

  // Initial check
  await checkHealth()

  // Poll periodically
  setInterval(checkHealth, intervalMs)
}

/**
 * Example 6: Batch Operations with Error Handling
 */
export async function batchCreateTopics(
  topicNames: string[],
  partitions: number = 3
) {
  const results = {
    successful: [] as string[],
    failed: [] as { name: string; error: string }[],
  }

  for (const name of topicNames) {
    try {
      await takhinApi.createTopic({ name, partitions })
      results.successful.push(name)
    } catch (error) {
      if (error instanceof TakhinApiError) {
        results.failed.push({
          name,
          error: error.message,
        })
      }
    }
  }

  console.log('Batch create results:', results)
  return results
}

/**
 * Example 7: Streaming Messages with Offset Tracking
 */
export async function* streamMessages(
  topicName: string,
  partition: number,
  startOffset: number = 0,
  batchSize: number = 100
) {
  let currentOffset = startOffset
  let hasMore = true

  while (hasMore) {
    try {
      const messages = await takhinApi.getMessages(topicName, {
        partition,
        offset: currentOffset,
        limit: batchSize,
      })

      if (messages.length === 0) {
        hasMore = false
        break
      }

      // Yield batch of messages
      yield messages

      // Update offset for next batch
      currentOffset = messages[messages.length - 1].offset + 1

      // If we got fewer messages than requested, we've reached the end
      if (messages.length < batchSize) {
        hasMore = false
      }
    } catch (error) {
      console.error('Error streaming messages:', error)
      throw error
    }
  }
}

/**
 * Example 8: Topic Statistics
 */
export async function getTopicStatistics(topicName: string) {
  try {
    const topic = await takhinApi.getTopic(topicName)

    const stats = {
      name: topic.name,
      totalPartitions: topic.partitionCount,
      totalMessages: topic.partitions.reduce(
        (sum, p) => sum + p.highWaterMark,
        0
      ),
      partitionStats: topic.partitions.map(p => ({
        partition: p.id,
        messages: p.highWaterMark,
        percentage: 0, // Will be calculated below
      })),
    }

    // Calculate distribution percentage
    const total = stats.totalMessages
    if (total > 0) {
      stats.partitionStats.forEach(p => {
        p.percentage = (p.messages / total) * 100
      })
    }

    return stats
  } catch (error) {
    if (error instanceof TakhinApiError) {
      console.error('Error fetching topic statistics:', error.message)
    }
    throw error
  }
}

/**
 * Example 9: Graceful Shutdown Handler
 */
export function setupGracefulShutdown() {
  const handleShutdown = async () => {
    console.log('Shutting down gracefully...')

    // Clear authentication
    authService.removeApiKey()

    // Additional cleanup here

    console.log('Shutdown complete')
  }

  // Listen for various shutdown signals
  window.addEventListener('beforeunload', handleShutdown)

  return () => {
    window.removeEventListener('beforeunload', handleShutdown)
  }
}

/**
 * Example 10: Unauthorized Event Handler
 */
export function setupAuthEventListener() {
  const handleUnauthorized = () => {
    console.log('Authentication expired or invalid')

    // Clear stored credentials
    authService.removeApiKey()

    // Redirect to login or show notification
    // window.location.href = '/login'

    // Or dispatch a custom event for React components
    window.dispatchEvent(new CustomEvent('app:logout'))
  }

  window.addEventListener('auth:unauthorized', handleUnauthorized)

  return () => {
    window.removeEventListener('auth:unauthorized', handleUnauthorized)
  }
}
