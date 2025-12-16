package handler

import "github.com/takhin-data/takhin/pkg/storage/topic"

// Backend defines the interface for handling topic operations
// This allows us to support both direct access and Raft-based replication
type Backend interface {
	// CreateTopic creates a new topic with the given number of partitions
	CreateTopic(name string, numPartitions int32) error

	// DeleteTopic deletes a topic
	DeleteTopic(name string) error

	// GetTopic retrieves a topic by name
	GetTopic(name string) (*topic.Topic, bool)

	// Append appends a message to a topic partition
	Append(topicName string, partition int32, key, value []byte) (int64, error)
}

// DirectBackend implements Backend by directly calling TopicManager
type DirectBackend struct {
	manager *topic.Manager
}

// NewDirectBackend creates a Backend that directly accesses the TopicManager
func NewDirectBackend(manager *topic.Manager) Backend {
	return &DirectBackend{manager: manager}
}

func (d *DirectBackend) CreateTopic(name string, numPartitions int32) error {
	return d.manager.CreateTopic(name, numPartitions)
}

func (d *DirectBackend) DeleteTopic(name string) error {
	return d.manager.DeleteTopic(name)
}

func (d *DirectBackend) GetTopic(name string) (*topic.Topic, bool) {
	return d.manager.GetTopic(name)
}

func (d *DirectBackend) Append(topicName string, partition int32, key, value []byte) (int64, error) {
	topic, exists := d.manager.GetTopic(topicName)
	if !exists {
		return -1, ErrTopicNotFound
	}
	return topic.Append(partition, key, value)
}
