import { useState, useEffect } from 'react'
import { Typography, Select, Card, message, Spin } from 'antd'
import apiClient from '../api/client'
import MessageBrowser from '../components/MessageBrowser'
import type { Topic, ApiResponse } from '../types'

const { Title } = Typography

export default function Messages() {
  const [topics, setTopics] = useState<Topic[]>([])
  const [selectedTopic, setSelectedTopic] = useState<Topic | null>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    fetchTopics()
  }, [])

  const fetchTopics = async () => {
    setLoading(true)
    try {
      const response = await apiClient.get<ApiResponse<{ topics: Topic[] }>>('/topics')
      setTopics(response.data.data.topics)
      if (response.data.data.topics.length > 0) {
        setSelectedTopic(response.data.data.topics[0])
      }
    } catch (error) {
      message.error('Failed to fetch topics')
      console.error(error)
    } finally {
      setLoading(false)
    }
  }

  const handleTopicChange = (topicName: string) => {
    const topic = topics.find(t => t.name === topicName)
    if (topic) {
      setSelectedTopic(topic)
    }
  }

  return (
    <div>
      <Title level={2}>Message Browser</Title>

      <Card style={{ marginBottom: 16 }}>
        <Select
          style={{ width: 300 }}
          placeholder="Select a topic"
          value={selectedTopic?.name}
          onChange={handleTopicChange}
          loading={loading}
        >
          {topics.map(topic => (
            <Select.Option key={topic.name} value={topic.name}>
              {topic.name} ({topic.partitions} partitions)
            </Select.Option>
          ))}
        </Select>
      </Card>

      {loading ? (
        <Card>
          <Spin tip="Loading topics..." />
        </Card>
      ) : selectedTopic ? (
        <MessageBrowser topic={selectedTopic.name} partitions={selectedTopic.partitions} />
      ) : (
        <Card>
          <Typography.Text type="secondary">
            No topics available. Please create a topic first.
          </Typography.Text>
        </Card>
      )}
    </div>
  )
}
