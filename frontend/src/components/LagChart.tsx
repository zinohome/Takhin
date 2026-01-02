import { useMemo } from 'react'
import { Card, Empty, Space, Statistic, Tag, Row, Col } from 'antd'
import type { ConsumerGroupLag, PartitionLag } from '../api/types'

interface LagChartProps {
  lag: ConsumerGroupLag
}

export default function LagChart({ lag }: LagChartProps) {
  const chartData = useMemo(() => {
    return lag.topicLags.map(topicLag => ({
      topic: topicLag.topic,
      totalLag: topicLag.totalLag,
      partitionCount: topicLag.partitionLags.length,
      maxLag: Math.max(...topicLag.partitionLags.map(p => p.lag)),
      avgLag: topicLag.totalLag / topicLag.partitionLags.length,
      partitions: topicLag.partitionLags,
    }))
  }, [lag])

  if (chartData.length === 0) {
    return <Empty description="No lag data available" />
  }

  const maxLagAcrossTopics = Math.max(...chartData.map(d => d.totalLag))

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      {chartData.map(topic => {
        const lagPercentage = maxLagAcrossTopics > 0 
          ? (topic.totalLag / maxLagAcrossTopics) * 100 
          : 0

        const lagColor = topic.totalLag > 1000 ? '#ff4d4f' : topic.totalLag > 100 ? '#faad14' : '#52c41a'

        return (
          <Card 
            key={topic.topic} 
            size="small" 
            title={
              <Space>
                <span>{topic.topic}</span>
                <Tag color={lagColor}>{topic.totalLag.toLocaleString()} total lag</Tag>
              </Space>
            }
          >
            <Row gutter={16} style={{ marginBottom: 16 }}>
              <Col span={6}>
                <Statistic 
                  title="Total Lag" 
                  value={topic.totalLag.toLocaleString()} 
                  valueStyle={{ color: lagColor, fontSize: '20px' }}
                />
              </Col>
              <Col span={6}>
                <Statistic 
                  title="Avg Lag" 
                  value={Math.round(topic.avgLag).toLocaleString()} 
                />
              </Col>
              <Col span={6}>
                <Statistic 
                  title="Max Lag" 
                  value={topic.maxLag.toLocaleString()} 
                />
              </Col>
              <Col span={6}>
                <Statistic 
                  title="Partitions" 
                  value={topic.partitionCount} 
                />
              </Col>
            </Row>

            {/* Bar chart visualization */}
            <div style={{ 
              background: '#f0f0f0', 
              borderRadius: 4, 
              overflow: 'hidden',
              marginBottom: 12
            }}>
              <div 
                style={{ 
                  height: 24, 
                  background: lagColor,
                  width: `${lagPercentage}%`,
                  transition: 'width 0.3s ease',
                  display: 'flex',
                  alignItems: 'center',
                  paddingLeft: 8,
                  color: 'white',
                  fontSize: 12,
                  fontWeight: 'bold',
                  minWidth: lagPercentage > 5 ? 'auto' : 0,
                }}
              >
                {lagPercentage > 5 && `${lagPercentage.toFixed(1)}%`}
              </div>
            </div>

            {/* Partition details */}
            <div style={{ 
              display: 'grid', 
              gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', 
              gap: 8 
            }}>
              {topic.partitions.map((partition: PartitionLag) => {
                const partLagColor = partition.lag > 1000 ? 'red' : partition.lag > 100 ? 'orange' : 'green'
                const partLagPercent = topic.totalLag > 0 
                  ? (partition.lag / topic.totalLag) * 100 
                  : 0

                return (
                  <div 
                    key={partition.partition}
                    style={{ 
                      padding: '8px 12px',
                      border: '1px solid #d9d9d9',
                      borderRadius: 4,
                      fontSize: 12,
                    }}
                  >
                    <div style={{ 
                      display: 'flex', 
                      justifyContent: 'space-between',
                      marginBottom: 4
                    }}>
                      <span style={{ fontWeight: 500 }}>Partition {partition.partition}</span>
                      <Tag color={partLagColor} style={{ margin: 0 }}>
                        {partition.lag.toLocaleString()}
                      </Tag>
                    </div>
                    <div style={{ color: '#666', fontSize: 11 }}>
                      Current: {partition.currentOffset.toLocaleString()}
                    </div>
                    <div style={{ color: '#666', fontSize: 11 }}>
                      End: {partition.logEndOffset.toLocaleString()}
                    </div>
                    <div style={{ 
                      marginTop: 4,
                      height: 4,
                      background: '#f0f0f0',
                      borderRadius: 2,
                      overflow: 'hidden'
                    }}>
                      <div style={{
                        height: '100%',
                        width: `${Math.min(partLagPercent, 100)}%`,
                        background: partLagColor === 'red' ? '#ff4d4f' : 
                                   partLagColor === 'orange' ? '#faad14' : '#52c41a',
                        transition: 'width 0.3s ease'
                      }} />
                    </div>
                  </div>
                )
              })}
            </div>
          </Card>
        )
      })}
    </Space>
  )
}
