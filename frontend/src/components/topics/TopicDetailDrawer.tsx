import { Drawer, Descriptions, Table, Tag, Spin, Typography, Space } from 'antd'
import { useEffect, useState } from 'react'
import { topicApi, type TopicDetail } from '../../api/topics'
import { DatabaseOutlined } from '@ant-design/icons'

const { Title, Text } = Typography

interface TopicDetailDrawerProps {
  open: boolean
  topicName: string | null
  onClose: () => void
}

export default function TopicDetailDrawer({
  open,
  topicName,
  onClose,
}: TopicDetailDrawerProps) {
  const [topic, setTopic] = useState<TopicDetail | null>(null)
  const [loading, setLoading] = useState(false)

  const loadTopicDetails = async () => {
    if (!topicName) return
    
    setLoading(true)
    try {
      const data = await topicApi.get(topicName)
      setTopic(data)
    } catch (error) {
      console.error('Failed to load topic details:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (open && topicName) {
      loadTopicDetails()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, topicName])

  const partitionColumns = [
    {
      title: 'Partition ID',
      dataIndex: 'id',
      key: 'id',
      width: 120,
      render: (id: number) => <Tag color="blue">#{id}</Tag>,
    },
    {
      title: 'High Water Mark',
      dataIndex: 'highWaterMark',
      key: 'highWaterMark',
      align: 'right' as const,
      render: (hwm: number) => (
        <Space>
          <DatabaseOutlined style={{ color: '#1890ff' }} />
          <Text>{hwm.toLocaleString()}</Text>
        </Space>
      ),
    },
  ]

  const totalMessages = topic?.partitions.reduce(
    (sum, p) => sum + p.highWaterMark,
    0
  ) ?? 0

  return (
    <Drawer
      title="Topic Details"
      placement="right"
      width={720}
      open={open}
      onClose={onClose}
      destroyOnClose
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: '50px' }}>
          <Spin size="large" />
        </div>
      ) : topic ? (
        <div>
          <Title level={4}>{topic.name}</Title>
          
          <Descriptions bordered column={1} style={{ marginBottom: 24 }}>
            <Descriptions.Item label="Topic Name">
              <Text copyable>{topic.name}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="Partition Count">
              <Tag color="blue">{topic.partitionCount}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Total Messages">
              <Text strong>{totalMessages.toLocaleString()}</Text>
            </Descriptions.Item>
          </Descriptions>

          <Title level={5} style={{ marginTop: 24, marginBottom: 16 }}>
            Partition Information
          </Title>
          
          <Table
            columns={partitionColumns}
            dataSource={topic.partitions}
            rowKey="id"
            pagination={false}
            size="small"
          />
        </div>
      ) : null}
    </Drawer>
  )
}
