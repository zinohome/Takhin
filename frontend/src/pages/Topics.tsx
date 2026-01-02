import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Typography, Card, Table, Button, Space, Input, Tag, Modal, Form, InputNumber, message as antMessage } from 'antd'
import { PlusOutlined, SearchOutlined, ReloadOutlined, EyeOutlined, DeleteOutlined } from '@ant-design/icons'
import type { TableColumnsType } from 'antd'
import { takhinApi } from '../api'
import type { CreateTopicRequest } from '../api/types'

const { Title } = Typography

interface TopicData {
  key: string
  name: string
  partitions: number
  highWaterMark: number
}

export default function Topics() {
  const navigate = useNavigate()
  const [searchText, setSearchText] = useState('')
  const [loading, setLoading] = useState(false)
  const [topics, setTopics] = useState<TopicData[]>([])
  const [createModalVisible, setCreateModalVisible] = useState(false)
  const [form] = Form.useForm()

  useEffect(() => {
    loadTopics()
  }, [])

  const loadTopics = async () => {
    setLoading(true)
    try {
      const topicList = await takhinApi.listTopics()
      const topicData: TopicData[] = topicList.map(topic => ({
        key: topic.name,
        name: topic.name,
        partitions: topic.partitionCount,
        highWaterMark: topic.partitions?.reduce((sum, p) => sum + p.highWaterMark, 0) || 0,
      }))
      setTopics(topicData)
    } catch {
      antMessage.error('Failed to load topics')
    } finally {
      setLoading(false)
    }
  }

  const handleCreateTopic = async () => {
    try {
      const values = await form.validateFields()
      const request: CreateTopicRequest = {
        name: values.name,
        partitions: values.partitions || 1,
      }
      await takhinApi.createTopic(request)
      antMessage.success(`Topic "${request.name}" created successfully`)
      setCreateModalVisible(false)
      form.resetFields()
      loadTopics()
    } catch {
      antMessage.error('Failed to create topic')
    }
  }

  const handleDeleteTopic = (topicName: string) => {
    Modal.confirm({
      title: 'Delete Topic',
      content: `Are you sure you want to delete topic "${topicName}"? This action cannot be undone.`,
      okText: 'Delete',
      okType: 'danger',
      onOk: async () => {
        try {
          await takhinApi.deleteTopic(topicName)
          antMessage.success(`Topic "${topicName}" deleted successfully`)
          loadTopics()
        } catch {
          antMessage.error('Failed to delete topic')
        }
      },
    })
  }

  const handleViewMessages = (topicName: string) => {
    navigate(`/topics/${encodeURIComponent(topicName)}/messages`)
  }

  const columns: TableColumnsType<TopicData> = [
    {
      title: 'Topic Name',
      dataIndex: 'name',
      key: 'name',
      sorter: (a, b) => a.name.localeCompare(b.name),
    },
    {
      title: 'Partitions',
      dataIndex: 'partitions',
      key: 'partitions',
      width: 120,
      sorter: (a, b) => a.partitions - b.partitions,
    },
    {
      title: 'Total Messages',
      dataIndex: 'highWaterMark',
      key: 'highWaterMark',
      width: 150,
      render: (hwm: number) => hwm.toLocaleString(),
      sorter: (a, b) => a.highWaterMark - b.highWaterMark,
    },
    {
      title: 'Status',
      key: 'status',
      width: 120,
      render: () => <Tag color="green">HEALTHY</Tag>,
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 200,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => handleViewMessages(record.name)}
          >
            Messages
          </Button>
          <Button
            type="link"
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleDeleteTopic(record.name)}
          >
            Delete
          </Button>
        </Space>
      ),
    },
  ]

  const filteredData = topics.filter(topic =>
    topic.name.toLowerCase().includes(searchText.toLowerCase())
  )

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={2} style={{ margin: 0 }}>
          Topics
        </Title>
        <Space>
          <Input
            placeholder="Search topics..."
            prefix={<SearchOutlined />}
            value={searchText}
            onChange={e => setSearchText(e.target.value)}
            style={{ width: 250 }}
          />
          <Button icon={<ReloadOutlined />} onClick={loadTopics} loading={loading}>
            Refresh
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalVisible(true)}>
            Create Topic
          </Button>
        </Space>
      </div>

      <Card>
        <Table
          columns={columns}
          dataSource={filteredData}
          loading={loading}
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showTotal: total => `Total ${total} topics`,
          }}
          locale={{
            emptyText: 'No topics found. Create your first topic to get started.',
          }}
        />
      </Card>

      {/* Create Topic Modal */}
      <Modal
        title="Create Topic"
        open={createModalVisible}
        onOk={handleCreateTopic}
        onCancel={() => {
          setCreateModalVisible(false)
          form.resetFields()
        }}
        okText="Create"
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label="Topic Name"
            name="name"
            rules={[
              { required: true, message: 'Please enter topic name' },
              { pattern: /^[a-zA-Z0-9._-]+$/, message: 'Invalid topic name format' },
            ]}
          >
            <Input placeholder="my-topic" />
          </Form.Item>
          <Form.Item
            label="Number of Partitions"
            name="partitions"
            initialValue={1}
            rules={[{ required: true, message: 'Please enter number of partitions' }]}
          >
            <InputNumber min={1} max={100} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
