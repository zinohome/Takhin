import { useState } from 'react'
import { Typography, Card, Table, Button, Space, Tag, Badge } from 'antd'
import { ReloadOutlined, ApiOutlined } from '@ant-design/icons'
import type { TableColumnsType } from 'antd'

const { Title } = Typography

interface BrokerData {
  key: string
  id: number
  host: string
  port: number
  rack?: string
  status: 'online' | 'offline'
  version: string
  uptime: string
}

const mockData: BrokerData[] = []

const columns: TableColumnsType<BrokerData> = [
  {
    title: 'Broker ID',
    dataIndex: 'id',
    key: 'id',
    width: 100,
    sorter: (a, b) => a.id - b.id,
  },
  {
    title: 'Status',
    dataIndex: 'status',
    key: 'status',
    width: 100,
    render: (status: string) => (
      <Badge
        status={status === 'online' ? 'success' : 'error'}
        text={status === 'online' ? 'Online' : 'Offline'}
      />
    ),
  },
  {
    title: 'Host',
    dataIndex: 'host',
    key: 'host',
  },
  {
    title: 'Port',
    dataIndex: 'port',
    key: 'port',
    width: 100,
  },
  {
    title: 'Rack',
    dataIndex: 'rack',
    key: 'rack',
    width: 120,
    render: (rack?: string) => rack || '-',
  },
  {
    title: 'Version',
    dataIndex: 'version',
    key: 'version',
    width: 120,
    render: (version: string) => <Tag color="blue">{version}</Tag>,
  },
  {
    title: 'Uptime',
    dataIndex: 'uptime',
    key: 'uptime',
    width: 150,
  },
  {
    title: 'Actions',
    key: 'actions',
    width: 120,
    render: () => (
      <Space size="small">
        <Button type="link" size="small">
          Details
        </Button>
        <Button type="link" size="small">
          Metrics
        </Button>
      </Space>
    ),
  },
]

export default function Brokers() {
  const [loading, setLoading] = useState(false)

  const handleRefresh = () => {
    setLoading(true)
    setTimeout(() => setLoading(false), 1000)
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={2} style={{ margin: 0 }}>
          Brokers
        </Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={handleRefresh} loading={loading}>
            Refresh
          </Button>
          <Button icon={<ApiOutlined />}>Cluster Config</Button>
        </Space>
      </div>

      <Card>
        <Table
          columns={columns}
          dataSource={mockData}
          loading={loading}
          pagination={{
            pageSize: 10,
            showTotal: total => `Total ${total} brokers`,
          }}
          locale={{
            emptyText: 'No brokers found in the cluster.',
          }}
        />
      </Card>
    </div>
  )
}
