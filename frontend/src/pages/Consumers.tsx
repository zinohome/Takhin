import { useState } from 'react'
import { Typography, Card, Table, Button, Space, Tag } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import type { TableColumnsType } from 'antd'

const { Title } = Typography

interface ConsumerGroupData {
  key: string
  groupId: string
  state: string
  members: number
  topics: string[]
  lag: number
}

const mockData: ConsumerGroupData[] = []

const columns: TableColumnsType<ConsumerGroupData> = [
  {
    title: 'Group ID',
    dataIndex: 'groupId',
    key: 'groupId',
    sorter: (a, b) => a.groupId.localeCompare(b.groupId),
  },
  {
    title: 'State',
    dataIndex: 'state',
    key: 'state',
    width: 120,
    render: (state: string) => {
      const colorMap: Record<string, string> = {
        stable: 'green',
        rebalancing: 'orange',
        dead: 'red',
      }
      return <Tag color={colorMap[state.toLowerCase()]}>{state.toUpperCase()}</Tag>
    },
  },
  {
    title: 'Members',
    dataIndex: 'members',
    key: 'members',
    width: 100,
    sorter: (a, b) => a.members - b.members,
  },
  {
    title: 'Topics',
    dataIndex: 'topics',
    key: 'topics',
    render: (topics: string[]) => (
      <Space size={[0, 4]} wrap>
        {topics.map(topic => (
          <Tag key={topic}>{topic}</Tag>
        ))}
      </Space>
    ),
  },
  {
    title: 'Total Lag',
    dataIndex: 'lag',
    key: 'lag',
    width: 120,
    sorter: (a, b) => a.lag - b.lag,
    render: (lag: number) => {
      const color = lag > 1000 ? 'red' : lag > 100 ? 'orange' : 'green'
      return <Tag color={color}>{lag.toLocaleString()}</Tag>
    },
  },
  {
    title: 'Actions',
    key: 'actions',
    width: 150,
    render: () => (
      <Space size="small">
        <Button type="link" size="small">
          Details
        </Button>
        <Button type="link" size="small">
          Reset
        </Button>
        <Button type="link" danger size="small">
          Delete
        </Button>
      </Space>
    ),
  },
]

export default function Consumers() {
  const [loading, setLoading] = useState(false)

  const handleRefresh = () => {
    setLoading(true)
    setTimeout(() => setLoading(false), 1000)
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={2} style={{ margin: 0 }}>
          Consumer Groups
        </Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={handleRefresh} loading={loading}>
            Refresh
          </Button>
        </Space>
      </div>

      <Card>
        <Table
          columns={columns}
          dataSource={mockData}
          loading={loading}
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showTotal: total => `Total ${total} consumer groups`,
          }}
          locale={{
            emptyText: 'No consumer groups found.',
          }}
        />
      </Card>
    </div>
  )
}
