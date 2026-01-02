import { useState, useEffect, useCallback } from 'react'
import { useParams } from 'react-router-dom'
import {
  Typography,
  Card,
  Descriptions,
  Table,
  Tag,
  Space,
  Alert,
  Button,
  Modal,
  Radio,
  message,
  Statistic,
  Row,
  Col,
  Progress,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { ReloadOutlined, RollbackOutlined } from '@ant-design/icons'
import { consumerGroupsApi } from '../api/consumerGroups'
import type { ConsumerGroupDetail, ConsumerGroupOffsetCommit } from '../types'

const { Title, Paragraph } = Typography

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

export default function ConsumerGroupDetail() {
  const { groupId } = useParams<{ groupId: string }>()
  const [groupDetail, setGroupDetail] = useState<ConsumerGroupDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [resetModalVisible, setResetModalVisible] = useState(false)
  const [resetStrategy, setResetStrategy] = useState<'earliest' | 'latest'>('earliest')
  const [resetting, setResetting] = useState(false)

  const fetchGroupDetail = useCallback(async () => {
    if (!groupId) return
    try {
      setLoading(true)
      setError(null)
      const data = await consumerGroupsApi.get(groupId)
      setGroupDetail(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch consumer group details')
    } finally {
      setLoading(false)
    }
  }, [groupId])

  useEffect(() => {
    fetchGroupDetail()
    const interval = setInterval(fetchGroupDetail, 5000)
    return () => clearInterval(interval)
  }, [fetchGroupDetail])

  const handleResetOffsets = async () => {
    if (!groupId) return
    try {
      setResetting(true)
      await consumerGroupsApi.resetOffsets(groupId, { strategy: resetStrategy })
      message.success('Offsets reset successfully')
      setResetModalVisible(false)
      fetchGroupDetail()
    } catch (err) {
      message.error(err instanceof Error ? err.message : 'Failed to reset offsets')
    } finally {
      setResetting(false)
    }
  }

  // Calculate total lag
  const totalLag = groupDetail?.offsetCommits.reduce((sum, commit) => sum + commit.lag, 0) || 0
  const totalMessages =
    groupDetail?.offsetCommits.reduce((sum, commit) => sum + commit.highWaterMark, 0) || 0
  const consumedMessages =
    groupDetail?.offsetCommits.reduce((sum, commit) => sum + commit.offset, 0) || 0
  const progressPercent =
    totalMessages > 0 ? Math.round((consumedMessages / totalMessages) * 100) : 0

  const offsetColumns: ColumnsType<ConsumerGroupOffsetCommit> = [
    {
      title: 'Topic',
      dataIndex: 'topic',
      key: 'topic',
    },
    {
      title: 'Partition',
      dataIndex: 'partition',
      key: 'partition',
    },
    {
      title: 'Current Offset',
      dataIndex: 'offset',
      key: 'offset',
      render: (offset: number) => offset.toLocaleString(),
    },
    {
      title: 'High Water Mark',
      dataIndex: 'highWaterMark',
      key: 'highWaterMark',
      render: (hwm: number) => hwm.toLocaleString(),
    },
    {
      title: 'Lag',
      dataIndex: 'lag',
      key: 'lag',
      render: (lag: number) => {
        const color = lag === 0 ? 'green' : lag > 1000 ? 'red' : 'orange'
        return <Tag color={color}>{lag.toLocaleString()}</Tag>
      },
      sorter: (a, b) => a.lag - b.lag,
    },
    {
      title: 'Progress',
      key: 'progress',
      render: (_, record) => {
        const percent =
          record.highWaterMark > 0 ? Math.round((record.offset / record.highWaterMark) * 100) : 100
        return (
          <Progress percent={percent} size="small" status={percent < 100 ? 'active' : 'success'} />
        )
      },
    },
  ]

  const memberColumns: ColumnsType<ConsumerGroupDetail['members'][0]> = [
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
      render: (partitions: number[]) => partitions.length,
    },
  ]

  if (!groupId) {
    return <Alert message="Group ID is required" type="error" />
  }

  return (
    <div>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Space style={{ justifyContent: 'space-between', width: '100%' }}>
          <Title level={2}>Consumer Group: {groupId}</Title>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={fetchGroupDetail}>
              Refresh
            </Button>
            <Button
              icon={<RollbackOutlined />}
              onClick={() => setResetModalVisible(true)}
              disabled={groupDetail?.state !== 'Empty' && groupDetail?.state !== 'Dead'}
            >
              Reset Offsets
            </Button>
          </Space>
        </Space>

        {error && (
          <Alert
            message="Error"
            description={error}
            type="error"
            closable
            onClose={() => setError(null)}
          />
        )}

        {loading && !groupDetail ? (
          <Card loading />
        ) : groupDetail ? (
          <>
            <Card title="Overview">
              <Row gutter={16}>
                <Col span={6}>
                  <Statistic title="State" value={groupDetail.state} />
                  <Tag color={getStateColor(groupDetail.state)} style={{ marginTop: 8 }}>
                    {groupDetail.state}
                  </Tag>
                </Col>
                <Col span={6}>
                  <Statistic
                    title="Total Lag"
                    value={totalLag.toLocaleString()}
                    suffix="messages"
                  />
                </Col>
                <Col span={6}>
                  <Statistic title="Members" value={groupDetail.members.length} />
                </Col>
                <Col span={6}>
                  <Statistic title="Progress" value={progressPercent} suffix="%" />
                  <Progress
                    percent={progressPercent}
                    status={progressPercent === 100 ? 'success' : 'active'}
                  />
                </Col>
              </Row>

              <Descriptions bordered style={{ marginTop: 24 }}>
                <Descriptions.Item label="Group ID">{groupDetail.groupId}</Descriptions.Item>
                <Descriptions.Item label="Protocol Type">
                  {groupDetail.protocolType || 'N/A'}
                </Descriptions.Item>
                <Descriptions.Item label="Protocol">
                  {groupDetail.protocol || 'N/A'}
                </Descriptions.Item>
              </Descriptions>
            </Card>

            <Card title="Members">
              <Table
                columns={memberColumns}
                dataSource={groupDetail.members}
                rowKey="memberId"
                pagination={false}
                locale={{ emptyText: 'No active members' }}
              />
            </Card>

            <Card title="Offset Commits & Lag">
              <Table
                columns={offsetColumns}
                dataSource={groupDetail.offsetCommits}
                rowKey={record => `${record.topic}-${record.partition}`}
                pagination={{ pageSize: 20 }}
                locale={{ emptyText: 'No offset commits' }}
              />
            </Card>
          </>
        ) : null}

        <Modal
          title="Reset Consumer Group Offsets"
          open={resetModalVisible}
          onOk={handleResetOffsets}
          onCancel={() => setResetModalVisible(false)}
          confirmLoading={resetting}
          okText="Reset"
          okButtonProps={{ danger: true }}
        >
          <Space direction="vertical" style={{ width: '100%' }}>
            <Alert
              message="Warning"
              description="Resetting offsets will move the consumer group to a different position in the topic. The group must be in Empty or Dead state."
              type="warning"
              showIcon
            />
            <Paragraph>Select reset strategy:</Paragraph>
            <Radio.Group value={resetStrategy} onChange={e => setResetStrategy(e.target.value)}>
              <Space direction="vertical">
                <Radio value="earliest">Reset to Earliest - Start from beginning of topics</Radio>
                <Radio value="latest">
                  Reset to Latest - Start from end of topics (skip all messages)
                </Radio>
              </Space>
            </Radio.Group>
          </Space>
        </Modal>
      </Space>
    </div>
  )
}
