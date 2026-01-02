import { useState, useEffect } from 'react'
import { takhinApi } from '../api'
import type {
  ClusterConfig,
  TopicConfig,
  TopicSummary,
  UpdateClusterConfigRequest,
  UpdateTopicConfigRequest,
} from '../api/types'

type ConfigTab = 'cluster' | 'topics'

export default function Configuration() {
  const [activeTab, setActiveTab] = useState<ConfigTab>('cluster')
  const [clusterConfig, setClusterConfig] = useState<ClusterConfig | null>(null)
  const [topics, setTopics] = useState<TopicSummary[]>([])
  const [selectedTopics, setSelectedTopics] = useState<string[]>([])
  const [topicConfigs, setTopicConfigs] = useState<Map<string, TopicConfig>>(new Map())
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [editMode, setEditMode] = useState(false)
  const [saving, setSaving] = useState(false)
  const [successMessage, setSuccessMessage] = useState<string | null>(null)

  // Cluster config form state
  const [clusterForm, setClusterForm] = useState<UpdateClusterConfigRequest>({})

  // Topic config form state
  const [topicForm, setTopicForm] = useState<UpdateTopicConfigRequest>({})

  useEffect(() => {
    loadData()
  }, [activeTab])

  const loadData = async () => {
    try {
      setLoading(true)
      setError(null)

      if (activeTab === 'cluster') {
        const config = await takhinApi.getClusterConfig()
        setClusterConfig(config)
        setClusterForm({
          maxMessageBytes: config.maxMessageBytes,
          maxConnections: config.maxConnections,
          requestTimeoutMs: config.requestTimeoutMs,
          connectionTimeoutMs: config.connectionTimeoutMs,
          logRetentionHours: config.logRetentionHours,
        })
      } else {
        const topicList = await takhinApi.listTopics()
        setTopics(topicList)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load configuration')
    } finally {
      setLoading(false)
    }
  }

  const loadTopicConfig = async (topicName: string) => {
    try {
      const config = await takhinApi.getTopicConfig(topicName)
      setTopicConfigs(prev => new Map(prev).set(topicName, config))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load topic configuration')
    }
  }

  const handleSaveClusterConfig = async () => {
    if (!clusterConfig) return

    try {
      setSaving(true)
      setError(null)
      setSuccessMessage(null)

      const updates: UpdateClusterConfigRequest = {}
      if (clusterForm.maxMessageBytes !== clusterConfig.maxMessageBytes) {
        updates.maxMessageBytes = clusterForm.maxMessageBytes
      }
      if (clusterForm.maxConnections !== clusterConfig.maxConnections) {
        updates.maxConnections = clusterForm.maxConnections
      }
      if (clusterForm.requestTimeoutMs !== clusterConfig.requestTimeoutMs) {
        updates.requestTimeoutMs = clusterForm.requestTimeoutMs
      }
      if (clusterForm.connectionTimeoutMs !== clusterConfig.connectionTimeoutMs) {
        updates.connectionTimeoutMs = clusterForm.connectionTimeoutMs
      }
      if (clusterForm.logRetentionHours !== clusterConfig.logRetentionHours) {
        updates.logRetentionHours = clusterForm.logRetentionHours
      }

      const updated = await takhinApi.updateClusterConfig(updates)
      setClusterConfig(updated)
      setEditMode(false)
      setSuccessMessage('Cluster configuration updated successfully')
      setTimeout(() => setSuccessMessage(null), 3000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update configuration')
    } finally {
      setSaving(false)
    }
  }

  const handleSaveTopicConfig = async (topicName: string) => {
    try {
      setSaving(true)
      setError(null)
      setSuccessMessage(null)

      const updates: UpdateTopicConfigRequest = {}
      const currentConfig = topicConfigs.get(topicName)

      if (topicForm.compressionType && topicForm.compressionType !== currentConfig?.compressionType) {
        updates.compressionType = topicForm.compressionType
      }
      if (topicForm.cleanupPolicy && topicForm.cleanupPolicy !== currentConfig?.cleanupPolicy) {
        updates.cleanupPolicy = topicForm.cleanupPolicy
      }
      if (topicForm.retentionMs !== undefined && topicForm.retentionMs !== currentConfig?.retentionMs) {
        updates.retentionMs = topicForm.retentionMs
      }
      if (topicForm.maxMessageBytes !== undefined && topicForm.maxMessageBytes !== currentConfig?.maxMessageBytes) {
        updates.maxMessageBytes = topicForm.maxMessageBytes
      }

      const updated = await takhinApi.updateTopicConfig(topicName, updates)
      setTopicConfigs(prev => new Map(prev).set(topicName, updated))
      setSuccessMessage(`Configuration for topic "${topicName}" updated successfully`)
      setTimeout(() => setSuccessMessage(null), 3000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update topic configuration')
    } finally {
      setSaving(false)
    }
  }

  // Mark as used for future enhancements
  console.debug('handleSaveTopicConfig available for future use', handleSaveTopicConfig)

  const handleBatchUpdate = async () => {
    if (selectedTopics.length === 0) {
      setError('Please select at least one topic')
      return
    }

    try {
      setSaving(true)
      setError(null)
      setSuccessMessage(null)

      await takhinApi.batchUpdateTopicConfigs({
        topics: selectedTopics,
        config: topicForm,
      })

      // Reload configs for selected topics
      for (const topic of selectedTopics) {
        await loadTopicConfig(topic)
      }

      setSelectedTopics([])
      setSuccessMessage(`Updated configuration for ${selectedTopics.length} topics`)
      setTimeout(() => setSuccessMessage(null), 3000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to batch update configurations')
    } finally {
      setSaving(false)
    }
  }

  const toggleTopicSelection = (topicName: string) => {
    setSelectedTopics(prev =>
      prev.includes(topicName)
        ? prev.filter(t => t !== topicName)
        : [...prev, topicName]
    )
  }

  const selectAllTopics = () => {
    setSelectedTopics(topics.map(t => t.name))
  }

  const deselectAllTopics = () => {
    setSelectedTopics([])
  }

  if (loading) {
    return (
      <div className="config-container">
        <div className="loading">Loading configuration...</div>
      </div>
    )
  }

  return (
    <div className="config-container">
      <div className="config-header">
        <h1>Configuration Management</h1>
        <div className="config-tabs">
          <button
            className={`tab-button ${activeTab === 'cluster' ? 'active' : ''}`}
            onClick={() => setActiveTab('cluster')}
          >
            Cluster Configuration
          </button>
          <button
            className={`tab-button ${activeTab === 'topics' ? 'active' : ''}`}
            onClick={() => setActiveTab('topics')}
          >
            Topic Configuration
          </button>
        </div>
      </div>

      {error && (
        <div className="alert alert-error">
          <span>⚠️ {error}</span>
          <button onClick={() => setError(null)}>✕</button>
        </div>
      )}

      {successMessage && (
        <div className="alert alert-success">
          <span>✓ {successMessage}</span>
          <button onClick={() => setSuccessMessage(null)}>✕</button>
        </div>
      )}

      {activeTab === 'cluster' && clusterConfig && (
        <div className="config-section">
          <div className="section-header">
            <h2>Cluster Settings</h2>
            <div className="section-actions">
              {!editMode ? (
                <button className="btn btn-primary" onClick={() => setEditMode(true)}>
                  Edit Configuration
                </button>
              ) : (
                <>
                  <button className="btn btn-secondary" onClick={() => {
                    setEditMode(false)
                    setClusterForm({
                      maxMessageBytes: clusterConfig.maxMessageBytes,
                      maxConnections: clusterConfig.maxConnections,
                      requestTimeoutMs: clusterConfig.requestTimeoutMs,
                      connectionTimeoutMs: clusterConfig.connectionTimeoutMs,
                      logRetentionHours: clusterConfig.logRetentionHours,
                    })
                  }}>
                    Cancel
                  </button>
                  <button
                    className="btn btn-primary"
                    onClick={handleSaveClusterConfig}
                    disabled={saving}
                  >
                    {saving ? 'Saving...' : 'Save Changes'}
                  </button>
                </>
              )}
            </div>
          </div>

          <div className="config-grid">
            <div className="config-group">
              <h3>Broker Information</h3>
              <div className="config-item">
                <label>Broker ID</label>
                <div className="config-value">{clusterConfig.brokerId}</div>
              </div>
              <div className="config-item">
                <label>Advertised Host</label>
                <div className="config-value">{clusterConfig.advertisedHost}</div>
              </div>
              <div className="config-item">
                <label>Advertised Port</label>
                <div className="config-value">{clusterConfig.advertisedPort}</div>
              </div>
              <div className="config-item">
                <label>Listeners</label>
                <div className="config-value">{clusterConfig.listeners.join(', ')}</div>
              </div>
            </div>

            <div className="config-group">
              <h3>Connection Settings</h3>
              <div className="config-item">
                <label>Max Connections</label>
                {editMode ? (
                  <input
                    type="number"
                    value={clusterForm.maxConnections || ''}
                    onChange={(e) => setClusterForm({ ...clusterForm, maxConnections: parseInt(e.target.value) })}
                    min="1"
                  />
                ) : (
                  <div className="config-value">{clusterConfig.maxConnections}</div>
                )}
              </div>
              <div className="config-item">
                <label>Request Timeout (ms)</label>
                {editMode ? (
                  <input
                    type="number"
                    value={clusterForm.requestTimeoutMs || ''}
                    onChange={(e) => setClusterForm({ ...clusterForm, requestTimeoutMs: parseInt(e.target.value) })}
                    min="1000"
                  />
                ) : (
                  <div className="config-value">{clusterConfig.requestTimeoutMs}</div>
                )}
              </div>
              <div className="config-item">
                <label>Connection Timeout (ms)</label>
                {editMode ? (
                  <input
                    type="number"
                    value={clusterForm.connectionTimeoutMs || ''}
                    onChange={(e) => setClusterForm({ ...clusterForm, connectionTimeoutMs: parseInt(e.target.value) })}
                    min="1000"
                  />
                ) : (
                  <div className="config-value">{clusterConfig.connectionTimeoutMs}</div>
                )}
              </div>
            </div>

            <div className="config-group">
              <h3>Message Settings</h3>
              <div className="config-item">
                <label>Max Message Bytes</label>
                {editMode ? (
                  <input
                    type="number"
                    value={clusterForm.maxMessageBytes || ''}
                    onChange={(e) => setClusterForm({ ...clusterForm, maxMessageBytes: parseInt(e.target.value) })}
                    min="1024"
                  />
                ) : (
                  <div className="config-value">{clusterConfig.maxMessageBytes.toLocaleString()} bytes</div>
                )}
              </div>
            </div>

            <div className="config-group">
              <h3>Storage Settings</h3>
              <div className="config-item">
                <label>Data Directory</label>
                <div className="config-value">{clusterConfig.dataDir}</div>
              </div>
              <div className="config-item">
                <label>Log Segment Size</label>
                <div className="config-value">{(clusterConfig.logSegmentSize / 1024 / 1024).toFixed(0)} MB</div>
              </div>
              <div className="config-item">
                <label>Log Retention Hours</label>
                {editMode ? (
                  <input
                    type="number"
                    value={clusterForm.logRetentionHours || ''}
                    onChange={(e) => setClusterForm({ ...clusterForm, logRetentionHours: parseInt(e.target.value) })}
                    min="1"
                  />
                ) : (
                  <div className="config-value">{clusterConfig.logRetentionHours} hours</div>
                )}
              </div>
              <div className="config-item">
                <label>Log Retention Bytes</label>
                <div className="config-value">
                  {clusterConfig.logRetentionBytes === -1 ? 'Unlimited' : `${clusterConfig.logRetentionBytes.toLocaleString()} bytes`}
                </div>
              </div>
            </div>

            <div className="config-group">
              <h3>Monitoring</h3>
              <div className="config-item">
                <label>Metrics Enabled</label>
                <div className="config-value">{clusterConfig.metricsEnabled ? 'Yes' : 'No'}</div>
              </div>
              <div className="config-item">
                <label>Metrics Port</label>
                <div className="config-value">{clusterConfig.metricsPort}</div>
              </div>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'topics' && (
        <div className="config-section">
          <div className="section-header">
            <h2>Topic Configuration</h2>
            <div className="section-actions">
              {selectedTopics.length > 0 && (
                <span className="selected-count">
                  {selectedTopics.length} topic{selectedTopics.length !== 1 ? 's' : ''} selected
                </span>
              )}
            </div>
          </div>

          {selectedTopics.length > 0 && (
            <div className="batch-update-panel">
              <h3>Batch Update Configuration</h3>
              <div className="batch-form">
                <div className="form-group">
                  <label>Compression Type</label>
                  <select
                    value={topicForm.compressionType || ''}
                    onChange={(e) => setTopicForm({ ...topicForm, compressionType: e.target.value })}
                  >
                    <option value="">No change</option>
                    <option value="none">None</option>
                    <option value="gzip">GZIP</option>
                    <option value="snappy">Snappy</option>
                    <option value="lz4">LZ4</option>
                    <option value="zstd">ZSTD</option>
                    <option value="producer">Producer</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Cleanup Policy</label>
                  <select
                    value={topicForm.cleanupPolicy || ''}
                    onChange={(e) => setTopicForm({ ...topicForm, cleanupPolicy: e.target.value })}
                  >
                    <option value="">No change</option>
                    <option value="delete">Delete</option>
                    <option value="compact">Compact</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Retention (ms)</label>
                  <input
                    type="number"
                    placeholder="No change"
                    value={topicForm.retentionMs || ''}
                    onChange={(e) => setTopicForm({ ...topicForm, retentionMs: parseInt(e.target.value) })}
                    min="1"
                  />
                </div>
                <div className="form-group">
                  <label>Max Message Bytes</label>
                  <input
                    type="number"
                    placeholder="No change"
                    value={topicForm.maxMessageBytes || ''}
                    onChange={(e) => setTopicForm({ ...topicForm, maxMessageBytes: parseInt(e.target.value) })}
                    min="1024"
                  />
                </div>
              </div>
              <div className="batch-actions">
                <button className="btn btn-secondary" onClick={deselectAllTopics}>
                  Clear Selection
                </button>
                <button
                  className="btn btn-primary"
                  onClick={handleBatchUpdate}
                  disabled={saving}
                >
                  {saving ? 'Updating...' : 'Apply to Selected Topics'}
                </button>
              </div>
            </div>
          )}

          <div className="topics-list">
            <div className="topics-list-header">
              <button className="btn btn-sm btn-secondary" onClick={selectAllTopics}>
                Select All
              </button>
              <button className="btn btn-sm btn-secondary" onClick={deselectAllTopics}>
                Deselect All
              </button>
            </div>

            <div className="topics-table">
              <table>
                <thead>
                  <tr>
                    <th style={{ width: '40px' }}>
                      <input
                        type="checkbox"
                        checked={selectedTopics.length === topics.length && topics.length > 0}
                        onChange={(e) => e.target.checked ? selectAllTopics() : deselectAllTopics()}
                      />
                    </th>
                    <th>Topic Name</th>
                    <th>Partitions</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {topics.map((topic) => (
                    <tr key={topic.name}>
                      <td>
                        <input
                          type="checkbox"
                          checked={selectedTopics.includes(topic.name)}
                          onChange={() => toggleTopicSelection(topic.name)}
                        />
                      </td>
                      <td>{topic.name}</td>
                      <td>{topic.partitionCount}</td>
                      <td>
                        <button
                          className="btn btn-sm btn-primary"
                          onClick={() => {
                            loadTopicConfig(topic.name)
                          }}
                        >
                          View Config
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {Array.from(topicConfigs.entries()).map(([topicName, config]) => (
              <div key={topicName} className="topic-config-detail">
                <h3>Configuration for "{topicName}"</h3>
                <div className="config-grid">
                  <div className="config-item">
                    <label>Compression Type</label>
                    <div className="config-value">{config.compressionType}</div>
                  </div>
                  <div className="config-item">
                    <label>Cleanup Policy</label>
                    <div className="config-value">{config.cleanupPolicy}</div>
                  </div>
                  <div className="config-item">
                    <label>Retention (ms)</label>
                    <div className="config-value">{config.retentionMs.toLocaleString()}</div>
                  </div>
                  <div className="config-item">
                    <label>Segment (ms)</label>
                    <div className="config-value">{config.segmentMs.toLocaleString()}</div>
                  </div>
                  <div className="config-item">
                    <label>Max Message Bytes</label>
                    <div className="config-value">{config.maxMessageBytes.toLocaleString()}</div>
                  </div>
                  <div className="config-item">
                    <label>Min In-Sync Replicas</label>
                    <div className="config-value">{config.minInSyncReplicas}</div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <style>{`
        .config-container {
          padding: 24px;
          max-width: 1400px;
          margin: 0 auto;
        }

        .config-header {
          margin-bottom: 24px;
        }

        .config-header h1 {
          margin: 0 0 16px 0;
          font-size: 28px;
          font-weight: 600;
        }

        .config-tabs {
          display: flex;
          gap: 8px;
          border-bottom: 2px solid #e5e7eb;
        }

        .tab-button {
          padding: 12px 24px;
          background: none;
          border: none;
          border-bottom: 3px solid transparent;
          color: #6b7280;
          font-size: 14px;
          font-weight: 500;
          cursor: pointer;
          transition: all 0.2s;
        }

        .tab-button:hover {
          color: #374151;
        }

        .tab-button.active {
          color: #2563eb;
          border-bottom-color: #2563eb;
        }

        .alert {
          padding: 12px 16px;
          border-radius: 8px;
          margin-bottom: 16px;
          display: flex;
          justify-content: space-between;
          align-items: center;
        }

        .alert-error {
          background-color: #fef2f2;
          color: #991b1b;
          border: 1px solid #fecaca;
        }

        .alert-success {
          background-color: #f0fdf4;
          color: #166534;
          border: 1px solid #bbf7d0;
        }

        .alert button {
          background: none;
          border: none;
          color: inherit;
          cursor: pointer;
          font-size: 18px;
          padding: 0 4px;
        }

        .config-section {
          background: white;
          border-radius: 12px;
          padding: 24px;
          box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
        }

        .section-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 24px;
          padding-bottom: 16px;
          border-bottom: 1px solid #e5e7eb;
        }

        .section-header h2 {
          margin: 0;
          font-size: 20px;
          font-weight: 600;
        }

        .section-actions {
          display: flex;
          gap: 8px;
          align-items: center;
        }

        .selected-count {
          color: #6b7280;
          font-size: 14px;
          margin-right: 8px;
        }

        .config-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
          gap: 24px;
        }

        .config-group {
          background: #f9fafb;
          padding: 16px;
          border-radius: 8px;
        }

        .config-group h3 {
          margin: 0 0 16px 0;
          font-size: 16px;
          font-weight: 600;
          color: #374151;
        }

        .config-item {
          margin-bottom: 12px;
        }

        .config-item:last-child {
          margin-bottom: 0;
        }

        .config-item label {
          display: block;
          font-size: 13px;
          font-weight: 500;
          color: #6b7280;
          margin-bottom: 4px;
        }

        .config-value {
          font-size: 14px;
          color: #111827;
          font-weight: 500;
        }

        .config-item input {
          width: 100%;
          padding: 8px 12px;
          border: 1px solid #d1d5db;
          border-radius: 6px;
          font-size: 14px;
        }

        .batch-update-panel {
          background: #f9fafb;
          padding: 20px;
          border-radius: 8px;
          margin-bottom: 24px;
        }

        .batch-update-panel h3 {
          margin: 0 0 16px 0;
          font-size: 16px;
          font-weight: 600;
        }

        .batch-form {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
          gap: 16px;
          margin-bottom: 16px;
        }

        .form-group label {
          display: block;
          font-size: 13px;
          font-weight: 500;
          color: #6b7280;
          margin-bottom: 4px;
        }

        .form-group input,
        .form-group select {
          width: 100%;
          padding: 8px 12px;
          border: 1px solid #d1d5db;
          border-radius: 6px;
          font-size: 14px;
        }

        .batch-actions {
          display: flex;
          justify-content: flex-end;
          gap: 8px;
        }

        .topics-list-header {
          display: flex;
          gap: 8px;
          margin-bottom: 16px;
        }

        .topics-table {
          overflow-x: auto;
        }

        .topics-table table {
          width: 100%;
          border-collapse: collapse;
        }

        .topics-table th,
        .topics-table td {
          text-align: left;
          padding: 12px;
          border-bottom: 1px solid #e5e7eb;
        }

        .topics-table th {
          background: #f9fafb;
          font-weight: 600;
          color: #374151;
          font-size: 13px;
          text-transform: uppercase;
          letter-spacing: 0.05em;
        }

        .topics-table td {
          font-size: 14px;
        }

        .topic-config-detail {
          margin-top: 24px;
          padding: 20px;
          background: #f9fafb;
          border-radius: 8px;
        }

        .topic-config-detail h3 {
          margin: 0 0 16px 0;
          font-size: 16px;
          font-weight: 600;
        }

        .btn {
          padding: 8px 16px;
          border: none;
          border-radius: 6px;
          font-size: 14px;
          font-weight: 500;
          cursor: pointer;
          transition: all 0.2s;
        }

        .btn:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .btn-primary {
          background-color: #2563eb;
          color: white;
        }

        .btn-primary:hover:not(:disabled) {
          background-color: #1d4ed8;
        }

        .btn-secondary {
          background-color: #f3f4f6;
          color: #374151;
        }

        .btn-secondary:hover:not(:disabled) {
          background-color: #e5e7eb;
        }

        .btn-sm {
          padding: 6px 12px;
          font-size: 13px;
        }

        .loading {
          text-align: center;
          padding: 48px;
          color: #6b7280;
        }
      `}</style>
    </div>
  )
}
