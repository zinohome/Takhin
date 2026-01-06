package handler

import (
	"fmt"
	"time"

	raftlib "github.com/takhin-data/takhin/pkg/raft"
	"github.com/takhin-data/takhin/pkg/storage/topic"
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

func (r *RaftBackend) AppendBatch(topicName string, partition int32, records []BatchRecord) ([]int64, error) {
	// For Raft backend, batch operations need to be implemented through Raft consensus
	// For now, fall back to individual appends
	// TODO: Implement batch consensus protocol
	offsets := make([]int64, len(records))
	for i, rec := range records {
		offset, err := r.Append(topicName, partition, rec.Key, rec.Value)
		if err != nil {
			return offsets[:i], err
		}
		offsets[i] = offset
	}
	return offsets, nil
}
