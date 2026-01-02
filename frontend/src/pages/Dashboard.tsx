import { Typography } from 'antd'

const { Title, Paragraph } = Typography

export default function Dashboard() {
  return (
    <div>
      <Title level={2}>Dashboard</Title>
      <Paragraph>Welcome to Takhin Console - Kafka-compatible streaming platform</Paragraph>
    </div>
  )
}
