package handler

import (
	"errors"
	"fmt"
	"time"

	raftlib "github.com/takhin-data/takhin/pkg/raft"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

var (
	// ErrTopicNotFound indicates the requested topic does not exist
	ErrTopicNotFound = errors.New("topic not found")
)

// RaftBackend implements Backend by routing operations through Raft consensus
type RaftBackend struct {
	node    *raftlib.Node
	timeout time.Duration
}

// NewRaftBackend creates a Backend that routes operations through Raft
func NewRaftBackend(node *raftlib.Node, timeout time.Duration) Backend {
	return &RaftBackend{
		node:    node,
		timeout: timeout,
	}
}

func (r *RaftBackend) CreateTopic(name string, numPartitions int32) error {
	return r.node.CreateTopic(name, numPartitions, r.timeout)
}

func (r *RaftBackend) DeleteTopic(name string) error {
	return r.node.DeleteTopic(name, r.timeout)
}

func (r *RaftBackend) GetTopic(name string) (*topic.Topic, bool) {
	// Read operations go directly to the FSM without consensus
	fsm := r.node.GetFSM()
	return fsm.TopicManager().GetTopic(name)
}

func (r *RaftBackend) Append(topicName string, partition int32, key, value []byte) (int64, error) {
	result, err := r.node.AppendMessage(topicName, partition, key, value, r.timeout)
	if err != nil {
		return -1, err
	}

	// The FSM returns the offset as interface{}
	if offset, ok := result.(int64); ok {
		return offset, nil
	}

	return -1, fmt.Errorf("unexpected result type: %T", result)
}
