import { useState, useEffect } from 'react'
import { Typography, Button, Space, Card, Statistic, Row, Col, Alert } from 'antd'
import { PlusOutlined, ReloadOutlined, DatabaseOutlined } from '@ant-design/icons'
import { topicApi, type TopicSummary } from '../api/topics'
import TopicList from '../components/topics/TopicList'
import CreateTopicModal from '../components/topics/CreateTopicModal'
import TopicDetailDrawer from '../components/topics/TopicDetailDrawer'
import DeleteTopicModal from '../components/topics/DeleteTopicModal'

const { Title } = Typography

export default function Topics() {
  const [topics, setTopics] = useState<TopicSummary[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [detailDrawerOpen, setDetailDrawerOpen] = useState(false)
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)
  
  const [selectedTopicName, setSelectedTopicName] = useState<string | null>(null)
  const [selectedTopic, setSelectedTopic] = useState<TopicSummary | null>(null)

  useEffect(() => {
    loadTopics()
  }, [])

  const loadTopics = async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await topicApi.list()
      setTopics(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load topics')
    } finally {
      setLoading(false)
    }
  }

  const handleViewTopic = (topic: TopicSummary) => {
    setSelectedTopicName(topic.name)
    setDetailDrawerOpen(true)
  }

  const handleDeleteTopic = (topic: TopicSummary) => {
    setSelectedTopic(topic)
    setDeleteModalOpen(true)
  }

  const totalPartitions = topics.reduce((sum, t) => sum + t.partitionCount, 0)
  const totalMessages = topics.reduce(
    (sum, t) => sum + (t.partitions?.reduce((s, p) => s + p.highWaterMark, 0) ?? 0),
    0
  )

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={2} style={{ margin: 0 }}>Topics</Title>
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={loadTopics}
            loading={loading}
          >
            Refresh
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateModalOpen(true)}
          >
            Create Topic
          </Button>
        </Space>
      </div>

      {error && (
        <Alert
          message="Error"
          description={error}
          type="error"
          closable
          onClose={() => setError(null)}
          style={{ marginBottom: 16 }}
        />
      )}

      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={8}>
          <Card>
            <Statistic
              title="Total Topics"
              value={topics.length}
              prefix={<DatabaseOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="Total Partitions"
              value={totalPartitions}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="Total Messages"
              value={totalMessages}
              formatter={(value) => value.toLocaleString()}
            />
          </Card>
        </Col>
      </Row>

      <Card>
        <TopicList
          topics={topics}
          loading={loading}
          onViewTopic={handleViewTopic}
          onDeleteTopic={handleDeleteTopic}
        />
      </Card>

      <CreateTopicModal
        open={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        onSuccess={loadTopics}
      />

      <TopicDetailDrawer
        open={detailDrawerOpen}
        topicName={selectedTopicName}
        onClose={() => setDetailDrawerOpen(false)}
      />

      <DeleteTopicModal
        open={deleteModalOpen}
        topic={selectedTopic}
        onClose={() => setDeleteModalOpen(false)}
        onSuccess={loadTopics}
      />
    </div>
  )
}
