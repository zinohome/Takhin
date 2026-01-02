import { useState } from 'react'
import { Typography, Card, Table, Button, Space, Input, Tag } from 'antd'
import { PlusOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import type { TableColumnsType } from 'antd'

const { Title } = Typography

interface TopicData {
  key: string
  name: string
  partitions: number
  replicas: number
  size: string
  status: string
}

const mockData: TopicData[] = []

const columns: TableColumnsType<TopicData> = [
  {
    title: 'Topic Name',
    dataIndex: 'name',
    key: 'name',
    sorter: (a, b) => a.name.localeCompare(b.name),
  },
  {
    title: 'Partitions',
    dataIndex: 'partitions',
    key: 'partitions',
    width: 120,
    sorter: (a, b) => a.partitions - b.partitions,
  },
  {
    title: 'Replicas',
    dataIndex: 'replicas',
    key: 'replicas',
    width: 120,
    sorter: (a, b) => a.replicas - b.replicas,
  },
  {
    title: 'Size',
    dataIndex: 'size',
    key: 'size',
    width: 120,
  },
  {
    title: 'Status',
    dataIndex: 'status',
    key: 'status',
    width: 120,
    render: (status: string) => (
      <Tag color={status === 'healthy' ? 'green' : 'red'}>{status.toUpperCase()}</Tag>
    ),
  },
  {
    title: 'Actions',
    key: 'actions',
    width: 150,
    render: () => (
      <Space size="small">
        <Button type="link" size="small">
          View
        </Button>
        <Button type="link" size="small">
          Edit
        </Button>
        <Button type="link" danger size="small">
          Delete
        </Button>
      </Space>
    ),
  },
]

export default function Topics() {
  const [searchText, setSearchText] = useState('')
  const [loading, setLoading] = useState(false)

  const handleRefresh = () => {
    setLoading(true)
    setTimeout(() => setLoading(false), 1000)
  }

  const filteredData = mockData.filter(topic =>
    topic.name.toLowerCase().includes(searchText.toLowerCase())
  )

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={2} style={{ margin: 0 }}>
          Topics
        </Title>
        <Space>
          <Input
            placeholder="Search topics..."
            prefix={<SearchOutlined />}
            value={searchText}
            onChange={e => setSearchText(e.target.value)}
            style={{ width: 250 }}
          />
          <Button icon={<ReloadOutlined />} onClick={handleRefresh} loading={loading}>
            Refresh
          </Button>
          <Button type="primary" icon={<PlusOutlined />}>
            Create Topic
          </Button>
        </Space>
      </div>

      <Card>
        <Table
          columns={columns}
          dataSource={filteredData}
          loading={loading}
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showTotal: total => `Total ${total} topics`,
          }}
          locale={{
            emptyText: 'No topics found. Create your first topic to get started.',
          }}
        />
      </Card>
    </div>
  )
}
