import { useState, useEffect } from 'react'
import { Typography, Card, Row, Col, Badge, Descriptions, Button, Space, Statistic, message } from 'antd'
import { ReloadOutlined, ClusterOutlined, DatabaseOutlined } from '@ant-design/icons'
import { takhinApi } from '../api'
import type { BrokerInfo, ClusterStats } from '../api/types'

const { Title } = Typography

export default function Brokers() {
  const [loading, setLoading] = useState(false)
  const [brokers, setBrokers] = useState<BrokerInfo[]>([])
  const [clusterStats, setClusterStats] = useState<ClusterStats | null>(null)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    setLoading(true)
    try {
      const [brokersData, statsData] = await Promise.all([
        takhinApi.listBrokers(),
        takhinApi.getClusterStats(),
      ])
      setBrokers(brokersData)
      setClusterStats(statsData)
    } catch (error) {
      message.error('Failed to load broker information')
      console.error(error)
    } finally {
      setLoading(false)
    }
  }

  const handleRefresh = () => {
    loadData()
  }

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
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
        </Space>
      </div>

      {/* Cluster Statistics */}
      {clusterStats && (
        <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
          <Col xs={24} sm={12} lg={6}>
            <Card>
              <Statistic
                title="Brokers"
                value={clusterStats.brokerCount}
                prefix={<ClusterOutlined />}
                valueStyle={{ color: '#3f8600' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card>
              <Statistic
                title="Topics"
                value={clusterStats.topicCount}
                prefix={<DatabaseOutlined />}
                valueStyle={{ color: '#1890ff' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card>
              <Statistic
                title="Total Messages"
                value={clusterStats.totalMessages}
                valueStyle={{ color: '#722ed1' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card>
              <Statistic
                title="Total Size"
                value={formatBytes(clusterStats.totalSizeBytes)}
                valueStyle={{ color: '#fa8c16' }}
              />
            </Card>
          </Col>
        </Row>
      )}

      {/* Broker Cards */}
      <Row gutter={[16, 16]}>
        {brokers.map((broker) => (
          <Col xs={24} lg={12} key={broker.id}>
            <Card
              title={
                <Space>
                  <span>Broker {broker.id}</span>
                  {broker.isController && (
                    <Badge status="processing" text="Controller" />
                  )}
                </Space>
              }
              extra={
                <Badge
                  status={broker.status === 'online' ? 'success' : 'error'}
                  text={broker.status === 'online' ? 'Online' : 'Offline'}
                />
              }
              loading={loading}
            >
              <Descriptions column={1} size="small">
                <Descriptions.Item label="Address">
                  {broker.host}:{broker.port}
                </Descriptions.Item>
                <Descriptions.Item label="Topics">
                  {broker.topicCount}
                </Descriptions.Item>
                <Descriptions.Item label="Partitions">
                  {broker.partitionCount}
                </Descriptions.Item>
                <Descriptions.Item label="Status">
                  <Badge
                    status={broker.status === 'online' ? 'success' : 'error'}
                    text={broker.status === 'online' ? 'Running' : 'Down'}
                  />
                </Descriptions.Item>
              </Descriptions>
            </Card>
          </Col>
        ))}
      </Row>

      {brokers.length === 0 && !loading && (
        <Card>
          <div style={{ textAlign: 'center', padding: '40px 0' }}>
            <ClusterOutlined style={{ fontSize: 48, color: '#d9d9d9', marginBottom: 16 }} />
            <p style={{ color: '#999' }}>No brokers found in the cluster.</p>
          </div>
        </Card>
      )}
    </div>
  )
}
