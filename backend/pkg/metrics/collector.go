// Copyright 2025 Takhin Data, Inc.

package metrics

import (
	"time"

	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/logger"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

// Collector periodically collects metrics from various components
type Collector struct {
	topicManager *topic.Manager
	coordinator  *coordinator.Coordinator
	logger       *logger.Logger
	stopChan     chan struct{}
	interval     time.Duration
}

// NewCollector creates a new metrics collector
func NewCollector(topicMgr *topic.Manager, coord *coordinator.Coordinator, interval time.Duration) *Collector {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	return &Collector{
		topicManager: topicMgr,
		coordinator:  coord,
		logger:       logger.Default().WithComponent("metrics-collector"),
		stopChan:     make(chan struct{}),
		interval:     interval,
	}
}

// Start begins periodic metrics collection
func (c *Collector) Start() {
	go c.collectLoop()
	c.logger.Info("metrics collector started", "interval", c.interval)
}

// Stop stops the metrics collector
func (c *Collector) Stop() {
	close(c.stopChan)
	c.logger.Info("metrics collector stopped")
}

func (c *Collector) collectLoop() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.collectMetrics()
		case <-c.stopChan:
			return
		}
	}
}

func (c *Collector) collectMetrics() {
	c.collectStorageMetrics()
	c.collectConsumerGroupMetrics()
}

func (c *Collector) collectStorageMetrics() {
	if c.topicManager == nil {
		return
	}

	topics := c.topicManager.ListTopics()
	for _, topicName := range topics {
		topic, exists := c.topicManager.GetTopic(topicName)
		if !exists {
			continue
		}

		numPartitions := topic.NumPartitions()
		for partitionID := int32(0); partitionID < int32(numPartitions); partitionID++ {
			// Get partition size
			partitionSize, err := topic.PartitionSize(partitionID)
			if err != nil {
				c.logger.Debug("failed to get partition size",
					"topic", topicName,
					"partition", partitionID,
					"error", err)
				continue
			}

			// Get high water mark (log end offset)
			hwm, err := topic.HighWaterMark(partitionID)
			if err != nil {
				c.logger.Debug("failed to get high water mark",
					"topic", topicName,
					"partition", partitionID,
					"error", err)
				continue
			}

			// Get log segments count (simplified - would need access to log internals)
			// For now, estimate based on size
			segments := 1
			if partitionSize > 0 {
				segments = int(partitionSize/(100*1024*1024)) + 1
			}

			// Update storage metrics
			UpdateStorageMetrics(topicName, partitionID, partitionSize, segments, hwm, partitionSize)

			// Update replication metrics
			c.collectReplicationMetrics(topic, topicName, partitionID, hwm)
		}
	}
}

func (c *Collector) collectReplicationMetrics(t *topic.Topic, topicName string, partitionID int32, leaderLEO int64) {
	// Get replicas
	replicas := t.GetReplicas(partitionID)
	isr := t.GetISR(partitionID)

	if replicas == nil || len(replicas) == 0 {
		return
	}

	// Update ISR and replica count
	UpdateReplicationMetrics(topicName, partitionID, 0, -1, len(isr), len(replicas))

	// Calculate lag for each follower
	for i := 1; i < len(replicas); i++ {
		followerID := replicas[i]
		followerLEO, exists := t.GetFollowerLEO(partitionID, followerID)

		var lag int64
		if exists {
			lag = leaderLEO - followerLEO
			if lag < 0 {
				lag = 0
			}
		} else {
			lag = leaderLEO
		}

		UpdateReplicationMetrics(topicName, partitionID, followerID, lag, len(isr), len(replicas))
	}
}

func (c *Collector) collectConsumerGroupMetrics() {
	if c.coordinator == nil {
		return
	}

	groups := c.coordinator.GetAllGroups()
	for groupID, groupInfo := range groups {
		group, exists := c.coordinator.GetGroup(groupID)
		if !exists {
			continue
		}

		// Update group state and member count
		memberCount := group.GetMemberCount()
		state := string(group.GetState())

		UpdateConsumerGroupMetrics(groupID, memberCount, state)

		// Calculate lag for each topic/partition
		c.collectConsumerGroupLag(groupID, groupInfo)
	}
}

func (c *Collector) collectConsumerGroupLag(groupID string, groupInfo *coordinator.GroupInfo) {
	if c.topicManager == nil {
		return
	}

	// Get topics for this group
	topics := c.coordinator.GetGroupTopics(groupID)

	for topicName, partitions := range topics {
		topic, exists := c.topicManager.GetTopic(topicName)
		if !exists {
			continue
		}

		for _, partitionID := range partitions {
			// Get committed offset
			offsetMeta, exists := c.coordinator.FetchOffset(groupID, topicName, partitionID)
			if !exists {
				continue
			}

			// Get high water mark
			hwm, err := topic.HighWaterMark(partitionID)
			if err != nil {
				continue
			}

			// Calculate lag
			lag := hwm - offsetMeta.Offset
			if lag < 0 {
				lag = 0
			}

			UpdateConsumerGroupLag(groupID, topicName, partitionID, lag)
		}
	}
}
