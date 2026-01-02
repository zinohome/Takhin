import { useState, useEffect } from 'react'
import { Card, Row, Col, Statistic, Typography, Skeleton, Table, Progress } from 'antd'
import {
  DatabaseOutlined,
  ClusterOutlined,
  TeamOutlined,
  CloudServerOutlined,
  ThunderboltOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons'
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'
import { takhinApi } from '../api'
import type { MonitoringMetrics, TopicLag } from '../api/types'

const { Title, Paragraph } = Typography

interface ChartDataPoint {
  timestamp: string
  produceRate: number
  fetchRate: number
}

interface LatencyDataPoint {
  timestamp: string
  produceP50: number
  produceP95: number
  produceP99: number
  fetchP50: number
  fetchP95: number
  fetchP99: number
}

export default function Dashboard() {
  const [loading, setLoading] = useState(true)
  const [metrics, setMetrics] = useState<MonitoringMetrics | null>(null)
  const [throughputData, setThroughputData] = useState<ChartDataPoint[]>([])
  const [latencyData, setLatencyData] = useState<LatencyDataPoint[]>([])

  useEffect(() => {
    let ws: WebSocket | null = null

    const connectWebSocket = () => {
      ws = takhinApi.connectMonitoringWebSocket(
        (newMetrics) => {
          setMetrics(newMetrics)
          setLoading(false)

          const timestamp = new Date(newMetrics.timestamp * 1000).toLocaleTimeString()

          setThroughputData((prev) => {
            const newData = [
              ...prev,
              {
                timestamp,
                produceRate: newMetrics.throughput.produceRate,
                fetchRate: newMetrics.throughput.fetchRate,
              },
            ]
            return newData.slice(-30)
          })

          setLatencyData((prev) => {
            const newData = [
              ...prev,
              {
                timestamp,
                produceP50: newMetrics.latency.produceP50 * 1000,
                produceP95: newMetrics.latency.produceP95 * 1000,
                produceP99: newMetrics.latency.produceP99 * 1000,
                fetchP50: newMetrics.latency.fetchP50 * 1000,
                fetchP95: newMetrics.latency.fetchP95 * 1000,
                fetchP99: newMetrics.latency.fetchP99 * 1000,
              },
            ]
            return newData.slice(-30)
          })
        },
        (error) => {
          console.error('WebSocket error:', error)
        },
        () => {
          console.log('WebSocket closed, attempting to reconnect...')
          setTimeout(connectWebSocket, 3000)
        }
      )
    }

    connectWebSocket()

    return () => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.close()
      }
    }
  }, [])

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }

  const formatRate = (rate: number): string => {
    return rate.toFixed(2)
  }

  const consumerLagColumns = [
    {
      title: 'Consumer Group',
      dataIndex: 'groupId',
      key: 'groupId',
    },
    {
      title: 'Total Lag',
      dataIndex: 'totalLag',
      key: 'totalLag',
      render: (lag: number) => lag.toLocaleString(),
    },
    {
      title: 'Topics',
      dataIndex: 'topicLags',
      key: 'topics',
      render: (topicLags: TopicLag[]) => topicLags.length,
    },
  ]

  const topicStatsColumns = [
    {
      title: 'Topic',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'Partitions',
      dataIndex: 'partitions',
      key: 'partitions',
    },
    {
      title: 'Messages',
      dataIndex: 'totalMessages',
      key: 'totalMessages',
      render: (total: number) => total.toLocaleString(),
    },
    {
      title: 'Size',
      dataIndex: 'totalBytes',
      key: 'totalBytes',
      render: (bytes: number) => formatBytes(bytes),
    },
    {
      title: 'Produce Rate',
      dataIndex: 'produceRate',
      key: 'produceRate',
      render: (rate: number) => `${formatRate(rate)} msg/s`,
    },
  ]

  return (
    <div>
      <Title level={2}>Real-time Monitoring Dashboard</Title>
      <Paragraph>
        Live cluster metrics updated every 2 seconds via WebSocket
      </Paragraph>

      <Row gutter={[16, 16]} style={{ marginTop: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false}>
            {loading ? (
              <Skeleton active paragraph={{ rows: 1 }} />
            ) : (
              <Statistic
                title="Topics"
                value={metrics?.clusterHealth.totalTopics || 0}
                prefix={<DatabaseOutlined />}
                valueStyle={{ color: '#3f8600' }}
              />
            )}
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false}>
            {loading ? (
              <Skeleton active paragraph={{ rows: 1 }} />
            ) : (
              <Statistic
                title="Partitions"
                value={metrics?.clusterHealth.totalPartitions || 0}
                prefix={<ClusterOutlined />}
                valueStyle={{ color: '#1890ff' }}
              />
            )}
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false}>
            {loading ? (
              <Skeleton active paragraph={{ rows: 1 }} />
            ) : (
              <Statistic
                title="Consumer Groups"
                value={metrics?.clusterHealth.totalConsumers || 0}
                prefix={<TeamOutlined />}
                valueStyle={{ color: '#722ed1' }}
              />
            )}
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false}>
            {loading ? (
              <Skeleton active paragraph={{ rows: 1 }} />
            ) : (
              <Statistic
                title="Active Connections"
                value={metrics?.clusterHealth.activeConnections || 0}
                prefix={<CloudServerOutlined />}
                valueStyle={{ color: '#fa8c16' }}
              />
            )}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 24 }}>
        <Col xs={24} lg={12}>
          <Card 
            title={
              <>
                <ThunderboltOutlined /> Throughput (Messages/Second)
              </>
            } 
            bordered={false}
          >
            {loading || throughputData.length === 0 ? (
              <Skeleton active paragraph={{ rows: 4 }} />
            ) : (
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={throughputData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="timestamp" />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <Line 
                    type="monotone" 
                    dataKey="produceRate" 
                    stroke="#8884d8" 
                    name="Produce Rate"
                    strokeWidth={2}
                  />
                  <Line 
                    type="monotone" 
                    dataKey="fetchRate" 
                    stroke="#82ca9d" 
                    name="Fetch Rate"
                    strokeWidth={2}
                  />
                </LineChart>
              </ResponsiveContainer>
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card 
            title={
              <>
                <ClockCircleOutlined /> Latency (Milliseconds)
              </>
            } 
            bordered={false}
          >
            {loading || latencyData.length === 0 ? (
              <Skeleton active paragraph={{ rows: 4 }} />
            ) : (
              <ResponsiveContainer width="100%" height={300}>
                <AreaChart data={latencyData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="timestamp" />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <Area 
                    type="monotone" 
                    dataKey="produceP50" 
                    stackId="1"
                    stroke="#8884d8" 
                    fill="#8884d8"
                    name="Produce P50"
                  />
                  <Area 
                    type="monotone" 
                    dataKey="produceP95" 
                    stackId="2"
                    stroke="#82ca9d" 
                    fill="#82ca9d"
                    name="Produce P95"
                  />
                  <Area 
                    type="monotone" 
                    dataKey="produceP99" 
                    stackId="3"
                    stroke="#ffc658" 
                    fill="#ffc658"
                    name="Produce P99"
                  />
                </AreaChart>
              </ResponsiveContainer>
            )}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 24 }}>
        <Col xs={24} lg={12}>
          <Card title="Topic Statistics" bordered={false}>
            {loading ? (
              <Skeleton active paragraph={{ rows: 3 }} />
            ) : (
              <Table
                dataSource={metrics?.topicStats || []}
                columns={topicStatsColumns}
                rowKey="name"
                pagination={{ pageSize: 5 }}
                size="small"
              />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="Consumer Group Lag" bordered={false}>
            {loading ? (
              <Skeleton active paragraph={{ rows: 3 }} />
            ) : (
              <Table
                dataSource={metrics?.consumerLags || []}
                columns={consumerLagColumns}
                rowKey="groupId"
                pagination={{ pageSize: 5 }}
                size="small"
              />
            )}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 24 }}>
        <Col xs={24} lg={12}>
          <Card title="System Resources" bordered={false}>
            {loading ? (
              <Skeleton active paragraph={{ rows: 3 }} />
            ) : (
              <div>
                <div style={{ marginBottom: 16 }}>
                  <Paragraph strong>Memory Usage</Paragraph>
                  <Paragraph>
                    {formatBytes(metrics?.clusterHealth.memoryUsageBytes || 0)}
                  </Paragraph>
                  <Progress 
                    percent={Math.min(
                      ((metrics?.clusterHealth.memoryUsageBytes || 0) / (1024 * 1024 * 1024)) * 100,
                      100
                    )} 
                    status="active"
                  />
                </div>
                <div style={{ marginBottom: 16 }}>
                  <Paragraph strong>Disk Usage</Paragraph>
                  <Paragraph>
                    {formatBytes(metrics?.clusterHealth.diskUsageBytes || 0)}
                  </Paragraph>
                  <Progress 
                    percent={Math.min(
                      ((metrics?.clusterHealth.diskUsageBytes || 0) / (10 * 1024 * 1024 * 1024)) * 100,
                      100
                    )} 
                    status="active"
                  />
                </div>
                <div>
                  <Paragraph strong>Goroutines</Paragraph>
                  <Paragraph>
                    {metrics?.clusterHealth.goroutineCount || 0}
                  </Paragraph>
                </div>
              </div>
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="Throughput Statistics" bordered={false}>
            {loading ? (
              <Skeleton active paragraph={{ rows: 3 }} />
            ) : (
              <ResponsiveContainer width="100%" height={250}>
                <BarChart 
                  data={[
                    {
                      name: 'Current',
                      Produce: metrics?.throughput.produceRate || 0,
                      Fetch: metrics?.throughput.fetchRate || 0,
                    },
                  ]}
                >
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <Bar dataKey="Produce" fill="#8884d8" />
                  <Bar dataKey="Fetch" fill="#82ca9d" />
                </BarChart>
              </ResponsiveContainer>
            )}
          </Card>
        </Col>
      </Row>
    </div>
  )
}
