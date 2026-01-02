import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Typography, Card, Table, Button, Space, Tag, message } from 'antd'
import { ReloadOutlined, ArrowLeftOutlined } from '@ant-design/icons'
import type { TableColumnsType } from 'antd'
import { takhinApi } from '../api'
import type { ConsumerGroupSummary, ConsumerGroupLag } from '../api/types'
import ConsumerGroupDetail from '../components/ConsumerGroupDetail'

const { Title } = Typography

interface ConsumerGroupData {
  key: string
  groupId: string
  state: string
  members: number
  topics: string[]
  lag: number
}

export default function Consumers() {
  const { groupId } = useParams<{ groupId: string }>()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [groups, setGroups] = useState<ConsumerGroupSummary[]>([])
  const [lags, setLags] = useState<Map<string, ConsumerGroupLag>>(new Map())
  const [autoRefresh, setAutoRefresh] = useState(true)

  const fetchGroups = async () => {
    try {
      setLoading(true)
      const [groupsData, metrics] = await Promise.all([
        takhinApi.listConsumerGroups(),
        takhinApi.getMonitoringMetrics(),
      ])
      
      setGroups(groupsData)
      
      const lagMap = new Map<string, ConsumerGroupLag>()
      metrics.consumerLags.forEach(lag => {
        lagMap.set(lag.groupId, lag)
      })
      setLags(lagMap)
    } catch (error) {
      message.error('Failed to fetch consumer groups')
      console.error(error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchGroups()
    
    if (autoRefresh) {
      const interval = setInterval(fetchGroups, 5000)
      return () => clearInterval(interval)
    }
  }, [autoRefresh])

  const handleRefresh = () => {
    fetchGroups()
  }

  const handleViewDetails = (groupId: string) => {
    navigate(`/consumers/${groupId}`)
  }

  const handleBack = () => {
    navigate('/consumers')
  }

  if (groupId) {
    return (
      <div>
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: 16 }}>
          <Button 
            icon={<ArrowLeftOutlined />} 
            onClick={handleBack}
            style={{ marginRight: 16 }}
          >
            Back
          </Button>
          <Title level={2} style={{ margin: 0, flex: 1 }}>
            Consumer Group: {groupId}
          </Title>
          <Space>
            <Button 
              type={autoRefresh ? 'primary' : 'default'}
              onClick={() => setAutoRefresh(!autoRefresh)}
            >
              {autoRefresh ? 'Auto-Refresh: ON' : 'Auto-Refresh: OFF'}
            </Button>
            <Button icon={<ReloadOutlined />} onClick={handleRefresh} loading={loading}>
              Refresh
            </Button>
          </Space>
        </div>
        <ConsumerGroupDetail groupId={groupId} onRefresh={fetchGroups} />
      </div>
    )
  }

  const tableData: ConsumerGroupData[] = groups.map(group => {
    const lagInfo = lags.get(group.groupId)
    const topics = lagInfo?.topicLags.map(tl => tl.topic) || []
    
    return {
      key: group.groupId,
      groupId: group.groupId,
      state: group.state,
      members: group.members,
      topics,
      lag: lagInfo?.totalLag || 0,
    }
  })

  const columns: TableColumnsType<ConsumerGroupData> = [
    {
      title: 'Group ID',
      dataIndex: 'groupId',
      key: 'groupId',
      sorter: (a, b) => a.groupId.localeCompare(b.groupId),
      render: (groupId: string) => (
        <Button type="link" onClick={() => handleViewDetails(groupId)} style={{ padding: 0 }}>
          {groupId}
        </Button>
      ),
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
          empty: 'default',
        }
        return <Tag color={colorMap[state.toLowerCase()] || 'default'}>{state.toUpperCase()}</Tag>
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
          {topics.length > 0 ? (
            topics.map(topic => (
              <Tag key={topic}>{topic}</Tag>
            ))
          ) : (
            <span style={{ color: '#999' }}>None</span>
          )}
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
      width: 100,
      render: (_: unknown, record: ConsumerGroupData) => (
        <Button type="link" size="small" onClick={() => handleViewDetails(record.groupId)}>
          Details
        </Button>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={2} style={{ margin: 0 }}>
          Consumer Groups
        </Title>
        <Space>
          <Button 
            type={autoRefresh ? 'primary' : 'default'}
            onClick={() => setAutoRefresh(!autoRefresh)}
          >
            {autoRefresh ? 'Auto-Refresh: ON' : 'Auto-Refresh: OFF'}
          </Button>
          <Button icon={<ReloadOutlined />} onClick={handleRefresh} loading={loading}>
            Refresh
          </Button>
        </Space>
      </div>

      <Card>
        <Table
          columns={columns}
          dataSource={tableData}
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
