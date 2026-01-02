import { Modal, Typography, Space, Alert, message } from 'antd'
import { ExclamationCircleOutlined } from '@ant-design/icons'
import { useState } from 'react'
import { topicApi, type TopicSummary } from '../../api/topics'

const { Text } = Typography

interface DeleteTopicModalProps {
  open: boolean
  topic: TopicSummary | null
  onClose: () => void
  onSuccess: () => void
}

export default function DeleteTopicModal({
  open,
  topic,
  onClose,
  onSuccess,
}: DeleteTopicModalProps) {
  const [loading, setLoading] = useState(false)

  const handleDelete = async () => {
    if (!topic) return

    setLoading(true)
    try {
      await topicApi.delete(topic.name)
      message.success(`Topic "${topic.name}" deleted successfully`)
      onSuccess()
      onClose()
    } catch (error) {
      if (error instanceof Error) {
        message.error(`Failed to delete topic: ${error.message}`)
      }
    } finally {
      setLoading(false)
    }
  }

  const totalMessages = topic?.partitions?.reduce(
    (sum, p) => sum + p.highWaterMark,
    0
  ) ?? 0

  return (
    <Modal
      title={
        <Space>
          <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />
          <span>Delete Topic</span>
        </Space>
      }
      open={open}
      onOk={handleDelete}
      onCancel={onClose}
      confirmLoading={loading}
      okText="Delete"
      cancelText="Cancel"
      okButtonProps={{ danger: true }}
    >
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        <Alert
          message="Warning: This action cannot be undone!"
          description="Deleting a topic will permanently remove all its data and configurations."
          type="warning"
          showIcon
        />

        {topic && (
          <div>
            <Text>Are you sure you want to delete the following topic?</Text>
            <div style={{ marginTop: 16, padding: 16, background: '#f5f5f5', borderRadius: 4 }}>
              <Space direction="vertical">
                <div>
                  <Text strong>Topic Name: </Text>
                  <Text code>{topic.name}</Text>
                </div>
                <div>
                  <Text strong>Partitions: </Text>
                  <Text>{topic.partitionCount}</Text>
                </div>
                <div>
                  <Text strong>Total Messages: </Text>
                  <Text>{totalMessages.toLocaleString()}</Text>
                </div>
              </Space>
            </div>
          </div>
        )}
      </Space>
    </Modal>
  )
}
