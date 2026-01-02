import { Card, Row, Col, Statistic, Typography, Skeleton } from 'antd'
import {
  DatabaseOutlined,
  ClusterOutlined,
  TeamOutlined,
  CloudServerOutlined,
} from '@ant-design/icons'

const { Title, Paragraph } = Typography

export default function Dashboard() {
  const loading = false

  return (
    <div>
      <Title level={2}>Dashboard</Title>
      <Paragraph>
        Welcome to Takhin Console - Kafka-compatible streaming platform written in Go
      </Paragraph>

      <Row gutter={[16, 16]} style={{ marginTop: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card bordered={false}>
            {loading ? (
              <Skeleton active paragraph={{ rows: 1 }} />
            ) : (
              <Statistic
                title="Topics"
                value={0}
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
                title="Brokers"
                value={0}
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
                value={0}
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
                title="Total Messages"
                value={0}
                prefix={<CloudServerOutlined />}
                valueStyle={{ color: '#fa8c16' }}
              />
            )}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 24 }}>
        <Col xs={24} lg={12}>
          <Card title="System Health" bordered={false}>
            {loading ? (
              <Skeleton active paragraph={{ rows: 3 }} />
            ) : (
              <Paragraph>Cluster status and health metrics will be displayed here</Paragraph>
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="Recent Activity" bordered={false}>
            {loading ? (
              <Skeleton active paragraph={{ rows: 3 }} />
            ) : (
              <Paragraph>Recent topic activity and events will be displayed here</Paragraph>
            )}
          </Card>
        </Col>
      </Row>
    </div>
  )
}
