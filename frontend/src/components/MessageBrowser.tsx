import { useState, useEffect } from 'react'
import {
  Card,
  Table,
  Input,
  Select,
  DatePicker,
  Button,
  Space,
  Modal,
  Typography,
  message as antMessage,
  Descriptions,
  Tag,
  Row,
  Col,
  InputNumber,
} from 'antd'
import {
  SearchOutlined,
  DownloadOutlined,
  ReloadOutlined,
  EyeOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import { messagesApi } from '../api/messages'
import type { Message, MessageQueryParams } from '../types'

const { RangePicker } = DatePicker
const { TextArea } = Input
const { Text, Paragraph } = Typography

interface MessageBrowserProps {
  topic: string
  partitions: number
}

export default function MessageBrowser({ topic, partitions }: MessageBrowserProps) {
  const [messages, setMessages] = useState<Message[]>([])
  const [loading, setLoading] = useState(false)
  const [totalCount, setTotalCount] = useState(0)
  const [hasMore, setHasMore] = useState(false)
  const [selectedMessage, setSelectedMessage] = useState<Message | null>(null)
  const [detailVisible, setDetailVisible] = useState(false)

  const [filters, setFilters] = useState<MessageQueryParams>({
    topic,
    partition: 0,
    limit: 100,
  })

  const [searchKey, setSearchKey] = useState('')
  const [searchValue, setSearchValue] = useState('')
  const [offsetRange, setOffsetRange] = useState<[number | null, number | null]>([null, null])
  const [timeRange, setTimeRange] = useState<[dayjs.Dayjs | null, dayjs.Dayjs | null] | null>(null)

  const fetchMessages = async () => {
    setLoading(true)
    try {
      const params: MessageQueryParams = {
        ...filters,
        topic,
      }

      if (searchKey) params.key = searchKey
      if (searchValue) params.value = searchValue
      if (offsetRange[0] !== null) params.startOffset = offsetRange[0]
      if (offsetRange[1] !== null) params.endOffset = offsetRange[1]
      if (timeRange?.[0]) params.startTime = timeRange[0].valueOf()
      if (timeRange?.[1]) params.endTime = timeRange[1].valueOf()

      const result = await messagesApi.fetchMessages(params)
      setMessages(result.messages)
      setTotalCount(result.totalCount)
      setHasMore(result.hasMore)
      antMessage.success(`Loaded ${result.messages.length} messages`)
    } catch (error) {
      antMessage.error('Failed to fetch messages')
      console.error(error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchMessages()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [topic])

  const handleExport = async (format: 'json' | 'csv') => {
    try {
      const params: MessageQueryParams = { ...filters, topic }
      if (searchKey) params.key = searchKey
      if (searchValue) params.value = searchValue
      if (offsetRange[0] !== null) params.startOffset = offsetRange[0]
      if (offsetRange[1] !== null) params.endOffset = offsetRange[1]
      if (timeRange?.[0]) params.startTime = timeRange[0].valueOf()
      if (timeRange?.[1]) params.endTime = timeRange[1].valueOf()

      const blob = await messagesApi.exportMessages(params, format)
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `${topic}-messages-${Date.now()}.${format}`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
      antMessage.success(`Exported as ${format.toUpperCase()}`)
    } catch (error) {
      antMessage.error('Failed to export messages')
      console.error(error)
    }
  }

  const formatValue = (value: string): string => {
    try {
      const parsed = JSON.parse(value)
      return JSON.stringify(parsed, null, 2)
    } catch {
      return value
    }
  }

  const isJsonValue = (value: string): boolean => {
    try {
      JSON.parse(value)
      return true
    } catch {
      return false
    }
  }

  const showMessageDetail = (record: Message) => {
    setSelectedMessage(record)
    setDetailVisible(true)
  }

  const columns: ColumnsType<Message> = [
    {
      title: 'Offset',
      dataIndex: 'offset',
      key: 'offset',
      width: 120,
      sorter: (a, b) => a.offset - b.offset,
    },
    {
      title: 'Partition',
      dataIndex: 'partition',
      key: 'partition',
      width: 100,
    },
    {
      title: 'Timestamp',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 180,
      render: (timestamp: number) => dayjs(timestamp).format('YYYY-MM-DD HH:mm:ss'),
      sorter: (a, b) => a.timestamp - b.timestamp,
    },
    {
      title: 'Key',
      dataIndex: 'key',
      key: 'key',
      width: 200,
      ellipsis: true,
      render: (key: string | null) => key || <Text type="secondary">null</Text>,
    },
    {
      title: 'Value',
      dataIndex: 'value',
      key: 'value',
      ellipsis: true,
      render: (value: string) => {
        const isJson = isJsonValue(value)
        return (
          <Space>
            <Text ellipsis style={{ maxWidth: 400 }}>
              {value}
            </Text>
            {isJson && <Tag color="blue">JSON</Tag>}
          </Space>
        )
      },
    },
    {
      title: 'Action',
      key: 'action',
      width: 100,
      render: (_, record) => (
        <Button
          type="link"
          icon={<EyeOutlined />}
          onClick={() => showMessageDetail(record)}
        >
          Detail
        </Button>
      ),
    },
  ]

  return (
    <div>
      <Card>
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <Row gutter={16}>
            <Col span={6}>
              <Select
                style={{ width: '100%' }}
                placeholder="Select Partition"
                value={filters.partition}
                onChange={partition => setFilters({ ...filters, partition })}
              >
                {Array.from({ length: partitions }, (_, i) => (
                  <Select.Option key={i} value={i}>
                    Partition {i}
                  </Select.Option>
                ))}
              </Select>
            </Col>
            <Col span={6}>
              <InputNumber
                style={{ width: '100%' }}
                placeholder="Limit"
                min={1}
                max={10000}
                value={filters.limit}
                onChange={limit => setFilters({ ...filters, limit: limit || 100 })}
              />
            </Col>
            <Col span={12}>
              <Space>
                <Button
                  type="primary"
                  icon={<SearchOutlined />}
                  onClick={fetchMessages}
                  loading={loading}
                >
                  Search
                </Button>
                <Button icon={<ReloadOutlined />} onClick={fetchMessages}>
                  Refresh
                </Button>
                <Button
                  icon={<DownloadOutlined />}
                  onClick={() => handleExport('json')}
                >
                  Export JSON
                </Button>
                <Button
                  icon={<DownloadOutlined />}
                  onClick={() => handleExport('csv')}
                >
                  Export CSV
                </Button>
              </Space>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Space.Compact style={{ width: '100%' }}>
                <InputNumber
                  style={{ width: '50%' }}
                  placeholder="Start Offset"
                  value={offsetRange[0]}
                  onChange={value => setOffsetRange([value, offsetRange[1]])}
                />
                <InputNumber
                  style={{ width: '50%' }}
                  placeholder="End Offset"
                  value={offsetRange[1]}
                  onChange={value => setOffsetRange([offsetRange[0], value])}
                />
              </Space.Compact>
            </Col>
            <Col span={12}>
              <RangePicker
                style={{ width: '100%' }}
                showTime
                value={timeRange}
                onChange={setTimeRange}
                placeholder={['Start Time', 'End Time']}
              />
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Input
                placeholder="Search by Key"
                prefix={<SearchOutlined />}
                value={searchKey}
                onChange={e => setSearchKey(e.target.value)}
                allowClear
              />
            </Col>
            <Col span={12}>
              <Input
                placeholder="Search by Value"
                prefix={<SearchOutlined />}
                value={searchValue}
                onChange={e => setSearchValue(e.target.value)}
                allowClear
              />
            </Col>
          </Row>
        </Space>
      </Card>

      <Card
        style={{ marginTop: 16 }}
        title={`Messages (${totalCount} total${hasMore ? ', more available' : ''})`}
      >
        <Table
          columns={columns}
          dataSource={messages}
          rowKey="offset"
          loading={loading}
          pagination={{
            pageSize: filters.limit || 100,
            total: totalCount,
            showSizeChanger: true,
            showTotal: total => `Total ${total} messages`,
          }}
        />
      </Card>

      <Modal
        title="Message Details"
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={[
          <Button key="close" onClick={() => setDetailVisible(false)}>
            Close
          </Button>,
        ]}
        width={800}
      >
        {selectedMessage && (
          <Space direction="vertical" size="large" style={{ width: '100%' }}>
            <Descriptions bordered column={2}>
              <Descriptions.Item label="Offset" span={1}>
                {selectedMessage.offset}
              </Descriptions.Item>
              <Descriptions.Item label="Partition" span={1}>
                {selectedMessage.partition}
              </Descriptions.Item>
              <Descriptions.Item label="Timestamp" span={2}>
                {dayjs(selectedMessage.timestamp).format('YYYY-MM-DD HH:mm:ss.SSS')}
              </Descriptions.Item>
              <Descriptions.Item label="Key" span={2}>
                {selectedMessage.key || <Text type="secondary">null</Text>}
              </Descriptions.Item>
            </Descriptions>

            <Card title="Value" size="small">
              {isJsonValue(selectedMessage.value) ? (
                <Paragraph>
                  <pre style={{ maxHeight: 400, overflow: 'auto' }}>
                    {formatValue(selectedMessage.value)}
                  </pre>
                </Paragraph>
              ) : (
                <TextArea
                  value={selectedMessage.value}
                  readOnly
                  autoSize={{ minRows: 3, maxRows: 20 }}
                />
              )}
            </Card>

            {selectedMessage.headers && Object.keys(selectedMessage.headers).length > 0 && (
              <Card title="Headers" size="small">
                <Descriptions bordered column={1}>
                  {Object.entries(selectedMessage.headers).map(([key, value]) => (
                    <Descriptions.Item key={key} label={key}>
                      {value}
                    </Descriptions.Item>
                  ))}
                </Descriptions>
              </Card>
            )}
          </Space>
        )}
      </Modal>
    </div>
  )
}
