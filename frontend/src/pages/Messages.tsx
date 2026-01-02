import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Typography,
  Card,
  Table,
  Button,
  Space,
  Input,
  Select,
  DatePicker,
  Drawer,
  Tag,
  message as antMessage,
  InputNumber,
  Form,
  Modal,
  Divider,
} from 'antd'
import {
  SearchOutlined,
  ReloadOutlined,
  DownloadOutlined,
  EyeOutlined,
  ArrowLeftOutlined,
  FilterOutlined,
} from '@ant-design/icons'
import type { TableColumnsType } from 'antd'
import dayjs from 'dayjs'
import { takhinApi } from '../api'
import type { Message, TopicDetail } from '../api/types'

const { Title, Text } = Typography
const { RangePicker } = DatePicker
const { TextArea } = Input

interface MessageFilter {
  partition?: number
  startOffset?: number
  endOffset?: number
  startTime?: number
  endTime?: number
  keySearch?: string
  valueSearch?: string
}

export default function Messages() {
  const { topicName } = useParams<{ topicName: string }>()
  const navigate = useNavigate()
  
  const [loading, setLoading] = useState(false)
  const [messages, setMessages] = useState<Message[]>([])
  const [topicDetail, setTopicDetail] = useState<TopicDetail | null>(null)
  const [selectedMessage, setSelectedMessage] = useState<Message | null>(null)
  const [drawerVisible, setDrawerVisible] = useState(false)
  const [filterVisible, setFilterVisible] = useState(false)
  
  const [form] = Form.useForm()
  const [filter, setFilter] = useState<MessageFilter>({
    partition: 0,
    startOffset: 0,
  })

  useEffect(() => {
    if (topicName) {
      loadTopicDetail()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [topicName])

  const loadTopicDetail = async () => {
    if (!topicName) return
    
    try {
      const detail = await takhinApi.getTopic(topicName)
      setTopicDetail(detail)
      if (detail.partitions.length > 0) {
        setFilter(prev => ({ ...prev, partition: detail.partitions[0].id }))
      }
    } catch {
      antMessage.error('Failed to load topic details')
    }
  }

  const loadMessages = async () => {
    if (!topicName || filter.partition === undefined) return
    
    setLoading(true)
    try {
      const params = {
        partition: filter.partition,
        offset: filter.startOffset || 0,
        limit: 100,
      }
      
      let fetchedMessages = await takhinApi.getMessages(topicName, params)
      
      // Apply client-side filters
      fetchedMessages = applyFilters(fetchedMessages)
      
      setMessages(fetchedMessages)
    } catch {
      antMessage.error('Failed to load messages')
      setMessages([])
    } finally {
      setLoading(false)
    }
  }

  const applyFilters = (msgs: Message[]): Message[] => {
    let filtered = msgs

    // Offset range filter
    if (filter.startOffset !== undefined) {
      filtered = filtered.filter(m => m.offset >= filter.startOffset!)
    }
    if (filter.endOffset !== undefined) {
      filtered = filtered.filter(m => m.offset <= filter.endOffset!)
    }

    // Time range filter
    if (filter.startTime !== undefined) {
      filtered = filtered.filter(m => m.timestamp >= filter.startTime!)
    }
    if (filter.endTime !== undefined) {
      filtered = filtered.filter(m => m.timestamp <= filter.endTime!)
    }

    // Key search
    if (filter.keySearch) {
      const search = filter.keySearch.toLowerCase()
      filtered = filtered.filter(m => m.key.toLowerCase().includes(search))
    }

    // Value search
    if (filter.valueSearch) {
      const search = filter.valueSearch.toLowerCase()
      filtered = filtered.filter(m => m.value.toLowerCase().includes(search))
    }

    return filtered
  }

  const handleViewMessage = (record: Message) => {
    setSelectedMessage(record)
    setDrawerVisible(true)
  }

  const handleExport = () => {
    if (messages.length === 0) {
      antMessage.warning('No messages to export')
      return
    }

    const jsonData = JSON.stringify(messages, null, 2)
    const blob = new Blob([jsonData], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${topicName}_messages_${Date.now()}.json`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
    
    antMessage.success('Messages exported successfully')
  }

  const handleApplyFilter = () => {
    const values = form.getFieldsValue()
    
    const newFilter: MessageFilter = {
      partition: values.partition ?? filter.partition,
      startOffset: values.startOffset,
      endOffset: values.endOffset,
      keySearch: values.keySearch,
      valueSearch: values.valueSearch,
    }

    if (values.timeRange && values.timeRange.length === 2) {
      newFilter.startTime = values.timeRange[0].valueOf()
      newFilter.endTime = values.timeRange[1].valueOf()
    }

    setFilter(newFilter)
    setFilterVisible(false)
    loadMessages()
  }

  const handleResetFilter = () => {
    const resetFilter: MessageFilter = {
      partition: topicDetail?.partitions[0]?.id ?? 0,
      startOffset: 0,
    }
    setFilter(resetFilter)
    form.resetFields()
    setFilterVisible(false)
  }

  const formatValue = (value: string): string => {
    if (!value) return ''
    
    // Try to parse as JSON for pretty display
    try {
      const parsed = JSON.parse(value)
      return JSON.stringify(parsed, null, 2)
    } catch {
      return value
    }
  }

  const isJSON = (str: string): boolean => {
    try {
      JSON.parse(str)
      return true
    } catch {
      return false
    }
  }

  const columns: TableColumnsType<Message> = [
    {
      title: 'Partition',
      dataIndex: 'partition',
      key: 'partition',
      width: 100,
      render: (partition: number) => <Tag color="blue">{partition}</Tag>,
    },
    {
      title: 'Offset',
      dataIndex: 'offset',
      key: 'offset',
      width: 120,
      sorter: (a, b) => a.offset - b.offset,
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
      render: (key: string) => <Text code>{key || '(empty)'}</Text>,
    },
    {
      title: 'Value',
      dataIndex: 'value',
      key: 'value',
      ellipsis: true,
      render: (value: string) => {
        const display = value.length > 100 ? value.substring(0, 100) + '...' : value
        return (
          <Space>
            <Text code>{display}</Text>
            {isJSON(value) && <Tag color="green">JSON</Tag>}
          </Space>
        )
      },
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 100,
      fixed: 'right',
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => handleViewMessage(record)}
        >
          View
        </Button>
      ),
    },
  ]

  if (!topicName) {
    return (
      <div>
        <Title level={2}>Messages</Title>
        <Card>
          <Text>Please select a topic to view messages.</Text>
        </Card>
      </div>
    )
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/topics')}>
            Back
          </Button>
          <Title level={2} style={{ margin: 0 }}>
            Messages: {topicName}
          </Title>
        </Space>
        <Space>
          <Button icon={<FilterOutlined />} onClick={() => setFilterVisible(true)}>
            Filter
          </Button>
          <Button icon={<ReloadOutlined />} onClick={loadMessages} loading={loading}>
            Refresh
          </Button>
          <Button icon={<DownloadOutlined />} onClick={handleExport} disabled={messages.length === 0}>
            Export
          </Button>
        </Space>
      </div>

      {topicDetail && (
        <Card size="small" style={{ marginBottom: 16 }}>
          <Space split={<Divider type="vertical" />}>
            <Text>
              <strong>Partitions:</strong> {topicDetail.partitionCount}
            </Text>
            <Text>
              <strong>Current Partition:</strong> {filter.partition}
            </Text>
            <Text>
              <strong>Messages Loaded:</strong> {messages.length}
            </Text>
          </Space>
        </Card>
      )}

      <Card>
        <Table
          columns={columns}
          dataSource={messages}
          loading={loading}
          rowKey={record => `${record.partition}-${record.offset}`}
          pagination={{
            pageSize: 50,
            showSizeChanger: true,
            showTotal: total => `Total ${total} messages`,
            pageSizeOptions: ['20', '50', '100', '200'],
          }}
          scroll={{ x: 1200 }}
          locale={{
            emptyText: 'No messages found. Try adjusting your filters or load messages.',
          }}
        />
      </Card>

      {/* Filter Modal */}
      <Modal
        title="Filter Messages"
        open={filterVisible}
        onOk={handleApplyFilter}
        onCancel={() => setFilterVisible(false)}
        width={600}
        okText="Apply"
        cancelText="Cancel"
        footer={[
          <Button key="reset" onClick={handleResetFilter}>
            Reset
          </Button>,
          <Button key="cancel" onClick={() => setFilterVisible(false)}>
            Cancel
          </Button>,
          <Button key="apply" type="primary" onClick={handleApplyFilter}>
            Apply & Load
          </Button>,
        ]}
      >
        <Form form={form} layout="vertical" initialValues={filter}>
          <Form.Item label="Partition" name="partition">
            <Select>
              {topicDetail?.partitions.map(p => (
                <Select.Option key={p.id} value={p.id}>
                  Partition {p.id} (HWM: {p.highWaterMark})
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item label="Offset Range">
            <Space.Compact style={{ width: '100%' }}>
              <Form.Item name="startOffset" noStyle>
                <InputNumber placeholder="Start offset" style={{ width: '50%' }} min={0} />
              </Form.Item>
              <Form.Item name="endOffset" noStyle>
                <InputNumber placeholder="End offset" style={{ width: '50%' }} min={0} />
              </Form.Item>
            </Space.Compact>
          </Form.Item>

          <Form.Item label="Time Range" name="timeRange">
            <RangePicker showTime style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item label="Search by Key" name="keySearch">
            <Input placeholder="Search in message keys" prefix={<SearchOutlined />} />
          </Form.Item>

          <Form.Item label="Search by Value" name="valueSearch">
            <Input placeholder="Search in message values" prefix={<SearchOutlined />} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Message Detail Drawer */}
      <Drawer
        title="Message Details"
        placement="right"
        width={720}
        onClose={() => setDrawerVisible(false)}
        open={drawerVisible}
      >
        {selectedMessage && (
          <div>
            <Space direction="vertical" size="large" style={{ width: '100%' }}>
              <div>
                <Text strong>Partition:</Text>
                <div>
                  <Tag color="blue">{selectedMessage.partition}</Tag>
                </div>
              </div>

              <div>
                <Text strong>Offset:</Text>
                <div>
                  <Text copyable>{selectedMessage.offset}</Text>
                </div>
              </div>

              <div>
                <Text strong>Timestamp:</Text>
                <div>
                  <Text>{dayjs(selectedMessage.timestamp).format('YYYY-MM-DD HH:mm:ss.SSS')}</Text>
                  <br />
                  <Text type="secondary" style={{ fontSize: '12px' }}>
                    {selectedMessage.timestamp}
                  </Text>
                </div>
              </div>

              <div>
                <Text strong>Key:</Text>
                <div style={{ marginTop: 8 }}>
                  {selectedMessage.key ? (
                    <Text code copyable style={{ display: 'block', whiteSpace: 'pre-wrap' }}>
                      {selectedMessage.key}
                    </Text>
                  ) : (
                    <Text type="secondary">(empty)</Text>
                  )}
                </div>
              </div>

              <div>
                <Space>
                  <Text strong>Value:</Text>
                  {isJSON(selectedMessage.value) && <Tag color="green">JSON</Tag>}
                </Space>
                <div style={{ marginTop: 8 }}>
                  {isJSON(selectedMessage.value) ? (
                    <pre
                      style={{
                        background: '#f5f5f5',
                        padding: '12px',
                        borderRadius: '4px',
                        overflow: 'auto',
                        maxHeight: '400px',
                      }}
                    >
                      <code>{formatValue(selectedMessage.value)}</code>
                    </pre>
                  ) : (
                    <TextArea
                      value={selectedMessage.value}
                      readOnly
                      autoSize={{ minRows: 4, maxRows: 20 }}
                      style={{ fontFamily: 'monospace' }}
                    />
                  )}
                </div>
              </div>

              <div>
                <Button
                  icon={<DownloadOutlined />}
                  onClick={() => {
                    const blob = new Blob([JSON.stringify(selectedMessage, null, 2)], {
                      type: 'application/json',
                    })
                    const url = URL.createObjectURL(blob)
                    const link = document.createElement('a')
                    link.href = url
                    link.download = `message_${selectedMessage.partition}_${selectedMessage.offset}.json`
                    document.body.appendChild(link)
                    link.click()
                    document.body.removeChild(link)
                    URL.revokeObjectURL(url)
                  }}
                >
                  Export Message
                </Button>
              </div>
            </Space>
          </div>
        )}
      </Drawer>
    </div>
  )
}
