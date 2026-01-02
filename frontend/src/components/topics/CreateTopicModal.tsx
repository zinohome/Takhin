import { Modal, Form, Input, InputNumber, message } from 'antd'
import { useState } from 'react'
import { topicApi, type CreateTopicRequest } from '../../api/topics'

interface CreateTopicModalProps {
  open: boolean
  onClose: () => void
  onSuccess: () => void
}

export default function CreateTopicModal({
  open,
  onClose,
  onSuccess,
}: CreateTopicModalProps) {
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setLoading(true)

      const request: CreateTopicRequest = {
        name: values.name,
        partitions: values.partitions,
      }

      await topicApi.create(request)
      message.success(`Topic "${values.name}" created successfully`)
      form.resetFields()
      onSuccess()
      onClose()
    } catch (error) {
      if (error instanceof Error) {
        message.error(`Failed to create topic: ${error.message}`)
      }
    } finally {
      setLoading(false)
    }
  }

  const handleCancel = () => {
    form.resetFields()
    onClose()
  }

  return (
    <Modal
      title="Create New Topic"
      open={open}
      onOk={handleSubmit}
      onCancel={handleCancel}
      confirmLoading={loading}
      okText="Create"
      cancelText="Cancel"
      destroyOnClose
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={{ partitions: 1 }}
      >
        <Form.Item
          name="name"
          label="Topic Name"
          rules={[
            { required: true, message: 'Please enter topic name' },
            {
              pattern: /^[a-zA-Z0-9._-]+$/,
              message: 'Only alphanumeric, dots, underscores, and hyphens allowed',
            },
            {
              max: 249,
              message: 'Topic name must be less than 249 characters',
            },
          ]}
        >
          <Input placeholder="e.g., user-events" />
        </Form.Item>

        <Form.Item
          name="partitions"
          label="Number of Partitions"
          rules={[
            { required: true, message: 'Please enter number of partitions' },
            {
              type: 'number',
              min: 1,
              max: 1000,
              message: 'Partitions must be between 1 and 1000',
            },
          ]}
        >
          <InputNumber
            min={1}
            max={1000}
            style={{ width: '100%' }}
            placeholder="1"
          />
        </Form.Item>

        <Form.Item>
          <div style={{ color: '#666', fontSize: '12px' }}>
            <p>Note: Partition count cannot be changed after topic creation.</p>
            <p>Choose carefully based on expected throughput and parallelism needs.</p>
          </div>
        </Form.Item>
      </Form>
    </Modal>
  )
}
