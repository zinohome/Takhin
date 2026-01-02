import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Table, Typography, Tag, Card, Space, Alert } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { consumerGroupsApi } from '../api/consumerGroups'
import type { ConsumerGroup } from '../types'

const { Title } = Typography

const getStateColor = (state: string) => {
  const stateColors: Record<string, string> = {
    Stable: 'green',
    PreparingRebalance: 'orange',
    CompletingRebalance: 'orange',
    Empty: 'blue',
    Dead: 'red',
  }
  return stateColors[state] || 'default'
}

export default function ConsumerGroups() {
  const navigate = useNavigate()
  const [groups, setGroups] = useState<ConsumerGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchGroups = async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await consumerGroupsApi.list()
      setGroups(data || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch consumer groups')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchGroups()
    const interval = setInterval(fetchGroups, 5000)
    return () => clearInterval(interval)
  }, [])

  const columns: ColumnsType<ConsumerGroup> = [
    {
      title: 'Group ID',
      dataIndex: 'groupId',
      key: 'groupId',
      render: (groupId: string) => (
        <a onClick={() => navigate(`/consumer-groups/${groupId}`)}>{groupId}</a>
      ),
    },
    {
      title: 'State',
      dataIndex: 'state',
      key: 'state',
      render: (state: string) => <Tag color={getStateColor(state)}>{state}</Tag>,
    },
    {
      title: 'Members',
      dataIndex: 'members',
      key: 'members',
    },
  ]

  return (
    <div>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Title level={2}>Consumer Groups</Title>

        {error && (
          <Alert
            message="Error"
            description={error}
            type="error"
            closable
            onClose={() => setError(null)}
          />
        )}

        <Card>
          <Table
            columns={columns}
            dataSource={groups}
            rowKey="groupId"
            loading={loading}
            pagination={{ pageSize: 20 }}
          />
        </Card>
      </Space>
    </div>
  )
}
