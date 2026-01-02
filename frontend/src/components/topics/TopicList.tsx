import { Table, Button, Space, Tag, Typography, Input, Tooltip } from 'antd'
import { DeleteOutlined, EyeOutlined, SearchOutlined } from '@ant-design/icons'
import type { TopicSummary } from '../../api/topics'
import { useState, useMemo } from 'react'

const { Text } = Typography

interface TopicListProps {
  topics: TopicSummary[]
  loading: boolean
  onViewTopic: (topic: TopicSummary) => void
  onDeleteTopic: (topic: TopicSummary) => void
}

export default function TopicList({
  topics,
  loading,
  onViewTopic,
  onDeleteTopic,
}: TopicListProps) {
  const [searchText, setSearchText] = useState('')

  const filteredTopics = useMemo(() => {
    if (!searchText) return topics
    const search = searchText.toLowerCase()
    return topics.filter(topic => topic.name.toLowerCase().includes(search))
  }, [topics, searchText])

  const columns = [
    {
      title: 'Topic Name',
      dataIndex: 'name',
      key: 'name',
      sorter: (a: TopicSummary, b: TopicSummary) => a.name.localeCompare(b.name),
      render: (name: string) => <Text strong>{name}</Text>,
    },
    {
      title: 'Partitions',
      dataIndex: 'partitionCount',
      key: 'partitionCount',
      width: 120,
      align: 'center' as const,
      sorter: (a: TopicSummary, b: TopicSummary) => a.partitionCount - b.partitionCount,
      render: (count: number) => <Tag color="blue">{count}</Tag>,
    },
    {
      title: 'Total Messages',
      key: 'totalMessages',
      width: 150,
      align: 'right' as const,
      render: (record: TopicSummary) => {
        const total = record.partitions?.reduce(
          (sum, p) => sum + p.highWaterMark,
          0
        ) ?? 0
        return <Text>{total.toLocaleString()}</Text>
      },
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 150,
      align: 'center' as const,
      render: (_: unknown, record: TopicSummary) => (
        <Space>
          <Tooltip title="View Details">
            <Button
              type="text"
              icon={<EyeOutlined />}
              onClick={() => onViewTopic(record)}
            />
          </Tooltip>
          <Tooltip title="Delete Topic">
            <Button
              type="text"
              danger
              icon={<DeleteOutlined />}
              onClick={() => onDeleteTopic(record)}
            />
          </Tooltip>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Input
          placeholder="Search topics..."
          prefix={<SearchOutlined />}
          value={searchText}
          onChange={e => setSearchText(e.target.value)}
          allowClear
          style={{ width: 300 }}
        />
      </div>
      <Table
        columns={columns}
        dataSource={filteredTopics}
        loading={loading}
        rowKey="name"
        pagination={{
          pageSize: 10,
          showSizeChanger: true,
          showTotal: total => `Total ${total} topics`,
        }}
      />
    </div>
  )
}
