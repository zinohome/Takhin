import { useState, useEffect } from 'react'
import { Card, Descriptions, Table, Tag, Space, Spin, message, Row, Col, Statistic } from 'antd'
import type { TableColumnsType } from 'antd'
import { takhinApi } from '../api'
import type { ConsumerGroupDetail, ConsumerGroupMember, ConsumerGroupOffsetCommit, ConsumerGroupLag } from '../api/types'
import LagChart from './LagChart'

interface ConsumerGroupDetailProps {
  groupId: string
  onRefresh?: () => void
}

export default function ConsumerGroupDetailComponent({ groupId }: ConsumerGroupDetailProps) {
  const [loading, setLoading] = useState(false)
  const [detail, setDetail] = useState<ConsumerGroupDetail | null>(null)
  const [lag, setLag] = useState<ConsumerGroupLag | null>(null)

  useEffect(() => {
    const fetchDetail = async () => {
      try {
        setLoading(true)
        const [detailData, metrics] = await Promise.all([
          takhinApi.getConsumerGroup(groupId),
          takhinApi.getMonitoringMetrics(),
        ])
        
        setDetail(detailData)
        
        const groupLag = metrics.consumerLags.find(l => l.groupId === groupId)
        setLag(groupLag || null)
      } catch (error) {
        message.error('Failed to fetch consumer group details')
        console.error(error)
      } finally {
        setLoading(false)
      }
    }

    fetchDetail()
    
    const interval = setInterval(fetchDetail, 5000)
    return () => clearInterval(interval)
  }, [groupId])

  if (loading && !detail) {
    return (
      <div style={{ textAlign: 'center', padding: 50 }}>
        <Spin size="large" />
      </div>
    )
  }

  if (!detail) {
    return <Card>Consumer group not found</Card>
  }

  const memberColumns: TableColumnsType<ConsumerGroupMember> = [
    {
      title: 'Member ID',
      dataIndex: 'memberId',
      key: 'memberId',
      ellipsis: true,
    },
    {
      title: 'Client ID',
      dataIndex: 'clientId',
      key: 'clientId',
    },
    {
      title: 'Client Host',
      dataIndex: 'clientHost',
      key: 'clientHost',
    },
    {
      title: 'Assigned Partitions',
      dataIndex: 'partitions',
      key: 'partitions',
      render: (partitions: number[]) => (
        <Space size={[0, 4]} wrap>
          {partitions.length > 0 ? (
            partitions.map(p => <Tag key={p}>{p}</Tag>)
          ) : (
            <span style={{ color: '#999' }}>None</span>
          )}
        </Space>
      ),
    },
  ]

  const offsetColumns: TableColumnsType<ConsumerGroupOffsetCommit> = [
    {
      title: 'Topic',
      dataIndex: 'topic',
      key: 'topic',
      sorter: (a, b) => a.topic.localeCompare(b.topic),
    },
    {
      title: 'Partition',
      dataIndex: 'partition',
      key: 'partition',
      width: 100,
      sorter: (a, b) => a.partition - b.partition,
    },
    {
      title: 'Current Offset',
      dataIndex: 'offset',
      key: 'offset',
      width: 150,
      render: (offset: number) => offset.toLocaleString(),
    },
    {
      title: 'Lag',
      key: 'lag',
      width: 120,
      render: (_: unknown, record: ConsumerGroupOffsetCommit) => {
        const topicLag = lag?.topicLags.find(tl => tl.topic === record.topic)
        const partitionLag = topicLag?.partitionLags.find(pl => pl.partition === record.partition)
        
        if (!partitionLag) {
          return <span style={{ color: '#999' }}>-</span>
        }
        
        const lagValue = partitionLag.lag
        const color = lagValue > 1000 ? 'red' : lagValue > 100 ? 'orange' : 'green'
        return <Tag color={color}>{lagValue.toLocaleString()}</Tag>
      },
    },
    {
      title: 'Log End Offset',
      key: 'logEndOffset',
      width: 150,
      render: (_: unknown, record: ConsumerGroupOffsetCommit) => {
        const topicLag = lag?.topicLags.find(tl => tl.topic === record.topic)
        const partitionLag = topicLag?.partitionLags.find(pl => pl.partition === record.partition)
        
        if (!partitionLag) {
          return <span style={{ color: '#999' }}>-</span>
        }
        
        return partitionLag.logEndOffset.toLocaleString()
      },
    },
  ]

  const totalLag = lag?.totalLag || 0
  const lagColor = totalLag > 1000 ? '#ff4d4f' : totalLag > 100 ? '#faad14' : '#52c41a'

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      {/* Summary Statistics */}
      <Card>
        <Row gutter={16}>
          <Col span={6}>
            <Statistic 
              title="Group State" 
              value={detail.state} 
              valueStyle={{ 
                color: detail.state === 'Stable' ? '#52c41a' : '#faad14',
                fontSize: '24px'
              }}
            />
          </Col>
          <Col span={6}>
            <Statistic title="Members" value={detail.members.length} />
          </Col>
          <Col span={6}>
            <Statistic title="Topics" value={lag?.topicLags.length || 0} />
          </Col>
          <Col span={6}>
            <Statistic 
              title="Total Lag" 
              value={totalLag.toLocaleString()} 
              valueStyle={{ color: lagColor }}
            />
          </Col>
        </Row>
      </Card>

      {/* Group Information */}
      <Card title="Group Information">
        <Descriptions column={2} bordered>
          <Descriptions.Item label="Group ID">{detail.groupId}</Descriptions.Item>
          <Descriptions.Item label="State">
            <Tag color={detail.state === 'Stable' ? 'green' : 'orange'}>
              {detail.state}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Protocol Type">{detail.protocolType}</Descriptions.Item>
          <Descriptions.Item label="Protocol">{detail.protocol}</Descriptions.Item>
        </Descriptions>
      </Card>

      {/* Lag Visualization */}
      {lag && lag.topicLags.length > 0 && (
        <Card title="Consumer Lag Chart">
          <LagChart lag={lag} />
        </Card>
      )}

      {/* Members */}
      <Card title="Members">
        <Table
          columns={memberColumns}
          dataSource={detail.members.map((m, idx) => ({ ...m, key: idx }))}
          pagination={false}
          locale={{
            emptyText: 'No members in this group',
          }}
        />
      </Card>

      {/* Offset Commits */}
      <Card title="Offset Commits">
        <Table
          columns={offsetColumns}
          dataSource={detail.offsetCommits.map((o, idx) => ({ ...o, key: idx }))}
          pagination={{
            pageSize: 20,
            showSizeChanger: true,
            showTotal: total => `Total ${total} partitions`,
          }}
          locale={{
            emptyText: 'No offset commits',
          }}
        />
      </Card>
    </Space>
  )
}
